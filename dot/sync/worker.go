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
	mtx         sync.Mutex
	status      byte
	peerID      peer.ID
	sharedGuard chan struct{}

	punishment chan time.Duration
	stopCh     chan struct{}
	doneCh     chan struct{}

	queue        chan *syncTask
	requestMaker network.RequestMaker
}

func newWorker(pID peer.ID, sharedGuard chan struct{}, network network.RequestMaker) *worker {
	return &worker{
		peerID:       pID,
		sharedGuard:  sharedGuard,
		punishment:   make(chan time.Duration),
		stopCh:       make(chan struct{}),
		doneCh:       make(chan struct{}),
		queue:        make(chan *syncTask, maxRequestsAllowed),
		requestMaker: network,
		status:       available,
	}
}

func (w *worker) start() {
	go func() {
		defer func() {
			logger.Debugf("[STOPPED] worker %s", w.peerID)
			close(w.doneCh)
		}()

		logger.Debugf("[STARTED] worker %s", w.peerID)
		for {
			select {
			case punishmentDuration := <-w.punishment:
				logger.Debugf("⏱️ punishement time for peer %s: %.2fs", w.peerID, punishmentDuration.Seconds())
				punishmentTimer := time.NewTimer(punishmentDuration)
				select {
				case <-punishmentTimer.C:
					w.mtx.Lock()
					w.status = available
					w.mtx.Unlock()

				case <-w.stopCh:
					return
				}

			case <-w.stopCh:
				return
			case task := <-w.queue:
				executeRequest(w.peerID, w.requestMaker, task, w.sharedGuard)
			}
		}
	}()
}

func (w *worker) processTask(task *syncTask) (enqueued bool) {
	if w.isPunished() {
		return false
	}

	select {
	case w.queue <- task:
		logger.Debugf("[ENQUEUED] worker %s, block request: %s", w.peerID, task.request)
		return true
	default:
		return false
	}
}

func (w *worker) punish(duration time.Duration) {
	w.punishment <- duration

	w.mtx.Lock()
	defer w.mtx.Unlock()
	w.status = punished
}

func (w *worker) isPunished() bool {
	w.mtx.Lock()
	defer w.mtx.Unlock()
	return w.status == punished
}

func (w *worker) stop() error {
	close(w.stopCh)

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
