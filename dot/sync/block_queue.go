// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package sync

import (
	"context"
	"sync"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
)

type blockQueue struct {
	queue          chan *types.BlockData
	hashesSet      map[common.Hash]struct{}
	hashesSetMutex sync.RWMutex
}

// newBlockQueue initialises a queue of *types.BlockData with the given capacity.
func newBlockQueue(capacity int) *blockQueue {
	return &blockQueue{
		queue:     make(chan *types.BlockData, capacity),
		hashesSet: make(map[common.Hash]struct{}, capacity),
	}
}

// push pushes an item into the queue. It blocks if the queue is at capacity.
func (bq *blockQueue) push(blockData *types.BlockData) {
	bq.hashesSetMutex.Lock()
	bq.hashesSet[blockData.Hash] = struct{}{}
	bq.hashesSetMutex.Unlock()

	bq.queue <- blockData
}

// pop pops the next item from the queue. It blocks if the queue is empty
// until the context is cancelled. If the context is canceled, it returns
// the error from the context.
func (bq *blockQueue) pop(ctx context.Context) (blockData *types.BlockData, err error) {
	select {
	case <-ctx.Done():
		return blockData, ctx.Err()
	case blockData = <-bq.queue:
	}
	bq.hashesSetMutex.Lock()
	delete(bq.hashesSet, blockData.Hash)
	bq.hashesSetMutex.Unlock()
	return blockData, nil
}

func (bq *blockQueue) has(blockHash common.Hash) (has bool) {
	bq.hashesSetMutex.RLock()
	defer bq.hashesSetMutex.RUnlock()
	_, has = bq.hashesSet[blockHash]
	return has
}
