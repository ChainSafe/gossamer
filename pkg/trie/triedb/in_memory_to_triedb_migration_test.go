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

func TestTrieDB_Get(t *testing.T) {
	t.Run("Read successful from db created using v1 trie", func(t *testing.T) {
		db := newTestDB(t)
		inMemoryTrie := inmemory.NewEmptyTrie()
		inMemoryTrie.SetVersion(trie.V1)

		entries := map[string][]byte{
			"no":           make([]byte, 20),
			"not":          make([]byte, 40),
			"nothing":      make([]byte, 20),
			"notification": make([]byte, 40),
			"test":         make([]byte, 40),
		}

		for k, v := range entries {
			inMemoryTrie.Put([]byte(k), v)
		}

		err := inMemoryTrie.WriteDirty(db)
		assert.NoError(t, err)

		root, err := inMemoryTrie.Hash()
		assert.NoError(t, err)

		trieDB := NewTrieDB(root, db)

		for k, v := range entries {
			value := trieDB.Get([]byte(k))
			assert.Equal(t, v, value)
		}

		assert.Equal(t, root, trieDB.MustHash())
	})
}

func TestTrieDB_Lookup(t *testing.T) {
	t.Run("Root not exists in db", func(t *testing.T) {
		db := newTestDB(t)
		trieDB := NewTrieDB(trie.EmptyHash, db)

		value, err := trieDB.lookup([]byte("test"))
		assert.Nil(t, value)
		assert.ErrorIs(t, err, ErrIncompleteDB)
	})
}
