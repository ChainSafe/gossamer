// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package triedb

import (
	"testing"

	"github.com/ChainSafe/gossamer/internal/database"
	"github.com/ChainSafe/gossamer/pkg/trie"
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

	// Use at least 1 value with more than 32 bytes to test trie V1
	entries := map[string][]byte{
		"no":           make([]byte, 10),
		"noot":         make([]byte, 20),
		"not":          make([]byte, 30),
		"notable":      make([]byte, 40),
		"notification": make([]byte, 50),
		"test":         make([]byte, 60),
		"dimartiro":    make([]byte, 70),
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
}
