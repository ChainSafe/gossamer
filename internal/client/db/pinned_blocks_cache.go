// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package db

import (
	"log"
	"math"

	"github.com/ChainSafe/gossamer/internal/primitives/runtime"
	lru "github.com/hashicorp/golang-lru/v2"
)

// Entry for pinned blocks cache.
type pinnedBlocksCacheEntry struct {
	// How many times this item has been pinned
	refCount uint32

	// Cached justifications for this block
	Justifications runtime.Justifications

	// Cached body for this block
	Body *[]runtime.Extrinsic
}

func (pbce *pinnedBlocksCacheEntry) DecreaseRef() {
	if pbce.refCount > 0 {
		pbce.refCount--
	}
}

func (pbce *pinnedBlocksCacheEntry) IncreaseRef() {
	if pbce.refCount < math.MaxUint32 {
		pbce.refCount++
	}
}

func (pbce *pinnedBlocksCacheEntry) HasNoReferences() bool {
	return pbce.refCount == 0
}

// Reference counted cache for pinned block bodies and justifications.
type pinnedBlocksCache[H comparable] struct {
	cache *lru.Cache[H, *pinnedBlocksCacheEntry]
}

func newPinnedBlocksCache[H comparable]() pinnedBlocksCache[H] {
	cache, err := lru.NewWithEvict[H, *pinnedBlocksCacheEntry](1024, func(key H, value *pinnedBlocksCacheEntry) {
		// If reference count was larger than 0 on removal,
		// the item was removed due to capacity limitations.
		// Since the cache should be large enough for pinned items,
		// we want to know about these evictions.
		if value.refCount > 0 {
			log.Printf("TRACE: Pinned block cache limit reached. Evicting value. hash = %v\n", key)
		} else {
			log.Printf("TRACE: Evicting value from pinned block cache. hash = %v\n", key)
		}
	})
	if err != nil {
		panic(err)
	}
	return pinnedBlocksCache[H]{cache}
}

// Increase reference count of an item.
// Create an entry with empty value in the cache if necessary.
func (pbc *pinnedBlocksCache[H]) Pin(hash H) {
	prev, ok, _ := pbc.cache.PeekOrAdd(hash, &pinnedBlocksCacheEntry{refCount: 1})
	if ok {
		prev.IncreaseRef()
		log.Printf("TRACE: Bumped cache refcount. hash = %v, num_entries = %v\n", hash, pbc.cache.Len())
		pbc.cache.Add(hash, prev)
	} else {
		log.Printf("TRACE: Unable to bump reference count. hash = %v\n", hash)
	}
}

// Clear the cache
func (pbc *pinnedBlocksCache[H]) Clear() {
	pbc.cache.Purge()
}

// Check if item is contained in the cache
func (pbc *pinnedBlocksCache[H]) Contains(hash H) bool {
	return pbc.cache.Contains(hash)
}

// Attach body to an existing cache item
func (pbc *pinnedBlocksCache[H]) InsertBody(hash H, extrinsics []runtime.Extrinsic) {
	val, ok := pbc.cache.Peek(hash)
	if ok {
		val.Body = &extrinsics
		log.Printf("TRACE: Cached body. hash = %v, num_entries = %v\n", hash, pbc.cache.Len())
	} else {
		log.Printf("TRACE: Unable to insert body for uncached item. hash = %v\n", hash)
	}
}

// Attach justification to an existing cache item
func (pbc *pinnedBlocksCache[H]) InsertJustifications(hash H, justifications runtime.Justifications) {
	val, ok := pbc.cache.Peek(hash)
	if ok {
		val.Justifications = justifications
		log.Printf("TRACE: Cached justification. hash = %v, num_entries = %v\n", hash, pbc.cache.Len())
	} else {
		log.Printf("TRACE: Unable to insert justifications for uncached item. hash = %v\n", hash)
	}
}

// Decreases reference count of an item.
// If the count hits 0, the item is removed.
func (pbc *pinnedBlocksCache[H]) Unpin(hash H) {
	val, ok := pbc.cache.Peek(hash)
	if ok {
		val.DecreaseRef()
		if val.HasNoReferences() {
			pbc.cache.Remove(hash)
		}
	}
}

// Get justifications for cached block
func (pbc *pinnedBlocksCache[H]) Justifications(hash H) runtime.Justifications {
	val, ok := pbc.cache.Peek(hash)
	if ok {
		return val.Justifications
	}
	return nil
}

// Get body for cached block
func (pbc *pinnedBlocksCache[H]) Body(hash H) *[]runtime.Extrinsic {
	val, ok := pbc.cache.Peek(hash)
	if ok {
		return val.Body
	}
	return nil
}
