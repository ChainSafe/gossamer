// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package statemachine

import (
	"bytes"
	"testing"

	"github.com/ChainSafe/gossamer/internal/primitives/core/hash"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime"
	"github.com/ChainSafe/gossamer/internal/primitives/storage"
	"github.com/ChainSafe/gossamer/internal/primitives/trie"
	"github.com/ChainSafe/gossamer/internal/primitives/trie/cache"
	"github.com/ChainSafe/gossamer/internal/primitives/trie/recorder"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/ChainSafe/gossamer/pkg/trie/triedb"
	"github.com/ChainSafe/gossamer/pkg/trie/triedb/codec"
	"github.com/stretchr/testify/require"
)

var (
	_ StorageIterator[hash.H256, runtime.BlakeTwo256] = &rawIter[hash.H256, runtime.BlakeTwo256]{}
	_ Backend[hash.H256, runtime.BlakeTwo256]         = &TrieBackend[hash.H256, runtime.BlakeTwo256]{}
)

var ChildKey1 = []byte("sub1")

func testDB(
	t *testing.T,
	stateVersion storage.StateVersion,
) (*trie.PrefixedMemoryDB[hash.H256, runtime.BlakeTwo256], hash.H256) {
	t.Helper()
	childInfo := storage.NewDefaultChildInfo(ChildKey1)
	mdb := trie.NewPrefixedMemoryDB[hash.H256, runtime.BlakeTwo256]()
	var root hash.H256

	{
		ksdb := trie.NewKeyspacedDB(mdb, childInfo.Keyspace())
		trie := triedb.NewEmptyTrieDB[hash.H256, runtime.BlakeTwo256](ksdb)
		trie.SetVersion(stateVersion.TrieLayout())
		require.NoError(t, trie.Set([]byte("value3"), bytes.Repeat([]byte{142}, 33)))
		require.NoError(t, trie.Set([]byte("value4"), bytes.Repeat([]byte{124}, 33)))
		root = trie.MustHash()
	}

	{
		subRoot := scale.MustMarshal(root)

		var build = func(trie *triedb.TrieDB[hash.H256, runtime.BlakeTwo256], childInfo storage.ChildInfo, subRoot []byte) {
			require.NoError(t, trie.Set(childInfo.PrefixedStorageKey(), subRoot))
			require.NoError(t, trie.Set([]byte("key"), []byte("value")))
			require.NoError(t, trie.Set([]byte("value1"), []byte{42}))
			require.NoError(t, trie.Set([]byte("value2"), []byte{24}))
			require.NoError(t, trie.Set([]byte(":code"), []byte("return 42")))
			for i := uint8(128); i < 255; i++ {
				require.NoError(t, trie.Set([]byte{i}, []byte{i}))
			}
		}

		trie := triedb.NewEmptyTrieDB[hash.H256, runtime.BlakeTwo256](mdb)
		trie.SetVersion(stateVersion.TrieLayout())
		build(trie, childInfo, subRoot)
		root = trie.MustHash()
	}

	return mdb, root
}

func testTrie(
	t *testing.T,
	hashedValue storage.StateVersion,
	cache *cache.LocalTrieCache[hash.H256],
	recorder *recorder.Recorder[hash.H256],
) *TrieBackend[hash.H256, runtime.BlakeTwo256] {
	t.Helper()
	mdb, root := testDB(t, hashedValue)

	var tb *TrieBackend[hash.H256, runtime.BlakeTwo256]
	if cache == nil {
		tb = NewTrieBackend[hash.H256, runtime.BlakeTwo256](HashDBTrieBackendStorage[hash.H256]{mdb}, root, nil, recorder)
	} else {
		tb = NewTrieBackend[hash.H256, runtime.BlakeTwo256](HashDBTrieBackendStorage[hash.H256]{mdb}, root, cache, recorder)
	}
	return tb
}
func TestTrieBackend(t *testing.T) {
	type parameters struct {
		storage.StateVersion
		*cache.SharedTrieCache[hash.H256]
		*recorder.Recorder[hash.H256]
	}
	params := []parameters{
		{StateVersion: storage.StateVersionV0},
		{StateVersion: storage.StateVersionV0, SharedTrieCache: cache.NewSharedTrieCache[hash.H256](1024 * 10)},
		{
			StateVersion:    storage.StateVersionV0,
			SharedTrieCache: cache.NewSharedTrieCache[hash.H256](1024 * 10),
			Recorder:        recorder.NewRecorder[hash.H256](),
		},
		{StateVersion: storage.StateVersionV1},
		{StateVersion: storage.StateVersionV1, SharedTrieCache: cache.NewSharedTrieCache[hash.H256](1024 * 10)},
		{
			StateVersion:    storage.StateVersionV1,
			SharedTrieCache: cache.NewSharedTrieCache[hash.H256](1024 * 10),
			Recorder:        recorder.NewRecorder[hash.H256](),
		},
	}

	t.Run("read_from_returns_some", func(t *testing.T) {
		for _, param := range params {
			var cache *cache.LocalTrieCache[hash.H256]
			if param.SharedTrieCache != nil {
				local := param.SharedTrieCache.LocalTrieCache()
				cache = &local
			}
			tb := testTrie(t, param.StateVersion, cache, param.Recorder)
			val, err := tb.Storage([]byte("key"))
			require.NoError(t, err)
			require.Equal(t, StorageValue("value"), val)
		}
	})

	t.Run("read_from_child_storage_returns_some", func(t *testing.T) {
		for _, param := range params {
			var cache *cache.LocalTrieCache[hash.H256]
			if param.SharedTrieCache != nil {
				local := param.SharedTrieCache.LocalTrieCache()
				cache = &local
			}
			tb := testTrie(t, param.StateVersion, cache, param.Recorder)
			val, err := tb.ChildStorage(storage.NewDefaultChildInfo(ChildKey1), []byte("value3"))
			require.NoError(t, err)
			require.Equal(t, bytes.Repeat([]byte{142}, 33), []byte(val))

			// Change cache entry to check that caching is active.
			tb.essence.cache.childRoot["sub1"] = nil
			val, err = tb.ChildStorage(storage.NewDefaultChildInfo(ChildKey1), []byte("value3"))
			require.NoError(t, err)
			require.Nil(t, val)
		}
	})

	t.Run("read_from_storage_returns_none", func(t *testing.T) {
		for _, param := range params {
			var cache *cache.LocalTrieCache[hash.H256]
			if param.SharedTrieCache != nil {
				local := param.SharedTrieCache.LocalTrieCache()
				cache = &local
			}
			tb := testTrie(t, param.StateVersion, cache, param.Recorder)
			val, err := tb.ChildStorage(storage.NewDefaultChildInfo(ChildKey1), []byte("non-existing-key"))
			require.NoError(t, err)
			require.Nil(t, val)
		}
	})

	t.Run("pairs_are_not_empty_on_non_empty_storage", func(t *testing.T) {
		for _, param := range params {
			var cache *cache.LocalTrieCache[hash.H256]
			if param.SharedTrieCache != nil {
				local := param.SharedTrieCache.LocalTrieCache()
				cache = &local
			}
			tb := testTrie(t, param.StateVersion, cache, param.Recorder)
			iter, err := tb.Pairs(IterArgs{})
			require.NoError(t, err)
			skv, err := iter.Next()
			require.NoError(t, err)
			require.NotNil(t, skv)
		}
	})

	t.Run("pairs_are_empty_on_empty_storage", func(t *testing.T) {
		mdb := trie.NewPrefixedMemoryDB[hash.H256, runtime.BlakeTwo256]()
		var root hash.H256
		tb := NewTrieBackend[hash.H256, runtime.BlakeTwo256](HashDBTrieBackendStorage[hash.H256]{mdb}, root, nil, nil)
		iter, err := tb.Pairs(IterArgs{})
		require.NoError(t, err)
		skv, err := iter.Next()
		require.NoError(t, err)
		require.Nil(t, skv)
	})

	t.Run("storage_iteration_works", func(t *testing.T) {
		for _, param := range params {
			var cache *cache.LocalTrieCache[hash.H256]
			if param.SharedTrieCache != nil {
				local := param.SharedTrieCache.LocalTrieCache()
				cache = &local
			}
			tb := testTrie(t, param.StateVersion, cache, param.Recorder)
			iter, err := tb.Keys(IterArgs{})
			require.NoError(t, err)

			// Fetch everything.
			sk, err := iter.Next()
			require.NoError(t, err)
			require.Equal(t, StorageKey(":child_storage:default:sub1"), sk)

			sk, err = iter.Next()
			require.NoError(t, err)
			require.Equal(t, StorageKey(":code"), sk)

			sk, err = iter.Next()
			require.NoError(t, err)
			require.Equal(t, StorageKey("key"), sk)

			sk, err = iter.Next()
			require.NoError(t, err)
			require.Equal(t, StorageKey("value1"), sk)

			sk, err = iter.Next()
			require.NoError(t, err)
			require.Equal(t, StorageKey("value2"), sk)

			// Fetch starting at a given key (full key).
			iter, err = tb.Keys(IterArgs{StartAt: []byte("key")})
			require.NoError(t, err)
			sk, err = iter.Next()
			require.NoError(t, err)
			require.Equal(t, StorageKey("key"), sk)

			sk, err = iter.Next()
			require.NoError(t, err)
			require.Equal(t, StorageKey("value1"), sk)

			sk, err = iter.Next()
			require.NoError(t, err)
			require.Equal(t, StorageKey("value2"), sk)

			// Fetch starting at a given key (partial key).
			iter, err = tb.Keys(IterArgs{StartAt: []byte("ke")})
			require.NoError(t, err)
			sk, err = iter.Next()
			require.NoError(t, err)
			require.Equal(t, StorageKey("key"), sk)

			sk, err = iter.Next()
			require.NoError(t, err)
			require.Equal(t, StorageKey("value1"), sk)

			sk, err = iter.Next()
			require.NoError(t, err)
			require.Equal(t, StorageKey("value2"), sk)

			// Fetch starting at a given key and with prefix which doesn't match that key.
			// (Start *before* the prefix.)
			iter, err = tb.Keys(IterArgs{StartAt: []byte("key"), Prefix: []byte("value")})
			require.NoError(t, err)
			sk, err = iter.Next()
			require.NoError(t, err)
			require.Equal(t, StorageKey("value1"), sk)

			sk, err = iter.Next()
			require.NoError(t, err)
			require.Equal(t, StorageKey("value2"), sk)

			sk, err = iter.Next()
			require.NoError(t, err)
			require.Nil(t, sk)

			// Fetch starting at a given key and with prefix which doesn't match that key.
			// (Start *after* the prefix.)
			iter, err = tb.Keys(IterArgs{StartAt: []byte("vblue"), Prefix: []byte("value")})
			require.NoError(t, err)
			sk, err = iter.Next()
			require.NoError(t, err)
			require.Nil(t, sk)

			// Fetch starting at a given key and with prefix which does match that key.
			iter, err = tb.Keys(IterArgs{StartAt: []byte("value"), Prefix: []byte("value")})
			require.NoError(t, err)
			sk, err = iter.Next()
			require.NoError(t, err)
			require.Equal(t, StorageKey("value1"), sk)

			sk, err = iter.Next()
			require.NoError(t, err)
			require.Equal(t, StorageKey("value2"), sk)

			sk, err = iter.Next()
			require.NoError(t, err)
			require.Nil(t, sk)
		}
	})

	t.Run("storage_root_is_non_default", func(t *testing.T) {
		for _, param := range params {
			var cache *cache.LocalTrieCache[hash.H256]
			if param.SharedTrieCache != nil {
				local := param.SharedTrieCache.LocalTrieCache()
				cache = &local
			}
			tb := testTrie(t, param.StateVersion, cache, param.Recorder)
			h, _ := tb.StorageRoot(nil, param.StateVersion)
			var expected = hash.H256(bytes.Repeat([]byte{0}, 32))
			require.NotEqual(t, expected, h)
		}
	})

	t.Run("storage_root_transaction_is_non_empty", func(t *testing.T) {
		for _, param := range params {
			var cache *cache.LocalTrieCache[hash.H256]
			if param.SharedTrieCache != nil {
				local := param.SharedTrieCache.LocalTrieCache()
				cache = &local
			}
			tb := testTrie(t, param.StateVersion, cache, param.Recorder)
			newRoot, tx := tb.StorageRoot([]Delta{{Key: []byte("new-key"), Value: []byte("new-value")}}, param.StateVersion)
			expected, _ := testTrie(t, param.StateVersion, cache, param.Recorder).StorageRoot(nil, param.StateVersion)
			require.NotEmpty(t, tx.Drain())
			require.NotEqual(t, expected, newRoot)
		}
	})

	t.Run("keys_with_empty_prefix_returns_all_keys", func(t *testing.T) {
		for _, param := range params {
			var cache *cache.LocalTrieCache[hash.H256]
			if param.SharedTrieCache != nil {
				local := param.SharedTrieCache.LocalTrieCache()
				cache = &local
			}
			testDB, testRoot := testDB(t, param.StateVersion)
			iter, err := triedb.NewTrieDB[hash.H256, runtime.BlakeTwo256](testRoot, testDB).Iterator()
			require.NoError(t, err)
			expected := make([][]byte, 0)
			for {
				item, err := iter.Next()
				require.NoError(t, err)
				if item != nil {
					expected = append(expected, item.Key)
				} else {
					break
				}
			}

			tb := testTrie(t, param.StateVersion, cache, param.Recorder)
			keysIter, err := tb.Keys(IterArgs{})
			require.NoError(t, err)
			keys := make([][]byte, 0)
			for {
				item, err := keysIter.Next()
				require.NoError(t, err)
				if item != nil {
					keys = append(keys, item)
				} else {
					break
				}
			}

			require.Equal(t, expected, keys)
		}
	})

	t.Run("proof_is_empty_until_value_is_read", func(t *testing.T) {
		for _, param := range params {
			var cache *cache.LocalTrieCache[hash.H256]
			if param.SharedTrieCache != nil {
				local := param.SharedTrieCache.LocalTrieCache()
				cache = &local
			}
			tb := testTrie(t, param.StateVersion, cache, param.Recorder)
			tb1 := NewWrappedTrieBackend(tb, nil, recorder.NewRecorder[hash.H256]())
			proof := tb1.ExtractProof()
			require.NotNil(t, proof)
			require.True(t, proof.Empty())
		}
	})

	t.Run("proof_is_non_empty_after_value_is_read", func(t *testing.T) {
		for _, param := range params {
			var cache *cache.LocalTrieCache[hash.H256]
			if param.SharedTrieCache != nil {
				local := param.SharedTrieCache.LocalTrieCache()
				cache = &local
			}
			tb := testTrie(t, param.StateVersion, cache, param.Recorder)
			tb1 := NewWrappedTrieBackend(tb, nil, recorder.NewRecorder[hash.H256]())

			sv, err := tb1.Storage([]byte("key"))
			require.NoError(t, err)
			require.Equal(t, StorageValue("value"), sv)
			proof := tb1.ExtractProof()
			require.NotNil(t, proof)
			require.False(t, proof.Empty())
		}
	})

	t.Run("proof_is_invalid_when_does_not_contains_root", func(t *testing.T) {
		_, err := NewProofCheckTrieBackend[hash.H256, runtime.BlakeTwo256](
			hash.NewH256FromLowUint64BigEndian(1), trie.NewStorageProof(nil))
		require.Error(t, err)
	})

	t.Run("passes_through_backend_calls_inner", func(t *testing.T) {
		for _, param := range params {
			var cache *cache.LocalTrieCache[hash.H256]
			if param.SharedTrieCache != nil {
				local := param.SharedTrieCache.LocalTrieCache()
				cache = &local
			}
			tb := testTrie(t, param.StateVersion, cache, param.Recorder)
			provingBackend := NewWrappedTrieBackend(tb, nil, recorder.NewRecorder[hash.H256]())

			sv, err := provingBackend.Storage([]byte("key"))
			require.NoError(t, err)
			require.Equal(t, StorageValue("value"), sv)

			pairs, err := tb.Pairs(IterArgs{})
			require.NoError(t, err)
			tbPairs := make([]StorageKeyValue, 0)
			for pair, err := range pairs.All() {
				require.NoError(t, err)
				tbPairs = append(tbPairs, pair)
			}
			pairs, err = provingBackend.Pairs(IterArgs{})
			require.NoError(t, err)
			pbPairs := make([]StorageKeyValue, 0)
			for pair, err := range pairs.All() {
				require.NoError(t, err)
				pbPairs = append(pbPairs, pair)
			}
			require.Equal(t, tbPairs, pbPairs)

			trieRoot, trieMDB := tb.StorageRoot(nil, param.StateVersion)
			provingRoot, provingMDB := provingBackend.StorageRoot(nil, param.StateVersion)
			require.Equal(t, trieRoot, provingRoot)
			require.Equal(t, trieMDB.Drain(), provingMDB.Drain())
		}
	})

	t.Run("proof_recorded_and_checked", func(t *testing.T) {
		for _, stateVersion := range []storage.StateVersion{storage.StateVersionV0, storage.StateVersionV1} {
			contents := make([]StorageKeyValue, 0)
			for i := uint8(0); i < 64; i++ {
				contents = append(contents, StorageKeyValue{[]byte{i}, bytes.Repeat([]byte{i}, 34)})
			}

			inMemory := NewMemoryDBTrieBackend[hash.H256, runtime.BlakeTwo256]()
			inMemory = inMemory.update([]change{{
				ChildInfo:         nil,
				StorageCollection: contents,
			}}, stateVersion)
			inMemoryRoot, _ := inMemory.StorageRoot(nil, stateVersion)
			for i := uint8(0); i < 64; i++ {
				val, err := inMemory.Storage([]byte{i})
				require.NoError(t, err)
				require.Equal(t, bytes.Repeat([]byte{i}, 34), []byte(val))
			}

			trie := inMemory.TrieBackend
			trieRoot, _ := trie.StorageRoot(nil, stateVersion)
			require.Equal(t, inMemoryRoot, trieRoot)
			for i := uint8(0); i < 64; i++ {
				val, err := trie.Storage([]byte{i})
				require.NoError(t, err)
				require.Equal(t, bytes.Repeat([]byte{i}, 34), []byte(val))
			}

			for _, cache := range []*cache.SharedTrieCache[hash.H256]{cache.NewSharedTrieCache[hash.H256](1024 * 10), nil} {
				// Run multiple times to have a different cache conditions.
				for i := 0; i < 5; i++ {
					if cache != nil {
						if i == 2 {
							cache.ResetNodeCache()
						} else if i == 3 {
							cache.ResetValueCache()
						}
					}

					proving := NewWrappedTrieBackend(trie, nil, recorder.NewRecorder[hash.H256]())
					if cache != nil {
						local := cache.LocalTrieCache()
						proving.essence.trieNodeCache = &local
					}
					val, err := proving.Storage([]byte{42})
					require.NoError(t, err)
					require.NotNil(t, val)
					require.Equal(t, StorageValue(bytes.Repeat([]byte{42}, 34)), val)

					proof := proving.ExtractProof()
					require.NotNil(t, proof)

					proofCheck, err := NewProofCheckTrieBackend[hash.H256, runtime.BlakeTwo256](inMemoryRoot, *proof)
					require.NoError(t, err)

					val, err = proofCheck.Storage([]byte{42})
					require.NoError(t, err)
					require.NotNil(t, val)
					require.Equal(t, StorageValue(bytes.Repeat([]byte{42}, 34)), val)
				}
			}
		}
	})

	t.Run("proof_record_works_with_iter", func(t *testing.T) {
		for _, stateVersion := range []storage.StateVersion{storage.StateVersionV0, storage.StateVersionV1} {
			for _, sharedCache := range []*cache.SharedTrieCache[hash.H256]{
				cache.NewSharedTrieCache[hash.H256](1024 * 10), nil} {
				// Run multiple times to have a different cache conditions.
				for i := 0; i < 5; i++ {
					if sharedCache != nil {
						if i == 2 {
							sharedCache.ResetNodeCache()
						} else if i == 3 {
							sharedCache.ResetValueCache()
						}
					}

					contents := make([]StorageKeyValue, 0)
					for i := uint8(0); i < 64; i++ {
						contents = append(contents, StorageKeyValue{[]byte{i}, []byte{i}})
					}
					inMemory := NewMemoryDBTrieBackend[hash.H256, runtime.BlakeTwo256]()
					inMemory = inMemory.update([]change{{
						ChildInfo:         nil,
						StorageCollection: contents,
					}}, stateVersion)
					inMemoryRoot, _ := inMemory.StorageRoot(nil, stateVersion)
					for i := uint8(0); i < 64; i++ {
						val, err := inMemory.Storage([]byte{i})
						require.NoError(t, err)
						require.Equal(t, []byte{i}, []byte(val))
					}

					trie := inMemory.TrieBackend
					trieRoot, _ := trie.StorageRoot(nil, stateVersion)
					require.Equal(t, inMemoryRoot, trieRoot)
					for i := uint8(0); i < 64; i++ {
						val, err := trie.Storage([]byte{i})
						require.NoError(t, err)
						require.Equal(t, []byte{i}, []byte(val))
					}

					var local TrieCacheProvider[hash.H256, *cache.TrieCache[hash.H256]]
					if sharedCache != nil {
						ltc := sharedCache.LocalTrieCache()
						local = &ltc
					}
					proving := NewWrappedTrieBackend(trie, local, recorder.NewRecorder[hash.H256]())

					for i := uint8(0); i < 63; i++ {
						sk, err := proving.NextStorageKey([]byte{i})
						require.NoError(t, err)
						require.Equal(t, StorageKey([]byte{i + 1}), sk)
					}

					proof := proving.ExtractProof()
					require.NotNil(t, proof)

					proofCheck, err := NewProofCheckTrieBackend[hash.H256, runtime.BlakeTwo256](inMemoryRoot, *proof)
					require.NoError(t, err)
					for i := uint8(0); i < 63; i++ {
						sk, err := proofCheck.NextStorageKey([]byte{i})
						require.NoError(t, err)
						require.Equal(t, StorageKey([]byte{i + 1}), sk)
					}
				}
			}
		}
	})

	t.Run("proof_recorded_and_checked_with_child", func(t *testing.T) {
		for _, stateVersion := range []storage.StateVersion{storage.StateVersionV0, storage.StateVersionV1} {
			childInfo1 := storage.NewDefaultChildInfo([]byte("sub1"))
			childInfo2 := storage.NewDefaultChildInfo([]byte("sub2"))
			var contents []change
			var sc StorageCollection
			for i := uint8(0); i < 64; i++ {
				sc = append(sc, StorageKeyValue{[]byte{i}, []byte{i}})
			}
			contents = append(contents, change{StorageCollection: sc})
			sc = nil
			for i := uint8(28); i < 65; i++ {
				sc = append(sc, StorageKeyValue{[]byte{i}, []byte{i}})
			}
			contents = append(contents, change{ChildInfo: childInfo1, StorageCollection: sc})
			sc = nil
			for i := uint8(10); i < 15; i++ {
				sc = append(sc, StorageKeyValue{[]byte{i}, []byte{i}})
			}
			contents = append(contents, change{ChildInfo: childInfo2, StorageCollection: sc})
			inMemory := NewMemoryDBTrieBackend[hash.H256, runtime.BlakeTwo256]()
			inMemory = inMemory.update(contents, stateVersion)
			childStorageKeys := []storage.ChildInfo{childInfo1, childInfo2}
			var childDeltas []ChildDelta
			for _, childStorageKey := range childStorageKeys {
				childDeltas = append(childDeltas, ChildDelta{
					ChildInfo: childStorageKey,
					Deltas:    nil,
				})
			}
			inMemoryRoot, _ := inMemory.FullStorageRoot(nil, childDeltas, stateVersion)
			for i := uint8(0); i < 64; i++ {
				val, err := inMemory.Storage([]byte{i})
				require.NoError(t, err)
				require.Equal(t, []byte{i}, []byte(val))
			}
			for i := uint8(28); i < 65; i++ {
				val, err := inMemory.ChildStorage(childInfo1, []byte{i})
				require.NoError(t, err)
				require.Equal(t, []byte{i}, []byte(val))
			}
			for i := uint8(10); i < 15; i++ {
				val, err := inMemory.ChildStorage(childInfo2, []byte{i})
				require.NoError(t, err)
				require.Equal(t, []byte{i}, []byte(val))
			}

			for _, sharedCache := range []*cache.SharedTrieCache[hash.H256]{
				cache.NewSharedTrieCache[hash.H256](1024 * 10), nil} {
				// Run multiple times to have a different cache conditions.
				for i := 0; i < 5; i++ {
					if sharedCache != nil {
						if i == 2 {
							sharedCache.ResetNodeCache()
						} else if i == 3 {
							sharedCache.ResetValueCache()
						}
					}

					trie := inMemory.TrieBackend
					trieRoot, _ := trie.StorageRoot(nil, stateVersion)
					require.Equal(t, inMemoryRoot, trieRoot)
					for i := uint8(0); i < 64; i++ {
						val, err := trie.Storage([]byte{i})
						require.NoError(t, err)
						require.Equal(t, []byte{i}, []byte(val))
					}

					var local TrieCacheProvider[hash.H256, *cache.TrieCache[hash.H256]]
					if sharedCache != nil {
						ltc := sharedCache.LocalTrieCache()
						local = &ltc
					}
					proving := NewWrappedTrieBackend(trie, local, recorder.NewRecorder[hash.H256]())
					val, err := proving.Storage([]byte{42})
					require.NoError(t, err)
					require.Equal(t, []byte{42}, []byte(val))

					proof := proving.ExtractProof()
					require.NotNil(t, proof)

					proofCheck, err := NewProofCheckTrieBackend[hash.H256, runtime.BlakeTwo256](inMemoryRoot, *proof)
					require.NoError(t, err)
					_, err = proofCheck.Storage([]byte{0})
					require.Error(t, err)
					val, err = proofCheck.Storage([]byte{42})
					require.NoError(t, err)
					require.Equal(t, []byte{42}, []byte(val))
					// note that it is include in root because proof close
					val, err = proofCheck.Storage([]byte{41})
					require.NoError(t, err)
					require.Equal(t, []byte{41}, []byte(val))
					val, err = proofCheck.Storage([]byte{64})
					require.NoError(t, err)
					require.Nil(t, val)

					proving = NewWrappedTrieBackend(trie, nil, recorder.NewRecorder[hash.H256]())
					val, err = proving.ChildStorage(childInfo1, []byte{64})
					require.NoError(t, err)
					require.Equal(t, []byte{64}, []byte(val))
					val, err = proving.ChildStorage(childInfo1, []byte{25})
					require.NoError(t, err)
					require.Nil(t, val)
					val, err = proving.ChildStorage(childInfo2, []byte{14})
					require.NoError(t, err)
					require.Equal(t, []byte{14}, []byte(val))
					val, err = proving.ChildStorage(childInfo2, []byte{25})
					require.NoError(t, err)
					require.Nil(t, val)

					proof = proving.ExtractProof()
					require.NotNil(t, proof)

					proofCheck, err = NewProofCheckTrieBackend[hash.H256, runtime.BlakeTwo256](inMemoryRoot, *proof)
					require.NoError(t, err)
					val, err = proofCheck.ChildStorage(childInfo1, []byte{64})
					require.NoError(t, err)
					require.Equal(t, []byte{64}, []byte(val))
					val, err = proofCheck.ChildStorage(childInfo1, []byte{25})
					require.NoError(t, err)
					require.Nil(t, val)

					val, err = proofCheck.ChildStorage(childInfo2, []byte{14})
					require.NoError(t, err)
					require.Equal(t, []byte{14}, []byte(val))
					val, err = proofCheck.ChildStorage(childInfo2, []byte{25})
					require.NoError(t, err)
					require.Nil(t, val)
				}
			}
		}
	})

	// This tests an edge case when recording a child trie access with a cache.
	//
	// The accessed value/node is in the cache, but not the nodes to get to this value. So,
	// the recorder will need to traverse the trie to access these nodes from the backend when the
	// storage proof is generated.
	t.Run("child_proof_recording_with_edge_cases_works", func(t *testing.T) {
		for _, stateVersion := range []storage.StateVersion{storage.StateVersionV0, storage.StateVersionV1} {
			childInfo1 := storage.NewDefaultChildInfo([]byte("sub1"))
			var contents []change
			var sc StorageCollection
			for i := uint8(0); i < 64; i++ {
				sc = append(sc, StorageKeyValue{[]byte{i}, []byte{i}})
			}
			contents = append(contents, change{StorageCollection: sc})
			sc = nil
			for i := uint8(28); i < 65; i++ {
				sc = append(sc, StorageKeyValue{[]byte{i}, []byte{i}})
			}
			sc = append(sc, StorageKeyValue{[]byte{65}, bytes.Repeat([]byte{65}, 128)})
			contents = append(contents, change{ChildInfo: childInfo1, StorageCollection: sc})

			inMemory := NewMemoryDBTrieBackend[hash.H256, runtime.BlakeTwo256]()
			inMemory = inMemory.update(contents, stateVersion)
			childStorageKeys := []storage.ChildInfo{childInfo1}
			var childDeltas []ChildDelta
			for _, childStorageKey := range childStorageKeys {
				childDeltas = append(childDeltas, ChildDelta{
					ChildInfo: childStorageKey,
					Deltas:    nil,
				})
			}
			inMemoryRoot, _ := inMemory.FullStorageRoot(nil, childDeltas, stateVersion)
			child1Root, _, _ := inMemory.ChildStorageRoot(childInfo1, nil, stateVersion)
			trie := inMemory.TrieBackend

			type hashNode struct {
				hash.H256
				triedb.CachedNode[hash.H256]
			}
			var nodes []hashNode
			{
				backend := NewWrappedTrieBackend(trie, nil, recorder.NewRecorder[hash.H256]())
				val, err := backend.ChildStorage(childInfo1, []byte{65})
				require.NoError(t, err)
				require.NotNil(t, val)
				require.Equal(t, bytes.Repeat([]byte{65}, 128), []byte(val))
				valueHash := runtime.BlakeTwo256{}.Hash(val)

				proof := backend.ExtractProof()
				require.NotNil(t, proof)

				for _, node := range proof.Nodes() {
					h := runtime.BlakeTwo256{}.Hash(node)
					// Only insert the node/value that contains the important data.
					if h != valueHash {
						node, err := codec.Decode[hash.H256](bytes.NewBuffer(node))
						require.NoError(t, err)
						cachedNode, err := triedb.NewCachedNodeFromNode[hash.H256, runtime.BlakeTwo256](node)
						require.NoError(t, err)

						if data := cachedNode.Data(); data != nil {
							if bytes.Equal(data, bytes.Repeat([]byte{65}, 128)) {
								nodes = append(nodes, hashNode{h, cachedNode})
							}
						}
					} else if h == valueHash {
						nodes = append(nodes, hashNode{h, triedb.ValueCachedNode[hash.H256]{Value: node, Hash: h}})
					}
				}
			}

			cache := cache.NewSharedTrieCache[hash.H256](1024 * 10)
			{
				localCache := cache.LocalTrieCache()
				trieCache, unlock := localCache.TrieCache(child1Root)

				// Put the value/node into the cache.
				for _, hn := range nodes {
					trieCache.GetOrInsertNode(hn.H256, func() (triedb.CachedNode[hash.H256], error) {
						return hn.CachedNode, nil
					})

					if data := hn.CachedNode.Data(); data != nil {
						trieCache.SetValue([]byte{65}, triedb.ExistingCachedValue[hash.H256]{Hash: hn.H256, Data: data})
					}
				}
				unlock()
				localCache.Commit()
			}

			{
				// Record the access
				proving := NewWrappedTrieBackend(trie, nil, recorder.NewRecorder[hash.H256]())
				if cache != nil {
					local := cache.LocalTrieCache()
					proving.essence.trieNodeCache = &local
				}
				val, err := proving.ChildStorage(childInfo1, []byte{65})
				require.NoError(t, err)
				require.Equal(t, bytes.Repeat([]byte{65}, 128), []byte(val))

				proof := proving.ExtractProof()
				require.NotNil(t, proof)

				proofCheck, err := NewProofCheckTrieBackend[hash.H256, runtime.BlakeTwo256](inMemoryRoot, *proof)
				require.NoError(t, err)
				val, err = proofCheck.ChildStorage(childInfo1, []byte{65})
				require.NoError(t, err)
				require.Equal(t, bytes.Repeat([]byte{65}, 128), []byte(val))
			}
		}
	})

	t.Run("storage_proof_encoded_size_estimation_works", func(t *testing.T) {
		for _, param := range params {
			var cache *cache.LocalTrieCache[hash.H256]
			if param.SharedTrieCache != nil {
				local := param.SharedTrieCache.LocalTrieCache()
				cache = &local
			}

			hasCache := cache != nil
			tb := testTrie(t, param.StateVersion, cache, param.Recorder)
			keys := [][]byte{
				[]byte("key"),
				[]byte("value1"),
				[]byte("value2"),
				[]byte("doesnotexist"),
				[]byte("doesnotexist2"),
			}

			var checkEstimation = func(backend *TrieBackend[hash.H256, runtime.BlakeTwo256], hasCache bool) {
				estimation := backend.essence.recorder.EstimateEncodedSize()
				storageProof := backend.ExtractProof()
				require.NotNil(t, storageProof)
				var storageProofSize uint
				for _, n := range storageProof.Nodes() {
					storageProofSize += uint(len(scale.MustMarshal(n)))
				}

				if hasCache {
					// Estimation is not entirely correct when we have values already cached.
					require.True(t, estimation >= storageProofSize)
				} else {
					require.Equal(t, storageProofSize, estimation)
				}
			}

			for n := 0; n < len(keys); n++ {
				backend := NewWrappedTrieBackend(tb, nil, recorder.NewRecorder[hash.H256]())
				// Read n keys
				for i := 0; i < n; i++ {
					_, err := backend.Storage(keys[i])
					require.NoError(t, err)
				}

				// Check the estimation
				checkEstimation(backend, hasCache)
			}
		}
	})

	// Test to ensure that recording the same `key` for different tries works as expected.
	//
	// Each trie stores a different value under the same key. The values are big enough to
	// be not inlined with `StateVersion::V1`, this is important to test the expected behavior. The
	// trie recorder is expected to differentiate key access based on the different storage roots
	// of the tries.
	t.Run("recording_same_key_access_in_different_tries", func(t *testing.T) {
		for _, stateVersion := range []storage.StateVersion{storage.StateVersionV0, storage.StateVersionV1} {
			key := []byte("test_key")
			// Use some big values to ensure that we don't keep them inline
			topTrieVal := bytes.Repeat([]byte{1}, 1024)
			childTrie1Val := bytes.Repeat([]byte{2}, 1024)
			childTrie2Val := bytes.Repeat([]byte{3}, 1024)

			childInfo1 := storage.NewDefaultChildInfo([]byte("sub1"))
			childInfo2 := storage.NewDefaultChildInfo([]byte("sub2"))
			contents := []change{
				{StorageCollection: StorageCollection{StorageKeyValue{key, topTrieVal}}},
				{ChildInfo: childInfo1, StorageCollection: StorageCollection{StorageKeyValue{key, childTrie1Val}}},
				{ChildInfo: childInfo2, StorageCollection: StorageCollection{StorageKeyValue{key, childTrie2Val}}},
			}

			inMemory := NewMemoryDBTrieBackend[hash.H256, runtime.BlakeTwo256]()
			inMemory = inMemory.update(contents, stateVersion)
			childStorageKeys := []storage.ChildInfo{childInfo1, childInfo2}
			var childDeltas []ChildDelta
			for _, childStorageKey := range childStorageKeys {
				childDeltas = append(childDeltas, ChildDelta{
					ChildInfo: childStorageKey,
					Deltas:    nil,
				})
			}
			inMemoryRoot, _ := inMemory.FullStorageRoot(nil, childDeltas, stateVersion)
			val, err := inMemory.Storage(key)
			require.NoError(t, err)
			require.Equal(t, topTrieVal, []byte(val))

			val, err = inMemory.ChildStorage(childInfo1, key)
			require.NoError(t, err)
			require.Equal(t, childTrie1Val, []byte(val))

			val, err = inMemory.ChildStorage(childInfo2, key)
			require.NoError(t, err)
			require.Equal(t, childTrie2Val, []byte(val))

			for _, sharedCache := range []*cache.SharedTrieCache[hash.H256]{
				cache.NewSharedTrieCache[hash.H256](1024 * 10), nil} {
				// Run multiple times to have a different cache conditions.
				for i := 0; i < 5; i++ {
					if sharedCache != nil {
						if i == 2 {
							sharedCache.ResetNodeCache()
						} else if i == 3 {
							sharedCache.ResetValueCache()
						}
					}

					trie := inMemory.TrieBackend
					trieRoot, _ := trie.StorageRoot(nil, stateVersion)
					require.Equal(t, inMemoryRoot, trieRoot)

					var local TrieCacheProvider[hash.H256, *cache.TrieCache[hash.H256]]
					if sharedCache != nil {
						ltc := sharedCache.LocalTrieCache()
						local = &ltc
					}
					proving := NewWrappedTrieBackend(trie, local, recorder.NewRecorder[hash.H256]())

					val, err := proving.Storage(key)
					require.NoError(t, err)
					require.Equal(t, topTrieVal, []byte(val))
					val, err = proving.ChildStorage(childInfo1, key)
					require.NoError(t, err)
					require.Equal(t, childTrie1Val, []byte(val))
					val, err = proving.ChildStorage(childInfo2, key)
					require.NoError(t, err)
					require.Equal(t, childTrie2Val, []byte(val))

					proof := proving.ExtractProof()
					require.NotNil(t, proof)

					proofCheck, err := NewProofCheckTrieBackend[hash.H256, runtime.BlakeTwo256](inMemoryRoot, *proof)
					require.NoError(t, err)
					val, err = proofCheck.Storage(key)
					require.NoError(t, err)
					require.Equal(t, topTrieVal, []byte(val))
					val, err = proofCheck.ChildStorage(childInfo1, key)
					require.NoError(t, err)
					require.Equal(t, childTrie1Val, []byte(val))
					val, err = proofCheck.ChildStorage(childInfo2, key)
					require.NoError(t, err)
					require.Equal(t, childTrie2Val, []byte(val))

				}
			}
		}
	})
}
