package sync

import (
	"fmt"
	"sort"
	"sync"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/libp2p/go-libp2p/core/peer"
	"golang.org/x/exp/maps"
)

const maxRetry = 3

type SyncerWorker interface {
	Stop() error
	Dispatch() error
}

type workerManager struct {
	locker     sync.RWMutex
	workersSet map[peer.ID]*syncerWorkerByNumber
}

func newWorkerManager(workers map[peer.ID]*syncerWorkerByNumber) *workerManager {
	return &workerManager{
		workersSet: workers,
	}
}

func (w *workerManager) Start() []*types.BlockData {
	currentWorkers := maps.Values(w.workersSet)
	importedBlocks, allWorkersComplete := w.dispatchWorkers(currentWorkers)
	if !allWorkersComplete {
		blockDataFromRetries := w.resolveRetries()
		if len(blockDataFromRetries) > 0 {
			importedBlocks = append(importedBlocks, blockDataFromRetries...)
		}
	}

	// loop until there is no gaps
	for {
		gaps := ensureNoGapsBetweenSyncedSets(importedBlocks)
		if len(gaps) == 0 {
			return join(importedBlocks)
		}

		bootstrapSyncerGapWorkers := make([]*syncerWorkerByNumber, len(gaps))
		for idx, gap := range gaps {
			startAt := gap[0]
			target := startAt + gap[1]

			availableWorker := nextAvailableWorker(currentWorkers)

			bootstrapSyncerGapWorkers[idx] = newSyncerWorkerByNumber(
				availableWorker.peerID, startAt, target, availableWorker.dataToRequest,
				availableWorker.direction, 0, availableWorker.network,
			)
		}

		importedGapsBlocks, allWorkersComplete := w.dispatchWorkers(bootstrapSyncerGapWorkers)
		importedBlocks = append(importedBlocks, importedGapsBlocks...)

		if !allWorkersComplete {
			importedGapsFromRetries := w.resolveRetries()
			if len(importedGapsFromRetries) > 0 {
				importedBlocks = append(importedBlocks, importedGapsFromRetries...)
			}
		}
	}
}

func join(s [][]*types.BlockData) []*types.BlockData {
	totalAmount := 0
	for _, element := range s {
		totalAmount += len(element)
	}

	result := make([]*types.BlockData, 0, totalAmount)
	for _, element := range s {
		result = append(result, element...)
	}
	return result
}

func ensureNoGapsBetweenSyncedSets(syncedSets [][]*types.BlockData) (missingBlocksAfter [][2]uint) {
	if len(syncedSets) < 2 {
		return nil
	}

	sort.Slice(syncedSets, func(i, j int) bool {
		syncedSetA := syncedSets[i]
		firstElementOfA := syncedSetA[0]
		lastElemenenOfA := syncedSetA[len(syncedSetA)-1]

		syncedSetB := syncedSets[j]
		firstElementOfB := syncedSetB[0]
		lastElemenenOfB := syncedSetB[len(syncedSetB)-1]

		lowerFirstElement := firstElementOfA.Number() < firstElementOfB.Number()
		lowerLastElement := lastElemenenOfA.Number() < lastElemenenOfB.Number()

		return lowerFirstElement && lowerLastElement
	})

	missingBlocksAfter = make([][2]uint, 0, len(syncedSets))
	previousSyncedSet := syncedSets[0]
	for _, currentSyncedSet := range syncedSets[1:] {
		lastFromPrevious := previousSyncedSet[len(previousSyncedSet)-1]
		firstFromCurrent := currentSyncedSet[0]

		equal := lastFromPrevious.Number() == firstFromCurrent.Number()
		greaterByOne := (lastFromPrevious.Number() + 1) == firstFromCurrent.Number()
		fmt.Printf("CHECKING A GAP!!!\nlast from previous: %d, first from current: %d\n",
			lastFromPrevious.Number(), firstFromCurrent.Number())
		if equal || greaterByOne {
			previousSyncedSet = currentSyncedSet
			continue
		}

		fmt.Printf("FOUND A GAP!!!\nlast from previous: %d, first from current: %d, diff: %d\n",
			lastFromPrevious.Number(), firstFromCurrent.Number(),
			firstFromCurrent.Number()-lastFromPrevious.Number())

		difference := firstFromCurrent.Number() - lastFromPrevious.Number()
		//lets exclude both nodes that are already synced

		gapStartAt := lastFromPrevious.Number()
		missingBlocksAfter = append(missingBlocksAfter, [2]uint{gapStartAt, difference})
		previousSyncedSet = currentSyncedSet
	}

	return missingBlocksAfter
}

func (w *workerManager) dispatchWorkers(workers []*syncerWorkerByNumber) (results [][]*types.BlockData, allWorkersComplete bool) {
	wg := new(sync.WaitGroup)
	workersBlockDataCh := make([]chan []*types.BlockData, 0, len(workers))

	for _, worker := range workers {
		blockDataCh := make(chan []*types.BlockData, 1)
		workersBlockDataCh = append(workersBlockDataCh, blockDataCh)
		wg.Add(1)

		go func(wg *sync.WaitGroup, worker *syncerWorkerByNumber, blockDataCh chan []*types.BlockData) {
			defer func() {
				wg.Done()
				close(blockDataCh)
			}()

			blockDataResponse, err := worker.Dispatch()
			if err != nil {
				logger.Errorf("worker error: %s", err)

				logger.Warnf("waiting for an available worker...")
				availableWorker := nextAvailableWorker(workers)
				logger.Warnf("found!!")

				transferErr := w.transferRequestTo(worker, availableWorker.peerID)
				if transferErr != nil {
					return
				}
			}

			blockDataCh <- blockDataResponse
		}(wg, worker, blockDataCh)
	}

	wg.Wait()

	blocksDataSetPerWorker := make([][]*types.BlockData, 0, len(workers))

	allWorkersComplete = true
	for _, workerCh := range workersBlockDataCh {
		blockDataFromWorker := <-workerCh
		if len(blockDataFromWorker) == 0 {
			allWorkersComplete = false
			continue
		}

		blocksDataSetPerWorker = append(blocksDataSetPerWorker, blockDataFromWorker)
	}

	return blocksDataSetPerWorker, allWorkersComplete
}

func (w *workerManager) resolveRetries() (importedBlocksFromRetries [][]*types.BlockData) {
	importedBlocksFromRetries = make([][]*types.BlockData, 0)

	for {
		workersToRetry := make([]*syncerWorkerByNumber, 0, len(w.workersSet))
		for _, worker := range w.workersSet {
			if worker.retryNumber > 0 && worker.retryNumber <= maxRetry {
				worker.done = make(chan struct{})
				workersToRetry = append(workersToRetry, worker)
			}
		}

		if len(workersToRetry) < 1 {
			return importedBlocksFromRetries
		}

		fmt.Printf("resolving retries %d\n", len(workersToRetry))

		blocksFromWorkers, allWorkersComplete := w.dispatchWorkers(workersToRetry)
		if allWorkersComplete {
			return blocksFromWorkers
		} else {
			importedBlocksFromRetries = append(importedBlocksFromRetries, blocksFromWorkers...)
		}
	}
}

func (w *workerManager) transferRequestTo(from *syncerWorkerByNumber, to peer.ID) error {
	w.locker.Lock()
	defer w.locker.Unlock()

	fmt.Printf("transfering request from %s to %s\n", from.peerID, to)

	w.workersSet[to] = newSyncerWorkerByNumber(
		to, from.startNumber, from.targetNumber,
		from.dataToRequest, from.direction, from.retryNumber+1, from.network)

	return nil
}

func nextAvailableWorker(workers []*syncerWorkerByNumber) *syncerWorkerByNumber {
	foundAvailable := make(chan *syncerWorkerByNumber, len(workers))
	defer close(foundAvailable)

	doneCh := make(chan struct{})

	wg := sync.WaitGroup{}
	for _, worker := range workers {
		wg.Add(1)
		go func(worker *syncerWorkerByNumber, wg *sync.WaitGroup) {
			defer wg.Done()

			select {
			case <-doneCh:
				return
			case <-worker.done:
				if worker.available && worker.retryNumber <= maxRetry {
					foundAvailable <- worker
				}
			}
		}(worker, &wg)
	}

	availableWorker := <-foundAvailable
	close(doneCh)

	wg.Wait()
	return availableWorker
}
