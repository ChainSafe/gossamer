// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package inmemory

import (
	"github.com/karlseguin/ccache/v3"
)

// cacheValue is a helper alias over []byte to implement ccache.Sized
type cacheValue []byte

// Size returns the size of the cacheValue + 350 bytes since ccache use it to
// store each entry
func (cv cacheValue) Size() int64 {
	return int64(len(cv) + 350)
}

// TrieValueCache is an in-memory cache for trie values
// consider that the values are deleted asyncronously so there is a change that
// the maxSize can be exceeded
// we can use lru.GC() to force the deletion of the items that should be deleted
type TrieValueCache struct {
	lru *ccache.Cache[cacheValue]
}

// newTrieValueCache creates a new TrieValueCache
// maxSize is the cache max size in bytes
func newTrieValueCache(maxSize int64) *TrieValueCache {
	cache := ccache.New(ccache.Configure[cacheValue]().MaxSize(maxSize))
	return &TrieValueCache{
		lru: cache,
	}
}

func (cache *TrieValueCache) getValue(key []byte) []byte {
	item := cache.lru.Get(string(key))
	if item != nil {
		return item.Value()

	}
	return nil
}

func (cache *TrieValueCache) setValue(key []byte, value []byte) {
	cache.lru.Set(string(key), cacheValue(value), 0)
}
