// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package state

import (
	"bytes"
	"testing"

	"github.com/ChainSafe/gossamer/internal/database"
	"github.com/ChainSafe/gossamer/internal/trie/node"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/trie"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func Test_NewTrieDB(t *testing.T) {
	t.Parallel()

	db := NewInMemoryDB(t)
	trieDB := NewTrieDB(db)

	expected := &TrieDB{
		db: db,
		tries: &tries{
			rootToTrie:    map[common.Hash]*trie.Trie{},
			triesGauge:    triesGauge,
			setCounter:    setCounter,
			deleteCounter: deleteCounter,
		},
		trieDBGetDuration: trieDBGetDuration,
	}

	assert.Equal(t, expected, trieDB)
}

func Test_Get(t *testing.T) {
	root := &trie.Node{
		PartialKey:   []byte{0},
		StorageValue: []byte{17},
		Dirty:        true,
	}

	// Create trie DB using in memory db table
	db := NewInMemoryDB(t)
	table := database.NewTable(db, "storage")
	trieDB := NewTrieDB(table)

	// Build trie with test root node
	testTrie := trie.NewTrie(root, table)

	// Store trie in trieDB
	err := trieDB.Put(testTrie)
	assert.NoError(t, err)

	// Encode trie to check later
	encoded := bytes.NewBuffer(nil)
	err = root.Encode(encoded, trie.NoMaxInlineValueSize)
	assert.NoError(t, err)

	ctrl := gomock.NewController(t)

	t.Run("get_and_update_histogram", func(t *testing.T) {
		t.Parallel()

		trieDBGetDuration := NewMockHistogram(ctrl)

		trieDB := TrieDB{
			db:                table,
			tries:             newTries(),
			trieDBGetDuration: trieDBGetDuration,
		}

		trieDBGetDuration.EXPECT().Observe(gomock.Any()).AnyTimes()

		_, err = trieDB.Get(trie.V0.MustHash(*testTrie))
		assert.NoError(t, err)
	})

	t.Run("decode trie and refresh cache", func(t *testing.T) {
		t.Parallel()
		// Cache should be empty
		assert.Len(t, trieDB.tries.rootToTrie, 0)

		// Get trie from trieDB table and check if it matches the encoded trie
		trieFromTrieDB, err := trieDB.Get(trie.V0.MustHash(*testTrie))
		assert.NoError(t, err)
		assert.Equal(t, testTrie.String(), trieFromTrieDB.String())

		// Trie should be added to cache
		assert.Len(t, trieDB.tries.rootToTrie, 1)

		// Get from cache
		fromCache := trieDB.tries.get(trie.V0.MustHash(*testTrie))
		assert.Equal(t, testTrie.String(), fromCache.String())
	})
}

func Test_Put(t *testing.T) {
	t.Parallel()

	testNode := &trie.Node{
		PartialKey:   []byte{0},
		StorageValue: []byte{17},
		Dirty:        true,
	}

	testCases := map[string]struct {
		trie    *trie.Trie
		encoded []byte
		success bool
		err     string
	}{
		"dirty_trie_should_be_stored_encoded": {
			trie: trie.NewTrie(testNode, nil),
			encoded: func() []byte {
				encoded := bytes.NewBuffer(nil)
				err := testNode.Encode(encoded, trie.NoMaxInlineValueSize)
				assert.NoError(t, err)

				return encoded.Bytes()
			}(),
			success: true,
			err:     "",
		},
		"do_not_store_not_dirty_nodes": {
			trie: func() *trie.Trie {
				notDirty := testNode.Copy(node.DeepCopySettings)
				notDirty.Dirty = false
				return trie.NewTrie(notDirty, nil)
			}(),
			encoded: func() []byte {
				encoded := bytes.NewBuffer(nil)
				err := testNode.Encode(encoded, trie.NoMaxInlineValueSize)
				assert.NoError(t, err)

				return encoded.Bytes()
			}(),
			success: false,
			err:     "not found",
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			db := NewInMemoryDB(t)
			table := database.NewTable(db, "storage")
			trieDB := NewTrieDB(table)

			err := trieDB.Put(testCase.trie)
			assert.NoError(t, err)

			trieFromDB, err := table.Get(trie.V0.MustHash(*testCase.trie).ToBytes())
			if testCase.success {
				assert.NoError(t, err)
				assert.Equal(t, testCase.encoded, trieFromDB)
			} else {
				assert.ErrorContains(t, err, testCase.err)
			}
		})
	}
}

func Test_GetDeletedTrieFromDBShouldReturnError(t *testing.T) {
	t.Parallel()

	// Create trie DB using in memory db table
	db := NewInMemoryDB(t)
	table := database.NewTable(db, "storage")
	trieDB := NewTrieDB(table)

	// Build trie with test root node
	root := &trie.Node{
		PartialKey:   []byte{0},
		StorageValue: []byte{17},
		Dirty:        true,
	}
	testTrie := trie.NewTrie(root, table)

	// Encode trie to check later
	encoded := bytes.NewBuffer(nil)
	err := root.Encode(encoded, trie.NoMaxInlineValueSize)
	assert.NoError(t, err)

	// Store trie in trieDB
	err = trieDB.Put(testTrie)
	assert.NoError(t, err)

	// Get trie from trieDB table and check if it matches the encoded trie
	trieFromTrieDB, err := trieDB.Get(trie.V0.MustHash(*testTrie))
	assert.NoError(t, err)
	assert.Equal(t, testTrie.String(), trieFromTrieDB.String())

	// Delete trie and try to get it again should return an error
	err = trieDB.Delete(trie.V0.MustHash(*testTrie))
	assert.NoError(t, err)

	_, err = trieDB.Get(trie.V0.MustHash(*testTrie))
	assert.ErrorContains(t, err, "not found")
}

func Test_GetKeyFromTrie(t *testing.T) {
	t.Parallel()

	testKey := []byte("testKey")
	testValue := []byte("testValue")

	// Create trie DB using in memory db table
	db := NewInMemoryDB(t)
	table := database.NewTable(db, "storage")
	trieDB := NewTrieDB(table)

	// Build trie with test root node
	root := &trie.Node{
		PartialKey:   []byte{0},
		StorageValue: []byte{17},
		Dirty:        true,
	}
	testTrie := trie.NewTrie(root, table)
	testTrie.Put(testKey, testValue)

	// Encode trie to check later
	encoded := bytes.NewBuffer(nil)
	err := root.Encode(encoded, trie.NoMaxInlineValueSize)
	assert.NoError(t, err)

	// Store trie in trieDB
	err = trieDB.Put(testTrie)
	assert.NoError(t, err)

	// Get trie from trieDB table and check if it matches the encoded trie
	valueFromTrie, err := trieDB.GetKey(trie.V0.MustHash(*testTrie), testKey)
	assert.NoError(t, err)
	assert.Equal(t, testValue, valueFromTrie)
}
