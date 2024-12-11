// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package offchain

import (
	"testing"

	memorykvdb "github.com/ChainSafe/gossamer/internal/kvdb/memory-kvdb"
	"github.com/ChainSafe/gossamer/internal/primitives/core/hash"
	"github.com/ChainSafe/gossamer/internal/primitives/database"
	"github.com/stretchr/testify/require"
)

// / Create new offchain storage for tests (backed by memorydb)
func NewTestLocalStorage(t *testing.T) *LocalStorage {
	t.Helper()
	kvdb := memorykvdb.New(13)
	db := database.NewDBAdapter[hash.H256](kvdb)
	return NewLocalStorage(db)
}

func TestLocalStorage(t *testing.T) {
	t.Run("compare_and_set_and_clear_the_locks_map", func(t *testing.T) {
		storage := NewTestLocalStorage(t)
		prefix := []byte("prefix")
		key := []byte("key")
		value := []byte("value")

		storage.Set(prefix, key, value)
		require.Equal(t, value, storage.Get(prefix, key))

		require.True(t, storage.CompareAndSet(prefix, key, value, []byte("asd")))
		require.Equal(t, []byte("asd"), storage.Get(prefix, key))
		require.Empty(t, storage.locks)
	})

	t.Run("compare_and_set_on_empty_field", func(t *testing.T) {
		storage := NewTestLocalStorage(t)
		prefix := []byte("prefix")
		key := []byte("key")

		require.True(t, storage.CompareAndSet(prefix, key, nil, []byte("asd")))
		require.Equal(t, []byte("asd"), storage.Get(prefix, key))
		require.Empty(t, storage.locks)
	})

	t.Run("remove", func(t *testing.T) {
		storage := NewTestLocalStorage(t)
		prefix := []byte("prefix")
		key := []byte("key")
		value := []byte("value")

		storage.Set(prefix, key, value)
		require.Equal(t, value, storage.Get(prefix, key))

		storage.Remove(prefix, key)
		require.Nil(t, storage.Get(prefix, key))
	})

}
