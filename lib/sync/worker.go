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
	logger.Debugf("[EXECUTING] worker %s, block request: %s", who, request)
	err := task.requestMaker.Do(who, request, task.response)
	if err != nil {
		logger.Debugf("[ERR] worker %s, err: %s", who, err)
		resCh <- &syncTaskResult{
			who:      who,
			request:  request,
			err:      err,
			response: nil,
		}
		return
	}

	logger.Debugf("[FINISHED] worker %s, response: %s", who, task.response.String())
	resCh <- &syncTaskResult{
		who:      who,
		request:  request,
		response: task.response,
	}
}
