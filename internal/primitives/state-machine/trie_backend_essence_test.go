// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package statemachine

import (
	"slices"
	"testing"

	hashdb "github.com/ChainSafe/gossamer/internal/hash-db"
	"github.com/ChainSafe/gossamer/internal/primitives/core/hash"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime"
	"github.com/ChainSafe/gossamer/internal/primitives/storage"
	"github.com/ChainSafe/gossamer/internal/primitives/trie"
	triedb "github.com/ChainSafe/gossamer/pkg/trie/triedb"
	"github.com/stretchr/testify/require"
)

var (
	_ hashdb.HashDB[hash.H256] = &trieBackendEssence[hash.H256, runtime.BlakeTwo256]{}
	_ hashdb.HashDB[hash.H256] = &ephemeral[hash.H256, runtime.BlakeTwo256]{}
)

func TestTrieBackendEssence(t *testing.T) {
	t.Run("next_storage_key_and_next_child_storage_key_work", func(t *testing.T) {
		childInfo := storage.NewDefaultChildInfo([]byte("MyChild"))
		// Contains values
		var root1 hash.H256
		// Contains child trie
		var root2 hash.H256

		mdb := trie.NewPrefixedMemoryDB[hash.H256, runtime.BlakeTwo256]()
		{
			trie := triedb.NewEmptyTrieDB[hash.H256, runtime.BlakeTwo256](mdb)
			trie.SetVersion(triedb.V1)
			require.NoError(t, trie.Set([]byte("3"), []byte{1}))
			require.NoError(t, trie.Set([]byte("4"), []byte{1}))
			require.NoError(t, trie.Set([]byte("6"), []byte{1}))
			root1 = trie.MustHash()

		}
		{
			ksdb := trie.NewKeyspacedDB(mdb, childInfo.Keyspace())
			// reuse of root_1 implicitly assert child trie root is same
			// as top trie (contents must remain the same).
			trie := triedb.NewEmptyTrieDB[hash.H256, runtime.BlakeTwo256](ksdb)
			trie.SetVersion(triedb.V1)
			err := trie.Set([]byte("3"), []byte{1})
			require.NoError(t, err)
			require.NoError(t, trie.Set([]byte("3"), []byte{1}))
			require.NoError(t, trie.Set([]byte("4"), []byte{1}))
			require.NoError(t, trie.Set([]byte("6"), []byte{1}))
			root := trie.MustHash()
			require.Equal(t, root1, root)
		}
		{
			trie := triedb.NewEmptyTrieDB[hash.H256, runtime.BlakeTwo256](mdb)
			trie.SetVersion(triedb.V1)
			bleh := childInfo.PrefixedStorageKey()
			require.NoError(t, trie.Set(slices.Clone(bleh), root1.Bytes()))
			root2 = trie.MustHash()

			val, err := trie.Get(bleh)
			require.NoError(t, err)
			require.Equal(t, root1.Bytes(), val)
		}

		essence1 := newTrieBackendEssence[hash.H256, runtime.BlakeTwo256](
			HashDBTrieBackendStorage[hash.H256]{mdb}, root1, nil, nil)
		tb1 := TrieBackend[hash.H256, runtime.BlakeTwo256]{essence: essence1} //nolint:govet

		key, err := tb1.NextStorageKey([]byte("2"))
		require.NoError(t, err)
		require.Equal(t, StorageKey("3"), key)

		key, err = tb1.NextStorageKey([]byte("3"))
		require.NoError(t, err)
		require.Equal(t, StorageKey("4"), key)

		key, err = tb1.NextStorageKey([]byte("4"))
		require.NoError(t, err)
		require.Equal(t, StorageKey("6"), key)

		key, err = tb1.NextStorageKey([]byte("5"))
		require.NoError(t, err)
		require.Equal(t, StorageKey("6"), key)

		key, err = tb1.NextStorageKey([]byte("6"))
		require.NoError(t, err)
		require.Nil(t, key)

		essence2 := newTrieBackendEssence[hash.H256, runtime.BlakeTwo256](
			HashDBTrieBackendStorage[hash.H256]{mdb}, root2, nil, nil)

		key, err = essence2.NextChildStorageKey(childInfo, []byte("2"))
		require.NoError(t, err)
		require.Equal(t, StorageKey("3"), key)

		key, err = essence2.NextChildStorageKey(childInfo, []byte("3"))
		require.NoError(t, err)
		require.Equal(t, StorageKey("4"), key)

		key, err = essence2.NextChildStorageKey(childInfo, []byte("4"))
		require.NoError(t, err)
		require.Equal(t, StorageKey("6"), key)

		key, err = essence2.NextChildStorageKey(childInfo, []byte("5"))
		require.NoError(t, err)
		require.Equal(t, StorageKey("6"), key)

		key, err = essence2.NextChildStorageKey(childInfo, []byte("6"))
		require.NoError(t, err)
		require.Nil(t, key)
	})
}
