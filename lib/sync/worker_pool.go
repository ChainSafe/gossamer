// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package sync

import (
	"sync"
	"time"

	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/lib/common"
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
	wg  sync.WaitGroup

	network     Network
	workers     map[peer.ID]*worker
	ignorePeers map[peer.ID]struct{}

	sharedGuard chan struct{}
}

func newSyncWorkerPool(net Network) *syncWorkerPool {
	swp := &syncWorkerPool{
		network:     net,
		workers:     make(map[peer.ID]*worker),
		ignorePeers: make(map[peer.ID]struct{}),
		sharedGuard: make(chan struct{}, maxRequestsAllowed),
	}

	return swp
}

func (s *syncWorkerPool) fromBlockAnnounceHandshake(who peer.ID, bestBlockHash common.Hash, bestBlockNumber uint) {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	if _, ok := s.ignorePeers[who]; ok {
		return
	}

	_, has := s.workers[who]
	if has {
		return
	}

	s.workers[who] = newWorker(who)
	logger.Tracef("potential worker added, total in the pool %d", len(s.workers))
}

func (s *syncWorkerPool) removeWorker(who peer.ID) {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	delete(s.workers, who)
}

// submitRequests takes an set of requests and will submit to the pool through submitRequest
// the response will be dispatch in the resultCh
func (s *syncWorkerPool) submitRequests(tasks []*syncTask) []*syncTaskResult {
	s.mtx.RLock()
	defer s.mtx.RUnlock()

	wg := sync.WaitGroup{}
	resCh := make(chan *syncTaskResult, len(tasks))

	allWorkers := maps.Values(s.workers)
	for idx, task := range tasks {
		workerID := idx % len(allWorkers)
		worker := allWorkers[workerID]
		if worker.status != available {
			continue
		}

		worker.status = busy
		wg.Add(1)
		go executeRequest(&wg, worker, task, resCh)
	}

	go func() {
		wg.Wait()
		close(resCh)
	}()

	results := make([]*syncTaskResult, 0)
	for r := range resCh {
		results = append(results, r)
	}

	return results
}

func (s *syncWorkerPool) ignorePeerAsWorker(who peer.ID) {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	delete(s.workers, who)
	s.ignorePeers[who] = struct{}{}
}

// totalWorkers only returns available or busy workers
func (s *syncWorkerPool) totalWorkers() (total int) {
	s.mtx.RLock()
	defer s.mtx.RUnlock()

	return len(s.workers)
}
