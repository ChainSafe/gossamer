// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package inmemory

import (
	"github.com/karlseguin/ccache/v3"
)

// ccache uses 350 bytes of overhead per entry
const cacheValueOverheadSize = 350

// cacheValue is a helper alias over []byte to implement ccache.Sized
type cacheValue []byte

// Size returns the size of the cacheValue taking the overhead into account
func (cv cacheValue) Size() int64 {
	return int64(len(cv) + cacheValueOverheadSize)
}

// maxBytesLRUCache is an in-memory lru cache
// consider that the values are deleted asyncronously so there is a chance that
// the maxSize can be exceeded
// we can use lru.GC() to force the deletion of the items that should be deleted
type maxBytesLRUCache struct {
	lru *ccache.Cache[cacheValue]
}

// newlruCache creates a new lruCache
// maxSize is the cache max size in bytes
func newLruCache(maxSize int64) *maxBytesLRUCache {
	cache := ccache.New(ccache.Configure[cacheValue]().MaxSize(maxSize))
	return &maxBytesLRUCache{
		lru: cache,
	}
}

func (cache *maxBytesLRUCache) get(key string) []byte {
	item := cache.lru.Get(key)
	if item != nil {
		return item.Value()

	}
	return nil
}

func (cache *maxBytesLRUCache) set(key string, value []byte) {
	cache.lru.Set(key, cacheValue(value), 0)
}
