// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package blockchain

import (
	"sync"

	"github.com/ChainSafe/gossamer/internal/primitives/runtime"
	lru "github.com/hashicorp/golang-lru/v2"
)

// Handles header metadata: hash, number, parent hash, etc.
type HeaderMetadata[H, N any] interface {
	HeaderMetadata(hash H) (CachedHeaderMetadata[H, N], error)
	InsertHeaderMetadata(hash H, headerMetadata CachedHeaderMetadata[H, N])
	RemoveHeaderMetadata(hash H)
}

// Caches header metadata in an in-memory LRU cache.
type HeaderMetadataCache[H comparable, N any] struct {
	cache *lru.Cache[H, CachedHeaderMetadata[H, N]]
	sync.RWMutex
}

// NewHeaderMetadataCache is constructor for HeaderMetadataCache.
func NewHeaderMetadataCache[H comparable, N any](capacity ...uint32) HeaderMetadataCache[H, N] {
	var cap int = 5000
	if len(capacity) > 0 && capacity[0] > 0 {
		cap = int(capacity[0])
	}
	cache, err := lru.New[H, CachedHeaderMetadata[H, N]](cap)
	if err != nil {
		panic(err)
	}
	return HeaderMetadataCache[H, N]{
		cache: cache,
	}
}

// HeaderMetadata returns the CachedHeaderMetadata for a given hash or `nil` if not found.
func (hmc *HeaderMetadataCache[H, N]) HeaderMetadata(hash H) *CachedHeaderMetadata[H, N] {
	hmc.RLock()
	defer hmc.RUnlock()
	val, ok := hmc.cache.Get(hash)
	if !ok {
		return nil
	}
	return &val
}

// InsertHeaderMetadata inserts a supplied `metadata` for a `hash`.
func (hmc *HeaderMetadataCache[H, N]) InsertHeaderMetadata(hash H, metadata CachedHeaderMetadata[H, N]) {
	hmc.Lock()
	defer hmc.Unlock()
	hmc.cache.Add(hash, metadata)
}

// RemoveHeaderMetadata removes the `metadata` for a `hash`.
func (hmc *HeaderMetadataCache[H, N]) RemoveHeaderMetadata(hash H) {
	hmc.Lock()
	defer hmc.Unlock()
	hmc.cache.Remove(hash)
}

// CachedHeaderMeatadata is used to efficiently traverse the tree.
type CachedHeaderMetadata[H, N any] struct {
	/// Hash of the header.
	Hash H
	/// Block number.
	Number N
	/// Hash of parent header.
	Parent H
	/// Block state root.
	StateRoot H
	/// Hash of an ancestor header. Used to jump through the tree.
	ancestor H
}

// NewCachedHeaderMetadata is constructor for CachedHeaderMetadata
func NewCachedHeaderMetadata[H runtime.Hash, N runtime.Number](header runtime.Header[N, H]) CachedHeaderMetadata[H, N] {
	return CachedHeaderMetadata[H, N]{
		Hash:      header.Hash(),
		Number:    header.Number(),
		Parent:    header.ParentHash(),
		StateRoot: header.StateRoot(),
		ancestor:  header.ParentHash(),
	}
}
