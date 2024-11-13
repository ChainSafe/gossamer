// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package triedb

import (
	"testing"

	"github.com/ChainSafe/gossamer/internal/primitives/core/hash"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime"
	"github.com/ChainSafe/gossamer/pkg/trie"
	"github.com/stretchr/testify/require"
)

func Benchmark_ValueCache(b *testing.B) {
	entries := map[string][]byte{
		"no":           make([]byte, 100),
		"noot":         make([]byte, 200),
		"not":          make([]byte, 300),
		"notable":      make([]byte, 400),
		"notification": make([]byte, 500),
		"test":         make([]byte, 600),
		"dimartiro":    make([]byte, 700),
	}
	version := trie.V1

	db := NewMemoryDB()
	trie := NewEmptyTrieDB[hash.H256, runtime.BlakeTwo256](db)
	trie.SetVersion(version)

	for k, v := range entries {
		require.NoError(b, trie.Set([]byte(k), v))
	}
	err := trie.commit()
	require.NoError(b, err)
	require.NotEmpty(b, trie.rootHash)
	root := trie.rootHash

	b.Run("get_value_without_cache", func(b *testing.B) {
		trieDB := NewTrieDB[hash.H256, runtime.BlakeTwo256](root, db)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// Use the deepest key to ensure the trie is traversed fully
			val, err := GetWith(trieDB, []byte("notification"), func(d []byte) []byte { return d })
			require.NoError(b, err)
			require.NotNil(b, val)
		}
	})

	b.Run("get_value_with_cache", func(b *testing.B) {
		cache := NewTestTrieCache[hash.H256]()
		trieDB := NewTrieDB(
			root, db, WithCache[hash.H256, runtime.BlakeTwo256](cache))
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// Use the deepest key to ensure the trie is traversed fully
			val, err := GetWith(trieDB, []byte("notification"), func(d []byte) []byte { return d })
			require.NoError(b, err)
			require.NotNil(b, val)
		}
	})
}

func Benchmark_NodesCache(b *testing.B) {
	entries := map[string][]byte{
		"no":           make([]byte, 100),
		"noot":         make([]byte, 200),
		"not":          make([]byte, 300),
		"notable":      make([]byte, 400),
		"notification": make([]byte, 500),
		"test":         make([]byte, 600),
		"dimartiro":    make([]byte, 700),
	}
	version := trie.V1

	db := NewMemoryDB()
	trie := NewEmptyTrieDB[hash.H256, runtime.BlakeTwo256](db)
	trie.SetVersion(version)

	for k, v := range entries {
		require.NoError(b, trie.Set([]byte(k), v))
	}
	err := trie.commit()
	require.NoError(b, err)
	require.NotEmpty(b, trie.rootHash)
	root := trie.rootHash

	b.Run("iterate_all_entries_without_cache", func(b *testing.B) {
		trieDB := NewTrieDB[hash.H256, runtime.BlakeTwo256](root, db)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// Iterate through all keys
			iter, err := NewTrieDBRawIterator(trieDB)
			require.NoError(b, err)
			for entry, err := iter.NextItem(); entry != nil && err == nil; entry, err = iter.NextItem() {
			}
		}
	})

	// This is the same as iterate_all_entries_without_cache since the raw iterator calls TrieDB.getNodeOrLookup
	b.Run("iterate_all_entries_with_cache", func(b *testing.B) {
		cache := NewTestTrieCache[hash.H256]()
		trieDB := NewTrieDB(root, db, WithCache[hash.H256, runtime.BlakeTwo256](cache))
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// Iterate through all keys
			iter, err := NewTrieDBRawIterator(trieDB)
			require.NoError(b, err)
			for entry, err := iter.NextItem(); entry != nil && err == nil; entry, err = iter.NextItem() {
			}
		}
	})
}
