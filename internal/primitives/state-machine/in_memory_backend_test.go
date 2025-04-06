// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package statemachine

import (
	"testing"

	"github.com/ChainSafe/gossamer/internal/primitives/core/hash"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime"
	"github.com/ChainSafe/gossamer/internal/primitives/storage"
	"github.com/stretchr/testify/require"
)

func TestMemoryDBTrieBackend(t *testing.T) {
	t.Run("in_memory_with_child_trie_only", func(t *testing.T) {
		mdbtb := NewMemoryDBTrieBackend[hash.H256, runtime.BlakeTwo256]()
		childInfo := storage.NewDefaultChildInfo([]byte("1"))
		tb := mdbtb.update([]change{
			{
				ChildInfo: childInfo,
				StorageCollection: StorageCollection{
					StorageKeyValue{
						StorageKey:   []byte("2"),
						StorageValue: []byte("3"),
					},
				},
			},
		}, storage.StateVersionV1)
		val, err := tb.ChildStorage(childInfo, []byte("2"))
		require.NoError(t, err)
		require.Equal(t, StorageValue([]byte("3")), val)
		require.NotNil(t, childInfo.PrefixedStorageKey())
	})

	t.Run("insert_multiple_times_child_data_works", func(t *testing.T) {
		mdbtb := NewMemoryDBTrieBackend[hash.H256, runtime.BlakeTwo256]()
		childInfo := storage.NewDefaultChildInfo([]byte("1"))

		mdbtb.insert([]change{
			{
				ChildInfo:         childInfo,
				StorageCollection: StorageCollection{{StorageKey: []byte("2"), StorageValue: []byte("3")}},
			},
		}, storage.StateVersionV1)
		mdbtb.insert([]change{
			{
				ChildInfo:         childInfo,
				StorageCollection: StorageCollection{{StorageKey: []byte("1"), StorageValue: []byte("3")}},
			},
		}, storage.StateVersionV1)

		val, err := mdbtb.ChildStorage(childInfo, []byte("2"))
		require.NoError(t, err)
		require.Equal(t, StorageValue([]byte("3")), val)

		val, err = mdbtb.ChildStorage(childInfo, []byte("1"))
		require.NoError(t, err)
		require.Equal(t, StorageValue([]byte("3")), val)
	})
}
