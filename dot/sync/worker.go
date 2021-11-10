// Copyright 2019 ChainSafe Systems (ON) Corp.
// This file is part of gossamer.
//
// The gossamer library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The gossamer library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the gossamer library. If not, see <http://www.gnu.org/licenses/>.

package sync

import (
	"context"
	"math/big"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p-core/peer"

	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/lib/common"
)

// workerState helps track the current worker set and set the upcoming worker ID
type workerState struct {
	ctx    context.Context
	cancel context.CancelFunc

	sync.Mutex
	nextWorker uint64
	workers    map[uint64]*worker
}

func newWorkerState() *workerState {
	ctx, cancel := context.WithCancel(context.Background())
	return &workerState{
		ctx:     ctx,
		cancel:  cancel,
		workers: make(map[uint64]*worker),
	}
}

func (s *workerState) add(w *worker) {
	s.Lock()
	defer s.Unlock()

	w.id = s.nextWorker
	w.ctx = s.ctx
	s.nextWorker++
	s.workers[w.id] = w
}

func (s *workerState) delete(id uint64) {
	s.Lock()
	defer s.Unlock()
	delete(s.workers, id)
}

func (s *workerState) reset() {
	s.cancel()
	s.ctx, s.cancel = context.WithCancel(context.Background())

	s.Lock()
	defer s.Unlock()

	for id := range s.workers {
		delete(s.workers, id)
	}
	s.nextWorker = 0
}

// worker respresents a process that is attempting to sync from the specified start block to target block
// if it fails for some reason, `err` is set.
// otherwise, we can assume all the blocks have been received and added to the `readyBlocks` queue
type worker struct {
	ctx        context.Context
	id         uint64
	retryCount uint16
	peersTried map[peer.ID]struct{}

	startHash    common.Hash
	startNumber  *big.Int
	targetHash   common.Hash
	targetNumber *big.Int

	// bitmap of fields to request
	requestData byte
	direction   network.SyncDirection

	duration time.Duration
	err      *workerError
}

type workerError struct {
	err error
	who peer.ID // whose response caused the error, if any
}
