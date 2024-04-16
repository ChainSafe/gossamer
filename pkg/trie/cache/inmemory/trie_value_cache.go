package inmemory

import (
	"github.com/karlseguin/ccache/v3"
)

type cacheValue []byte

func (cv cacheValue) Size() int64 {
	return int64(len(cv))
}

type TrieValueCache struct {
	lru *ccache.Cache[[]byte]
}

func NewTrieValueCache(maxSize int64) *TrieValueCache {
	cache := ccache.New(ccache.Configure[[]byte]().MaxSize(maxSize))
	return &TrieValueCache{
		lru: cache,
	}
}

func (cache *TrieValueCache) getValue(key []byte) []byte {
	item := cache.lru.Get(string(key))
	if item != nil {
		return item.Value()

	}
	return nil
}

func (cache *TrieValueCache) setValue(key []byte, value []byte) {
	cache.lru.Set(string(key), cacheValue(value), 0)
}
