// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package sync

import (
	"container/list"
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
)

const defaultWorkerPoolCapacity = 100

var (
	ErrNoPeers     = errors.New("no peers available")
	ErrPeerIgnored = errors.New("peer ignored")
)

type TaskID string
type Result any

type Task interface {
	ID() TaskID
	Do(p peer.ID) (Result, error)
	String() string
}

type TaskResult struct {
	Task      Task
	Completed bool
	Result    Result
	Error     error
	Retries   int
	Who       peer.ID
}

func (t TaskResult) Failed() bool {
	return t.Error != nil
}

type BatchStatus struct {
	Failed  map[TaskID]TaskResult
	Success map[TaskID]TaskResult
}

func (bs BatchStatus) Completed(todo int) bool {
	if len(bs.Failed)+len(bs.Success) < todo {
		return false
	}

	for _, tr := range bs.Failed {
		if !tr.Completed {
			return false
		}
	}

	for _, tr := range bs.Success {
		if !tr.Completed {
			return false
		}
	}

	return true
}

type BatchID string

type WorkerPool interface {
	SubmitBatch(tasks []Task) (id BatchID, err error)
	GetBatch(id BatchID) (status BatchStatus, ok bool)
	Results() chan TaskResult
	AddPeer(p peer.ID) error
	RemovePeer(p peer.ID)
	IgnorePeer(p peer.ID)
	NumPeers() int
	Shutdown()
}

type WorkerPoolConfig struct {
	Capacity   int
	MaxRetries int
}

// NewWorkerPool creates a new worker pool with the given configuration.
func NewWorkerPool(cfg WorkerPoolConfig) WorkerPool {
	ctx, cancel := context.WithCancel(context.Background())

	if cfg.Capacity <= 0 {
		cfg.Capacity = defaultWorkerPoolCapacity
	}

	return &workerPool{
		maxRetries:   cfg.MaxRetries,
		ignoredPeers: make(map[peer.ID]struct{}),
		statuses:     make(map[BatchID]BatchStatus),
		resChan:      make(chan TaskResult, cfg.Capacity),
		ctx:          ctx,
		cancel:       cancel,
	}
}

type workerPool struct {
	mtx sync.RWMutex
	wg  sync.WaitGroup

	maxRetries   int
	peers        list.List
	ignoredPeers map[peer.ID]struct{}
	statuses     map[BatchID]BatchStatus
	resChan      chan TaskResult
	ctx          context.Context
	cancel       context.CancelFunc
}

// SubmitBatch accepts a list of tasks and immediately returns a batch ID. The batch ID can be used to query the status
// of the batch using [GetBatchStatus].
// TODO
// If tasks are submitted faster than they are completed, resChan will run full, blocking the calling goroutine.
// Ideally this method would provide backpressure to the caller in that case. The rejected tasks should then stay in
// FullSyncStrategy.requestQueue until the next round. But this would need to be supported in all sync strategies.
func (w *workerPool) SubmitBatch(tasks []Task) (id BatchID, err error) {
	w.mtx.Lock()
	defer w.mtx.Unlock()

	bID := BatchID(fmt.Sprintf("%d", time.Now().UnixNano()))

	w.statuses[bID] = BatchStatus{
		Failed:  make(map[TaskID]TaskResult),
		Success: make(map[TaskID]TaskResult),
	}

	w.wg.Add(1)
	go func() {
		defer w.wg.Done()
		w.executeBatch(tasks, bID)
	}()

	return bID, nil
}

// GetBatch returns the status of a batch previously submitted using [SubmitBatch].
func (w *workerPool) GetBatch(id BatchID) (status BatchStatus, ok bool) {
	w.mtx.RLock()
	defer w.mtx.RUnlock()

	status, ok = w.statuses[id]
	return
}

// Results returns a channel that can be used to receive the results of completed tasks.
func (w *workerPool) Results() chan TaskResult {
	return w.resChan
}

// AddPeer adds a peer to the worker pool unless it has been ignored previously.
func (w *workerPool) AddPeer(who peer.ID) error {
	w.mtx.Lock()
	defer w.mtx.Unlock()

	if _, ok := w.ignoredPeers[who]; ok {
		return ErrPeerIgnored
	}

	for e := w.peers.Front(); e != nil; e = e.Next() {
		if e.Value.(peer.ID) == who {
			return nil
		}
	}

	w.peers.PushBack(who)
	logger.Tracef("peer added, total in the pool %d", w.peers.Len())
	return nil
}

// RemovePeer removes a peer from the worker pool.
func (w *workerPool) RemovePeer(who peer.ID) {
	w.mtx.Lock()
	defer w.mtx.Unlock()

	w.removePeer(who)
}

// IgnorePeer removes a peer from the worker pool and prevents it from being added again.
func (w *workerPool) IgnorePeer(who peer.ID) {
	w.mtx.Lock()
	defer w.mtx.Unlock()

	w.removePeer(who)
	w.ignoredPeers[who] = struct{}{}
}

// NumPeers returns the number of peers in the worker pool, both busy and free.
func (w *workerPool) NumPeers() int {
	w.mtx.RLock()
	defer w.mtx.RUnlock()

	return w.peers.Len()
}

// Shutdown stops the worker pool and waits for all tasks to complete.
func (w *workerPool) Shutdown() {
	w.cancel()
	w.wg.Wait()
}

func (w *workerPool) executeBatch(tasks []Task, bID BatchID) {
	batchResults := make(chan TaskResult, len(tasks))

	for _, t := range tasks {
		w.wg.Add(1)
		go func(t Task) {
			defer w.wg.Done()
			w.executeTask(t, batchResults)
		}(t)
	}

	for {
		select {
		case <-w.ctx.Done():
			return

		case tr := <-batchResults:
			if tr.Failed() {
				w.handleFailedTask(tr, bID, batchResults)
			} else {
				w.handleSuccessfulTask(tr, bID)
			}

			if w.batchCompleted(bID, len(tasks)) {
				return
			}
		}
	}
}

func (w *workerPool) executeTask(task Task, ch chan TaskResult) {
	if errors.Is(w.ctx.Err(), context.Canceled) {
		logger.Tracef("[CANCELED] task=%s, shutting down", task.String())
		return
	}

	who, err := w.reservePeer()
	if errors.Is(err, ErrNoPeers) {
		logger.Tracef("no peers available for task=%s", task.String())
		ch <- TaskResult{Task: task, Error: ErrNoPeers}
		return
	}

	logger.Infof("[EXECUTING] task=%s", task.String())

	result, err := task.Do(who)
	if err != nil {
		logger.Tracef("[FAILED] task=%s peer=%s, err=%s", task.String(), who, err.Error())
	} else {
		logger.Tracef("[FINISHED] task=%s peer=%s", task.String(), who)
	}

	w.mtx.Lock()
	w.peers.PushBack(who)
	w.mtx.Unlock()

	ch <- TaskResult{
		Task:    task,
		Who:     who,
		Result:  result,
		Error:   err,
		Retries: 0,
	}
}

func (w *workerPool) reservePeer() (who peer.ID, err error) {
	w.mtx.Lock()
	defer w.mtx.Unlock()

	peerElement := w.peers.Front()

	if peerElement == nil {
		return who, ErrNoPeers
	}

	w.peers.Remove(peerElement)
	return peerElement.Value.(peer.ID), nil
}

func (w *workerPool) removePeer(who peer.ID) {
	var toRemove *list.Element
	for e := w.peers.Front(); e != nil; e = e.Next() {
		if e.Value.(peer.ID) == who {
			toRemove = e
			break
		}
	}

	if toRemove != nil {
		w.peers.Remove(toRemove)
	}
}

func (w *workerPool) handleSuccessfulTask(tr TaskResult, batchID BatchID) {
	w.mtx.Lock()
	defer w.mtx.Unlock()

	tID := tr.Task.ID()

	if failedTr, ok := w.statuses[batchID].Failed[tID]; ok {
		tr.Retries = failedTr.Retries + 1
		delete(w.statuses[batchID].Failed, tID)
	}

	tr.Completed = true
	w.statuses[batchID].Success[tID] = tr
	logger.Infof("handleSuccessfulTask(): len(w.resChan)=%d", len(w.resChan)) // TODO: remove
	w.resChan <- tr
}

func (w *workerPool) handleFailedTask(tr TaskResult, batchID BatchID, batchResults chan TaskResult) {
	w.mtx.Lock()
	defer w.mtx.Unlock()

	tID := tr.Task.ID()

	if oldTr, ok := w.statuses[batchID].Failed[tID]; ok {
		// It is only considered a retry if the task was actually executed.
		if errors.Is(oldTr.Error, ErrNoPeers) {
			// TODO Should we sleep a bit to wait for peers?
		} else {
			tr.Retries = oldTr.Retries + 1
			tr.Completed = tr.Retries >= w.maxRetries
		}
	}

	w.statuses[batchID].Failed[tID] = tr

	if tr.Completed {
		logger.Infof("handleFailedTask(): len(w.resChan)=%d", len(w.resChan)) // TODO: remove
		w.resChan <- tr
		return
	}

	// retry task
	w.wg.Add(1)
	go func() {
		defer w.wg.Done()
		w.executeTask(tr.Task, batchResults)
	}()
}

func (w *workerPool) batchCompleted(id BatchID, todo int) bool {
	w.mtx.Lock()
	defer w.mtx.Unlock()

	b, ok := w.statuses[id]
	return !ok || b.Completed(todo)
}
