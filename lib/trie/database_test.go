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
	err := trie.Store(db)
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

	err := trie.Store(db)
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
	err := trie.Store(db)
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
	err := trie.Store(db)
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

func Test_PopulateMerkleValues(t *testing.T) {
	t.Parallel()

	someNode := &Node{Key: []byte{1}, SubValue: []byte{2}}

	testCases := map[string]struct {
		trie         *Trie
		node         *Node
		merkleValues map[string]struct{}
		errSentinel  error
		errMessage   string
	}{
		"nil node": {
			trie:         &Trie{},
			merkleValues: map[string]struct{}{},
		},
		"leaf node": {
			trie: &Trie{},
			node: &Node{MerkleValue: []byte("a")},
			merkleValues: map[string]struct{}{
				"a": {},
			},
		},
		"leaf node without Merkle value": {
			trie: &Trie{},
			node: &Node{Key: []byte{1}, SubValue: []byte{2}},
			merkleValues: map[string]struct{}{
				"A\x01\x04\x02": {},
			},
		},
		"root leaf node without Merkle value": {
			trie: &Trie{
				root: someNode,
			},
			node: someNode,
			merkleValues: map[string]struct{}{
				"`Qm\v\xb6\xe1\xbb\xfb\x12\x93\xf1\xb2v\xea\x95\x05\xe9\xf4\xa4\xe7ُb\r\x05\x11^\v\x85'J\xe1": {},
			},
		},
		"branch node": {
			trie: &Trie{},
			node: &Node{
				MerkleValue: []byte("a"),
				Children: padRightChildren([]*Node{
					{MerkleValue: []byte("b")},
				}),
			},
			merkleValues: map[string]struct{}{
				"a": {},
				"b": {},
			},
		},
		"nested branch node": {
			trie: &Trie{},
			node: &Node{
				MerkleValue: []byte("a"),
				Children: padRightChildren([]*Node{
					{MerkleValue: []byte("b")},
					{
						MerkleValue: []byte("c"),
						Children: padRightChildren([]*Node{
							{MerkleValue: []byte("d")},
						}),
					},
				}),
			},
			merkleValues: map[string]struct{}{
				"a": {},
				"b": {},
				"c": {},
				"d": {},
			},
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			merkleValues := make(map[string]struct{})

			err := testCase.trie.PopulateMerkleValues(testCase.node, merkleValues)

			assert.ErrorIs(t, err, testCase.errSentinel)
			if testCase.errSentinel != nil {
				assert.EqualError(t, err, testCase.errMessage)
			}
			assert.Equal(t, testCase.merkleValues, merkleValues)
		})
	}
}

func Test_GetFromDB(t *testing.T) {
	t.Parallel()

	const size = 1000
	trie, keyValues := makeSeededTrie(t, size)

	db := newTestDB(t)
	err := trie.Store(db)
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
		err := trie.PutChild(keyToChildTrie, childTrie)
		require.NoError(t, err)

		err = trie.Store(db)
		require.NoError(t, err)

		trieFromDB := NewEmptyTrie()
		err = trieFromDB.Load(db, trie.MustHash())
		require.NoError(t, err)

		assert.Equal(t, trie.childTries, trieFromDB.childTries)
		assert.Equal(t, trie.String(), trieFromDB.String())
	}
}
