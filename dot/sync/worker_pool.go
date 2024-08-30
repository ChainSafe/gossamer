// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package sync

import (
	"errors"
	"sync"
	"time"

	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/dot/network/messages"
	"github.com/libp2p/go-libp2p/core/peer"
	"golang.org/x/exp/maps"
)

var (
	ErrNoPeersToMakeRequest = errors.New("no peers to make requests")
	ErrPeerIgnored          = errors.New("peer ignored")
)

const (
	punishmentBaseTimeout      = 5 * time.Minute
	maxRequestsAllowed    uint = 3
)

type syncTask struct {
	requestMaker network.RequestMaker
	request      messages.P2PMessage
	response     messages.P2PMessage
}

type syncTaskResult struct {
	who       peer.ID
	completed bool
	request   messages.P2PMessage
	response  messages.P2PMessage
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
func (s *syncWorkerPool) submitRequests(tasks []*syncTask) []*syncTaskResult {
	if len(tasks) == 0 {
		return nil
	}

	s.mtx.RLock()
	defer s.mtx.RUnlock()

	pids := maps.Keys(s.workers)
	workerPool := make(chan peer.ID, len(pids))
	for _, worker := range pids {
		workerPool <- worker
	}

	failedTasks := make(chan *syncTask, len(tasks))
	results := make(chan *syncTaskResult, len(tasks))

	var wg sync.WaitGroup
	for _, task := range tasks {
		wg.Add(1)
		go func(t *syncTask) {
			defer wg.Done()
			executeTask(t, workerPool, failedTasks, results)
		}(task)
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		for task := range failedTasks {
			if len(workerPool) > 0 {
				wg.Add(1)
				go func(t *syncTask) {
					defer wg.Done()
					executeTask(t, workerPool, failedTasks, results)
				}(task)
			} else {
				results <- &syncTaskResult{
					completed: false,
					request:   task.request,
					response:  nil,
				}
			}
		}
	}()

	allResults := make(chan []*syncTaskResult, 1)
	wg.Add(1)
	go func(expectedResults int) {
		defer wg.Done()
		var taskResults []*syncTaskResult
		for result := range results {
			taskResults = append(taskResults, result)
			if len(taskResults) == expectedResults {
				close(failedTasks)
				break
			}
		}

		allResults <- taskResults
	}(len(tasks))

	wg.Wait()
	close(workerPool)
	close(results)

	return <-allResults
}

func executeTask(task *syncTask, workerPool chan peer.ID, failedTasks chan *syncTask, results chan *syncTaskResult) {
	worker := <-workerPool
	logger.Infof("[EXECUTING] worker %s", worker)

	err := task.requestMaker.Do(worker, task.request, task.response)
	if err != nil {
		logger.Infof("[ERR] worker %s, request: %s, err: %s", worker, task.request, err.Error())
		failedTasks <- task
	} else {
		logger.Infof("[FINISHED] worker %s, request: %s", worker, task.request)
		results <- &syncTaskResult{
			who:       worker,
			completed: true,
			request:   task.request,
			response:  task.response,
		}
	}
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
