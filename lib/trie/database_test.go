// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package trie

import (
	"testing"

	"github.com/ChainSafe/gossamer/internal/database"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestDB(t *testing.T) database.Table {
	db, err := database.NewPebble("", true)
	require.NoError(t, err)
	return database.NewTable(db, "trie")
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

		trie.Put(key, value, V0)

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
	trie.Put(existingKey, newValue, V0)
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

	var (
		merkleValue32Zeroes = common.Hash{}
		merkleValue32Ones   = common.Hash{
			1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1,
			1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1}
		merkleValue32Twos = common.Hash{
			2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2,
			2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2}
		merkleValue32Threes = common.Hash{
			3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3,
			3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3}
	)

	testCases := map[string]struct {
		node       *Node
		nodeHashes map[common.Hash]struct{}
		panicValue interface{}
	}{
		"nil_node": {
			nodeHashes: map[common.Hash]struct{}{},
		},
		"inlined_leaf_node": {
			node:       &Node{MerkleValue: []byte("a")},
			nodeHashes: map[common.Hash]struct{}{},
		},
		"leaf_node": {
			node: &Node{MerkleValue: merkleValue32Zeroes.ToBytes()},
			nodeHashes: map[common.Hash]struct{}{
				merkleValue32Zeroes: {},
			},
		},
		"leaf_node_without_Merkle_value": {
			node:       &Node{PartialKey: []byte{1}, StorageValue: []byte{2}},
			panicValue: "node with partial key 0x01 has no Merkle value computed",
		},
		"inlined_branch_node": {
			node: &Node{
				MerkleValue: []byte("a"),
				Children: padRightChildren([]*Node{
					{MerkleValue: []byte("b")},
				}),
			},
			nodeHashes: map[common.Hash]struct{}{},
		},
		"branch_node": {
			node: &Node{
				MerkleValue: merkleValue32Zeroes.ToBytes(),
				Children: padRightChildren([]*Node{
					{MerkleValue: merkleValue32Ones.ToBytes()},
				}),
			},
			nodeHashes: map[common.Hash]struct{}{
				merkleValue32Zeroes: {},
				merkleValue32Ones:   {},
			},
		},
		"nested_branch_node": {
			node: &Node{
				MerkleValue: merkleValue32Zeroes.ToBytes(),
				Children: padRightChildren([]*Node{
					{MerkleValue: merkleValue32Ones.ToBytes()},
					{
						MerkleValue: merkleValue32Twos.ToBytes(),
						Children: padRightChildren([]*Node{
							{MerkleValue: merkleValue32Threes.ToBytes()},
						}),
					},
				}),
			},
			nodeHashes: map[common.Hash]struct{}{
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

			nodeHashes := make(map[common.Hash]struct{})

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
