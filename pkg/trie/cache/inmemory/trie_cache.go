// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package inmemory

import (
	"github.com/ChainSafe/gossamer/pkg/trie/cache"
)

// https://github.com/paritytech/polkadot-sdk/blob/a8f4f4f00f8fc0da512a09e1450bf4cda954d70d/substrate/primitives/trie/src/cache/mod.rs#L98
const defaultValueCacheMaxSize = 2 * 1024 * 1024 // 2MB

// TrieInMemoryCache is an in-memory cache for trie nodes
type TrieInMemoryCache struct {
	valueCache *TrieValueCache
}

func NewTrieInMemoryCache() *TrieInMemoryCache {
	return &TrieInMemoryCache{
		valueCache: NewTrieValueCache(defaultValueCacheMaxSize),
	}
}

func (tc *TrieInMemoryCache) GetValue(key []byte) []byte {
	return tc.valueCache.getValue(key)
}

func (tc *TrieInMemoryCache) SetValue(key []byte, value []byte) {
	tc.valueCache.setValue(key, value)
}

var _ cache.TrieCache = (*TrieInMemoryCache)(nil)
