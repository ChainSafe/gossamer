package sync

import (
	"context"
	"errors"
	"math/big"
	"sync"

	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/libp2p/go-libp2p/core/peer"
	"golang.org/x/exp/maps"
)

type syncTask struct {
	request  *network.BlockRequestMessage
	resultCh chan<- *syncTaskResult
}

type syncWorkerPool struct {
	ctx context.Context
	l   sync.RWMutex
	wg  sync.WaitGroup

	network   Network
	taskQueue chan *syncTask
	workers   map[peer.ID]*syncWorker
}

func newSyncWorkerPool(net Network) *syncWorkerPool {
	return &syncWorkerPool{
		network:   net,
		workers:   make(map[peer.ID]*syncWorker),
		taskQueue: make(chan *syncTask),
	}
}

func (s *syncWorkerPool) useConnectedPeers() {
	connectedPeers := s.network.TotalConnectedPeers()

	s.l.Lock()
	defer s.l.Unlock()

	for _, connectedPeer := range connectedPeers {
		_, has := s.workers[connectedPeer]
		if has {
			continue
		}

		// they are ephemeral because once we reach the tip we
		// should remove them and use only peers who send us
		// block announcements
		ephemeralSyncWorker := newSyncWorker(s.ctx, connectedPeer, common.Hash{}, 0, s.network)
		ephemeralSyncWorker.isEphemeral = true
		ephemeralSyncWorker.Start(s.taskQueue, &s.wg)
		s.workers[connectedPeer] = ephemeralSyncWorker
	}
}

func (s *syncWorkerPool) addWorker(who peer.ID, bestHash common.Hash, bestNumber uint) error {
	s.l.Lock()
	defer s.l.Unlock()

	worker, has := s.workers[who]
	if has {
		worker.update(bestHash, bestNumber)
		return nil
	}

	syncWorker := newSyncWorker(s.ctx, who, bestHash, bestNumber, s.network)
	logger.Tracef("potential worker added, total in the pool %d", len(s.workers))

	syncWorker.Start(s.taskQueue, &s.wg)
	s.workers[who] = syncWorker
	return nil
}

func (s *syncWorkerPool) submitRequest(request *network.BlockRequestMessage, resultCh chan<- *syncTaskResult) {
	s.taskQueue <- &syncTask{
		request:  request,
		resultCh: resultCh,
	}
}

func (s *syncWorkerPool) submitRequests(requests []*network.BlockRequestMessage, resultCh chan<- *syncTaskResult) {
	for _, request := range requests {
		s.submitRequest(request, resultCh)
	}
}

func (s *syncWorkerPool) shutdownWorker(who peer.ID) {
	s.l.Lock()
	defer s.l.Unlock()

	peer, has := s.workers[who]
	if !has {
		return
	}

	peer.Stop()
	delete(s.workers, who)
}

func (s *syncWorkerPool) totalWorkers() (total uint) {
	s.l.RLock()
	defer s.l.RUnlock()

	total = 0
	for range s.workers {
		total++
	}

	return total
}

// getTargetBlockNumber takes the average of all peer heads
// TODO: should we just return the highest? could be an attack vector potentially, if a peer reports some very large
// head block number, it would leave us in bootstrap mode forever
// it would be better to have some sort of standard deviation calculation and discard any outliers (#1861)
func (s *syncWorkerPool) getTargetBlockNumber() (uint, error) {
	s.l.RLock()
	activeWorkers := maps.Values(s.workers)
	s.l.RUnlock()

	// in practice, this shouldn't happen, as we only start the module once we have some peer states
	if len(activeWorkers) == 0 {
		// return max uint32 instead of 0, as returning 0 would switch us to tip mode unexpectedly
		return 0, errors.New("no active workers yet")
	}

	// we are going to sort the data and remove the outliers then we will return the avg of all the valid elements
	blockNumbers := make([]uint, 0, len(activeWorkers))
	for _, worker := range activeWorkers {
		// we don't count ephemeral workers since they don't have
		// a best block hash/number informations, they are connected peers
		// who can help us sync blocks faster
		if worker.isEphemeral {
			continue
		}

		blockNumbers = append(blockNumbers, worker.bestNumber)
	}

	if len(blockNumbers) < 1 {
		return 0, errors.New("no active workers yet")
	}

	sum, count := nonOutliersSumCount(blockNumbers)
	quotientBigInt := big.NewInt(0).Div(sum, big.NewInt(int64(count)))
	return uint(quotientBigInt.Uint64()), nil
}
