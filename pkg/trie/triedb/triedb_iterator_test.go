// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package triedb

import (
	"testing"

	"github.com/ChainSafe/gossamer/internal/database"
	"github.com/ChainSafe/gossamer/internal/primitives/core/hash"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime"
	"github.com/ChainSafe/gossamer/pkg/trie"
	"github.com/ChainSafe/gossamer/pkg/trie/inmemory"
	"github.com/stretchr/testify/assert"
)

func newTestDB(t assert.TestingT) database.Table {
	db, err := database.NewPebble("", true)
	assert.NoError(t, err)
	return database.NewTable(db, "trie")
}

func TestIterator(t *testing.T) {
	db := newTestDB(t)
	inMemoryTrie := inmemory.NewEmptyTrie()
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

	inmemoryDB := NewMemoryDB()
	trieDB := NewEmptyTrieDB[hash.H256, runtime.BlakeTwo256](inmemoryDB)

	for k, v := range entries {
		err := trieDB.Set([]byte(k), v)
		assert.NoError(t, err)
	}
	assert.NoError(t, trieDB.commit())

	// check that the root hashes are the same
	assert.Equal(t, root.ToBytes(), trieDB.rootHash.Bytes())

	t.Run("iterate_over_all_entries", func(t *testing.T) {
		iter, err := NewTrieDBRawIterator(trieDB)
		assert.NoError(t, err)

		expected := inMemoryTrie.NextKey([]byte{})
		i := 0
		for {
			item, err := iter.NextItem()
			assert.NoError(t, err)
			if item == nil {
				break
			}
			assert.Equal(t, expected, item.Key)
			expected = inMemoryTrie.NextKey(expected)
			i++
		}
		assert.Equal(t, len(entries), i)
	})

	t.Run("iterate_after_seeking", func(t *testing.T) {
		iter, err := NewTrieDBRawIterator(trieDB)
		assert.NoError(t, err)

		found, err := iter.seek([]byte("not"), true)
		assert.NoError(t, err)
		assert.True(t, found)

		expected := inMemoryTrie.NextKey([]byte("not"))
		actual, err := iter.NextItem()
		assert.NoError(t, err)
		assert.NotNil(t, actual)

		assert.Equal(t, []byte("not"), actual.Key)
		actual, err = iter.NextItem()
		assert.NoError(t, err)
		assert.NotNil(t, actual)
		assert.Equal(t, expected, actual.Key)
	})
}
