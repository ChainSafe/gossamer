// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package triedb

import (
	"testing"

	"github.com/ChainSafe/gossamer/internal/database"
	"github.com/ChainSafe/gossamer/pkg/trie"
	"github.com/ChainSafe/gossamer/pkg/trie/cache/inmemory"
	inmemory_trie "github.com/ChainSafe/gossamer/pkg/trie/inmemory"
	"github.com/stretchr/testify/assert"
)

func newTestDB(t assert.TestingT) database.Table {
	db, err := database.NewPebble("", true)
	assert.NoError(t, err)
	return database.NewTable(db, "trie")
}

func TestTrieDB_Migration(t *testing.T) {
	db := newTestDB(t)
	inMemoryTrie := inmemory_trie.NewEmptyTrie()
	inMemoryTrie.SetVersion(trie.V1)

	entries := map[string][]byte{
		"no":           make([]byte, 1),
		"noot":         make([]byte, 2),
		"not":          make([]byte, 3),
		"notable":      make([]byte, 4),
		"notification": make([]byte, 5),
		"test":         make([]byte, 6),
		"dimartiro":    make([]byte, 7),
	}

	for k, v := range entries {
		inMemoryTrie.Put([]byte(k), v)
	}

	err := inMemoryTrie.WriteDirty(db)
	assert.NoError(t, err)

	root, err := inMemoryTrie.Hash()
	assert.NoError(t, err)
	trieDB := NewTrieDB(root, db, nil)

	t.Run("read_successful_from_db_created_using_v1_trie", func(t *testing.T) {
		for k, v := range entries {
			value := trieDB.Get([]byte(k))
			assert.Equal(t, v, value)
		}

		assert.Equal(t, root, trieDB.MustHash())
	})

	t.Run("next_key_are_the_same", func(t *testing.T) {
		key := []byte("no")

		for key != nil {
			expected := inMemoryTrie.NextKey(key)
			actual := trieDB.NextKey(key)
			assert.Equal(t, expected, actual)

			key = actual
		}
	})

	t.Run("get_keys_with_prefix_are_the_same", func(t *testing.T) {
		key := []byte("no")

		expected := inMemoryTrie.GetKeysWithPrefix(key)
		actual := trieDB.GetKeysWithPrefix(key)

		assert.Equal(t, expected, actual)
	})

	t.Run("entries_are_the_same", func(t *testing.T) {
		expected := inMemoryTrie.Entries()
		actual := trieDB.Entries()

		assert.Equal(t, expected, actual)
	})

	t.Run("cache_has_cache_value", func(t *testing.T) {
		cache := inmemory.NewTrieInMemoryCache()
		trieDB := NewTrieDB(root, db, cache)

		val := trieDB.Get([]byte("no"))
		assert.NotNil(t, val)

		valueFromCache := cache.GetValue([]byte("no"))
		assert.Equal(t, val, valueFromCache)
	})
}

func TestTrieDB_Lookup(t *testing.T) {
	t.Run("root_not_exists_in_db", func(t *testing.T) {
		db := newTestDB(t)
		trieDB := NewTrieDB(trie.EmptyHash, db, nil)

		value, err := trieDB.lookup([]byte("test"))
		assert.Nil(t, value)
		assert.ErrorIs(t, err, ErrIncompleteDB)
	})
}

func Benchmark_GetKeyWithoutCache(b *testing.B) {
	db := newTestDB(b)
	inMemoryTrie := inmemory_trie.NewEmptyTrie()
	inMemoryTrie.SetVersion(trie.V1)

	entries := map[string][]byte{
		"no":           make([]byte, 1),
		"not":          make([]byte, 2),
		"nota":         make([]byte, 3),
		"notab":        make([]byte, 4),
		"notification": make([]byte, 5),
	}

	for k, v := range entries {
		inMemoryTrie.Put([]byte(k), v)
	}

	err := inMemoryTrie.WriteDirty(db)
	assert.NoError(b, err)

	root, err := inMemoryTrie.Hash()
	assert.NoError(b, err)

	// 3358 ns/op	    2949 B/op	     117 allocs/op
	b.Run("get_key_without_cache", func(b *testing.B) {
		trieDB := NewTrieDB(root, db, nil)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// Use the deepest key to ensure the trie is traversed fully
			_ = trieDB.Get([]byte("notification"))
		}
	})

	// 81.43 ns/op	      32 B/op	       3 allocs/op
	b.Run("get_key_with_cache", func(b *testing.B) {
		cache := inmemory.NewTrieInMemoryCache()
		trieDB := NewTrieDB(root, db, cache)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// Use the deepest key to ensure the trie is traversed fully
			_ = trieDB.Get([]byte("notification"))
		}
	})
}
