// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package blocktree

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/lib/common"
	runtime "github.com/ChainSafe/gossamer/lib/runtime"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func Test_newHashToRuntime(t *testing.T) {
	t.Parallel()

	hti := newHashToRuntime()

	expected := &hashToRuntime{
		mapping: make(map[Hash]runtime.Instance),
	}
	assert.Equal(t, expected, hti)
}

func Test_hashToRuntime_get(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		htr      *hashToRuntime
		hash     Hash
		instance runtime.Instance
	}{
		"hash_does_not_exist": {
			htr: &hashToRuntime{
				mapping: map[Hash]runtime.Instance{
					{4, 5, 6}: NewMockInstance(nil),
				},
			},
			hash: common.Hash{1, 2, 3},
		},
		"hash_exists": {
			htr: &hashToRuntime{
				mapping: map[Hash]runtime.Instance{
					{1, 2, 3}: NewMockInstance(nil),
				},
			},
			hash:     common.Hash{1, 2, 3},
			instance: NewMockInstance(nil),
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			instance := testCase.htr.get(testCase.hash)

			assert.Equal(t, testCase.instance, instance)
		})
	}
}

func Test_hashToRuntime_hashes(t *testing.T) {
	t.Parallel()

	htr := &hashToRuntime{
		mapping: map[Hash]runtime.Instance{
			{4, 5, 6}: NewMockInstance(nil),
			{7, 8, 9}: NewMockInstance(nil),
			{1, 2, 3}: NewMockInstance(nil),
		},
	}

	expectedHashes := []common.Hash{
		{7, 8, 9},
		{4, 5, 6},
		{1, 2, 3},
	}

	hashes := htr.hashes()
	assert.ElementsMatch(t, expectedHashes, hashes)
}

func Test_hashToRuntime_set(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		initialHtr  *hashToRuntime
		hash        Hash
		instance    runtime.Instance
		expectedHtr *hashToRuntime
	}{
		"set_new_instance": {
			initialHtr: &hashToRuntime{
				mapping: map[Hash]runtime.Instance{},
			},
			hash:     common.Hash{1, 2, 3},
			instance: NewMockInstance(nil),
			expectedHtr: &hashToRuntime{
				mapping: map[Hash]runtime.Instance{
					{1, 2, 3}: NewMockInstance(nil),
				},
			},
		},
		"override_instance": {
			initialHtr: &hashToRuntime{
				mapping: map[Hash]runtime.Instance{
					{1, 2, 3}: NewMockInstance(nil),
				},
			},
			hash:     common.Hash{1, 2, 3},
			instance: nil,
			expectedHtr: &hashToRuntime{
				mapping: map[Hash]runtime.Instance{
					{1, 2, 3}: nil,
				},
			},
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			htr := testCase.initialHtr

			htr.set(testCase.hash, testCase.instance)

			assert.Equal(t, testCase.expectedHtr, htr)
		})
	}
}

func Test_hashToRuntime_delete(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		initialHtr  *hashToRuntime
		hash        common.Hash
		expectedHtr *hashToRuntime
	}{
		"hash_does_not_exist": {
			initialHtr: &hashToRuntime{
				mapping: map[Hash]runtime.Instance{},
			},
			hash: common.Hash{1, 2, 3},
			expectedHtr: &hashToRuntime{
				mapping: map[Hash]runtime.Instance{},
			},
		},
		"hash_deleted": {
			initialHtr: &hashToRuntime{
				mapping: map[Hash]runtime.Instance{
					{1, 2, 3}: NewMockInstance(nil),
				},
			},
			hash: common.Hash{1, 2, 3},
			expectedHtr: &hashToRuntime{
				mapping: map[Hash]runtime.Instance{},
			},
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			htr := testCase.initialHtr

			htr.delete(testCase.hash)

			assert.Equal(t, testCase.expectedHtr, htr)
		})
	}
}

func Test_hashToRuntime_onFinalisation(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		makeParameters          func(ctrl *gomock.Controller) (initial, expected *hashToRuntime)
		newCanonicalBlockHashes []Hash
	}{
		"new_finalised_runtime_not_found": {
			makeParameters: func(ctrl *gomock.Controller) (initial, expected *hashToRuntime) {
				return &hashToRuntime{}, &hashToRuntime{}
			},
			newCanonicalBlockHashes: []Hash{{1}},
		},
		"prune_fork_runtime_with_a_unique_instance": {
			makeParameters: func(ctrl *gomock.Controller) (initial, expected *hashToRuntime) {
				finalisedRuntime := NewMockInstance(ctrl)
				initial = &hashToRuntime{
					mapping: map[Hash]runtime.Instance{
						{1}: finalisedRuntime,
					},
				}

				// keep the instance but update the key
				expected = &hashToRuntime{
					mapping: map[Hash]runtime.Instance{
						{2}: finalisedRuntime,
					},
				}
				return initial, expected
			},
			newCanonicalBlockHashes: []Hash{{2}},
		},
		"prune_fork_runtimes_only": {
			makeParameters: func(ctrl *gomock.Controller) (initial, expected *hashToRuntime) {
				finalisedRuntime := NewMockInstance(ctrl)
				prunedForkRuntime := NewMockInstance(ctrl)
				prunedForkRuntime.EXPECT().Stop()
				initial = &hashToRuntime{
					mapping: map[Hash]runtime.Instance{
						{1}: finalisedRuntime,
						{3}: prunedForkRuntime,
					},
				}
				expected = &hashToRuntime{
					mapping: map[Hash]runtime.Instance{
						{1}: finalisedRuntime,
					},
				}
				return initial, expected
			},
			newCanonicalBlockHashes: []Hash{{1}},
		},
		"new_canonical_block_hash_not_found": {
			makeParameters: func(ctrl *gomock.Controller) (initial, expected *hashToRuntime) {
				newFinalisedRuntime := NewMockInstance(ctrl)
				initial = &hashToRuntime{
					mapping: map[Hash]runtime.Instance{
						// missing {1}
						{2}: newFinalisedRuntime,
					},
				}
				expected = &hashToRuntime{
					mapping: map[Hash]runtime.Instance{
						{2}: newFinalisedRuntime,
					},
				}
				return initial, expected
			},
			newCanonicalBlockHashes: []Hash{{1}, {2}},
		},
		"prune_fork_and_canonical_runtimes": {
			makeParameters: func(ctrl *gomock.Controller) (initial, expected *hashToRuntime) {
				finalisedRuntime := NewMockInstance(ctrl)
				unfinalisedRuntime := NewMockInstance(ctrl)
				newFinalisedRuntime := NewMockInstance(ctrl)
				prunedForkRuntime := NewMockInstance(ctrl)

				finalisedRuntime.EXPECT().Stop()
				unfinalisedRuntime.EXPECT().Stop()
				prunedForkRuntime.EXPECT().Stop()

				initial = &hashToRuntime{
					mapping: map[Hash]runtime.Instance{
						// Previously finalised chain
						{0}: finalisedRuntime,
						// Newly finalised chain
						{3}: unfinalisedRuntime,
						{5}: newFinalisedRuntime,
						// Runtimes from forks
						{100}: prunedForkRuntime,
					},
				}
				expected = &hashToRuntime{
					mapping: map[Hash]runtime.Instance{
						{6}: newFinalisedRuntime,
					},
				}
				return initial, expected
			},
			newCanonicalBlockHashes: []Hash{{2}, {3}, {4}, {5}, {6}},
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)

			htr, expectedHtr := testCase.makeParameters(ctrl)
			htr.onFinalisation(testCase.newCanonicalBlockHashes)

			assert.Equal(t, expectedHtr, htr)
		})
	}
}

func Test_hashToRuntime_threadSafety(t *testing.T) {
	// This test consists in checking for concurrent access
	// using the -race detector.
	t.Parallel()

	var startWg, endWg sync.WaitGroup
	ctx, cancel := context.WithCancel(context.Background())

	const parallelism = 4
	const operations = 3
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

	htr := newHashToRuntime()
	hash := common.Hash{1, 2, 3}
	instance := NewMockInstance(nil)

	for i := 0; i < parallelism; i++ {
		go runInLoop(func() {
			htr.get(hash)
		})

		go runInLoop(func() {
			htr.set(hash, instance)
		})

		go runInLoop(func() {
			htr.delete(hash)
		})
	}

	endWg.Wait()
}
