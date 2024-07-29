// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package sync

import (
	"errors"
	"sync"
	"time"

	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"golang.org/x/exp/maps"
)

var ErrNoPeersToMakeRequest = errors.New("no peers to make requests")

const (
	punishmentBaseTimeout      = 5 * time.Minute
	maxRequestsAllowed    uint = 60
)

type syncTask struct {
	requestMaker network.RequestMaker
	request      network.Message
	response     network.ResponseMessage
}

type syncTaskResult struct {
	who      peer.ID
	err      error
	request  network.Message
	response network.ResponseMessage
}

type syncWorkerPool struct {
	mtx sync.RWMutex

	network     Network
	workers     map[peer.ID]struct{}
	ignorePeers map[peer.ID]struct{}

	sharedGuard chan struct{}
}

func newSyncWorkerPool(net Network) *syncWorkerPool {
	swp := &syncWorkerPool{
		network:     net,
		workers:     make(map[peer.ID]struct{}),
		ignorePeers: make(map[peer.ID]struct{}),
		sharedGuard: make(chan struct{}, maxRequestsAllowed),
	}

	return swp
}

// fromBlockAnnounceHandshake stores the peer which send us a handshake as
// a possible source for requesting blocks/state/warp proofs
func (s *syncWorkerPool) fromBlockAnnounceHandshake(who peer.ID) {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	if _, ok := s.ignorePeers[who]; ok {
		return
	}

	_, has := s.workers[who]
	if has {
		return
	}

	s.workers[who] = struct{}{}
	logger.Tracef("potential worker added, total in the pool %d", len(s.workers))
}

// submitRequests takes an set of requests and will submit to the pool through submitRequest
// the response will be dispatch in the resultCh
func (s *syncWorkerPool) submitRequests(tasks []*syncTask) ([]*syncTaskResult, error) {
	peers := s.network.AllConnectedPeersIDs()
	connectedPeers := make(map[peer.ID]struct{}, len(peers))
	for _, peer := range peers {
		connectedPeers[peer] = struct{}{}
	}

	s.mtx.RLock()
	defer s.mtx.RUnlock()

	wg := sync.WaitGroup{}
	resCh := make(chan *syncTaskResult, len(tasks))

	for pid, w := range s.workers {
		_, ok := connectedPeers[pid]
		if ok {
			continue
		}
		connectedPeers[pid] = w
	}

	allWorkers := maps.Keys(connectedPeers)
	if len(allWorkers) == 0 {
		return nil, ErrNoPeersToMakeRequest
	}

	guard := make(chan struct{}, len(allWorkers))
	for idx, task := range tasks {
		guard <- struct{}{}

		workerID := idx % len(allWorkers)
		worker := allWorkers[workerID]

		wg.Add(1)
		go executeRequest(&wg, worker, task, guard, resCh)
	}

	go func() {
		wg.Wait()
		close(resCh)
	}()

	results := make([]*syncTaskResult, 0)
	for r := range resCh {
		results = append(results, r)
	}

	return results, nil
}

func (s *syncWorkerPool) ignorePeerAsWorker(who peer.ID) {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	delete(s.workers, who)
	s.ignorePeers[who] = struct{}{}
}

func (s *syncWorkerPool) removeWorker(who peer.ID) {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	delete(s.workers, who)
}

// totalWorkers only returns available or busy workers
func (s *syncWorkerPool) totalWorkers() (total int) {
	s.mtx.RLock()
	defer s.mtx.RUnlock()

	return len(s.workers)
}
