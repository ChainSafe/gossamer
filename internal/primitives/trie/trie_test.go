// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package trie

import (
	"testing"

	hashdb "github.com/ChainSafe/gossamer/internal/hash-db"
	"github.com/ChainSafe/gossamer/internal/primitives/core/hash"
	"github.com/ChainSafe/gossamer/internal/primitives/core/hasher"
	"github.com/ChainSafe/gossamer/pkg/trie/triedb"
	"github.com/stretchr/testify/require"
)

var (
	_ hashdb.HashDB[hash.H256] = &KeyspacedDB[hash.H256]{}
)

func checkEquivalent(t *testing.T, input []KeyValue, layout Layout[hash.H256]) {
	closedForm := layout.TrieRoot(input)

	t.Logf("%s", closedForm)

	memDB := NewMemoryDB[hash.H256, hasher.Blake2Hasher]()
	trieDB := triedb.NewEmptyTrieDB[hash.H256, hasher.Blake2Hasher](memDB, layout)
	for i := len(input) - 1; i >= 0; i-- {
		kv := input[i]
		trieDB.Set(kv.Key, kv.Value)
	}
	persistent := trieDB.MustHash()
	require.Equal(t, closedForm, persistent)
}
func checkInput(t *testing.T, input []KeyValue) {
	checkEquivalent(t, input, LayoutV0[hasher.Blake2Hasher, hash.H256]{})
	checkEquivalent(t, input, LayoutV1[hasher.Blake2Hasher, hash.H256]{})
}

func TestTrieRoot(t *testing.T) {
	t.Run("default_trie_root", func(t *testing.T) {
		db := NewMemoryDB[hash.H256, hasher.Blake2Hasher]()
		empty := triedb.NewEmptyTrieDB[hash.H256, hasher.Blake2Hasher](db, LayoutV1[hasher.Blake2Hasher, hash.H256]{})
		root1 := empty.MustHash()
		root2 := LayoutV1[hasher.Blake2Hasher, hash.H256]{}.TrieRoot(nil)
		require.Equal(t, root1, root2)
	})

	t.Run("empty_is_equivalent", func(t *testing.T) {
		checkInput(t, nil)
	})

	t.Run("leaf_is_equivalent", func(t *testing.T) {
		checkInput(t, []KeyValue{
			{Key: []byte{0xaa}, Value: []byte{0xbb}},
		})
	})

	t.Run("branch_is_equivalent", func(t *testing.T) {
		input := []KeyValue{
			{Key: []byte{0xaa}, Value: []byte{0x10}},
			{Key: []byte{0xba}, Value: []byte{0x11}},
		}
		checkInput(t, input)
	})

	t.Run("extension_and_branch_is_equivalent", func(t *testing.T) {
		input := []KeyValue{
			{Key: []byte{0xaa}, Value: []byte{0x10}},
			{Key: []byte{0xab}, Value: []byte{0x11}},
		}
		checkInput(t, input)
	})

	t.Run("extension_and_branch_with_value_is_equivalent", func(t *testing.T) {
		input := []KeyValue{
			{Key: []byte{0xaa}, Value: []byte{0xa0}},
			{Key: []byte{0xaa, 0xaa}, Value: []byte{0xaa}},
			{Key: []byte{0xaa, 0xbb}, Value: []byte{0xab}},
		}
		checkInput(t, input)
	})

	t.Run("bigger_extension_and_branch_with_value_is_equivalent", func(t *testing.T) {
		input := []KeyValue{
			{Key: []byte{0xaa}, Value: []byte{0xa0}},
			{Key: []byte{0xaa, 0xaa}, Value: []byte{0xaa}},
			{Key: []byte{0xaa, 0xbb}, Value: []byte{0xab}},
			{Key: []byte{0xbb}, Value: []byte{0xb0}},
			{Key: []byte{0xbb, 0xbb}, Value: []byte{0xbb}},
			{Key: []byte{0xbb, 0xcc}, Value: []byte{0xbc}},
		}
		checkInput(t, input)
	})

	t.Run("single_long_leaf_is_equivalent", func(t *testing.T) {
		input := []KeyValue{
			{Key: []byte{0xaa}, Value: []byte("ABCABCABCABCABCABCABCABCABCABCABCABCABCABCABCABCABCABCABCABCABCABCABCABC")},
			{Key: []byte{0xba}, Value: []byte{0x11}},
		}
		checkInput(t, input)
	})

	t.Run("two_long_leaves_is_equivalent", func(t *testing.T) {
		input := []KeyValue{
			{Key: []byte{0xaa}, Value: []byte("ABCABCABCABCABCABCABCABCABCABCABCABCABCABCABCABCABCABCABCABCABCABCABCABC")},
			{Key: []byte{0xba}, Value: []byte("ABCABCABCABCABCABCABCABCABCABCABCABCABCABCABCABCABCABCABCABCABCABCABCABC")},
		}
		checkInput(t, input)
	})

}
