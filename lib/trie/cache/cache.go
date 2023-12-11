// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package cache

import (
	"bytes"
	"container/list"
	"sync"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/trie"
)

const DefaultCapacity = 8 * 1024 * 1024 // bytes

// LRUCache represents the LRU cache.
type LRUCache struct {
	sync.RWMutex
	capacity   uint
	memoryUsed uint
	cache      map[common.Hash]*list.Element
	lruList    *list.List
}

// Entry represents an item in the cache.
type Entry struct {
	key   common.Hash
	value trie.Node
}

// NewLRUCache creates a new LRU cache with the specified capacity.
func NewLRUCache(capacity uint) *LRUCache {
	if capacity < 1 {
		capacity = DefaultCapacity
	}

	return &LRUCache{
		capacity: capacity,
		cache:    make(map[common.Hash]*list.Element),
		lruList:  list.New(),
	}
}

// Get retrieves the value associated with the given key from the cache.
func (c *LRUCache) Get(key common.Hash) trie.Node {
	c.RLock()
	defer c.RUnlock()

	if elem, exists := c.cache[key]; exists {
		c.lruList.MoveToFront(elem)
		return elem.Value.(*Entry).value
	}

	var zeroV trie.Node
	return zeroV
}

// Has checks if the cache contains the given key.
func (c *LRUCache) Has(key common.Hash) bool {
	c.RLock()
	defer c.RUnlock()

	_, exists := c.cache[key]
	return exists
}

// Put adds a key-value pair to the cache.
func (c *LRUCache) Put(key common.Hash, value trie.Node) error {
	c.Lock()
	defer c.Unlock()

	return c.insertEntry(key, value)
}

// SoftPut adds a key-value pair to the cache if it does not already exist.
func (c *LRUCache) SoftPut(key common.Hash, value trie.Node) (bool, error) {
	c.Lock()
	defer c.Unlock()

	if _, exists := c.cache[key]; exists {
		return false, nil
	}

	err := c.insertEntry(key, value)
	if err != nil {
		return false, err
	}
	return true, nil
}

// Delete removes the given key from the cache.
func (c *LRUCache) Delete(key common.Hash) bool {
	c.Lock()
	defer c.Unlock()

	val, exists := c.cache[key]
	if !exists {
		return false
	}

	c.lruList.Remove(val)

	delete(c.cache, key)
	return true
}

// Len returns the number of items in the cache.
func (c *LRUCache) Len() int {
	c.Lock()
	defer c.Unlock()

	return len(c.cache)
}

func (c *LRUCache) insertEntry(key common.Hash, value trie.Node) error {
	// If the key already exists in the cache, update its value and move it to the front.
	if elem, exists := c.cache[key]; exists {
		elem.Value.(*Entry).value = value
		c.lruList.MoveToFront(elem)
		return nil
	}

	buffer := bytes.NewBuffer(nil)
	err := value.Encode(buffer, trie.NoMaxInlineValueSize)
	if err != nil {
		return err
	}

	// If the cache is full, remove the least recently used item (from the back of the list).
	if buffer.Len() >= int(c.capacity) {
		// Get the least recently used item (back of the list).
		lastElem := c.lruList.Back()
		if lastElem != nil {
			delete(c.cache, lastElem.Value.(*Entry).key)
			c.lruList.Remove(lastElem)
		}
	}

	// Add the new key-value pair to the cache (at the front of the list).
	newEntry := &Entry{key: key, value: value}
	newElem := c.lruList.PushFront(newEntry)
	c.cache[key] = newElem

	c.memoryUsed += uint(buffer.Len())

	return nil
}
