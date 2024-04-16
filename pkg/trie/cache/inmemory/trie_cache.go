// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package inmemory

import (
	"github.com/ChainSafe/gossamer/pkg/trie/cache"
)

// https://github.com/paritytech/polkadot-sdk/blob/a8f4f4f00f8fc0da512a09e1450bf4cda954d70d/substrate/primitives/trie/src/cache/mod.rs#L98
const defaultNodeCacheMaxSize = 8 * 1024 * 1024  // 8MB
const defaultValueCacheMaxSize = 2 * 1024 * 1024 // 2MB

// TrieInMemoryCache is an in-memory cache for trie nodes
type trieInMemoryCache struct {
	nodeCache  *lruCache
	valueCache *lruCache
}

// NewTrieInMemoryCache creates a new TrieInMemoryCache
func NewTrieInMemoryCache() *trieInMemoryCache {
	return &trieInMemoryCache{
		nodeCache:  newLruCache(defaultNodeCacheMaxSize),
		valueCache: newLruCache(defaultValueCacheMaxSize),
	}
}

// GetValue returns the value for the given key
func (tc *trieInMemoryCache) GetValue(key []byte) []byte {
	return tc.valueCache.get(string(key))
}

// SetValue sets the value for the given key
func (tc *trieInMemoryCache) SetValue(key []byte, value []byte) {
	tc.valueCache.set(string(key), value)
}

// GetNode returns the node for the given key
func (tc *trieInMemoryCache) GetNode(key []byte) []byte {
	return tc.nodeCache.get(string(key))
}

// SetNode sets the node for the given key
func (tc *trieInMemoryCache) SetNode(key, value []byte) {
	tc.nodeCache.set(string(key), value)
}

var _ cache.TrieCache = (*trieInMemoryCache)(nil)
