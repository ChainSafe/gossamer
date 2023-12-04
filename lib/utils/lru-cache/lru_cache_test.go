// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package lrucache

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLRUCache(t *testing.T) {
	cache := NewLRUCache[int, string](2)

	t.Run("TestBasicOperations", func(t *testing.T) {
		cache.Put(1, "Alice")
		cache.Put(2, "Bob")

		v := cache.Get(1)
		require.Equal(t, "Alice", v)

		v = cache.Get(2)
		require.Equal(t, "Bob", v)
	})

	t.Run("TestUpdateExistingKey", func(t *testing.T) {
		cache.Put(1, "Alice")

		// Update the value of an existing key.
		cache.Put(1, "Alice Smith")

		v := cache.Get(1)
		require.Equal(t, "Alice Smith", v)
	})

	t.Run("TestCacheEviction", func(t *testing.T) {
		cache.Put(1, "Alice")
		cache.Put(2, "Bob")

		// This will evict 1 (least recently used).
		cache.Put(4, "Dave")

		v := cache.Get(1)
		require.Equal(t, "", v)

		v = cache.Get(2)
		require.Equal(t, "Bob", v)

		v = cache.Get(4)
		require.Equal(t, "Dave", v)
	})

	t.Run("TestRetrieveNonExistingKey", func(t *testing.T) {
		cache.Put(1, "Alice")

		v := cache.Get(999)
		require.Equal(t, "", v)
	})

	t.Run("TestPutAndGetLen", func(t *testing.T) {
		cache.Put(1, "Alice")
		cache.Put(2, "Bob")

		len := cache.Len()
		require.Equal(t, 2, len)
	})

	t.Run("TestPutAndDelete", func(t *testing.T) {
		cache.Put(1, "Alice")
		cache.Put(2, "Bob")

		require.True(t, cache.Has(1))

		cache.Delete(1)

		require.False(t, cache.Has(1))
	})

	t.Run("TestSoftPut", func(t *testing.T) {
		cache.Put(1, "Alice")
		cache.SoftPut(1, "Bob")

		require.Equal(t, "Alice", cache.Get(1))
	})
}
