package sync

import (
	"context"
	"sync"
	"time"

	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/libp2p/go-libp2p/core/peer"
)

type syncTask struct {
	request  *network.BlockRequestMessage
	resultCh chan<- *syncTaskResult
}

type syncTaskResult struct {
	who      peer.ID
	request  *network.BlockRequestMessage
	response *network.BlockResponseMessage
	err      error
}

type syncWorkerPool struct {
	ctx context.Context
	l   sync.RWMutex
	wg  sync.WaitGroup

	network   Network
	taskQueue chan *syncTask
	workers   map[peer.ID]*syncWorker

	// TODO add this worker in a ignorePeers list, implement some expiration time for
	// peers added to it (peerJail where peers have a release date and maybe extend the punishment
	// if fail again ang again Jimmy's + Diego's idea)
	ignorePeers map[peer.ID]time.Time
}

const maxRequestAllowed uint = 40

func newSyncWorkerPool(net Network) *syncWorkerPool {
	return &syncWorkerPool{
		network:     net,
		workers:     make(map[peer.ID]*syncWorker),
		taskQueue:   make(chan *syncTask, maxRequestAllowed+1),
		ignorePeers: make(map[peer.ID]time.Time),
	}
}

const ignorePeerTimeout = 2 * time.Minute

func (s *syncWorkerPool) useConnectedPeers() {
	connectedPeers := s.network.AllConnectedPeers()

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

		worker := newSyncWorker(s.ctx, connectedPeer, s.network)
		worker.Start(s.taskQueue, &s.wg)
		s.workers[connectedPeer] = worker
	}
}

func (s *syncWorkerPool) addWorkerFromBlockAnnounce(who peer.ID) error {
	s.l.Lock()
	defer s.l.Unlock()

	_, has := s.ignorePeers[who]
	if has {
		delete(s.ignorePeers, who)
	}

	_, has = s.workers[who]
	if has {
		return nil
	}

	syncWorker := newSyncWorker(s.ctx, who, s.network)
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

	go func() {
		logger.Warnf("trying to stop %s (ignore=%v)", who, ignore)
		peer.Stop()
		logger.Warnf("peer %s stopped", who)
	}()

	delete(s.workers, who)

	if ignore {
		ignorePeerTimeout := time.Now().Add(ignorePeerTimeout)
		s.ignorePeers[who] = ignorePeerTimeout
	}
}

func (s *syncWorkerPool) totalWorkers() (total uint) {
	s.l.RLock()
	defer s.l.RUnlock()
	return uint(len(s.workers))
}
