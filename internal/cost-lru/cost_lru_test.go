// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package costlru

import (
	"testing"

	"github.com/ChainSafe/gossamer/internal/primitives/core/hash"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime"
	"github.com/dolthub/maphash"
	"github.com/stretchr/testify/require"
)

type Hasher struct {
	maphash.Hasher[ValueCacheKeyHash[hash.H256]]
}

func (h Hasher) Hash(key ValueCacheKeyHash[hash.H256]) uint32 {
	return uint32(h.Hasher.Hash(key))
}

type ValueCacheKeyHash[H runtime.Hash] struct {
	StorageRoot H
	StorageKey  string
}

func TestLRU(t *testing.T) {
	hasher := Hasher{maphash.NewHasher[ValueCacheKeyHash[hash.H256]]()}

	var costFunc = func(key ValueCacheKeyHash[hash.H256], val []byte) uint32 {
		keyCost := uint32(len(key.StorageKey))
		return keyCost + uint32(len(val))
	}

	maxNum := uint(1024)
	maxSize := uint(costFunc(ValueCacheKeyHash[hash.H256]{
		StorageRoot: hash.NewRandomH256(),
		StorageKey:  string(hash.NewRandomH256()),
	}, []byte{1})) * maxNum

	l, err := New[ValueCacheKeyHash[hash.H256], []byte](maxSize, hasher.Hash, costFunc)
	require.NoError(t, err)

	someRoot := hash.NewRandomH256()
	allKeys := make([]ValueCacheKeyHash[hash.H256], 0)
	for i := 0; i < int(maxNum)*2; i++ {
		hash := ValueCacheKeyHash[hash.H256]{
			StorageRoot: someRoot,
			StorageKey:  string(hash.NewRandomH256()),
		}
		allKeys = append(allKeys, hash)
		added, evicted := l.Add(hash, []byte{uint8(i)})
		require.True(t, added)
		if i >= int(maxNum) {
			require.True(t, evicted)
		} else {
			require.False(t, evicted)
		}
	}

	require.Equal(t, maxSize, l.currentCost)

	for i, key := range l.Keys() {
		require.Equal(t, allKeys[int(maxNum)+i], key)
	}
}

func TestLRU_EvictCallback(t *testing.T) {
	hasher := Hasher{maphash.NewHasher[ValueCacheKeyHash[hash.H256]]()}

	var costFunc = func(key ValueCacheKeyHash[hash.H256], val []byte) uint32 {
		keyCost := uint32(len(key.StorageKey))
		return keyCost + uint32(len(val))
	}

	maxNum := uint(1024)
	maxSize := uint(costFunc(ValueCacheKeyHash[hash.H256]{
		StorageRoot: hash.NewRandomH256(),
		StorageKey:  string(hash.NewRandomH256()),
	}, []byte{1})) * maxNum

	evictCount := 0
	l, err := New[ValueCacheKeyHash[hash.H256], []byte](maxSize, hasher.Hash, costFunc)
	require.NoError(t, err)
	l.SetOnEvict(func(vckh ValueCacheKeyHash[hash.H256], b []byte) {
		evictCount++
	})

	someRoot := hash.NewRandomH256()
	allKeys := make([]ValueCacheKeyHash[hash.H256], 0)
	for i := 0; i < int(maxNum)*2; i++ {
		hash := ValueCacheKeyHash[hash.H256]{
			StorageRoot: someRoot,
			StorageKey:  string(hash.NewRandomH256()),
		}
		allKeys = append(allKeys, hash)
		added, evicted := l.Add(hash, []byte{uint8(i)})
		require.True(t, added)
		if i >= int(maxNum) {
			require.True(t, evicted)
		} else {
			require.False(t, evicted)
		}
	}

	require.Equal(t, maxSize, l.currentCost)

	for i, key := range l.Keys() {
		require.Equal(t, allKeys[int(maxNum)+i], key)
	}

	require.Equal(t, int(maxNum), evictCount)
}

func TestLRU_Purge(t *testing.T) {
	hasher := Hasher{maphash.NewHasher[ValueCacheKeyHash[hash.H256]]()}

	var costFunc = func(key ValueCacheKeyHash[hash.H256], val []byte) uint32 {
		keyCost := uint32(len(key.StorageKey))
		return keyCost + uint32(len(val))
	}

	maxNum := uint(3)
	maxSize := uint(costFunc(ValueCacheKeyHash[hash.H256]{
		StorageRoot: hash.NewRandomH256(),
		StorageKey:  string(hash.NewRandomH256()),
	}, []byte{1})) * maxNum

	l, err := New[ValueCacheKeyHash[hash.H256], []byte](maxSize, hasher.Hash, costFunc)
	require.NoError(t, err)

	someRoot := hash.NewRandomH256()
	allKeys := make([]ValueCacheKeyHash[hash.H256], 0)
	for i := 0; i < int(maxNum)*2; i++ {
		hash := ValueCacheKeyHash[hash.H256]{
			StorageRoot: someRoot,
			StorageKey:  string(hash.NewRandomH256()),
		}
		allKeys = append(allKeys, hash)
		added, evicted := l.Add(hash, []byte{uint8(i)})
		require.True(t, added)
		if i >= int(maxNum) {
			require.True(t, evicted)
		} else {
			require.False(t, evicted)
		}
	}

	require.Equal(t, maxSize, l.currentCost)

	for i, key := range l.Keys() {
		require.Equal(t, allKeys[int(maxNum)+i], key)
	}

	l.Purge()
	require.Equal(t, 0, l.Len())
	require.Equal(t, uint(0), l.currentCost)
}

func TestLRU_Same_Entries(t *testing.T) {
	hasher := Hasher{maphash.NewHasher[ValueCacheKeyHash[hash.H256]]()}

	var costFunc = func(key ValueCacheKeyHash[hash.H256], val []byte) uint32 {
		keyCost := uint32(len(key.StorageKey))
		return keyCost + uint32(len(val))
	}

	someKey := ValueCacheKeyHash[hash.H256]{
		StorageRoot: hash.NewRandomH256(),
		StorageKey:  string(hash.NewRandomH256()),
	}

	maxNum := uint(5)
	maxSize := uint(costFunc(someKey, []byte{1})) * maxNum

	l, err := New[ValueCacheKeyHash[hash.H256], []byte](maxSize, hasher.Hash, costFunc)
	require.NoError(t, err)

	for i := 0; i < int(maxNum); i++ {
		added, evicted := l.Add(someKey, []byte{1})
		require.True(t, added)
		require.False(t, evicted)
	}
	require.Equal(t, 1, l.Len())
	require.Equal(t, int(l.Cost()), int(costFunc(someKey, []byte{1})))

}
