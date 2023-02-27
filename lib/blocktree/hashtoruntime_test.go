// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package blocktree

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func Test_newHashToRuntime(t *testing.T) {
	t.Parallel()

	hti := newHashToRuntime()

	expected := &hashToRuntime{
		mapping: make(map[Hash]Runtime),
	}
	assert.Equal(t, expected, hti)
}

func Test_hashToRuntime_get(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		htr      *hashToRuntime
		hash     Hash
		instance Runtime
	}{
		"hash_does_not_exist": {
			htr: &hashToRuntime{
				mapping: map[Hash]Runtime{
					{4, 5, 6}: NewMockRuntime(nil),
				},
			},
			hash: common.Hash{1, 2, 3},
		},
		"hash_exists": {
			htr: &hashToRuntime{
				mapping: map[Hash]Runtime{
					{1, 2, 3}: NewMockRuntime(nil),
				},
			},
			hash:     common.Hash{1, 2, 3},
			instance: NewMockRuntime(nil),
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

func Test_hashToRuntime_set(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		initialHtr  *hashToRuntime
		hash        Hash
		instance    Runtime
		expectedHtr *hashToRuntime
	}{
		"set_new_instance": {
			initialHtr: &hashToRuntime{
				mapping: map[Hash]Runtime{},
			},
			hash:     common.Hash{1, 2, 3},
			instance: NewMockRuntime(nil),
			expectedHtr: &hashToRuntime{
				mapping: map[Hash]Runtime{
					{1, 2, 3}: NewMockRuntime(nil),
				},
			},
		},
		"override_instance": {
			initialHtr: &hashToRuntime{
				mapping: map[Hash]Runtime{
					{1, 2, 3}: NewMockRuntime(nil),
				},
			},
			hash:     common.Hash{1, 2, 3},
			instance: nil,
			expectedHtr: &hashToRuntime{
				mapping: map[Hash]Runtime{
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
				mapping: map[Hash]Runtime{},
			},
			hash: common.Hash{1, 2, 3},
			expectedHtr: &hashToRuntime{
				mapping: map[Hash]Runtime{},
			},
		},
		"hash_deleted": {
			initialHtr: &hashToRuntime{
				mapping: map[Hash]Runtime{
					{1, 2, 3}: NewMockRuntime(nil),
				},
			},
			hash: common.Hash{1, 2, 3},
			expectedHtr: &hashToRuntime{
				mapping: map[Hash]Runtime{},
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
		prunedForkBlockHashes   []Hash
		panicString             string
	}{
		"new_finalised_runtime_not_found": {
			makeParameters: func(ctrl *gomock.Controller) (initial, expected *hashToRuntime) {
				return &hashToRuntime{}, nil
			},
			newCanonicalBlockHashes: []Hash{{1}},
			panicString: "runtime not found for finalised block hash " +
				"0x0100000000000000000000000000000000000000000000000000000000000000",
		},
		"pruned_fork_runtime_not_found": {
			makeParameters: func(ctrl *gomock.Controller) (initial, expected *hashToRuntime) {
				finalisedRuntime := NewMockRuntime(ctrl)
				return &hashToRuntime{
					mapping: map[Hash]Runtime{
						{1}: finalisedRuntime,
					},
				}, nil
			},
			newCanonicalBlockHashes: []Hash{{1}},
			prunedForkBlockHashes:   []Hash{{2}},
			panicString: "runtime not found for pruned forked block hash " +
				"0x0200000000000000000000000000000000000000000000000000000000000000",
		},
		"prune_fork_runtimes_only": {
			makeParameters: func(ctrl *gomock.Controller) (initial, expected *hashToRuntime) {
				finalisedRuntime := NewMockRuntime(ctrl)
				prunedForkRuntime := NewMockRuntime(ctrl)
				prunedForkRuntime.EXPECT().Stop()
				initial = &hashToRuntime{
					finalisedRuntime:   finalisedRuntime,
					currentBlockHashes: []Hash{{0}},
					mapping: map[Hash]Runtime{
						{1}: finalisedRuntime,
						{2}: finalisedRuntime,
						{3}: prunedForkRuntime,
						{4}: prunedForkRuntime,
					},
				}
				expected = &hashToRuntime{
					finalisedRuntime:   finalisedRuntime,
					currentBlockHashes: []Hash{{0}, {1}},
					mapping: map[Hash]Runtime{
						{1}: finalisedRuntime,
					},
				}
				return initial, expected
			},
			newCanonicalBlockHashes: []Hash{{1}},
			prunedForkBlockHashes:   []Hash{{2}, {3}, {4}},
		},
		"new_canonical_block_hash_not_found": {
			makeParameters: func(ctrl *gomock.Controller) (initial, expected *hashToRuntime) {
				newFinalisedRuntime := NewMockRuntime(ctrl)
				initial = &hashToRuntime{
					mapping: map[Hash]Runtime{
						// missing {1}
						{2}: newFinalisedRuntime,
					},
				}
				return initial, nil
			},
			newCanonicalBlockHashes: []Hash{{1}, {2}},
			panicString: "runtime not found for canonical chain block hash " +
				"0x0100000000000000000000000000000000000000000000000000000000000000",
		},
		"prune_fork_and_canonical_runtimes": {
			makeParameters: func(ctrl *gomock.Controller) (initial, expected *hashToRuntime) {
				finalisedRuntime := NewMockRuntime(ctrl)
				unfinalisedRuntime := NewMockRuntime(ctrl)
				newFinalisedRuntime := NewMockRuntime(ctrl)
				prunedForkRuntime := NewMockRuntime(ctrl)

				finalisedRuntime.EXPECT().Stop()
				unfinalisedRuntime.EXPECT().Stop()
				prunedForkRuntime.EXPECT().Stop()

				initial = &hashToRuntime{
					finalisedRuntime:   finalisedRuntime,
					currentBlockHashes: []Hash{{0}, {1}},
					mapping: map[Hash]Runtime{
						// Previously finalised chain
						{0}: finalisedRuntime,
						{1}: finalisedRuntime,
						// Newly finalised chain
						{2}: finalisedRuntime,
						{3}: unfinalisedRuntime,
						{4}: unfinalisedRuntime,
						{5}: newFinalisedRuntime,
						{6}: newFinalisedRuntime,
						// Runtimes from forks
						{100}: prunedForkRuntime,
						{101}: prunedForkRuntime,
						{102}: finalisedRuntime,
						{103}: newFinalisedRuntime,
					},
				}
				expected = &hashToRuntime{
					finalisedRuntime:   newFinalisedRuntime,
					currentBlockHashes: []Hash{{5}, {6}},
					mapping: map[Hash]Runtime{
						{5}: newFinalisedRuntime,
						{6}: newFinalisedRuntime,
					},
				}
				return initial, expected
			},
			newCanonicalBlockHashes: []Hash{{2}, {3}, {4}, {5}, {6}},
			prunedForkBlockHashes:   []Hash{{100}, {101}, {102}, {103}},
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)

			htr, expectedHtr := testCase.makeParameters(ctrl)

			if testCase.panicString != "" {
				assert.PanicsWithValue(t, testCase.panicString, func() {
					htr.onFinalisation(testCase.newCanonicalBlockHashes, testCase.prunedForkBlockHashes)
				})
				return
			}

			htr.onFinalisation(testCase.newCanonicalBlockHashes, testCase.prunedForkBlockHashes)

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
	instance := NewMockRuntime(nil)

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
