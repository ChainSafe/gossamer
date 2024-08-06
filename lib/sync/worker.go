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

func executeRequest(wg *sync.WaitGroup, who peer.ID, task *syncTask, guard chan struct{}, resCh chan<- *syncTaskResult) {
	defer func() {
		<-guard
		wg.Done()
	}()

	request := task.request
	//logger.Infof("[EXECUTING] worker %s", who, request)
	err := task.requestMaker.Do(who, request, task.response)
	if err != nil {
		logger.Infof("[ERR] worker %s, request: %s, err: %s", who, request, err.Error())
		resCh <- &syncTaskResult{
			who:      who,
			request:  request,
			response: nil,
		}
		return
	}

	logger.Infof("[FINISHED] worker %s, request: %s", who, request)
	resCh <- &syncTaskResult{
		who:      who,
		request:  request,
		response: task.response,
	}
}
