// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package state

import (
	"context"
	"math/big"
	"sync"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/stretchr/testify/assert"
)

func Test_newHashToBlockMap(t *testing.T) {
	t.Parallel()

	htb := newHashToBlockMap()

	expected := &hashToBlockMap{
		mapping: make(map[common.Hash]*types.Block),
	}
	assert.Equal(t, expected, htb)
}

func Test_hashToBlockMap_getBlock(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		htb   *hashToBlockMap
		hash  common.Hash
		block *types.Block
	}{
		"hash does not exist": {
			htb: &hashToBlockMap{
				mapping: map[common.Hash]*types.Block{
					{4, 5, 6}: {},
				},
			},
			hash: common.Hash{1, 2, 3},
		},
		"hash exists": {
			htb: &hashToBlockMap{
				mapping: map[common.Hash]*types.Block{
					{1, 2, 3}: {Header: types.Header{ParentHash: common.Hash{1}}},
				},
			},
			hash:  common.Hash{1, 2, 3},
			block: &types.Block{Header: types.Header{ParentHash: common.Hash{1}}},
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			block := testCase.htb.getBlock(testCase.hash)

			assert.Equal(t, testCase.block, block)
		})
	}
}

func Test_hashToBlockMap_getBlockHeader(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		htb    *hashToBlockMap
		hash   common.Hash
		header *types.Header
	}{
		"hash does not exist": {
			htb: &hashToBlockMap{
				mapping: map[common.Hash]*types.Block{
					{4, 5, 6}: {},
				},
			},
			hash: common.Hash{1, 2, 3},
		},
		"hash exists": {
			htb: &hashToBlockMap{
				mapping: map[common.Hash]*types.Block{
					{1, 2, 3}: {Header: types.Header{ParentHash: common.Hash{1}}},
				},
			},
			hash:   common.Hash{1, 2, 3},
			header: &types.Header{ParentHash: common.Hash{1}},
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			header := testCase.htb.getBlockHeader(testCase.hash)

			assert.Equal(t, testCase.header, header)
		})
	}
}

func Test_hashToBlockMap_getBlockBody(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		htb  *hashToBlockMap
		hash common.Hash
		body *types.Body
	}{
		"hash does not exist": {
			htb: &hashToBlockMap{
				mapping: map[common.Hash]*types.Block{
					{4, 5, 6}: {},
				},
			},
			hash: common.Hash{1, 2, 3},
		},
		"hash exists": {
			htb: &hashToBlockMap{
				mapping: map[common.Hash]*types.Block{
					{1, 2, 3}: {Body: types.Body{}},
				},
			},
			hash: common.Hash{1, 2, 3},
			body: &types.Body{},
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			body := testCase.htb.getBlockBody(testCase.hash)

			assert.Equal(t, testCase.body, body)
		})
	}
}

func Test_hashToBlockMap_store(t *testing.T) {
	t.Parallel()

	headerWithHash := func(header types.Header) types.Header {
		header.Hash()
		return header
	}

	testCases := map[string]struct {
		initialMap  *hashToBlockMap
		block       *types.Block
		expectedMap *hashToBlockMap
	}{
		"override block": {
			initialMap: &hashToBlockMap{
				mapping: map[common.Hash]*types.Block{
					{
						0x64, 0x43, 0xa0, 0xb4, 0x6e, 0x4, 0x12, 0xe6,
						0x26, 0x36, 0x30, 0x28, 0x11, 0x5a, 0x9f, 0x2c,
						0xf9, 0x63, 0xee, 0xed, 0x52, 0x6b, 0x8b, 0x33,
						0xe5, 0x31, 0x6f, 0x8, 0xb5, 0xd, 0xd, 0xc3,
					}: {Header: types.Header{Number: big.NewInt(99)}},
				},
			},
			block: &types.Block{Header: types.Header{Number: big.NewInt(1)}},
			expectedMap: &hashToBlockMap{
				mapping: map[common.Hash]*types.Block{
					{
						0x64, 0x43, 0xa0, 0xb4, 0x6e, 0x4, 0x12, 0xe6,
						0x26, 0x36, 0x30, 0x28, 0x11, 0x5a, 0x9f, 0x2c,
						0xf9, 0x63, 0xee, 0xed, 0x52, 0x6b, 0x8b, 0x33,
						0xe5, 0x31, 0x6f, 0x8, 0xb5, 0xd, 0xd, 0xc3,
					}: {Header: headerWithHash(types.Header{Number: big.NewInt(1)})},
				},
			},
		},
		"store new block": {
			initialMap: &hashToBlockMap{
				mapping: map[common.Hash]*types.Block{},
			},
			block: &types.Block{Header: types.Header{Number: big.NewInt(1)}},
			expectedMap: &hashToBlockMap{
				mapping: map[common.Hash]*types.Block{
					{
						0x64, 0x43, 0xa0, 0xb4, 0x6e, 0x4, 0x12, 0xe6,
						0x26, 0x36, 0x30, 0x28, 0x11, 0x5a, 0x9f, 0x2c,
						0xf9, 0x63, 0xee, 0xed, 0x52, 0x6b, 0x8b, 0x33,
						0xe5, 0x31, 0x6f, 0x8, 0xb5, 0xd, 0xd, 0xc3,
					}: {Header: headerWithHash(types.Header{Number: big.NewInt(1)})},
				},
			},
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			htb := testCase.initialMap

			htb.store(testCase.block)

			assert.Equal(t, testCase.expectedMap, htb)
		})
	}
}

func Test_hashToBlockMap_delete(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		initialMap    *hashToBlockMap
		hash          common.Hash
		deletedHeader *types.Header
		expectedMap   *hashToBlockMap
	}{
		"hash does not exist": {
			initialMap: &hashToBlockMap{
				mapping: map[common.Hash]*types.Block{},
			},
			hash: common.Hash{1, 2, 3},
			expectedMap: &hashToBlockMap{
				mapping: map[common.Hash]*types.Block{},
			},
		},
		"hash deleted": {
			initialMap: &hashToBlockMap{
				mapping: map[common.Hash]*types.Block{
					{1, 2, 3}: {Header: types.Header{ParentHash: common.Hash{1, 2, 3}}},
				},
			},
			hash:          common.Hash{1, 2, 3},
			deletedHeader: &types.Header{ParentHash: common.Hash{1, 2, 3}},
			expectedMap: &hashToBlockMap{
				mapping: map[common.Hash]*types.Block{},
			},
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			htb := testCase.initialMap

			deletedHeader := htb.delete(testCase.hash)

			assert.Equal(t, testCase.deletedHeader, deletedHeader)
			assert.Equal(t, testCase.expectedMap, htb)
		})
	}
}

func Test_hashToBlockMap_threadSafety(t *testing.T) {
	// This test consists in checking for concurrent access
	// using the -race detector.
	t.Parallel()

	var startWg, endWg sync.WaitGroup
	ctx, cancel := context.WithCancel(context.Background())

	const parallelism = 4
	const operations = 5
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

	htb := newHashToBlockMap()
	hash := common.Hash{
		0x64, 0x43, 0xa0, 0xb4, 0x6e, 0x4, 0x12, 0xe6,
		0x26, 0x36, 0x30, 0x28, 0x11, 0x5a, 0x9f, 0x2c,
		0xf9, 0x63, 0xee, 0xed, 0x52, 0x6b, 0x8b, 0x33,
		0xe5, 0x31, 0x6f, 0x8, 0xb5, 0xd, 0xd, 0xc3,
	}
	block := &types.Block{
		Header: types.Header{Number: big.NewInt(1)},
	}

	for i := 0; i < parallelism; i++ {
		go runInLoop(func() {
			htb.getBlock(hash)
		})

		go runInLoop(func() {
			htb.getBlockHeader(hash)
		})

		go runInLoop(func() {
			htb.getBlockBody(hash)
		})

		go runInLoop(func() {
			htb.store(block)
		})

		go runInLoop(func() {
			_ = htb.delete(hash)
		})
	}

	endWg.Wait()
}
