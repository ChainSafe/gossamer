// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package sync

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_newBlockQueue(t *testing.T) {
	t.Parallel()

	const capacity = 1
	bq := newBlockQueue(capacity)

	require.NotNil(t, bq.queue)
	assert.Equal(t, 1, cap(bq.queue))
	assert.Equal(t, 0, len(bq.queue))
	bq.queue = nil

	expectedBlockQueue := &blockQueue{
		hashesSet: make(map[common.Hash]struct{}, capacity),
	}
	assert.Equal(t, expectedBlockQueue, bq)
}

func Test_blockQueue_push(t *testing.T) {
	t.Parallel()

	const capacity = 1
	bq := newBlockQueue(capacity)
	blockData := &types.BlockData{
		Hash: common.Hash{1},
	}

	bq.push(blockData)

	// cannot compare channels
	require.NotNil(t, bq.queue)
	assert.Len(t, bq.queue, 1)

	receivedBlockData := <-bq.queue
	expectedBlockData := &types.BlockData{
		Hash: common.Hash{1},
	}
	assert.Equal(t, expectedBlockData, receivedBlockData)

	bq.queue = nil
	expectedBlockQueue := &blockQueue{
		hashesSet: map[common.Hash]struct{}{{1}: {}},
	}
	assert.Equal(t, expectedBlockQueue, bq)
}

func Test_blockQueue_pop(t *testing.T) {
	t.Parallel()

	t.Run("context_canceled", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		const capacity = 1
		bq := newBlockQueue(capacity)

		blockData, err := bq.pop(ctx)
		assert.Nil(t, blockData)
		assert.ErrorIs(t, err, context.Canceled)
	})

	t.Run("get_block_data_after_waiting", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()

		const capacity = 1
		bq := newBlockQueue(capacity)

		const afterDuration = 5 * time.Millisecond
		time.AfterFunc(afterDuration, func() {
			blockData := &types.BlockData{
				Hash: common.Hash{1},
			}
			bq.push(blockData)
		})

		blockData, err := bq.pop(ctx)

		expectedBlockData := &types.BlockData{
			Hash: common.Hash{1},
		}
		assert.Equal(t, expectedBlockData, blockData)
		assert.NoError(t, err)

		assert.Len(t, bq.queue, 0)
		bq.queue = nil
		expectedBlockQueue := &blockQueue{
			hashesSet: map[common.Hash]struct{}{},
		}
		assert.Equal(t, expectedBlockQueue, bq)
	})
}

func Test_blockQueue_has(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		blockQueue *blockQueue
		blockHash  common.Hash
		has        bool
	}{
		"absent": {
			blockQueue: &blockQueue{
				hashesSet: map[common.Hash]struct{}{},
			},
			blockHash: common.Hash{1},
		},
		"exists": {
			blockQueue: &blockQueue{
				hashesSet: map[common.Hash]struct{}{{1}: {}},
			},
			blockHash: common.Hash{1},
			has:       true,
		},
	}

	for name, tc := range testCases {
		testCase := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			has := testCase.blockQueue.has(testCase.blockHash)
			assert.Equal(t, testCase.has, has)
		})
	}
}

func Test_lockQueue_endToEnd(t *testing.T) {
	t.Parallel()

	const capacity = 10
	blockQueue := newBlockQueue(capacity)

	newBlockData := func(i byte) *types.BlockData {
		return &types.BlockData{
			Hash: common.Hash{i},
		}
	}

	blockQueue.push(newBlockData(1))
	blockQueue.push(newBlockData(2))
	blockQueue.push(newBlockData(3))

	blockData, err := blockQueue.pop(context.Background())
	assert.Equal(t, newBlockData(1), blockData)
	assert.NoError(t, err)

	has := blockQueue.has(newBlockData(2).Hash)
	assert.True(t, has)
	has = blockQueue.has(newBlockData(3).Hash)
	assert.True(t, has)

	blockQueue.push(newBlockData(4))

	has = blockQueue.has(newBlockData(4).Hash)
	assert.True(t, has)

	blockData, err = blockQueue.pop(context.Background())
	assert.Equal(t, newBlockData(2), blockData)
	assert.NoError(t, err)

	// drain queue
	for len(blockQueue.queue) > 0 {
		<-blockQueue.queue
	}
}

func Test_lockQueue_threadSafety(t *testing.T) {
	// This test consists in checking for concurrent access
	// using the -race detector.
	t.Parallel()

	var startWg, endWg sync.WaitGroup
	ctx, cancel := context.WithCancel(context.Background())

	const operations = 3
	const parallelism = 3
	const goroutines = parallelism * operations
	startWg.Add(goroutines)
	endWg.Add(goroutines)

	const testDuration = 50 * time.Millisecond
	go func() {
		timer := time.NewTimer(time.Hour)
		startWg.Wait()
		_ = timer.Reset(testDuration)
		<-timer.C
		cancel()
	}()

	runInLoop := func(f func()) {
		defer endWg.Done()
		startWg.Done()
		startWg.Wait()
		for ctx.Err() == nil {
			f()
		}
	}

	const capacity = 10
	blockQueue := newBlockQueue(capacity)
	blockData := &types.BlockData{
		Hash: common.Hash{1},
	}
	blockHash := common.Hash{1}

	endWg.Add(1)
	go func() {
		defer endWg.Done()
		<-ctx.Done()
		// Empty queue channel to make sure `push` does not block
		// when the context is cancelled.
		for len(blockQueue.queue) > 0 {
			<-blockQueue.queue
		}
	}()

	for i := 0; i < parallelism; i++ {
		go runInLoop(func() {
			blockQueue.push(blockData)
		})

		go runInLoop(func() {
			_, _ = blockQueue.pop(ctx)
		})

		go runInLoop(func() {
			_ = blockQueue.has(blockHash)
		})
	}

	endWg.Wait()
}
