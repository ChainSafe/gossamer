// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package state

import (
	"testing"

	"github.com/ChainSafe/gossamer/internal/trie/node"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/trie"
	lrucache "github.com/ChainSafe/gossamer/lib/utils/lru-cache"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

var emptyTrie = trie.NewEmptyTrie()

func Test_NewTries(t *testing.T) {
	t.Parallel()

	rootToTrie := NewTries()

	expectedTries := &Tries{
		rootToTrie:    lrucache.NewLRUCache[common.Hash, *trie.Trie](MaxInMemoryTries),
		triesGauge:    triesGauge,
		setCounter:    setCounter,
		deleteCounter: deleteCounter,
	}

	assert.Equal(t, expectedTries, rootToTrie)
}

func Test_Tries_SetEmptyTrie(t *testing.T) {
	t.Parallel()

	tries := NewTries()
	tries.SetEmptyTrie()

	expectedTries := &Tries{
		rootToTrie:    lrucache.NewLRUCache[common.Hash, *trie.Trie](MaxInMemoryTries),
		triesGauge:    triesGauge,
		setCounter:    setCounter,
		deleteCounter: deleteCounter,
	}
	expectedTries.rootToTrie.Put(trie.EmptyHash, trie.NewEmptyTrie())

	assert.Equal(t, expectedTries, tries)
}

func Test_Tries_SetTrie(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	db := NewMockDatabase(ctrl)
	db.EXPECT().Get(gomock.Any()).Times(0)

	tr := trie.NewTrie(&node.Node{PartialKey: []byte{1}}, db)

	tries := NewTries()
	tries.SetTrie(tr)

	expectedTries := &Tries{
		rootToTrie: func() *lrucache.LRUCache[common.Hash, *trie.Trie] {
			cache := lrucache.NewLRUCache[common.Hash, *trie.Trie](MaxInMemoryTries)
			cache.Put(tr.MustHash(trie.NoMaxInlineValueSize), tr)
			return cache
		}(),
		triesGauge:    triesGauge,
		setCounter:    setCounter,
		deleteCounter: deleteCounter,
	}

	expectedTries.rootToTrie.Put(tr.MustHash(trie.NoMaxInlineValueSize), tr)

	assert.Equal(t, expectedTries, tries)
}

func Test_Tries_softSet(t *testing.T) {
	t.Parallel()
	testCases := map[string]struct {
		rootToTrie         *lrucache.LRUCache[common.Hash, *trie.Trie]
		root               common.Hash
		trie               *trie.Trie
		triesGaugeInc      bool
		expectedRootToTrie *lrucache.LRUCache[common.Hash, *trie.Trie]
	}{
		"set_new_in_map": {
			rootToTrie:    lrucache.NewLRUCache[common.Hash, *trie.Trie](MaxInMemoryTries),
			root:          common.Hash{1, 2, 3},
			trie:          trie.NewEmptyTrie(),
			triesGaugeInc: true,
			expectedRootToTrie: func() *lrucache.LRUCache[common.Hash, *trie.Trie] {
				cache := lrucache.NewLRUCache[common.Hash, *trie.Trie](MaxInMemoryTries)
				cache.Put(common.Hash{1, 2, 3}, trie.NewEmptyTrie())
				return cache
			}(),
		},
		"do_not_override_in_map": {
			rootToTrie: func() *lrucache.LRUCache[common.Hash, *trie.Trie] {
				cache := lrucache.NewLRUCache[common.Hash, *trie.Trie](MaxInMemoryTries)
				cache.Put(common.Hash{1, 2, 3}, emptyTrie)
				return cache
			}(),
			root: common.Hash{1, 2, 3},
			trie: emptyTrie,
			expectedRootToTrie: func() *lrucache.LRUCache[common.Hash, *trie.Trie] {
				cache := lrucache.NewLRUCache[common.Hash, *trie.Trie](MaxInMemoryTries)
				cache.Put(common.Hash{1, 2, 3}, emptyTrie)
				return cache
			}(),
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)

			triesGauge := NewMockGauge(ctrl)
			if testCase.triesGaugeInc {
				triesGauge.EXPECT().Inc()
			}

			setCounter := NewMockCounter(ctrl)
			if testCase.triesGaugeInc {
				setCounter.EXPECT().Inc()
			}

			tries := &Tries{
				rootToTrie: testCase.rootToTrie,
				triesGauge: triesGauge,
				setCounter: setCounter,
			}

			tries.softSet(testCase.root, testCase.trie)

			assert.Equal(t, testCase.expectedRootToTrie, tries.rootToTrie)
		})
	}
}

func Test_Tries_delete(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		rootToTrie         *lrucache.LRUCache[common.Hash, *trie.Trie]
		root               common.Hash
		counterUpdated     bool
		expectedRootToTrie *lrucache.LRUCache[common.Hash, *trie.Trie]
		triesGaugeSet      float64
	}{
		"not_found": {
			rootToTrie: func() *lrucache.LRUCache[common.Hash, *trie.Trie] {
				cache := lrucache.NewLRUCache[common.Hash, *trie.Trie](MaxInMemoryTries)
				cache.Put(common.Hash{3, 4, 5}, emptyTrie)
				return cache
			}(),
			root:          common.Hash{1, 2, 3},
			triesGaugeSet: 1,
			expectedRootToTrie: func() *lrucache.LRUCache[common.Hash, *trie.Trie] {
				cache := lrucache.NewLRUCache[common.Hash, *trie.Trie](MaxInMemoryTries)
				cache.Put(common.Hash{3, 4, 5}, emptyTrie)
				return cache
			}(),
			counterUpdated: false,
		},
		"deleted": {
			rootToTrie: func() *lrucache.LRUCache[common.Hash, *trie.Trie] {
				cache := lrucache.NewLRUCache[common.Hash, *trie.Trie](MaxInMemoryTries)
				cache.Put(common.Hash{1, 2, 3}, emptyTrie)
				cache.Put(common.Hash{3, 4, 5}, emptyTrie)
				return cache
			}(),
			root:          common.Hash{1, 2, 3},
			triesGaugeSet: 1,
			expectedRootToTrie: func() *lrucache.LRUCache[common.Hash, *trie.Trie] {
				cache := lrucache.NewLRUCache[common.Hash, *trie.Trie](MaxInMemoryTries)
				cache.Put(common.Hash{3, 4, 5}, emptyTrie)
				return cache
			}(),
			counterUpdated: true,
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			triesGauge := NewMockGauge(ctrl)
			deleteCounter := NewMockCounter(ctrl)

			if testCase.counterUpdated {
				deleteCounter.EXPECT().Inc()
				triesGauge.EXPECT().Dec()
			}

			tries := &Tries{
				rootToTrie:    testCase.rootToTrie,
				triesGauge:    triesGauge,
				deleteCounter: deleteCounter,
			}

			tries.delete(testCase.root)

			assert.Equal(t, testCase.expectedRootToTrie, tries.rootToTrie)
		})
	}
}

func Test_Tries_get(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	db := NewMockDatabase(ctrl)
	db.EXPECT().Get(gomock.Any()).Times(0)

	testCases := map[string]struct {
		tries *Tries
		root  common.Hash
		trie  *trie.Trie
	}{
		"found_in_map": {
			tries: &Tries{
				rootToTrie: func() *lrucache.LRUCache[common.Hash, *trie.Trie] {
					cache := lrucache.NewLRUCache[common.Hash, *trie.Trie](MaxInMemoryTries)
					tr := trie.NewTrie(&node.Node{
						PartialKey:   []byte{1, 2, 3},
						StorageValue: []byte{1},
					}, db)
					cache.Put(common.Hash{1, 2, 3}, tr)
					return cache
				}(),
			},
			root: common.Hash{1, 2, 3},
			trie: trie.NewTrie(&node.Node{
				PartialKey:   []byte{1, 2, 3},
				StorageValue: []byte{1},
			}, db),
		},
		"not_found_in_map": {
			// similar to not found in database
			tries: &Tries{
				rootToTrie: lrucache.NewLRUCache[common.Hash, *trie.Trie](MaxInMemoryTries),
			},
			root: common.Hash{1, 2, 3},
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			trieFound := testCase.tries.get(testCase.root)

			assert.Equal(t, testCase.trie, trieFound)
		})
	}
}

func Test_Tries_len(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		tries  *Tries
		length int
	}{
		"empty_map": {
			tries: &Tries{
				rootToTrie: lrucache.NewLRUCache[common.Hash, *trie.Trie](MaxInMemoryTries),
			},
		},
		"non_empty_map": {
			tries: &Tries{
				rootToTrie: func() *lrucache.LRUCache[common.Hash, *trie.Trie] {
					cache := lrucache.NewLRUCache[common.Hash, *trie.Trie](MaxInMemoryTries)
					cache.Put(common.Hash{1, 2, 3}, emptyTrie)
					return cache
				}(),
			},
			length: 1,
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			length := testCase.tries.len()

			assert.Equal(t, testCase.length, length)
		})
	}
}
