package sync

import (
	"errors"
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
	maxRequestsAllowed uint = 40
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
	wg     sync.WaitGroup
	l      sync.RWMutex
	doneCh chan struct{}

	network     Network
	taskQueue   chan *syncTask
	workers     map[peer.ID]*peerSyncWorker
	ignorePeers map[peer.ID]struct{}

	waiting         bool
	availablePeerCh chan peer.ID

	waitingBounded   *peer.ID
	availableBounded chan struct{}
	waitBoundedLock  sync.Mutex
}

func newSyncWorkerPool(net Network) *syncWorkerPool {
	return &syncWorkerPool{
		network:          net,
		waiting:          false,
		doneCh:           make(chan struct{}),
		availablePeerCh:  make(chan peer.ID),
		availableBounded: make(chan struct{}),
		workers:          make(map[peer.ID]*peerSyncWorker),
		taskQueue:        make(chan *syncTask, maxRequestsAllowed),
		ignorePeers:      make(map[peer.ID]struct{}),
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

	_, toIgnore := s.ignorePeers[who]
	if toIgnore {
		delete(s.workers, who)
		return
	}

	peerSync, has := s.workers[who]
	if !has {
		peerSync = &peerSyncWorker{status: available}
	}

	// if the punishment is still valid we do nothing
	if peerSync.status == punished && peerSync.punishedTime.After(time.Now()) {
		return
	}

	if s.waitingBounded != nil && *s.waitingBounded == who {
		s.waitingBounded = nil
		s.workers[who] = &peerSyncWorker{status: busy}
		s.availableBounded <- struct{}{}
		return
	}

	s.workers[who] = &peerSyncWorker{status: available}

	if s.waiting {
		s.waiting = false
		s.availablePeerCh <- who
	}
}

func (s *syncWorkerPool) punishPeer(who peer.ID, ignore bool) {
	s.l.Lock()
	defer s.l.Unlock()

	if ignore {
		s.ignorePeers[who] = struct{}{}
		delete(s.workers, who)
		return
	}

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

var errPeerNotFound = errors.New("peer not found")
var errNoPeersAvailable = errors.New("no peers available")

// getAvailablePeer returns the very first peer available and changes
// its status from available to busy, if there is no peer avaible then
// the caller should wait for availablePeerCh
func (s *syncWorkerPool) searchForAvailable() (peer.ID, error) {
	s.l.RLock()
	defer s.l.RUnlock()

	for peerID, peerSync := range s.workers {
		switch peerSync.status {
		case punished:
			// if the punishedTime has passed then we mark it
			// as available and notify it availability if needed
			// otherwise we keep the peer in the punishment and don't notify
			if peerSync.punishedTime.Before(time.Now()) {
				peerSync.status = busy
				s.workers[peerID] = peerSync
				return peerID, nil
			}
		case available:
			peerSync.status = busy
			s.workers[peerID] = peerSync
			return peerID, nil
		default:
		}
	}

	s.waiting = true
	return peer.ID(""), errNoPeersAvailable //could not found an available peer to dispatch
}

func (s *syncWorkerPool) searchForExactAvailable(peerID peer.ID) (bool, error) {
	s.l.RLock()
	defer s.l.RUnlock()
	peerSync, has := s.workers[peerID]
	if !has {
		return false, errPeerNotFound
	}

	switch peerSync.status {
	case punished:
		// if the punishedTime has passed then we mark it
		// as available and notify it availability if needed
		// otherwise we keep the peer in the punishment and don't notify
		if peerSync.punishedTime.Before(time.Now()) {
			peerSync.status = busy
			s.workers[peerID] = peerSync
			return true, nil
		}
	case available:
		peerSync.status = busy
		s.workers[peerID] = peerSync
		return true, nil
	default:
	}

	return peerSync.status == available, nil
}

func (s *syncWorkerPool) waitPeerAndExecute(network Network, who peer.ID, availableBounded <-chan struct{}, task *syncTask, wg *sync.WaitGroup) {
	s.waitBoundedLock.Lock()
	s.waitingBounded = &who

	logger.Debugf("[WAITING] bounded task to peer %s in idle state: %s", who, task.request)
	<-availableBounded
	logger.Debugf("[WAITING] got the peer %s to handle task: %s", who, task)
	s.waitBoundedLock.Unlock()

	executeRequest(network, who, task, wg)
}

func (s *syncWorkerPool) listenForRequests(stopCh chan struct{}) {
	defer close(s.doneCh)
	for {
		select {
		case <-stopCh:
			//wait for ongoing requests to be finished before returning
			s.wg.Wait()
			return

		case task := <-s.taskQueue:
			var availablePeer peer.ID
			if task.boundTo != nil {
				isAvailable, err := s.searchForExactAvailable(*task.boundTo)
				if err != nil {
					logger.Errorf("while checking peer %s available: %s",
						*task.boundTo, task.request)
					continue
				}

				if isAvailable {
					availablePeer = *task.boundTo
				} else {
					s.wg.Add(1)
					go s.waitPeerAndExecute(s.network, *task.boundTo, s.availableBounded, task, &s.wg)
					continue
				}
			} else {
				var err error
				availablePeer, err = s.searchForAvailable()
				if err != nil {
					if errors.Is(err, errNoPeersAvailable) {
						logger.Debugf("[WAITING] task in idle state: %s", task.request)
						availablePeer = <-s.availablePeerCh
						logger.Debugf("[WAITING] got the peer %s to handle task: %s", availablePeer, task)
					} else {
						logger.Errorf("while searching for available peer: %s", task.request)
					}
				}
			}

			s.wg.Add(1)
			go executeRequest(s.network, availablePeer, task, &s.wg)
		}
	}
}

func executeRequest(network Network, who peer.ID, task *syncTask, wg *sync.WaitGroup) {
	defer wg.Done()
	request := task.request

	logger.Debugf("[EXECUTING] worker %s: block request: %s", who, request)
	response, err := network.DoBlockRequest(who, request)
	if err != nil {
		logger.Debugf("[FINISHED] worker %s: err: %s", who, err)
	} else if response != nil {
		logger.Debugf("[FINISHED] worker %s: block data amount: %d", who, len(response.BlockData))
	}

	task.resultCh <- &syncTaskResult{
		who:      who,
		request:  request,
		response: response,
		err:      err,
	}
}
