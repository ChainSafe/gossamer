// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package triedb

import (
	"testing"

	"github.com/ChainSafe/gossamer/internal/database"
	"github.com/ChainSafe/gossamer/pkg/trie"
	"github.com/ChainSafe/gossamer/pkg/trie/inmemory"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestDB(t *testing.T) database.Table {
	db, err := database.NewPebble("", true)
	require.NoError(t, err)
	return database.NewTable(db, "trie")
}

func TestTrieDB_Migration(t *testing.T) {
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
	trieDB := NewTrieDB(root, db)

	t.Run("read_successful_from_db_created_using_v1_trie", func(t *testing.T) {
		for k, v := range entries {
			value := trieDB.Get([]byte(k))
			assert.Equal(t, v, value)
		}

		assert.Equal(t, root, trieDB.MustHash())
	})
	t.Run("next_key", func(t *testing.T) {
		key := []byte("no")

		for key != nil {
			expected := inMemoryTrie.NextKey(key)
			actual := trieDB.NextKey(key)
			assert.Equal(t, expected, actual)

			key = actual
		}

	})
}

func TestTrieDB_Lookup(t *testing.T) {
	t.Run("root_not_exists_in_db", func(t *testing.T) {
		db := newTestDB(t)
		trieDB := NewTrieDB(trie.EmptyHash, db)

		value, err := trieDB.lookup([]byte("test"))
		assert.Nil(t, value)
		assert.ErrorIs(t, err, ErrIncompleteDB)
	})
}
