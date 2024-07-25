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

	stopCh chan struct{}
}

// newWorker creates and returns a new worker instance.
func newWorker(pID peer.ID) *worker {
	return &worker{
		peerID: pID,
		status: available,
	}
}

func executeRequest(wg *sync.WaitGroup, who *worker, task *syncTask, resCh chan<- *syncTaskResult) {
	defer func() {
		who.status = available
		wg.Done()
	}()

	request := task.request
	logger.Debugf("[EXECUTING] worker %s, block request: %s", who, request)
	err := task.requestMaker.Do(who.peerID, request, task.response)
	if err != nil {
		logger.Debugf("[ERR] worker %s, err: %s", who, err)
		resCh <- &syncTaskResult{
			who:      who.peerID,
			request:  request,
			err:      err,
			response: nil,
		}
		return
	}

	logger.Debugf("[FINISHED] worker %s, response: %s", who, task.response.String())
	resCh <- &syncTaskResult{
		who:      who.peerID,
		request:  request,
		response: task.response,
	}
}
