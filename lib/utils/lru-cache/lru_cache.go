// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

// Package lrucache provides a generic LRU (Least Recently Used) cache implementation in Go,
// capable of storing key-value pairs of any comparable key type and any value type. It supports
// concurrent read and write operations and automatically evicts the least recently used item
// when the cache reaches its capacity. The cache is backed by a doubly linked list and a map
// for fast access and eviction.
//
// Example usage:
//   cache := NewLRUCache[string, string](100) // Create a cache with a capacity of 100
//   cache.Put("key1", value1)
//   val1 := cache.Get("key1")
//
// Note: The LRUCache does not automatically resize, and values must be comparable.
//

package lrucache

import (
	"container/list"
	"sync"
)

// DefaultLRUCapacity is the default capacity of the LRU cache.
const DefaultLRUCapacity = 20

// LRUCache represents the LRU cache.
type LRUCache[K comparable, V any] struct {
	sync.RWMutex
	capacity uint
	cache    map[K]*list.Element
	lruList  *list.List
}

// Entry represents an item in the cache.
type Entry[K comparable, V any] struct {
	key   K
	value V
}

// NewLRUCache creates a new LRU cache with the specified capacity.
func NewLRUCache[K comparable, V any](capacity uint) *LRUCache[K, V] {
	if capacity < 1 {
		capacity = DefaultLRUCapacity
	}

	return &LRUCache[K, V]{
		capacity: capacity,
		cache:    make(map[K]*list.Element),
		lruList:  list.New(),
	}
}

// Get retrieves the value associated with the given key from the cache.
func (c *LRUCache[K, V]) Get(key K) V {
	c.RLock()
	defer c.RUnlock()

	if elem, exists := c.cache[key]; exists {
		c.lruList.MoveToFront(elem)
		return elem.Value.(*Entry[K, V]).value
	}

	var zeroV V
	return zeroV
}

// Put adds a key-value pair to the cache.
func (c *LRUCache[K, V]) Put(key K, value V) {
	c.Lock()
	defer c.Unlock()

	// If the key already exists in the cache, update its value and move it to the front.
	if elem, exists := c.cache[key]; exists {
		elem.Value.(*Entry[K, V]).value = value
		c.lruList.MoveToFront(elem)
		return
	}

	// If the cache is full, remove the least recently used item (from the back of the list).
	if len(c.cache) >= int(c.capacity) {
		// Get the least recently used item (back of the list).
		lastElem := c.lruList.Back()
		if lastElem != nil {
			delete(c.cache, lastElem.Value.(*Entry[K, V]).key)
			c.lruList.Remove(lastElem)
		}
	}

	// Add the new key-value pair to the cache (at the front of the list).
	newEntry := &Entry[K, V]{key: key, value: value}
	newElem := c.lruList.PushFront(newEntry)
	c.cache[key] = newElem
}
