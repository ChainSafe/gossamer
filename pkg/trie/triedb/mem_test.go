// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package triedb

import (
	"testing"

	"github.com/ChainSafe/gossamer/pkg/trie"
	inmemory_cache "github.com/ChainSafe/gossamer/pkg/trie/cache/inmemory"
	inmemory_trie "github.com/ChainSafe/gossamer/pkg/trie/inmemory"
	"github.com/stretchr/testify/assert"
)

func Benchmark_ValueCache(b *testing.B) {
	db := newTestDB(b)
	inMemoryTrie := inmemory_trie.NewEmptyTrie()
	inMemoryTrie.SetVersion(trie.V1)

	entries := map[string][]byte{
		"no":           make([]byte, 100),
		"noot":         make([]byte, 200),
		"not":          make([]byte, 300),
		"notable":      make([]byte, 400),
		"notification": make([]byte, 500),
		"test":         make([]byte, 600),
		"dimartiro":    make([]byte, 700),
	}

	for k, v := range entries {
		inMemoryTrie.Put([]byte(k), v)
	}

	err := inMemoryTrie.WriteDirty(db)
	assert.NoError(b, err)

	root, err := inMemoryTrie.Hash()
	assert.NoError(b, err)

	b.Run("get_value_without_cache", func(b *testing.B) {
		trieDB := NewTrieDB(root, db)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// Use the deepest key to ensure the trie is traversed fully
			_ = trieDB.Get([]byte("notification"))
		}
	})

	b.Run("get_value_with_cache", func(b *testing.B) {
		cache := inmemory_cache.NewTrieInMemoryCache()
		trieDB := NewTrieDB(root, db, WithCache(cache))
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// Use the deepest key to ensure the trie is traversed fully
			_ = trieDB.Get([]byte("notification"))
		}
	})
}

func Benchmark_NodesCache(b *testing.B) {
	db := newTestDB(b)
	inMemoryTrie := inmemory_trie.NewEmptyTrie()
	inMemoryTrie.SetVersion(trie.V1)

	entries := map[string][]byte{
		"no":           make([]byte, 100),
		"noot":         make([]byte, 200),
		"not":          make([]byte, 300),
		"notable":      make([]byte, 400),
		"notification": make([]byte, 500),
		"test":         make([]byte, 600),
		"dimartiro":    make([]byte, 700),
	}

	for k, v := range entries {
		inMemoryTrie.Put([]byte(k), v)
	}

	err := inMemoryTrie.WriteDirty(db)
	assert.NoError(b, err)

	root, err := inMemoryTrie.Hash()
	assert.NoError(b, err)

	b.Run("iterate_all_entries_without_cache", func(b *testing.B) {
		trieDB := NewTrieDB(root, db)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// Iterate through all keys
			iter := NewTrieDBIterator(trieDB)
			for entry := iter.NextEntry(); entry != nil; entry = iter.NextEntry() {
			}
		}
	})

	// TODO: we still have some room to improve here, we are caching the raw
	// node data and we need to decode it every time we access it. We could
	// cache the decoded node instead and avoid decoding it every time.
	b.Run("iterate_all_entries_with_cache", func(b *testing.B) {
		cache := inmemory_cache.NewTrieInMemoryCache()
		trieDB := NewTrieDB(root, db, WithCache(cache))
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// Iterate through all keys
			iter := NewTrieDBIterator(trieDB)
			for entry := iter.NextEntry(); entry != nil; entry = iter.NextEntry() {
			}
		}
	})
}
