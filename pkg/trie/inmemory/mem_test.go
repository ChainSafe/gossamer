// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package inmemory

import (
	"math"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_Trie_MemoryUsage(t *testing.T) {
	// Set skip to false to run the test.
	// This test should be run on its own since it interacts
	// with the Go garbage collector.
	const skip = true
	if skip {
		t.SkipNow()
	}

	triesMap := map[string]*InMemoryTrie{
		"first": NewEmptyInmemoryTrie(),
	}

	generator := newGenerator()
	const size = 10000
	kv := generateKeyValues(t, generator, size)

	// Populate a first branch branching out
	// from the root on the 'left'
	populateTrieAtPrefix(triesMap["first"], []byte{0, 1}, kv)

	// Check heap memory usage - it should be X
	halfFilledTrieHeap := getHeapUsage()

	// Populate a second branch branching out
	// from the root on the 'right'
	populateTrieAtPrefix(triesMap["first"], []byte{0, 2}, kv)

	// Check heap memory usage - it should be 2X
	filledTrieHeap := getHeapUsage()
	ratio := getApproximateRatio(halfFilledTrieHeap, filledTrieHeap)
	assert.Greater(t, ratio, 1.6)
	assert.Less(t, ratio, 1.7)

	// Snapshot the trie
	triesMap["second"] = triesMap["first"].Snapshot()

	// Modify all the leaves from the first branch in the new trie
	mutateTrieLeavesAtPrefix(triesMap["second"], []byte{0, 1}, kv)

	// Check heap memory usage - it should be 3X
	halfMutatedTrieHeap := getHeapUsage()
	ratio = getApproximateRatio(halfFilledTrieHeap, halfMutatedTrieHeap)
	assert.Greater(t, ratio, 2.2)
	assert.Less(t, ratio, 2.4)

	// Remove the older trie from our reference
	delete(triesMap, "first")

	// Check heap memory usage - it should be 2X
	prunedTrieHeap := getHeapUsage()
	ratio = getApproximateRatio(halfFilledTrieHeap, prunedTrieHeap)
	assert.Greater(t, ratio, 1.6)
	assert.Less(t, ratio, 1.8)

	// Dummy calls - has to be after prunedTrieHeap for
	// GC to keep them
	_, ok := triesMap["first"]
	require.False(t, ok)
	_, ok = kv["dummy"]
	require.False(t, ok)
}

func getApproximateRatio(old, new uint64) (ratio float64) {
	ratio = float64(new) / float64(old)
	ratio = math.Round(ratio*100) / 100
	return ratio
}

func getHeapUsage() (heapAlloc uint64) {
	runtime.GC()
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	return memStats.HeapAlloc
}

func populateTrieAtPrefix(trie *InMemoryTrie,
	prefix []byte, kv map[string][]byte) {
	for keyString, value := range kv {
		key := append(prefix, []byte(keyString)...) //skipcq: CRT-D0001

		trie.Put(key, value)
	}
}

func mutateTrieLeavesAtPrefix(trie *InMemoryTrie,
	prefix []byte, originalKV map[string][]byte) {
	for keyString, value := range originalKV {
		key := append(prefix, []byte(keyString)...) //skipcq: CRT-D0001

		var newValue []byte
		if len(value) == 0 {
			newValue = []byte{1}
		} else {
			newValue = make([]byte, len(value))
			copy(newValue, value)
			for i := range newValue {
				newValue[i]++
			}
		}

		trie.Put(key, newValue)
	}
}
