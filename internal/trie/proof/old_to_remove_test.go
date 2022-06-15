// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package proof

import (
	"testing"

	"github.com/ChainSafe/chaindb"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/trie"
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

	trie := trie.NewEmptyTrie()
	trie.Put([]byte("cat"), generateRandBytes(t, size, generator))
	trie.Put([]byte("catapulta"), generateRandBytes(t, size, generator))
	trie.Put([]byte("catapora"), expectedValue)
	trie.Put([]byte("dog"), generateRandBytes(t, size, generator))
	trie.Put([]byte("doguinho"), generateRandBytes(t, size, generator))

	err = trie.Store(memdb)
	require.NoError(t, err)

	hash, err := trie.Hash()
	require.NoError(t, err)

	proofCatapulta, err := Generate(hash, []byte("catapulta"), memdb)
	require.NoError(t, err)
	require.Equal(t, 4, len(proofCatapulta))
	proofCatapora, err := Generate(hash, []byte("catapora"), memdb)
	require.NoError(t, err)
	require.Equal(t, 4, len(proofCatapulta))

	proof := append(proofCatapulta, proofCatapora...)

	err = Verify(proof, hash.ToBytes(), []byte("catapora"), expectedValue)
	require.NoError(t, err)
}

type keyValue struct {
	Key   []byte
	Value []byte
}

func testGenerateProof(t *testing.T, entries []keyValue, keys [][]byte) ([]byte, [][]byte, []keyValue) {
	t.Helper()

	tmp := t.TempDir()

	memdb, err := chaindb.NewBadgerDB(&chaindb.Config{
		InMemory: true,
		DataDir:  tmp,
	})
	require.NoError(t, err)

	trie := trie.NewEmptyTrie()
	for _, e := range entries {
		trie.Put(e.Key, e.Value)
	}

	err = trie.Store(memdb)
	require.NoError(t, err)

	root := trie.RootNode().HashDigest

	var proof [][]byte
	for _, key := range keys {
		keyProof, err := Generate(common.BytesToHash(root), key, memdb)
		require.NoError(t, err)
		proof = append(proof, keyProof...)
	}

	items := make([]keyValue, len(keys))
	for idx, key := range keys {
		value := trie.Get(key)
		require.NotNil(t, value)

		items[idx] = keyValue{
			Key:   key,
			Value: value,
		}
	}

	return root, proof, items
}

func TestVerifyProof_ShouldReturnTrue(t *testing.T) {
	t.Parallel()

	entries := []keyValue{
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

	for _, keyValue := range pairs {
		err := Verify(proof, root, keyValue.Key, keyValue.Value)
		require.NoError(t, err)
	}
}

func TestVerifyProof_ShouldReturnTrueWithouCompareValues(t *testing.T) {
	t.Parallel()

	entries := []keyValue{
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

	pl := []keyValue{
		{Key: []byte("do"), Value: nil},
		{Key: []byte("dog"), Value: nil},
		{Key: []byte("doge"), Value: nil},
	}

	for _, keyValue := range pl {
		err := Verify(proof, root, keyValue.Key, keyValue.Value)
		require.NoError(t, err)
	}
}

func TestBranchNodes_SameHash_DifferentPaths_GenerateAndVerifyProof(t *testing.T) {
	value := []byte("somevalue")
	entries := []keyValue{
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

	for _, pair := range pairs {
		err := Verify(proof, root, pair.Key, pair.Value)
		require.NoError(t, err)
	}
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

	tt := trie.NewEmptyTrie()
	tt.Put(key1, value)
	tt.Put(key2, value)

	err = tt.Store(memdb)
	require.NoError(t, err)

	hash, err := tt.Hash()
	require.NoError(t, err)

	proofKey1, err := Generate(hash, key1, memdb)
	require.NoError(t, err)

	proofKey2, err := Generate(hash, key2, memdb)
	require.NoError(t, err)

	proof := append(proofKey1, proofKey2...)

	pairs := []keyValue{
		{Key: key1, Value: value},
		{Key: key2, Value: value},
	}

	for _, pair := range pairs {
		err := Verify(proof, hash.ToBytes(), pair.Key, pair.Value)
		require.NoError(t, err)
	}
}
