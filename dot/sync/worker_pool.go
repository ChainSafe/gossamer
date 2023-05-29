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
	wg            sync.WaitGroup
	l             sync.RWMutex
	doneCh        chan struct{}
	availableCond *sync.Cond

	network     Network
	taskQueue   chan *syncTask
	workers     map[peer.ID]*peerSyncWorker
	ignorePeers map[peer.ID]struct{}
}

func newSyncWorkerPool(net Network) *syncWorkerPool {
	swp := &syncWorkerPool{
		network:     net,
		doneCh:      make(chan struct{}),
		workers:     make(map[peer.ID]*peerSyncWorker),
		taskQueue:   make(chan *syncTask),
		ignorePeers: make(map[peer.ID]struct{}),
	}

	swp.availableCond = sync.NewCond(&swp.l)
	return swp
}

func (s *syncWorkerPool) useConnectedPeers() {
	connectedPeers := s.network.AllConnectedPeers()
	if len(connectedPeers) < 1 {
		return
	}

	s.l.Lock()
	defer s.l.Unlock()
	for _, connectedPeer := range connectedPeers {
		s.newPeer(connectedPeer, false)
	}
}

func (s *syncWorkerPool) fromBlockAnnounce(who peer.ID) {
	s.l.Lock()
	defer s.l.Unlock()
	s.newPeer(who, true)
}

func (s *syncWorkerPool) newPeer(who peer.ID, isFromBlockAnnounce bool) {
	_, exists := s.ignorePeers[who]
	if exists {
		if isFromBlockAnnounce {
			delete(s.ignorePeers, who)
		} else {
			return
		}
	}

	peerSync, has := s.workers[who]
	if !has {
		peerSync = &peerSyncWorker{status: available}
		s.workers[who] = peerSync

		logger.Tracef("potential worker added, total in the pool %d", len(s.workers))
	}

	// check if the punishment is not valid
	if peerSync.status == punished && peerSync.punishedTime.Before(time.Now()) {
		s.workers[who] = &peerSyncWorker{status: available}
	}
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

// getAvailablePeer returns the very first peer available and changes
// its status from available to busy, if there is no peer avaible then
// the caller should wait for availablePeerCh
func (s *syncWorkerPool) getAvailablePeer() peer.ID {
	for peerID, peerSync := range s.workers {
		switch peerSync.status {
		case punished:
			// if the punishedTime has passed then we mark it
			// as available and notify it availability if needed
			// otherwise we keep the peer in the punishment and don't notify
			if peerSync.punishedTime.Before(time.Now()) {
				return peerID
			}
		case available:
			return peerID
		default:
		}
	}

	//could not found an available peer to dispatch
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
			//wait for ongoing requests to be finished before returning
			s.wg.Wait()
			return

		case task := <-s.taskQueue:
			// whenever a task arrives we try to find an available peer
			// if the task is directed at some peer then we will wait for
			// that peer to become available, same happens a normal task
			// arrives and there is no available peer, then we should wait
			// for someone to become free and then use it.

			s.l.Lock()
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
					s.workers[peerID] = &peerSyncWorker{status: busy}
					s.l.Unlock()

					s.wg.Add(1)
					go s.executeRequest(peerID, task)
					break
				} else {
					s.availableCond.Wait()
				}
			}
		}
	}
}

func (s *syncWorkerPool) executeRequest(who peer.ID, task *syncTask) {
	defer func() {
		s.l.Lock()
		s.workers[who] = &peerSyncWorker{status: available}
		s.l.Unlock()

		s.availableCond.Signal()
		s.wg.Done()
	}()
	request := task.request

	logger.Debugf("[EXECUTING] worker %s: block request: %s", who, request)
	response, err := s.network.DoBlockRequest(who, request)
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
