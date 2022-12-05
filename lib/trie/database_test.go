// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package trie

import (
	"testing"

	"github.com/ChainSafe/chaindb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestDB(t *testing.T) chaindb.Database {
	chainDBConfig := &chaindb.Config{
		InMemory: true,
	}
	database, err := chaindb.NewBadgerDB(chainDBConfig)
	require.NoError(t, err)
	return chaindb.NewTable(database, "trie")
}

func Test_Trie_Store_Load(t *testing.T) {
	t.Parallel()

	const size = 1000
	trie, _ := makeSeededTrie(t, size)

	rootHash := trie.MustHash()

	db := newTestDB(t)
	err := trie.WriteDirty(db)
	require.NoError(t, err)

	trieFromDB := NewEmptyTrie()
	err = trieFromDB.Load(db, rootHash)
	require.NoError(t, err)
	assert.Equal(t, trie.String(), trieFromDB.String())
}

func Test_Trie_WriteDirty_Put(t *testing.T) {
	t.Parallel()

	generator := newGenerator()
	const size = 500
	keyValues := generateKeyValues(t, generator, size)

	trie := NewEmptyTrie()

	db := newTestDB(t)

	// Put, write dirty and get from DB
	for keyString, value := range keyValues {
		key := []byte(keyString)

		trie.Put(key, value)

		err := trie.WriteDirty(db)
		require.NoError(t, err)

		rootHash := trie.MustHash()
		valueFromDB, err := GetFromDB(db, rootHash, key)
		require.NoError(t, err)
		assert.Equalf(t, value, valueFromDB, "for key=%x", key)
	}

	err := trie.WriteDirty(db)
	require.NoError(t, err)

	// Pick an existing key and replace its value
	oneKeySet := pickKeys(keyValues, generator, 1)
	existingKey := oneKeySet[0]
	existingValue := keyValues[string(existingKey)]
	newValue := make([]byte, len(existingValue))
	copy(newValue, existingValue)
	newValue = append(newValue, 99)
	trie.Put(existingKey, newValue)
	err = trie.WriteDirty(db)
	require.NoError(t, err)

	rootHash := trie.MustHash()

	// Verify the trie in database is also modified.
	trieFromDB := NewEmptyTrie()
	err = trieFromDB.Load(db, rootHash)
	require.NoError(t, err)
	require.Equal(t, trie.String(), trieFromDB.String())
	value, err := GetFromDB(db, rootHash, existingKey)
	require.NoError(t, err)
	assert.Equal(t, newValue, value)
}

func Test_Trie_WriteDirty_Delete(t *testing.T) {
	t.Parallel()

	const size = 1000
	trie, keyValues := makeSeededTrie(t, size)

	generator := newGenerator()
	keysToDelete := pickKeys(keyValues, generator, size/50)

	db := newTestDB(t)
	err := trie.WriteDirty(db)
	require.NoError(t, err)

	deletedKeys := make(map[string]struct{}, len(keysToDelete))
	for _, keyToDelete := range keysToDelete {
		trie.Delete(keyToDelete)
		err = trie.WriteDirty(db)
		require.NoError(t, err)

		deletedKeys[string(keyToDelete)] = struct{}{}
	}

	rootHash := trie.MustHash()

	trieFromDB := NewEmptyTrie()
	err = trieFromDB.Load(db, rootHash)
	require.NoError(t, err)
	require.Equal(t, trie.String(), trieFromDB.String())

	for keyString, expectedValue := range keyValues {
		if _, deleted := deletedKeys[keyString]; deleted {
			expectedValue = nil
		}

		key := []byte(keyString)
		value, err := GetFromDB(db, rootHash, key)
		require.NoError(t, err)
		assert.Equal(t, expectedValue, value)
	}
}

func Test_Trie_WriteDirty_ClearPrefix(t *testing.T) {
	t.Parallel()

	const size = 2000
	trie, keyValues := makeSeededTrie(t, size)

	generator := newGenerator()
	keysToClearPrefix := pickKeys(keyValues, generator, size/50)

	db := newTestDB(t)
	err := trie.WriteDirty(db)
	require.NoError(t, err)

	for _, keyToClearPrefix := range keysToClearPrefix {
		trie.ClearPrefix(keyToClearPrefix)
		err = trie.WriteDirty(db)
		require.NoError(t, err)
	}

	rootHash := trie.MustHash()

	trieFromDB := NewEmptyTrie()
	err = trieFromDB.Load(db, rootHash)
	require.NoError(t, err)
	assert.Equal(t, trie.String(), trieFromDB.String())
}

func Test_PopulateNodeHashes(t *testing.T) {
	t.Parallel()

	const (
		merkleValue32Zeroes = "00000000000000000000000000000000"
		merkleValue32Ones   = "11111111111111111111111111111111"
		merkleValue32Twos   = "22222222222222222222222222222222"
		merkleValue32Threes = "33333333333333333333333333333333"
	)

	testCases := map[string]struct {
		node       *Node
		nodeHashes map[string]struct{}
		panicValue interface{}
	}{
		"nil node": {
			nodeHashes: map[string]struct{}{},
		},
		"inlined leaf node": {
			node:       &Node{MerkleValue: []byte("a")},
			nodeHashes: map[string]struct{}{},
		},
		"leaf node": {
			node: &Node{MerkleValue: []byte(merkleValue32Zeroes)},
			nodeHashes: map[string]struct{}{
				merkleValue32Zeroes: {},
			},
		},
		"leaf node without Merkle value": {
			node:       &Node{PartialKey: []byte{1}, StorageValue: []byte{2}},
			panicValue: "node with partial key 0x01 has no Merkle value computed",
		},
		"inlined branch node": {
			node: &Node{
				MerkleValue: []byte("a"),
				Children: padRightChildren([]*Node{
					{MerkleValue: []byte("b")},
				}),
			},
			nodeHashes: map[string]struct{}{},
		},
		"branch node": {
			node: &Node{
				MerkleValue: []byte(merkleValue32Zeroes),
				Children: padRightChildren([]*Node{
					{MerkleValue: []byte(merkleValue32Ones)},
				}),
			},
			nodeHashes: map[string]struct{}{
				merkleValue32Zeroes: {},
				merkleValue32Ones:   {},
			},
		},
		"nested branch node": {
			node: &Node{
				MerkleValue: []byte(merkleValue32Zeroes),
				Children: padRightChildren([]*Node{
					{MerkleValue: []byte(merkleValue32Ones)},
					{
						MerkleValue: []byte(merkleValue32Twos),
						Children: padRightChildren([]*Node{
							{MerkleValue: []byte(merkleValue32Threes)},
						}),
					},
				}),
			},
			nodeHashes: map[string]struct{}{
				merkleValue32Zeroes: {},
				merkleValue32Ones:   {},
				merkleValue32Twos:   {},
				merkleValue32Threes: {},
			},
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			nodeHashes := make(map[string]struct{})

			if testCase.panicValue != nil {
				assert.PanicsWithValue(t, testCase.panicValue, func() {
					PopulateNodeHashes(testCase.node, nodeHashes)
				})
				return
			}

			PopulateNodeHashes(testCase.node, nodeHashes)

			assert.Equal(t, testCase.nodeHashes, nodeHashes)
		})
	}
}

func Test_GetFromDB(t *testing.T) {
	t.Parallel()

	const size = 1000
	trie, keyValues := makeSeededTrie(t, size)

	db := newTestDB(t)
	err := trie.WriteDirty(db)
	require.NoError(t, err)

	root := trie.MustHash()

	for keyString, expectedValue := range keyValues {
		key := []byte(keyString)
		value, err := GetFromDB(db, root, key)
		assert.NoError(t, err)
		assert.Equal(t, expectedValue, value)
	}
}

func Test_Trie_PutChild_Store_Load(t *testing.T) {
	t.Parallel()

	const size = 100
	trie, _ := makeSeededTrie(t, size)

	const childTrieSize = 10
	childTrie, _ := makeSeededTrie(t, childTrieSize)

	db := newTestDB(t)

	// the hash is equal to the key if the key is less or equal to 32 bytes
	// and is the blake2b hash of the encoding of the node otherwise.
	// This is why we test with keys greater and smaller than 32 bytes below.
	keysToChildTries := [][]byte{
		[]byte("012345678901234567890123456789013"), // 33 bytes
		[]byte("01234567890123456789012345678901"),  // 32 bytes
		[]byte("0123456789012345678901234567890"),   // 31 bytes
	}

	for _, keyToChildTrie := range keysToChildTries {
		err := trie.SetChild(keyToChildTrie, childTrie)
		require.NoError(t, err)

		err = trie.WriteDirty(db)
		require.NoError(t, err)

		trieFromDB := NewEmptyTrie()
		err = trieFromDB.Load(db, trie.MustHash())
		require.NoError(t, err)

		assert.Equal(t, trie.childTries, trieFromDB.childTries)
		assert.Equal(t, trie.String(), trieFromDB.String())
	}
}
