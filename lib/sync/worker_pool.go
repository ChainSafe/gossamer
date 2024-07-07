// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package sync

import (
	"crypto/rand"
	"fmt"
	"math/big"
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
	resultCh     chan<- *syncTaskResult
}

type syncTaskResult struct {
	who      peer.ID
	request  network.Message
	response network.ResponseMessage
	err      error
}

type syncWorker struct {
	stopCh          chan struct{}
	bestBlockHash   common.Hash
	bestBlockNumber uint
	worker          *worker
	queue           chan *syncTask
}

func (s *syncWorker) stop() {

}

type syncWorkerPool struct {
	mtx sync.RWMutex
	wg  sync.WaitGroup

	network     Network
	workers     map[peer.ID]*syncWorker
	ignorePeers map[peer.ID]struct{}

	sharedGuard chan struct{}
}

func newSyncWorkerPool(net Network) *syncWorkerPool {
	swp := &syncWorkerPool{
		network:     net,
		workers:     make(map[peer.ID]*syncWorker),
		ignorePeers: make(map[peer.ID]struct{}),
		sharedGuard: make(chan struct{}, maxRequestsAllowed),
	}

	return swp
}

// stop will shutdown all the available workers goroutines
func (s *syncWorkerPool) stop() error {
	s.mtx.RLock()
	defer s.mtx.RUnlock()

	for _, sw := range s.workers {
		close(sw.queue)
	}

	allWorkersDoneCh := make(chan struct{})
	go func() {
		defer close(allWorkersDoneCh)
		s.wg.Wait()
	}()

	timeoutTimer := time.NewTimer(30 * time.Second)
	select {
	case <-timeoutTimer.C:
		return fmt.Errorf("timeout reached while finishing workers")
	case <-allWorkersDoneCh:
		if !timeoutTimer.Stop() {
			<-timeoutTimer.C
		}

		return nil
	}
}

func (s *syncWorkerPool) fromBlockAnnounceHandshake(who peer.ID, bestBlockHash common.Hash, bestBlockNumber uint) {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	if _, ok := s.ignorePeers[who]; ok {
		return
	}

	syncPeer, has := s.workers[who]
	if has {
		syncPeer.bestBlockHash = bestBlockHash
		syncPeer.bestBlockNumber = bestBlockNumber
		return
	}

	workerStopCh := make(chan struct{})
	worker := newWorker(who, s.sharedGuard, workerStopCh)
	workerQueue := make(chan *syncTask, maxRequestsAllowed)

	s.wg.Add(1)
	go worker.run(workerQueue, &s.wg)

	s.workers[who] = &syncWorker{
		worker:          worker,
		queue:           workerQueue,
		bestBlockHash:   bestBlockHash,
		bestBlockNumber: bestBlockNumber,
		stopCh:          workerStopCh,
	}
	logger.Tracef("potential worker added, total in the pool %d", len(s.workers))
}

func (s *syncWorkerPool) removeWorker(who peer.ID) {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	worker, ok := s.workers[who]
	if !ok {
		return
	}

	close(worker.stopCh)
	delete(s.workers, who)
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
		syncWorker, inMap := s.workers[*who]
		if inMap {
			if syncWorker == nil {
				panic("sync worker should not be nil")
			}
			syncWorker.queue <- task
			return
		}
	}

	// if the exact peer is not specified then
	// randomly select a worker and assign the
	// task to it, if the amount of workers is
	var selectedWorkerIdx int
	workers := maps.Values(s.workers)
	nBig, err := rand.Int(rand.Reader, big.NewInt(int64(len(workers))))
	if err != nil {
		panic(fmt.Errorf("fail to get a random number: %w", err))
	}
	selectedWorkerIdx = int(nBig.Int64())
	selectedWorker := workers[selectedWorkerIdx]
	selectedWorker.queue <- task
}

// submitRequests takes an set of requests and will submit to the pool through submitRequest
// the response will be dispatch in the resultCh
func (s *syncWorkerPool) submitRequests(tasks []*syncTask) {
	s.mtx.RLock()
	defer s.mtx.RUnlock()

	allWorkers := maps.Values(s.workers)
	for idx, task := range tasks {
		workerID := idx % len(allWorkers)
		syncWorker := allWorkers[workerID]

		syncWorker.queue <- task
	}
}

func (s *syncWorkerPool) ignorePeerAsWorker(who peer.ID) {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	worker, has := s.workers[who]
	if has {
		close(worker.queue)
		delete(s.workers, who)
		s.ignorePeers[who] = struct{}{}
	}
}

// totalWorkers only returns available or busy workers
func (s *syncWorkerPool) totalWorkers() (total int) {
	s.mtx.RLock()
	defer s.mtx.RUnlock()

	return len(s.workers)
}
