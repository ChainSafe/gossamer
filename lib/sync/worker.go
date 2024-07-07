// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package sync

import (
	"errors"
	"sync"

	"github.com/libp2p/go-libp2p/core/peer"
)

// ErrStopTimeout is an error indicating that the worker stop operation timed out.
var ErrStopTimeout = errors.New("stop timeout")

// worker represents a worker that processes sync tasks by making network requests to peers.
// It manages the synchronisation tasks between nodes in the Polkadot's peer-to-peer network.
// The primary goal of the worker is to handle and coordinate tasks related to network requests,
// ensuring that nodes stay synchronised with the blockchain state
type worker struct {
	// Status of the worker (e.g., available, busy, etc.)
	status byte

	// ID of the peer this worker is associated with
	peerID peer.ID

	// Channel used as a semaphore to limit concurrent tasks. By making the channel buffered with some size,
	// the creator of the channel can control how many workers can work concurrently and send requests.
	sharedGuard chan struct{}

	stopCh chan struct{}
}

// newWorker creates and returns a new worker instance.
func newWorker(pID peer.ID, sharedGuard chan struct{}, stopCh chan struct{}) *worker {
	return &worker{
		peerID:      pID,
		sharedGuard: sharedGuard,
		status:      available,
		stopCh:      stopCh,
	}
}

// run starts the worker to process tasks from the queue.
// queue: Channel from which the worker receives tasks
// wg: WaitGroup to signal when the worker has finished processing
func (w *worker) run(queue chan *syncTask, wg *sync.WaitGroup) {
	defer func() {
		logger.Debugf("[STOPPED] worker %s", w.peerID)
		wg.Done()
	}()

	for {
		select {
		case <-w.stopCh:
			return
		case task := <-queue:
			executeRequest(w.peerID, task, w.sharedGuard)
		}
	}
}

// executeRequest processes a sync task by making a network request to a peer.
// who: ID of the peer making the request
// requestMaker: Interface to make the network request
// task: Sync task to be processed
// sharedGuard: Channel used for concurrency control
func executeRequest(who peer.ID, task *syncTask, sharedGuard chan struct{}) {
	defer func() {
		<-sharedGuard // Release the semaphore slot after the request is processed
	}()

	sharedGuard <- struct{}{} // Acquire a semaphore slot before starting the request

	request := task.request
	logger.Debugf("[EXECUTING] worker %s, block request: %s", who, request)
	err := task.requestMaker.Do(who, request, task.response)
	if err != nil {
		logger.Debugf("[ERR] worker %s, err: %s", who, err)
	}

	task.resultCh <- &syncTaskResult{
		who:      who,
		request:  request,
		response: task.response,
		err:      err,
	}

	logger.Debugf("[FINISHED] worker %s, response: %s", who, task.response.String())
}
