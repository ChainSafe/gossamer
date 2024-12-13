// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package db

import (
	"math"

	"github.com/ChainSafe/gossamer/internal/primitives/runtime"
	lru "github.com/hashicorp/golang-lru/v2"
)

// Entry for pinned blocks cache.
type pinnedBlocksCacheEntry[E runtime.Extrinsic] struct {
	// How many times this item has been pinned
	refCount uint32

	// Cached justifications for this block
	Justifications *runtime.Justifications

	// Cached body for this block
	Body *[]E
}

func (pbce *pinnedBlocksCacheEntry[E]) DecreaseRef() {
	if pbce.refCount > 0 {
		pbce.refCount--
	} else {
		panic("can not decrease refCount less than 0")
	}
}

func (pbce *pinnedBlocksCacheEntry[E]) IncreaseRef() {
	if pbce.refCount < math.MaxUint32 {
		pbce.refCount++
	} else {
		panic("can not increase refCount greater than MaxUint32")
	}
}

func (pbce *pinnedBlocksCacheEntry[E]) HasNoReferences() bool {
	return pbce.refCount == 0
}

// Reference counted cache for pinned block bodies and justifications.
type pinnedBlocksCache[H comparable, E runtime.Extrinsic] struct {
	cache *lru.Cache[H, *pinnedBlocksCacheEntry[E]]
}

func newPinnedBlocksCache[H comparable, E runtime.Extrinsic]() pinnedBlocksCache[H, E] {
	cache, err := lru.NewWithEvict[H, *pinnedBlocksCacheEntry[E]](1024, func(key H, value *pinnedBlocksCacheEntry[E]) {
		// If reference count was larger than 0 on removal,
		// the item was removed due to capacity limitations.
		// Since the cache should be large enough for pinned items,
		// we want to know about these evictions.
		if value.refCount > 0 {
			logger.Tracef("Pinned block cache limit reached. Evicting value. hash = %v", key)
		} else {
			logger.Tracef("Evicting value from pinned block cache. hash = %v", key)
		}
	})
	if err != nil {
		panic(err)
	}
	return pinnedBlocksCache[H, E]{cache}
}

// Increase reference count of an item.
// Create an entry with empty value in the cache if necessary.
func (pbc *pinnedBlocksCache[H, E]) Pin(hash H) {
	prev, ok, _ := pbc.cache.PeekOrAdd(hash, &pinnedBlocksCacheEntry[E]{refCount: 1})
	if ok {
		prev.IncreaseRef()
		logger.Tracef("Bumped cache refcount. hash = %v, num_entries = %v", hash, pbc.cache.Len())
		pbc.cache.Add(hash, prev)
	} else {
		logger.Tracef("Unable to bump reference count. hash = %v", hash)
	}
}

// Clear the cache
func (pbc *pinnedBlocksCache[H, E]) Clear() {
	pbc.cache.Purge()
}

// Check if item is contained in the cache
func (pbc *pinnedBlocksCache[H, E]) Contains(hash H) bool {
	return pbc.cache.Contains(hash)
}

// Attach body to an existing cache item
func (pbc *pinnedBlocksCache[H, E]) InsertBody(hash H, extrinsics []E) {
	val, ok := pbc.cache.Peek(hash)
	if ok {
		val.Body = &extrinsics
		logger.Tracef("Cached body. hash = %v, num_entries = %v", hash, pbc.cache.Len())
	} else {
		logger.Tracef("Unable to insert body for uncached item. hash = %v", hash)
	}
}

// Attach justification to an existing cache item
func (pbc *pinnedBlocksCache[H, E]) InsertJustifications(hash H, justifications runtime.Justifications) {
	val, ok := pbc.cache.Peek(hash)
	if ok {
		val.Justifications = &justifications
		logger.Tracef("Cached justification. hash = %v, num_entries = %v", hash, pbc.cache.Len())
	} else {
		logger.Tracef("Unable to insert justifications for uncached item. hash = %v", hash)
	}
}

// Decreases reference count of an item.
// If the count hits 0, the item is removed.
func (pbc *pinnedBlocksCache[H, E]) Unpin(hash H) {
	val, ok := pbc.cache.Peek(hash)
	if ok {
		val.DecreaseRef()
		if val.HasNoReferences() {
			pbc.cache.Remove(hash)
		}
	}
}

// Get justifications for cached block
func (pbc *pinnedBlocksCache[H, E]) Justifications(hash H) *runtime.Justifications {
	val, ok := pbc.cache.Peek(hash)
	if ok {
		return val.Justifications
	}
	return nil
}

// Get body for cached block
func (pbc *pinnedBlocksCache[H, E]) Body(hash H) *[]E {
	val, ok := pbc.cache.Peek(hash)
	if ok {
		return val.Body
	}
	return nil
}
