// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package cache

import (
	"bytes"
	"slices"
	"testing"

	"github.com/ChainSafe/gossamer/internal/primitives/core/hash"
	"github.com/ChainSafe/gossamer/pkg/trie/triedb"
	"github.com/stretchr/testify/require"
)

func Test_sharedValueCache(t *testing.T) {
	cache := newSharedValueCache[hash.H256](110)

	key := bytes.Repeat([]byte{0}, 10)
	root0 := hash.NewRandomH256()
	root1 := hash.NewRandomH256()

	vck0 := ValueCacheKey[hash.H256]{
		StorageRoot: root0,
		StorageKey:  key,
	}
	vck1 := ValueCacheKey[hash.H256]{
		StorageRoot: root1,
		StorageKey:  key,
	}

	cache.Update([]sharedValueCacheAdded[hash.H256]{
		{
			ValueCacheKey: vck0,
			CachedValue:   triedb.NonExistingCachedValue[hash.H256]{},
		},
		{
			ValueCacheKey: vck1,
			CachedValue:   triedb.NonExistingCachedValue[hash.H256]{},
		},
	}, nil)

	// Ensure that the basics are working
	byStorageKey := slices.CompactFunc(cache.lru.Keys(), func(a, b ValueCacheKeyComparable[hash.H256]) bool {
		return a.StorageKey == b.StorageKey
	})
	require.Len(t, byStorageKey, 1)
	require.Equal(t, 2, cache.lru.Len())
	keys := cache.lru.Keys()
	newestKey := keys[len(keys)-1]
	require.Equal(t, root1, newestKey.StorageRoot)
	oldestKey := keys[0]
	require.Equal(t, root0, oldestKey.StorageRoot)
	require.Equal(t, uint(22), cache.lru.Cost())

	// Just accessing a key should not change anything on the size and number of entries.
	cache.Update(nil, []ValueCacheKeyComparable[hash.H256]{
		vck0.ValueCacheKeyComparable(),
	})
	byStorageKey = slices.CompactFunc(cache.lru.Keys(), func(a, b ValueCacheKeyComparable[hash.H256]) bool {
		return a.StorageKey == b.StorageKey
	})
	require.Len(t, byStorageKey, 1)
	require.Equal(t, 2, cache.lru.Len())
	keys = cache.lru.Keys()
	newestKey = keys[len(keys)-1]
	require.Equal(t, root0, newestKey.StorageRoot)
	oldestKey = keys[0]
	require.Equal(t, root1, oldestKey.StorageRoot)
	require.Equal(t, uint(22), cache.lru.Cost())

	// Updating the cache again with exactly the same data should not change anything.
	cache.Update([]sharedValueCacheAdded[hash.H256]{
		{
			ValueCacheKey: vck1,
			CachedValue:   triedb.NonExistingCachedValue[hash.H256]{},
		},
		{
			ValueCacheKey: vck0,
			CachedValue:   triedb.NonExistingCachedValue[hash.H256]{},
		},
	}, nil)
	byStorageKey = slices.CompactFunc(cache.lru.Keys(), func(a, b ValueCacheKeyComparable[hash.H256]) bool {
		return a.StorageKey == b.StorageKey
	})
	require.Len(t, byStorageKey, 1)
	require.Equal(t, 2, cache.lru.Len())
	keys = cache.lru.Keys()
	newestKey = keys[len(keys)-1]
	require.Equal(t, root0, newestKey.StorageRoot)
	oldestKey = keys[0]
	require.Equal(t, root1, oldestKey.StorageRoot)
	require.Equal(t, uint(22), cache.lru.Cost())

	var added []sharedValueCacheAdded[hash.H256]
	// Add 10 other entries and this should move out two of the initial entries.
	for i := 1; i < 11; i++ {
		added = append(added, sharedValueCacheAdded[hash.H256]{
			ValueCacheKey: ValueCacheKey[hash.H256]{
				StorageRoot: root0,
				StorageKey:  bytes.Repeat([]byte{uint8(i)}, 10),
			},
			CachedValue: triedb.NonExistingCachedValue[hash.H256]{},
		})
	}
	cache.Update(added, nil)

	require.Equal(t, uint64(2), cache.lru.Metrics().Removals) // removals instead of evictions
	require.Equal(t, 10, cache.lru.Len())
	byStorageKey = slices.CompactFunc(cache.lru.Keys(), func(a, b ValueCacheKeyComparable[hash.H256]) bool {
		return a.StorageKey == b.StorageKey
	})
	require.Len(t, byStorageKey, 10)
	require.False(t, slices.ContainsFunc(cache.lru.Keys(), func(a ValueCacheKeyComparable[hash.H256]) bool {
		return a.StorageKey == string(key)
	}))
	require.Equal(t, uint(110), cache.lru.Cost())

	val, ok := cache.lru.Peek(ValueCacheKey[hash.H256]{
		StorageRoot: root0,
		StorageKey:  bytes.Repeat([]byte{1}, 10),
	}.ValueCacheKeyComparable())
	require.True(t, ok)
	require.Equal(t, triedb.NonExistingCachedValue[hash.H256]{}, val)

	// this was never inserted
	_, ok = cache.lru.Peek(ValueCacheKey[hash.H256]{
		StorageRoot: root1,
		StorageKey:  bytes.Repeat([]byte{1}, 10),
	}.ValueCacheKeyComparable())
	require.False(t, ok)

	_, ok = cache.lru.Peek(vck0.ValueCacheKeyComparable())
	require.False(t, ok)
	_, ok = cache.lru.Peek(vck1.ValueCacheKeyComparable())
	require.False(t, ok)

	cache.Update([]sharedValueCacheAdded[hash.H256]{
		{
			ValueCacheKey: ValueCacheKey[hash.H256]{
				StorageRoot: root0,
				StorageKey:  bytes.Repeat([]byte{10}, 10),
			},
			CachedValue: triedb.NonExistingCachedValue[hash.H256]{},
		},
	}, nil)
	require.False(t, slices.ContainsFunc(cache.lru.Keys(), func(a ValueCacheKeyComparable[hash.H256]) bool {
		return a.StorageKey == string(key)
	}))
}
