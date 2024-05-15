// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package inmemory

import (
	lrucache "github.com/ChainSafe/gossamer/lib/utils/lru-cache"
	"github.com/ChainSafe/gossamer/pkg/trie/cache"
)

// https://github.com/paritytech/polkadot-sdk/blob/a8f4f4f00f8fc0da512a09e1450bf4cda954d70d/substrate/primitives/trie/src/cache/mod.rs#L98
const defaultNodeCacheMaxElements = 10000
const defaultValueCacheMaxSize = 2 * 1024 * 1024 // 2MB

// TrieInMemoryCache is an in-memory cache for trie nodes
type TrieInMemoryCache struct {
	nodeCache  *lrucache.LRUCache[string, []byte]
	valueCache *lruCache
}

// NewTrieInMemoryCache creates a new TrieInMemoryCache
func NewTrieInMemoryCache() *TrieInMemoryCache {
	return &TrieInMemoryCache{
		nodeCache:  lrucache.NewLRUCache[string, []byte](defaultNodeCacheMaxElements),
		valueCache: newLruCache(defaultValueCacheMaxSize),
	}
}

// GetValue returns the value for the given key
func (tc *TrieInMemoryCache) GetValue(key []byte) []byte {
	return tc.valueCache.get(string(key))
}

// SetValue sets the value for the given key
func (tc *TrieInMemoryCache) SetValue(key []byte, value []byte) {
	tc.valueCache.set(string(key), value)
}

// GetNode returns the node for the given key
func (tc *TrieInMemoryCache) GetNode(key []byte) []byte {
	return tc.nodeCache.Get(string(key))
}

// SetNode sets the node for the given key
func (tc *TrieInMemoryCache) SetNode(key, value []byte) {
	tc.nodeCache.Put(string(key), value)
}

var _ cache.TrieCache = (*TrieInMemoryCache)(nil)
