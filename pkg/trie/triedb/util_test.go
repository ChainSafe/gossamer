// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package triedb

import (
	memorydb "github.com/ChainSafe/gossamer/internal/memory-db"
	chash "github.com/ChainSafe/gossamer/internal/primitives/core/hash"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime"
	"github.com/ChainSafe/gossamer/pkg/trie/triedb/hash"
)

func NewMemoryDB() *memorydb.MemoryDB[
	chash.H256, runtime.BlakeTwo256, chash.H256, memorydb.HashKey[chash.H256]] {
	db := memorydb.NewMemoryDB[
		chash.H256, runtime.BlakeTwo256, chash.H256, memorydb.HashKey[chash.H256],
	]([]byte{0})
	return &db
}

type TestTrieCache[H hash.Hash] struct {
	valueCache map[string]CachedValue[H]
	nodeCache  map[H]CachedNode[H]
}

func NewTestTrieCache[H hash.Hash]() *TestTrieCache[H] {
	return &TestTrieCache[H]{
		valueCache: make(map[string]CachedValue[H]),
		nodeCache:  make(map[H]CachedNode[H]),
	}
}

func (ttc *TestTrieCache[H]) GetValue(key []byte) CachedValue[H] {
	cv, ok := ttc.valueCache[string(key)]
	if !ok {
		return nil
	}
	return cv
}

func (ttc *TestTrieCache[H]) SetValue(key []byte, value CachedValue[H]) {
	ttc.valueCache[string(key)] = value
}

func (ttc *TestTrieCache[H]) GetOrInsertNode(hash H, fetchNode func() (CachedNode[H], error)) (CachedNode[H], error) {
	node, ok := ttc.nodeCache[hash]
	if !ok {
		var err error
		node, err = fetchNode()
		if err != nil {
			return nil, err
		}
		ttc.nodeCache[hash] = node
	}
	return node, nil
}

func (ttc *TestTrieCache[H]) GetNode(hash H) CachedNode[H] {
	node, ok := ttc.nodeCache[hash]
	if !ok {
		return nil
	}
	return node
}

var _ TrieCache[chash.H256] = &TestTrieCache[chash.H256]{}
