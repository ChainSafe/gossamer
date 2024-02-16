// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package sync

import (
	"sync"
)

type requestData struct {
	resultsQueue           chan *syncTaskResult
	origin                 blockOrigin
	startRequestAt         uint
	expectedAmountOfBlocks uint32
}

func (cs *chainSync) startRequestQueue(wg *sync.WaitGroup) { // Can this be receiver?
	defer func() {
		logger.Debugf("[STOPPED] request queue \n")
		wg.Done()
	}()

	for data := range cs.requestQueueCh {
		// process item
		logger.Warnf("Processing request")
		err := cs.handleWorkersResults(data.resultsQueue, data.origin, data.startRequestAt, data.expectedAmountOfBlocks)
		if err != nil {
			logger.Errorf("ERROR %v\n", err)
			//panic(err) // change later
		}
		logger.Warnf("Done processing request")
	}
}
