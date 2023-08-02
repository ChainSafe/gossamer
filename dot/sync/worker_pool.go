// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package sync

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"golang.org/x/exp/maps"
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
	timesPunished int
	worker        *worker
}

func (p *peerSyncWorker) isPunished() bool {
	return p.worker.isPunished()
}

type syncWorkerPool struct {
	mtx sync.RWMutex

	network      Network
	requestMaker network.RequestMaker
	workers      map[peer.ID]*peerSyncWorker
	ignorePeers  map[peer.ID]struct{}

	sharedGuard chan struct{}
}

func newSyncWorkerPool(net Network, requestMaker network.RequestMaker) *syncWorkerPool {
	swp := &syncWorkerPool{
		network:      net,
		requestMaker: requestMaker,
		workers:      make(map[peer.ID]*peerSyncWorker),
		ignorePeers:  make(map[peer.ID]struct{}),
		sharedGuard:  make(chan struct{}, maxRequestsAllowed),
	}

	return swp
}

// stop will shutdown all the available workers goroutines
func (s *syncWorkerPool) stop() error {
	s.mtx.RLock()
	defer s.mtx.RUnlock()

	wg := sync.WaitGroup{}
	// make it buffered so the goroutines can write on it
	// without beign blocked
	errCh := make(chan error, len(s.workers))

	for _, syncWorker := range s.workers {
		if syncWorker.isPunished() {
			continue
		}

		wg.Add(1)
		go func(syncWorker *peerSyncWorker, wg *sync.WaitGroup) {
			defer wg.Done()
			errCh <- syncWorker.worker.stop()
		}(syncWorker, &wg)
	}

	wg.Wait()
	// closing the errCh then the following for loop don't
	// panic due to "all goroutines are asleep - deadlock"
	close(errCh)

	var errs error
	for err := range errCh {
		if err != nil {
			errs = errors.Join(errs, err)
		}
	}
	return errs
}

// useConnectedPeers will retrieve all connected peers
// through the network layer and use them as sources of blocks
func (s *syncWorkerPool) useConnectedPeers() {
	connectedPeers := s.network.AllConnectedPeersIDs()
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

	_, has := s.workers[who]
	if has {
		return
	}

	syncWorker := &peerSyncWorker{
		worker: newWorker(who, s.sharedGuard, s.requestMaker),
	}

	syncWorker.worker.start()
	s.workers[who] = syncWorker
	logger.Tracef("potential worker added, total in the pool %d", len(s.workers))
}

// submitRequest given a request, the worker pool will get the peer given the peer.ID
// parameter or if nil the very first available worker or
// to perform the request, the response will be dispatch in the resultCh.
func (s *syncWorkerPool) submitRequest(request *network.BlockRequestMessage,
	who *peer.ID, resultCh chan<- *syncTaskResult) {

	task := &syncTask{
		request:  request,
		resultCh: resultCh,
	}

	// if the request is bounded to a specific peer then just
	// request it and sent through its queue otherwise send
	// the request in the general queue where all worker are
	// listening on
	s.mtx.RLock()
	defer s.mtx.RUnlock()

	if who != nil {
		syncWorker := s.workers[*who]
		if !syncWorker.isPunished() {
			syncWorker.worker.processTask(task)
			return
		}
	}

	for syncWorkerPeerID, syncWorker := range s.workers {
		if who != nil && *who == syncWorkerPeerID {
			continue
		}

		if syncWorker.isPunished() {
			continue
		}

		enqueued := syncWorker.worker.processTask(task)
		if enqueued {
			break
		}
	}
}

// submitRequests takes an set of requests and will submit to the pool through submitRequest
// the response will be dispatch in the resultCh
func (s *syncWorkerPool) submitRequests(requests []*network.BlockRequestMessage) (resultCh chan *syncTaskResult) {
	logger.Debugf("[SENDING] %d requests", len(requests))
	resultCh = make(chan *syncTaskResult, maxRequestsAllowed+1)

	s.mtx.RLock()
	defer s.mtx.RUnlock()

	idx := 0
	allWorkers := maps.Values(s.workers)
	for idx < len(requests) {
		workerID := idx % len(allWorkers)
		syncWorker := allWorkers[workerID]

		if syncWorker.isPunished() {
			continue
		}

		enqueued := syncWorker.worker.processTask(&syncTask{
			request:  requests[idx],
			resultCh: resultCh,
		})

		if !enqueued {
			continue
		}

		idx++
	}

	return resultCh
}

// punishPeer given a peer.ID we check increase its times punished
// and apply the punishment time using the base timeout of 5m, so
// each time a peer is punished its timeout will increase by 5m
func (s *syncWorkerPool) punishPeer(who peer.ID) {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	syncWorker, has := s.workers[who]
	if !has || syncWorker.isPunished() {
		return
	}

	timesPunished := syncWorker.timesPunished + 1
	punishmentTime := time.Duration(timesPunished) * punishmentBaseTimeout

	syncWorker.timesPunished = timesPunished
	syncWorker.worker.punish(punishmentTime)

	s.workers[who] = syncWorker
}

func (s *syncWorkerPool) ignorePeerAsWorker(who peer.ID) error {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	syncWorker, has := s.workers[who]
	if !has {
		return nil
	}

	err := syncWorker.worker.stop()
	if err != nil {
		return fmt.Errorf("stopping worker: %w", err)
	}

	delete(s.workers, who)
	s.ignorePeers[who] = struct{}{}
	return nil
}

// totalWorkers only returns available or busy workers
func (s *syncWorkerPool) totalWorkers() (total uint) {
	s.mtx.RLock()
	defer s.mtx.RUnlock()

	for _, syncWorker := range s.workers {
		if !syncWorker.worker.isPunished() {
			total++
		}
	}

	return total
}
