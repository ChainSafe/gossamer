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
)

var ErrStopTimeout = errors.New("stop timeout")

type worker struct {
	mxt         sync.Mutex
	status      byte
	peerID      peer.ID
	sharedGuard chan struct{}

	punishment chan time.Duration
	doneCh     chan struct{}

	queue        chan *syncTask
	requestMaker network.RequestMaker
}

func newWorker(pID peer.ID, sharedGuard chan struct{}, network network.RequestMaker) *worker {
	return &worker{
		peerID:       pID,
		sharedGuard:  sharedGuard,
		punishment:   make(chan time.Duration),
		doneCh:       make(chan struct{}),
		queue:        make(chan *syncTask, maxRequestsAllowed),
		requestMaker: network,
		status:       available,
	}
}

func (w *worker) start() {
	defer func() {
		logger.Debugf("[STOPPED] worker %s", w.peerID)
		close(w.doneCh)
	}()

	logger.Debugf("[STARTED] worker %s", w.peerID)
	for task := range w.queue {
		executeRequest(w.peerID, w.requestMaker, task, w.sharedGuard)
	}
}

func (w *worker) processTask(task *syncTask) {
	w.mxt.Lock()
	defer w.mxt.Unlock()
	if w.queue != nil {
		select {
		case w.queue <- task:
		default:
			panic(fmt.Sprintf("worker %s queue is blocked, cannot enqueue task: %s",
				w.peerID, task.request.String()))
		}
	}
}

func (w *worker) stop() error {
	w.mxt.Lock()
	close(w.queue)
	w.queue = nil
	w.mxt.Unlock()

	timeoutTimer := time.NewTimer(30 * time.Second)
	select {
	case <-w.doneCh:
		if !timeoutTimer.Stop() {
			<-timeoutTimer.C
		}

		return nil
	case <-timeoutTimer.C:
		return fmt.Errorf("%w: worker %s", ErrStopTimeout, w.peerID)
	}
}

func executeRequest(who peer.ID, requestMaker network.RequestMaker,
	task *syncTask, sharedGuard chan struct{}) {
	defer func() {
		<-sharedGuard
	}()

	sharedGuard <- struct{}{}

	request := task.request
	logger.Debugf("[EXECUTING] worker %s, block request: %s", who, request)
	response := new(network.BlockResponseMessage)
	err := requestMaker.Do(who, request, response)

	task.resultCh <- &syncTaskResult{
		who:      who,
		request:  request,
		response: response,
		err:      err,
	}

	logger.Debugf("[FINISHED] worker %s, err: %s, block data amount: %d", who, err, len(response.BlockData))
}
