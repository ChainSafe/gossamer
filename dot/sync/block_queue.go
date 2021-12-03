// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package sync

import (
	"sync"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
)

type blockQueue struct {
	sync.RWMutex
	cap    int
	ch     chan *types.BlockData
	blocks map[common.Hash]*types.BlockData
}

// newBlockQueue initialises a queue of *types.BlockData with the given capacity.
func newBlockQueue(cap int) *blockQueue {
	return &blockQueue{
		cap:    cap,
		ch:     make(chan *types.BlockData, cap),
		blocks: make(map[common.Hash]*types.BlockData),
	}
}

// push pushes an item into the queue. it blocks if the queue is at capacity.
func (q *blockQueue) push(bd *types.BlockData) {
	q.Lock()
	q.blocks[bd.Hash] = bd
	q.Unlock()

	q.ch <- bd
}

// pop pops an item from the queue. it blocks if the queue is empty.
func (q *blockQueue) pop() *types.BlockData {
	bd := <-q.ch
	q.Lock()
	delete(q.blocks, bd.Hash)
	q.Unlock()
	return bd
}

func (q *blockQueue) has(hash common.Hash) bool {
	q.RLock()
	defer q.RUnlock()
	_, has := q.blocks[hash]
	return has
}
