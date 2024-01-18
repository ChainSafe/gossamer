// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package memorydb

import (
	"testing"

	"github.com/ChainSafe/gossamer/pkg/trie/test_support/keccak_hasher"
	"github.com/ChainSafe/gossamer/pkg/trie/test_support/reference_trie"
	"github.com/ChainSafe/gossamer/pkg/trie/triedb/nibble"
	"github.com/stretchr/testify/require"
)

type KeccakHash = keccak_hasher.KeccakHash

var hasher = keccak_hasher.NewKeccakHasher()
var V0Layout = reference_trie.LayoutV0[KeccakHash]{}

var nullNode = []byte{0x0}
var emptyPrefix = nibble.EmptyPrefix

func Test_New(t *testing.T) {
	db := NewMemoryDB[KeccakHash](hasher, HashKey[KeccakHash])
	hashedNullNode := hasher.Hash(nullNode)
	require.Equal(t, hashedNullNode, db.Insert(emptyPrefix, nullNode))

	db2, root := NewMemoryDBWithRoot[KeccakHash](hasher, HashKey[KeccakHash])
	require.True(t, db2.Contains(root, emptyPrefix))
	require.True(t, db.Contains(root, emptyPrefix))
}

func Test_Remove(t *testing.T) {
	helloBytes := []byte("hello world!")
	helloKey := hasher.Hash(helloBytes)

	t.Run("Remove purge insert purge", func(t *testing.T) {
		m := NewMemoryDB[KeccakHash](hasher, HashKey[KeccakHash])
		m.Remove(helloKey, emptyPrefix)
		dbValue := m.Raw(helloKey, emptyPrefix)
		require.NotNil(t, dbValue)
		require.Equal(t, -1, dbValue.rc)

		m.Purge()
		dbValue = m.Raw(helloKey, emptyPrefix)
		require.NotNil(t, dbValue)
		require.Equal(t, -1, dbValue.rc)

		m.Insert(emptyPrefix, helloBytes)
		dbValue = m.Raw(helloKey, emptyPrefix)
		require.NotNil(t, dbValue)
		require.Equal(t, 0, dbValue.rc)

		m.Purge()
		dbValue = m.Raw(helloKey, emptyPrefix)
		require.Nil(t, dbValue)
	})

	t.Run("Remove and purge", func(t *testing.T) {
		m := NewMemoryDB[KeccakHash](hasher, HashKey[KeccakHash])
		res := m.RemoveAndPurge(helloKey, emptyPrefix)
		require.Nil(t, res)

		dbValue := m.Raw(helloKey, emptyPrefix)
		require.NotNil(t, dbValue)
		require.Equal(t, -1, dbValue.rc)

		m.Insert(emptyPrefix, helloBytes)
		m.Insert(emptyPrefix, helloBytes)

		dbValue = m.Raw(helloKey, emptyPrefix)
		require.NotNil(t, dbValue)
		require.Equal(t, 1, dbValue.rc)

		res = m.RemoveAndPurge(helloKey, emptyPrefix)
		require.NotNil(t, res)
		require.Equal(t, helloBytes, *res)

		dbValue = m.Raw(helloKey, emptyPrefix)
		require.Nil(t, dbValue)

		res = m.RemoveAndPurge(helloKey, emptyPrefix)
		require.Nil(t, res)
	})
}

func Test_Consolidate(t *testing.T) {
	main := NewMemoryDB[KeccakHash](hasher, HashKey[KeccakHash])
	other := NewMemoryDB[KeccakHash](hasher, HashKey[KeccakHash])

	removeKey := other.Insert(emptyPrefix, []byte("doggo"))
	main.Remove(removeKey, emptyPrefix)

	insertKey := other.Insert(emptyPrefix, []byte("arf"))
	main.Emplace(insertKey, emptyPrefix, []byte("arf"))

	negativeRemoveKey := other.Insert(emptyPrefix, []byte("negative"))
	other.Remove(negativeRemoveKey, emptyPrefix) // rc = 0
	other.Remove(negativeRemoveKey, emptyPrefix) // rc = -1
	main.Remove(negativeRemoveKey, emptyPrefix)  // rc = -1

	main.Consolidate(other)

	t.Run("removeKey with rc=0", func(t *testing.T) {
		dbValue := main.Raw(removeKey, emptyPrefix)
		require.NotNil(t, dbValue)
		require.Equal(t, []byte("doggo"), dbValue.value)
		require.Equal(t, 0, dbValue.rc)
	})

	t.Run("insertKey with rc=2", func(t *testing.T) {
		dbValue := main.Raw(insertKey, emptyPrefix)
		require.NotNil(t, dbValue)
		require.Equal(t, []byte("arf"), dbValue.value)
		require.Equal(t, 2, dbValue.rc)
	})

	t.Run("negativeRemoveKey with rc=-2", func(t *testing.T) {
		dbValue := main.Raw(negativeRemoveKey, emptyPrefix)
		require.NotNil(t, dbValue)
		require.Equal(t, []byte("negative"), dbValue.value)
		require.Equal(t, -2, dbValue.rc)
	})
}
