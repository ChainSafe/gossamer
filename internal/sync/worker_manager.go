package sync

import (
	"errors"
	"fmt"
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
	importedBlocks, allWorkersComplete := w.dispatchWorkers(maps.Values(w.workersSet))
	if !allWorkersComplete {
		blockDataFromRetries := w.resolveRetries()
		if len(importedBlocks) > 0 {
			importedBlocks = append(importedBlocks, blockDataFromRetries...)
		}
	}

	return importedBlocks
}

func (w *workerManager) dispatchWorkers(workers []*syncerWorkerByNumber) (results []*types.BlockData, allWorkersComplete bool) {
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
				availableWorker := w.nextAvailableWorker(workers)
				logger.Warnf("found!!")

				transferErr := w.transferRequestTo(worker, availableWorker)
				if transferErr != nil {
					return
				}
			}

			blockDataCh <- blockDataResponse
		}(wg, worker, blockDataCh)
	}

	wg.Wait()

	expectedTotalSyncedBlocks := 128 * len(workers)
	importedBlocks := make([]*types.BlockData, 0, expectedTotalSyncedBlocks)

	allWorkersComplete = true
	for _, workerCh := range workersBlockDataCh {
		blockDataFromWorker := <-workerCh
		if len(blockDataFromWorker) == 0 {
			allWorkersComplete = false
			continue
		}

		importedBlocks = append(importedBlocks, blockDataFromWorker...)
	}

	return importedBlocks, allWorkersComplete
}

func (w *workerManager) resolveRetries() (importedBlocksFromRetries []*types.BlockData) {
	importedBlocksFromRetries = make([]*types.BlockData, 0)

	for {
		workersToRetry := make([]*syncerWorkerByNumber, 0, len(w.workersSet))
		for _, worker := range w.workersSet {
			if worker.retryNumber > 0 && worker.retryNumber <= maxRetry {
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

var errMaxRetryReached = errors.New("max retry reached: 3")

func (w *workerManager) transferRequestTo(from *syncerWorkerByNumber, to peer.ID) error {
	w.locker.Lock()
	defer w.locker.Unlock()

	fmt.Printf("transfering request from %s to %s\n", from.peerID, to)

	retryNumber := from.retryNumber + 1
	if retryNumber > maxRetry {
		return errMaxRetryReached
	}

	w.workersSet[to] = newSyncerWorkerByNumber(
		to, from.startNumber, from.targetNumber,
		from.dataToRequest, from.direction, retryNumber, from.network)

	return nil
}

func (w *workerManager) nextAvailableWorker(workers []*syncerWorkerByNumber) peer.ID {
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
				if worker.available {
					foundAvailable <- worker
				}
			}
		}(worker, &wg)
	}

	availableWorker := <-foundAvailable
	close(doneCh)

	wg.Wait()
	return availableWorker.peerID
}
