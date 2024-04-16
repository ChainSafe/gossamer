// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package inmemory

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_TrieValueCache_SetAndGet(t *testing.T) {
	const maxCacheSize = 10
	t.Run("set_and_get_value_successful", func(t *testing.T) {
		cache := NewTrieValueCache(maxCacheSize)
		key := []byte("key")
		value := []byte("value")

		cache.setValue(key, value)
		valueFromCache := cache.getValue(key)

		assert.Equal(t, value, valueFromCache)
	})

	t.Run("get_value_not_found", func(t *testing.T) {
		cache := NewTrieValueCache(maxCacheSize)
		valueFromCache := cache.getValue([]byte("missing"))
		assert.Nil(t, valueFromCache)
	})

	t.Run("replace_value_when_size_exceeded", func(t *testing.T) {
		cache := NewTrieValueCache(maxCacheSize)
		key1 := []byte("key1")
		key2 := []byte("key1")
		value1 := make([]byte, maxCacheSize/2+1)
		value2 := make([]byte, maxCacheSize/2+1)

		// First value is inserted successfully
		cache.setValue(key1, value1)
		valueFromCache := cache.getValue(key1)

		assert.Equal(t, value1, valueFromCache)

		// Second value is inserted successfully
		cache.setValue(key2, value2)
		valueFromCache = cache.getValue(key2)

		assert.Equal(t, value2, valueFromCache)

		// First value has been removed
		cache.lru.GC() // Force GC to remove items that should be deleted

		valueFromCache = cache.getValue(key1)
		assert.Nil(t, valueFromCache)
	})
}
