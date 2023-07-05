// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package sync

import (
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
	punishmentBaseTimeout      = 5 * time.Minute
	maxRequestsAllowed    uint = 60
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
	status         byte
	timesPunished  int
	punishmentTime time.Time
}

type syncWorkerPool struct {
	wg            sync.WaitGroup
	mtx           sync.RWMutex
	doneCh        chan struct{}
	availableCond *sync.Cond

	network      Network
	requestMaker network.RequestMaker
	taskQueue    chan *syncTask
	workers      map[peer.ID]*peerSyncWorker
	ignorePeers  map[peer.ID]struct{}
}

func newSyncWorkerPool(net Network, requestMaker network.RequestMaker) *syncWorkerPool {
	swp := &syncWorkerPool{
		network:      net,
		requestMaker: requestMaker,
		doneCh:       make(chan struct{}),
		workers:      make(map[peer.ID]*peerSyncWorker),
		taskQueue:    make(chan *syncTask, maxRequestsAllowed+1),
		ignorePeers:  make(map[peer.ID]struct{}),
	}

	swp.availableCond = sync.NewCond(&swp.mtx)
	return swp
}

// useConnectedPeers will retrieve all connected peers
// through the network layer and use them as sources of blocks
func (s *syncWorkerPool) useConnectedPeers() {
	connectedPeers := s.network.AllConnectedPeersID()
	if len(connectedPeers) < 1 {
		return
	}

	s.mtx.Lock()
	defer s.mtx.Unlock()
	for _, connectedPeer := range connectedPeers {
		s.newPeer(connectedPeer)
	}
}

func (s *syncWorkerPool) fromBlockAnnounce(who peer.ID) {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	s.newPeer(who)
}

// newPeer a new peer will be included in the worker
// pool if it is not a peer to ignore or is not punished
func (s *syncWorkerPool) newPeer(who peer.ID) {
	if _, ok := s.ignorePeers[who]; ok {
		return
	}

	peerSync, has := s.workers[who]
	if !has {
		peerSync = &peerSyncWorker{status: available}
		s.workers[who] = peerSync

		logger.Tracef("potential worker added, total in the pool %d", len(s.workers))
	}

	// check if the punishment is not valid
	if peerSync.status == punished && peerSync.punishmentTime.Before(time.Now()) {
		peerSync.status = available
		s.workers[who] = peerSync
	}
}

// submitBoundedRequest given a request the worker pool will driven it
// to the given peer.ID, used for tip sync when we receive a block announce
// from a peer and we want to use the exact same peer to request blocks
func (s *syncWorkerPool) submitBoundedRequest(request *network.BlockRequestMessage,
	who peer.ID, resultCh chan<- *syncTaskResult) {
	s.taskQueue <- &syncTask{
		boundTo:  &who,
		request:  request,
		resultCh: resultCh,
	}
}

// submitRequest given a request the worker pool will get the very first available worker
// to perform the request, the response will be dispatch in the resultCh
func (s *syncWorkerPool) submitRequest(request *network.BlockRequestMessage, resultCh chan<- *syncTaskResult) {
	s.taskQueue <- &syncTask{
		request:  request,
		resultCh: resultCh,
	}
}

// submitRequests takes an set of requests and will submit to the pool through submitRequest
// the response will be dispatch in the resultCh
func (s *syncWorkerPool) submitRequests(requests []*network.BlockRequestMessage, resultCh chan<- *syncTaskResult) {
	for _, request := range requests {
		s.submitRequest(request, resultCh)
	}
}

// punishPeer given a peer.ID we check increase its times punished
// and apply the punishment time using the base timeout of 5m, so
// each time a peer is punished its timeout will increase by 5m
func (s *syncWorkerPool) punishPeer(who peer.ID) {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	worker, has := s.workers[who]
	if !has {
		return
	}

	timesPunished := worker.timesPunished + 1
	punishmentTime := time.Duration(timesPunished) * punishmentBaseTimeout
	logger.Debugf("⏱️ punishement time for peer %s: %.2fs", who, punishmentTime.Seconds())

	s.workers[who] = &peerSyncWorker{
		status:         punished,
		timesPunished:  timesPunished,
		punishmentTime: time.Now().Add(punishmentTime),
	}
}

func (s *syncWorkerPool) ignorePeerAsWorker(who peer.ID) {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	delete(s.workers, who)
	s.ignorePeers[who] = struct{}{}
}

// totalWorkers only returns available or busy workers
func (s *syncWorkerPool) totalWorkers() (total uint) {
	s.mtx.RLock()
	defer s.mtx.RUnlock()

	for _, worker := range s.workers {
		if worker.status == available {
			total += 1
		}
	}

	return total
}

// getAvailablePeer returns the very first peer available, if there
// is no peer avaible then the caller should wait for availablePeerCh
func (s *syncWorkerPool) getAvailablePeer() peer.ID {
	for peerID, peerSync := range s.workers {
		switch peerSync.status {
		case punished:
			// if the punishedTime has passed then we can
			// use it as an available peer
			if peerSync.punishmentTime.Before(time.Now()) {
				return peerID
			}
		case available:
			return peerID
		default:
		}
	}

	return peer.ID("")
}

func (s *syncWorkerPool) getPeerByID(peerID peer.ID) *peerSyncWorker {
	peerSync, has := s.workers[peerID]
	if !has {
		return nil
	}

	return peerSync
}

func (s *syncWorkerPool) listenForRequests(stopCh chan struct{}) {
	defer close(s.doneCh)
	for {
		select {
		case <-stopCh:
			// wait for ongoing requests to be finished before returning
			s.wg.Wait()
			return

		case task := <-s.taskQueue:
			// whenever a task arrives we try to find an available peer
			// if the task is directed at some peer then we will wait for
			// that peer to become available, same happens a normal task
			// arrives and there is no available peer, then we should wait
			// for someone to become free and then use it.

			s.mtx.Lock()
			for {
				var peerID peer.ID
				if task.boundTo != nil {
					peerSync := s.getPeerByID(*task.boundTo)
					if peerSync != nil && peerSync.status == available {
						peerID = *task.boundTo
					}
				} else {
					peerID = s.getAvailablePeer()
				}

				if peerID != peer.ID("") {
					peerSync := s.workers[peerID]
					peerSync.status = busy
					s.workers[peerID] = peerSync

					s.mtx.Unlock()

					s.wg.Add(1)
					go s.executeRequest(peerID, task)
					break
				}

				s.availableCond.Wait()
			}
		}
	}
}

func (s *syncWorkerPool) executeRequest(who peer.ID, task *syncTask) {
	defer s.wg.Done()
	request := task.request

	logger.Debugf("[EXECUTING] worker %s, block request: %s", who, request)
	response := new(network.BlockResponseMessage)
	err := s.requestMaker.Do(who, request, response)
	logger.Debugf("[FINISHED] worker %s, err: %s, block data amount: %d", who, err, len(response.BlockData))

	s.mtx.Lock()
	peerSync, has := s.workers[who]
	if has {
		peerSync.status = available
		s.workers[who] = peerSync
	}
	s.mtx.Unlock()
	s.availableCond.Signal()

	task.resultCh <- &syncTaskResult{
		who:      who,
		request:  request,
		response: response,
		err:      err,
	}
}
