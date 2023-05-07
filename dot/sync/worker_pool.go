package sync

import (
	"context"
	"sync"
	"time"

	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/libp2p/go-libp2p/core/peer"
)

const (
	available byte = iota
	busy
	punished
)

const (
	ignorePeerTimeout       = 2 * time.Minute
	maxRequestsAllowed uint = 45
)

type syncTask struct {
	boundTo  *peer.ID
	request  *network.BlockRequestMessage
	resultCh chan<- *syncTaskResult
}

type syncTaskResult struct {
	who      peer.ID
	request  *network.BlockRequestMessage
	response *network.BlockResponseMessage
	err      error
}

type peerSyncWorker struct {
	status       byte
	punishedTime time.Time
}

type syncWorkerPool struct {
	wg sync.WaitGroup
	l  sync.RWMutex

	network   Network
	taskQueue chan *syncTask
	workers   map[peer.ID]*peerSyncWorker

	ctx context.Context
}

func newSyncWorkerPool(ctx context.Context, net Network) *syncWorkerPool {
	return &syncWorkerPool{
		network:   net,
		workers:   make(map[peer.ID]*peerSyncWorker),
		taskQueue: make(chan *syncTask, maxRequestsAllowed),
		ctx:       ctx,
	}
}

func (s *syncWorkerPool) useConnectedPeers() {
	connectedPeers := s.network.AllConnectedPeers()
	for _, connectedPeer := range connectedPeers {
		s.releaseWorker(connectedPeer)
	}
}

func (s *syncWorkerPool) fromBlockAnnounce(who peer.ID) {
	s.releaseWorker(who)
	logger.Tracef("potential worker added, total in the pool %d", len(s.workers))
}

func (s *syncWorkerPool) submitBoundedRequest(request *network.BlockRequestMessage, who peer.ID, resultCh chan<- *syncTaskResult) {
	s.taskQueue <- &syncTask{
		boundTo:  &who,
		request:  request,
		resultCh: resultCh,
	}
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

func (s *syncWorkerPool) releaseWorker(who peer.ID) {
	s.l.Lock()
	defer s.l.Unlock()

	peerSync, has := s.workers[who]
	if !has {
		s.workers[who] = &peerSyncWorker{status: available}
	} else {
		switch peerSync.status {
		case available:
		case busy:
			s.workers[who] = &peerSyncWorker{status: available}

		case punished:
			if peerSync.punishedTime.Before(time.Now()) {
				s.workers[who] = &peerSyncWorker{status: available}
			}
		}
	}
}

func (s *syncWorkerPool) punishPeer(who peer.ID) {
	s.l.Lock()
	defer s.l.Unlock()

	_, has := s.workers[who]
	if !has {
		return
	}

	s.workers[who] = &peerSyncWorker{
		status:       punished,
		punishedTime: time.Now().Add(ignorePeerTimeout),
	}
}

func (s *syncWorkerPool) totalWorkers() (total uint) {
	s.l.RLock()
	defer s.l.RUnlock()
	return uint(len(s.workers))
}

// getFirstAvailable returns the very first peer available and changes
// its status from available to busy, if there is no peer avaible then
// it blocks until find one
func (s *syncWorkerPool) getFirstAvailable(expected *peer.ID) (peer.ID, error) {
	for {
		select {
		case <-s.ctx.Done():
			return peer.ID(""), context.Canceled
		default:
			s.l.RLock()
			defer s.l.RUnlock()

			var availablePeer peer.ID
			for peerID, peerSync := range s.workers {
				switch peerSync.status {
				case available:
					if expected != nil {
						if availablePeer != *expected {
							availablePeer = peerID
						}
					} else {
						availablePeer = peerID
					}
				case punished:
					// if the punishedTime has passed then we mark it
					// as available and notify it availability if needed
					// otherwise we keep the peer in the punishment and don't notify
					if peerSync.punishedTime.Before(time.Now()) {
						s.workers[peerID].punishedTime = time.Time{}
						return peerID, nil
					}
				}
			}
			if availablePeer != peer.ID("") {
				s.workers[availablePeer].status = busy
				return availablePeer, nil
			}
		}
	}
}

func (s *syncWorkerPool) listenForRequests(stopCh chan struct{}) {
	for {
		select {
		case <-stopCh:
			//wait for ongoing requests to be finished before returning
			s.wg.Wait()
			return

		case task := <-s.taskQueue:
			s.wg.Add(1)
			go s.executeRequest(s.network, task, &s.wg)
		}
	}
}

func (s *syncWorkerPool) executeRequest(network Network, task *syncTask, wg *sync.WaitGroup) {
	defer wg.Done()
	request := task.request

	// Blocks until it find an available peer to use
	availablePeer, err := s.getFirstAvailable(task.boundTo)
	// If we get a context canceled error we return earlier since we are shutting down the workers
	if err != nil {
		if err == context.Canceled {
			return
		}
	}

	logger.Debugf("[EXECUTING] worker for peer %s: block request: %s", availablePeer, request)
	response, err := network.DoBlockRequest(availablePeer, request)
	if err != nil {
		logger.Debugf("[FINISHED] error getting blocks from peer %s: err: %s", request, availablePeer, err)
	} else if response != nil {
		logger.Debugf("[FINISHED] success getting blocks from peer %s: block data amount: %d", availablePeer, len(response.BlockData))
	}

	task.resultCh <- &syncTaskResult{
		who:      availablePeer,
		request:  request,
		response: response,
		err:      err,
	}

	s.releaseWorker(availablePeer)
}
