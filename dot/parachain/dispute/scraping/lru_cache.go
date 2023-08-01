package scraping

import (
	"container/list"
	"github.com/ChainSafe/gossamer/lib/common"
)

// DefaultLRUObservedBlockCapacity is the default capacity of the LRU cache.
const DefaultLRUObservedBlockCapacity = 20

// LRUCache represents the LRU cache.
type LRUCache struct {
	capacity int
	cache    map[common.Hash]*list.Element
	lruList  *list.List
}

// Entry represents an item in the cache.
type Entry struct {
	key   common.Hash
	value any
}

// NewLRUCache creates a new LRU cache with the specified capacity.
func NewLRUCache(capacity int) *LRUCache {
	if capacity <= 0 {
		capacity = DefaultLRUObservedBlockCapacity
	}

	return &LRUCache{
		capacity: capacity,
		cache:    make(map[common.Hash]*list.Element),
		lruList:  list.New(),
	}
}

// Get retrieves the value associated with the given key from the cache.
func (c *LRUCache) Get(key common.Hash) any {
	if elem, exists := c.cache[key]; exists {
		c.lruList.MoveToFront(elem)
		return elem.Value.(*Entry).value
	}
	return nil
}

// Put adds a key-value pair to the cache.
func (c *LRUCache) Put(key common.Hash, value any) {
	// If the key already exists in the cache, update its value and move it to the front.
	if elem, exists := c.cache[key]; exists {
		elem.Value.(*Entry).value = value
		c.lruList.MoveToFront(elem)
		return
	}

	// If the cache is full, remove the least recently used item (from the back of the list).
	if len(c.cache) >= c.capacity {
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
}
