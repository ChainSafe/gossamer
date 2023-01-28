// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package blocktree

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/lib/common"
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
