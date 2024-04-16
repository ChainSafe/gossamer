// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package inmemory

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_TrieCache_SetAndGet(t *testing.T) {
	t.Run("set_and_get_value_successful", func(t *testing.T) {
		cache := NewTrieInMemoryCache()
		key := []byte("key")
		value := []byte("value")

		cache.SetValue(key, value)
		valueFromCache := cache.GetValue(key)

		assert.Equal(t, value, valueFromCache)
	})

	t.Run("get_value_not_found", func(t *testing.T) {
		cache := NewTrieInMemoryCache()
		valueFromCache := cache.GetValue([]byte("missing"))
		assert.Nil(t, valueFromCache)
	})
}
