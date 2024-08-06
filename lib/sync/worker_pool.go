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

var (
	ErrNoPeersToMakeRequest = errors.New("no peers to make requests")
	ErrPeerIgnored          = errors.New("peer ignored")
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
	who       peer.ID
	completed bool
	request   network.Message
	response  network.ResponseMessage
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
func (s *syncWorkerPool) fromBlockAnnounceHandshake(who peer.ID) error {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	if _, ok := s.ignorePeers[who]; ok {
		return ErrPeerIgnored
	}

	_, has := s.workers[who]
	if has {
		return nil
	}

	s.workers[who] = struct{}{}
	logger.Tracef("potential worker added, total in the pool %d", len(s.workers))
	return nil
}

// submitRequests takes an set of requests and will submit to the pool through submitRequest
// the response will be dispatch in the resultCh
func (s *syncWorkerPool) submitRequests(tasks []*syncTask) ([]*syncTaskResult, error) {
	s.mtx.RLock()
	defer s.mtx.RUnlock()

	pids := maps.Keys(s.workers)
	results := make([]*syncTaskResult, 0, len(tasks))

	for _, task := range tasks {
		completed := false
		for _, pid := range pids {
			logger.Infof("[EXECUTING] worker %s", pid)
			err := task.requestMaker.Do(pid, task.request, task.response)
			if err != nil {
				logger.Infof("[ERR] worker %s, request: %s, err: %s", pid, task.request, err.Error())
				continue
			}

			completed = true
			results = append(results, &syncTaskResult{
				who:       pid,
				completed: completed,
				request:   task.request,
				response:  task.response,
			})
			logger.Infof("[FINISHED] worker %s, request: %s", pid, task.request)
			break
		}

		if !completed {
			results = append(results, &syncTaskResult{
				completed: completed,
				request:   task.request,
				response:  nil,
			})
		}
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
