// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package sync

import (
	"errors"
	"sync"

	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/libp2p/go-libp2p/core/peer"
)

var ErrStopTimeout = errors.New("stop timeout")

type worker struct {
	mxt          sync.Mutex
	status       byte
	peerID       peer.ID
	sharedGuard  chan struct{}
	requestMaker network.RequestMaker
}

func newWorker(pID peer.ID, sharedGuard chan struct{}, network network.RequestMaker) *worker {
	return &worker{
		peerID:       pID,
		sharedGuard:  sharedGuard,
		requestMaker: network,
		status:       available,
	}
}

func (w *worker) run(queue chan *syncTask, wg *sync.WaitGroup) {
	defer func() {
		logger.Debugf("[STOPPED] worker %s", w.peerID)
		wg.Done()
	}()

	for task := range queue {
		executeRequest(w.peerID, w.requestMaker, task, w.sharedGuard)
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
