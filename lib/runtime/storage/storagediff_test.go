// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package storage

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStorageDiff_Get(t *testing.T) {
	t.Run("Upsert", func(t *testing.T) {
		changes := newStorageDiff()

		key := "key"
		value := []byte("value")
		changes.upsert(key, value)

		val, deleted := changes.get(key)
		require.False(t, deleted)
		require.Equal(t, value, val)
	})

	t.Run("Upsert then delete", func(t *testing.T) {
		changes := newStorageDiff()

		key := "key"
		value := []byte("value")
		changes.upsert(key, value)
		changes.delete(key)

		val, deleted := changes.get(key)
		require.True(t, deleted)
		require.Nil(t, val)
	})
}
