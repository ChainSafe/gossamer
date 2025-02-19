// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package cache

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/ChainSafe/gossamer/internal/primitives/core/hash"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime"
	"github.com/ChainSafe/gossamer/internal/primitives/trie"
	"github.com/ChainSafe/gossamer/internal/primitives/trie/recorder"
	pkgtrie "github.com/ChainSafe/gossamer/pkg/trie"
	"github.com/ChainSafe/gossamer/pkg/trie/triedb"
	"github.com/stretchr/testify/require"
)

var testData = []triedb.TrieItem{
	{Key: []byte("key1"), Value: []byte("val1")},
	{Key: []byte("key2"), Value: bytes.Repeat([]byte{2}, 64)},
	{Key: []byte("key3"), Value: []byte("val3")},
	{Key: []byte("key4"), Value: bytes.Repeat([]byte{4}, 64)},
}

const cacheSize uint = 1024 * 10

func createTrie(t *testing.T) (*trie.MemoryDB[hash.H256, runtime.BlakeTwo256], hash.H256) {
	t.Helper()

	db := trie.NewMemoryDB[hash.H256, runtime.BlakeTwo256]()
	trie := triedb.NewEmptyTrieDB[hash.H256, runtime.BlakeTwo256](db)
	trie.SetVersion(pkgtrie.V1)
	for _, item := range testData {
		err := trie.Set(item.Key, item.Value)
		require.NoError(t, err)
	}
	hash := trie.MustHash()

	return db, hash
}

func Test_SharedTrieCache(t *testing.T) {
	t.Run("cache_works", func(t *testing.T) {
		db, root := createTrie(t)
		sharedCache := NewSharedTrieCache[hash.H256](cacheSize)
		localCache := sharedCache.LocalTrieCache()

		{
			cache, unlock := localCache.TrieCache(root)
			trie := triedb.NewTrieDB(
				root, db, triedb.WithCache[hash.H256, runtime.BlakeTwo256](cache))
			trie.SetVersion(pkgtrie.V1)

			val, err := triedb.GetWith(trie, testData[0].Key, func(d []byte) []byte { return d })
			require.NoError(t, err)
			require.NotNil(t, val)
			require.NotNil(t, *val)
			require.Equal(t, testData[0].Value, *val)
			unlock()
		}

		// Local cache wasn't committed yet, so there should nothing in the shared caches.
		require.Equal(t, 0, sharedCache.inner.valueCache.lru.Len())
		require.Equal(t, 0, sharedCache.inner.nodeCache.lru.Len())

		localCache.Commit()

		// Now we should have the cached items in the shared cache.
		require.GreaterOrEqual(t, sharedCache.inner.nodeCache.lru.Len(), 1)

		cachedVal, ok := sharedCache.inner.valueCache.lru.Peek(
			ValueCacheKey[hash.H256]{
				StorageRoot: root,
				StorageKey:  testData[0].Key,
			}.ValueCacheKeyComparable(),
		)
		require.True(t, ok)
		existing, ok := cachedVal.(triedb.ExistingCachedValue[hash.H256])
		require.True(t, ok)
		require.Equal(t, testData[0].Value, existing.Data)

		var fakeData = []byte("fake_data")

		localCache = sharedCache.LocalTrieCache()
		sharedCache.inner.valueCache.lru.Add(
			ValueCacheKey[hash.H256]{
				StorageRoot: root,
				StorageKey:  testData[1].Key,
			}.ValueCacheKeyComparable(),
			triedb.ExistingCachedValue[hash.H256]{
				Hash: hash.NewH256(),
				Data: fakeData,
			},
		)

		{
			cache, unlock := localCache.TrieCache(root)
			trie := triedb.NewTrieDB[hash.H256, runtime.BlakeTwo256](
				root, db, triedb.WithCache[hash.H256, runtime.BlakeTwo256](cache))
			trie.SetVersion(pkgtrie.V1)

			// We should now get the "fake_data", because we inserted this manually to the cache.
			val, err := triedb.GetWith(trie, testData[1].Key, func(d []byte) []byte { return d })
			require.NoError(t, err)
			require.NotNil(t, val)
			require.NotNil(t, *val)
			require.Equal(t, fakeData, *val)
			unlock()
		}
	})

	t.Run("TrieCacheMut", func(t *testing.T) {
		db, root := createTrie(t)
		newKey := []byte("new_key")
		// Use some long value to not have it inlined
		newValue := bytes.Repeat([]byte{23}, 64)

		sharedCache := NewSharedTrieCache[hash.H256](cacheSize)
		var newRoot hash.H256

		{
			localCache := sharedCache.LocalTrieCache()
			cache, unlock := localCache.TrieCacheMut()

			{
				trie := triedb.NewTrieDB[hash.H256, runtime.BlakeTwo256](
					root, db, triedb.WithCache[hash.H256, runtime.BlakeTwo256](cache))
				trie.SetVersion(pkgtrie.V1)
				require.NoError(t, trie.Set(newKey, newValue))
				newRoot = trie.MustHash()
			}

			cache.MergeInto(&localCache, newRoot)
			unlock()
			localCache.Commit()
		}

		// After the local cache is committed, all changes should have been merged back to the shared
		// cache.
		cachedVal, ok := sharedCache.inner.valueCache.lru.Peek(ValueCacheKey[hash.H256]{
			StorageRoot: newRoot,
			StorageKey:  newKey,
		}.ValueCacheKeyComparable())
		require.True(t, ok)
		existing, ok := cachedVal.(triedb.ExistingCachedValue[hash.H256])
		require.True(t, ok)
		require.Equal(t, newValue, existing.Data)
	})

	t.Run("TrieCache_cache_and_recorder_work_together", func(t *testing.T) {
		db, root := createTrie(t)
		sharedCache := NewSharedTrieCache[hash.H256](cacheSize)

		for i := 0; i < 5; i++ {
			// Clear some of the caches.
			if i == 2 {
				sharedCache.ResetNodeCache()
			} else if i == 3 {
				sharedCache.ResetValueCache()
			}

			localCache := sharedCache.LocalTrieCache()
			recorder := recorder.NewRecorder[hash.H256]()

			{
				cache, unlock := localCache.TrieCache(root)
				recorder := recorder.TrieRecorder(root)
				trie := triedb.NewTrieDB[hash.H256, runtime.BlakeTwo256](
					root, db,
					triedb.WithCache[hash.H256, runtime.BlakeTwo256](cache),
					triedb.WithRecorder[hash.H256, runtime.BlakeTwo256](recorder),
				)

				for _, item := range testData {
					val, err := triedb.GetWith(trie, item.Key, func(d []byte) []byte { return d })
					require.NoError(t, err)
					require.NotNil(t, val)
					require.NotNil(t, *val)
					require.Equal(t, item.Value, *val)
				}

				root = trie.MustHash()
				unlock()
			}

			storageProof := recorder.DrainStorageProof()
			memoryDB := trie.NewMemoryDBFromStorageProof[hash.H256, runtime.BlakeTwo256](storageProof)

			{
				trie := triedb.NewTrieDB[hash.H256, runtime.BlakeTwo256](root, memoryDB)
				for _, item := range testData {
					val, err := triedb.GetWith(trie, item.Key, func(d []byte) []byte { return d })
					require.NoError(t, err)
					require.NotNil(t, val)
					require.NotNil(t, *val)
					require.Equal(t, item.Value, *val)
				}
			}
		}
	})

	t.Run("TrieCacheMut_cache_and_recorder_work_together", func(t *testing.T) {
		var dataToAdd = []triedb.TrieItem{
			{Key: []byte("key11"), Value: bytes.Repeat([]byte{45}, 78)},
			{Key: []byte("key33"), Value: bytes.Repeat([]byte{78}, 89)},
		}

		db, root := createTrie(t)

		sharedCache := NewSharedTrieCache[hash.H256](cacheSize)

		// Run this twice so that we use the data cache in the second run.
		for i := 0; i < 5; i++ {
			// Clear some of the caches.
			if i == 2 {
				sharedCache.ResetNodeCache()
			} else if i == 3 {
				sharedCache.ResetValueCache()
			}

			localCache := sharedCache.LocalTrieCache()
			recorder := recorder.NewRecorder[hash.H256]()
			var newRoot hash.H256

			{
				db := db.Clone()
				cache, unlock := localCache.TrieCache(root)
				recorder := recorder.TrieRecorder(root)

				trie := triedb.NewTrieDB[hash.H256, runtime.BlakeTwo256](
					root, &db,
					triedb.WithCache[hash.H256, runtime.BlakeTwo256](cache),
					triedb.WithRecorder[hash.H256, runtime.BlakeTwo256](recorder),
				)

				for _, item := range dataToAdd {
					err := trie.Set(item.Key, item.Value)
					require.NoError(t, err)
				}

				newRoot = trie.MustHash()
				unlock()
			}

			storageProof := recorder.DrainStorageProof()
			memoryDB := trie.NewMemoryDBFromStorageProof[hash.H256, runtime.BlakeTwo256](storageProof)
			var proofRoot hash.H256
			{
				trie := triedb.NewTrieDB[hash.H256, runtime.BlakeTwo256](root, memoryDB)
				for _, item := range dataToAdd {
					err := trie.Set(item.Key, item.Value)
					require.NoError(t, err)
				}
				proofRoot = trie.MustHash()
			}
			require.Equal(t, newRoot, proofRoot)
		}
	})

	t.Run("cache_lru_works", func(t *testing.T) {
		db, root := createTrie(t)
		sharedCache := NewSharedTrieCache[hash.H256](cacheSize)

		{
			localCache := sharedCache.LocalTrieCache()
			cache, unlock := localCache.TrieCache(root)

			trie := triedb.NewTrieDB[hash.H256, runtime.BlakeTwo256](
				root, db, triedb.WithCache[hash.H256, runtime.BlakeTwo256](cache))
			trie.SetVersion(pkgtrie.V1)

			for _, item := range testData {
				val, err := triedb.GetWith(trie, item.Key, func(d []byte) []byte { return d })
				require.NoError(t, err)
				require.NotNil(t, val)
				require.NotNil(t, *val)
				require.Equal(t, item.Value, *val)
			}

			unlock()
			localCache.Commit()
		}

		// Check that all items are there.
		var allFound = true
		for _, key := range sharedCache.inner.valueCache.lru.Keys() {
			var found bool
			for _, item := range testData {
				if bytes.Equal([]byte(key.StorageKey), item.Key) {
					found = true
					break
				}
			}
			if !found {
				allFound = false
				break
			}
		}
		require.True(t, allFound)

		// Run this in a loop. The first time we check that with the filled value cache,
		// the expected values are at the top of the LRU.
		// The second run is using an empty value cache to ensure that we access the nodes.
		for i := 0; i < 2; i++ {
			{
				localCache := sharedCache.LocalTrieCache()
				cache, unlock := localCache.TrieCache(root)
				trie := triedb.NewTrieDB[hash.H256, runtime.BlakeTwo256](
					root, db, triedb.WithCache[hash.H256, runtime.BlakeTwo256](cache))
				trie.SetVersion(pkgtrie.V1)

				for _, item := range testData[:2] {
					val, err := triedb.GetWith(trie, item.Key, func(d []byte) []byte { return d })
					require.NoError(t, err)
					require.NotNil(t, val)
					require.NotNil(t, *val)
					require.Equal(t, item.Value, *val)
				}
				unlock()
				localCache.Commit()
			}

			// Ensure that the accessed items are part of the shared value
			// cache.
			for _, item := range testData[:2] {
				_, ok := sharedCache.inner.valueCache.lru.Peek(ValueCacheKey[hash.H256]{
					StorageRoot: root,
					StorageKey:  item.Key,
				}.ValueCacheKeyComparable())
				require.True(t, ok)
			}

			// Delete the value cache, so that we access the nodes.
			sharedCache.ResetValueCache()
		}

		mostRecentlyUsedNodes := sharedCache.inner.nodeCache.lru.Keys()

		{
			localCache := sharedCache.LocalTrieCache()
			cache, unlock := localCache.TrieCache(root)
			trie := triedb.NewTrieDB[hash.H256, runtime.BlakeTwo256](
				root, db, triedb.WithCache[hash.H256, runtime.BlakeTwo256](cache))
			trie.SetVersion(pkgtrie.V1)

			for _, item := range testData[2:] {
				val, err := triedb.GetWith(trie, item.Key, func(d []byte) []byte { return d })
				require.NoError(t, err)
				require.NotNil(t, val)
				require.NotNil(t, *val)
				require.Equal(t, item.Value, *val)
			}
			unlock()
			localCache.Commit()
		}

		// Ensure that the most recently used nodes changed as well.
		cachedNodes := sharedCache.inner.nodeCache.lru.Keys()

		require.NotEqual(t, mostRecentlyUsedNodes, cachedNodes)
	})

	t.Run("cache_respects_bounds", func(t *testing.T) {
		db, root := createTrie(t)
		sharedCache := NewSharedTrieCache[hash.H256](cacheSize)

		{
			localCache := sharedCache.LocalTrieCache()
			var newRoot hash.H256

			{
				cache, unlock := localCache.TrieCache(root)
				{
					trie := triedb.NewTrieDB[hash.H256, runtime.BlakeTwo256](
						root, db, triedb.WithCache[hash.H256, runtime.BlakeTwo256](cache))
					trie.SetVersion(pkgtrie.V1)
					value := bytes.Repeat([]byte{10}, 100)
					// Ensure we add enough data that would overflow the cache.
					for i := 0; i < (int(cacheSize) / 100 * 2); i++ {
						require.NoError(t, trie.Set([]byte(fmt.Sprintf("key%d", i)), value))
					}
					newRoot = trie.MustHash()
				}
				unlock()
				cache.MergeInto(&localCache, newRoot)
			}

			localCache.Commit()
		}

		require.Less(t, sharedCache.usedMemorySize(), cacheSize)
	})
}
