package sync

import (
	"context"
	"sync"
	"time"

	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/libp2p/go-libp2p/core/peer"
)

type syncTask struct {
	request  *network.BlockRequestMessage
	resultCh chan<- *syncTaskResult
}

type syncWorkerPool struct {
	ctx context.Context
	l   sync.RWMutex
	wg  sync.WaitGroup

	network     Network
	taskQueue   chan *syncTask
	workers     map[peer.ID]*syncWorker
	ignorePeers map[peer.ID]time.Time
}

func newSyncWorkerPool(net Network) *syncWorkerPool {
	return &syncWorkerPool{
		network:     net,
		workers:     make(map[peer.ID]*syncWorker),
		taskQueue:   make(chan *syncTask),
		ignorePeers: make(map[peer.ID]time.Time),
	}
}

const ignorePeerTimeout = 2 * time.Minute

func (s *syncWorkerPool) useConnectedPeers() {
	connectedPeers := s.network.TotalConnectedPeers()

	s.l.Lock()
	defer s.l.Unlock()

	for _, connectedPeer := range connectedPeers {
		_, has := s.workers[connectedPeer]
		if has {
			continue
		}

		releaseTime, has := s.ignorePeers[connectedPeer]
		if has {
			if time.Now().Before(releaseTime) {
				continue
			} else {
				delete(s.ignorePeers, connectedPeer)
			}
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

	// delete it since it sends a block announcement so it might be
	// a valid peer to request blocks for now
	_, has := s.ignorePeers[who]
	if has {
		delete(s.ignorePeers, who)
	}

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

func (s *syncWorkerPool) shutdownWorker(who peer.ID, ignore bool) {
	s.l.Lock()
	defer s.l.Unlock()

	peer, has := s.workers[who]
	if !has {
		return
	}

	peer.Stop()
	delete(s.workers, who)

	if ignore {
		ignorePeerTimeout := time.Now().Add(ignorePeerTimeout)
		s.ignorePeers[who] = ignorePeerTimeout
	}
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
