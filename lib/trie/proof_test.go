// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package trie

import (
	"testing"

	"github.com/ChainSafe/chaindb"
	"github.com/stretchr/testify/require"
)

func TestProofGeneration(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()

	memdb, err := chaindb.NewBadgerDB(&chaindb.Config{
		InMemory: true,
		DataDir:  tmp,
	})
	require.NoError(t, err)

	const size = 32
	generator := newGenerator()

	expectedValue := generateRandBytes(t, size, generator)

	trie := NewEmptyTrie()
	trie.Put([]byte("cat"), generateRandBytes(t, size, generator))
	trie.Put([]byte("catapulta"), generateRandBytes(t, size, generator))
	trie.Put([]byte("catapora"), expectedValue)
	trie.Put([]byte("dog"), generateRandBytes(t, size, generator))
	trie.Put([]byte("doguinho"), generateRandBytes(t, size, generator))

	err = trie.Store(memdb)
	require.NoError(t, err)

	hash, err := trie.Hash()
	require.NoError(t, err)

	proof, err := GenerateProof(hash.ToBytes(), [][]byte{[]byte("catapulta"), []byte("catapora")}, memdb)
	require.NoError(t, err)

	require.Equal(t, 5, len(proof))

	pl := []Pair{
		{Key: []byte("catapora"), Value: expectedValue},
	}

	v, err := VerifyProof(proof, hash.ToBytes(), pl)
	require.True(t, v)
	require.NoError(t, err)
}

func testGenerateProof(t *testing.T, entries []Pair, keys [][]byte) ([]byte, [][]byte, []Pair) {
	t.Helper()

	tmp := t.TempDir()

	memdb, err := chaindb.NewBadgerDB(&chaindb.Config{
		InMemory: true,
		DataDir:  tmp,
	})
	require.NoError(t, err)

	trie := NewEmptyTrie()
	for _, e := range entries {
		trie.Put(e.Key, e.Value)
	}

	err = trie.Store(memdb)
	require.NoError(t, err)

	root := trie.root.GetHash()
	proof, err := GenerateProof(root, keys, memdb)
	require.NoError(t, err)

	items := make([]Pair, len(keys))
	for idx, key := range keys {
		value := trie.Get(key)
		require.NotNil(t, value)

		items[idx] = Pair{
			Key:   key,
			Value: value,
		}
	}

	return root, proof, items
}

func TestVerifyProof_ShouldReturnTrue(t *testing.T) {
	t.Parallel()

	entries := []Pair{
		{Key: []byte("alpha"), Value: make([]byte, 32)},
		{Key: []byte("bravo"), Value: []byte("bravo")},
		{Key: []byte("do"), Value: []byte("verb")},
		{Key: []byte("dogea"), Value: []byte("puppy")},
		{Key: []byte("dogeb"), Value: []byte("puppy")},
		{Key: []byte("horse"), Value: []byte("stallion")},
		{Key: []byte("house"), Value: []byte("building")},
	}

	keys := [][]byte{
		[]byte("do"),
		[]byte("dogea"),
		[]byte("dogeb"),
	}

	root, proof, pairs := testGenerateProof(t, entries, keys)
	v, err := VerifyProof(proof, root, pairs)

	require.NoError(t, err)
	require.True(t, v)
}

func TestVerifyProof_ShouldReturnDuplicateKeysError(t *testing.T) {
	t.Parallel()

	pl := []Pair{
		{Key: []byte("do"), Value: []byte("verb")},
		{Key: []byte("do"), Value: []byte("puppy")},
	}

	v, err := VerifyProof([][]byte{}, []byte{}, pl)
	require.False(t, v)
	require.Error(t, err, ErrDuplicateKeys)
}

func TestVerifyProof_ShouldReturnTrueWithouCompareValues(t *testing.T) {
	t.Parallel()

	entries := []Pair{
		{Key: []byte("alpha"), Value: make([]byte, 32)},
		{Key: []byte("bravo"), Value: []byte("bravo")},
		{Key: []byte("do"), Value: []byte("verb")},
		{Key: []byte("dog"), Value: []byte("puppy")},
		{Key: []byte("doge"), Value: make([]byte, 32)},
		{Key: []byte("horse"), Value: []byte("stallion")},
		{Key: []byte("house"), Value: []byte("building")},
	}

	keys := [][]byte{
		[]byte("do"),
		[]byte("dog"),
		[]byte("doge"),
	}

	root, proof, _ := testGenerateProof(t, entries, keys)

	pl := []Pair{
		{Key: []byte("do"), Value: nil},
		{Key: []byte("dog"), Value: nil},
		{Key: []byte("doge"), Value: nil},
	}

	v, err := VerifyProof(proof, root, pl)
	require.True(t, v)
	require.NoError(t, err)
}

func TestBranchNodes_SameHash_DifferentPaths_GenerateAndVerifyProof(t *testing.T) {
	value := []byte("somevalue")
	entries := []Pair{
		{Key: []byte("d"), Value: value},
		{Key: []byte("b"), Value: value},
		{Key: []byte("dxyz"), Value: value},
		{Key: []byte("bxyz"), Value: value},
		{Key: []byte("dxyzi"), Value: value},
		{Key: []byte("bxyzi"), Value: value},
	}

	keys := [][]byte{
		[]byte("d"),
		[]byte("b"),
		[]byte("dxyz"),
		[]byte("bxyz"),
		[]byte("dxyzi"),
		[]byte("bxyzi"),
	}

	root, proof, pairs := testGenerateProof(t, entries, keys)

	ok, err := VerifyProof(proof, root, pairs)
	require.NoError(t, err)
	require.True(t, ok)
}

func TestLeafNodes_SameHash_DifferentPaths_GenerateAndVerifyProof(t *testing.T) {
	tmp := t.TempDir()

	memdb, err := chaindb.NewBadgerDB(&chaindb.Config{
		InMemory: true,
		DataDir:  tmp,
	})
	require.NoError(t, err)

	var (
		value = []byte("somevalue")
		key1  = []byte("worlda")
		key2  = []byte("worldb")
	)

	tt := NewEmptyTrie()
	tt.Put(key1, value)
	tt.Put(key2, value)

	err = tt.Store(memdb)
	require.NoError(t, err)

	hash, err := tt.Hash()
	require.NoError(t, err)

	proof, err := GenerateProof(hash.ToBytes(), [][]byte{key1, key2}, memdb)
	require.NoError(t, err)

	pairs := []Pair{
		{Key: key1, Value: value},
		{Key: key2, Value: value},
	}

	ok, err := VerifyProof(proof, hash.ToBytes(), pairs)
	require.NoError(t, err)
	require.True(t, ok)
}
