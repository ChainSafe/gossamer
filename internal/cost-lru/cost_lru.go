// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package costlru

import (
	"math"
	"time"

	"github.com/elastic/go-freelru"
)

// LRU is a cost based LRU wrapped around [freelru.LRU].
type LRU[K comparable, V any] struct {
	currentCost     uint
	maxCost         uint
	costFunc        func(K, V) uint32
	onEvictCallback freelru.OnEvictCallback[K, V]
	*freelru.LRU[K, V]
}

// Costructor for [LRU].
func New[K comparable, V any](
	maxCost uint, hash freelru.HashKeyCallback[K], costFunc func(K, V) uint32,
) (*LRU[K, V], error) {
	var capacity = uint32(math.MaxUint32)
	if maxCost < math.MaxUint32 {
		capacity = uint32(maxCost)
	}
	lru, err := freelru.New[K, V](capacity, hash)
	if err != nil {
		return nil, err
	}

	l := LRU[K, V]{
		maxCost:  maxCost,
		costFunc: costFunc,
		LRU:      lru,
	}
	lru.SetOnEvict(l.evictCallback)

	return &l, nil
}

func (l *LRU[K, V]) costRemove(key K, value V) (cost uint32, removed bool, canAdd bool) {
	if l.LRU.Contains(key) {
		oldVal, ok := l.LRU.Peek(key)
		if !ok {
			panic("should be in lru")
		}
		cost := l.costFunc(key, oldVal)
		l.currentCost -= uint(cost)
	}
	cost = l.costFunc(key, value)
	if uint(cost) > l.maxCost {
		return cost, removed, false
	}
	for uint(cost)+l.currentCost > l.maxCost {
		_, _, removed = l.LRU.RemoveOldest()
		if !removed {
			panic("huh?")
		}
	}
	return cost, removed, true
}

func (l *LRU[K, V]) Add(key K, value V) (added bool, evicted bool) {
	cost, removed, canAdd := l.costRemove(key, value)
	l.currentCost += uint(cost)
	evicted = l.LRU.Add(key, value)
	return canAdd, evicted || removed
}

func (l *LRU[K, V]) AddWithLifetime(key K, value V, lifetime time.Duration) (added bool, evicted bool) {
	cost, removed, canAdd := l.costRemove(key, value)
	l.currentCost += uint(cost)
	evicted = l.LRU.AddWithLifetime(key, value, lifetime)
	return canAdd, evicted || removed
}

func (l *LRU[K, V]) evictCallback(key K, value V) {
	cost := l.costFunc(key, value)
	if uint(cost) <= l.currentCost {
		l.currentCost -= uint(cost)
	} else {
		l.currentCost = 0
	}
	if l.onEvictCallback != nil {
		l.onEvictCallback(key, value)
	}
}

func (l *LRU[K, V]) SetOnEvict(onEvict freelru.OnEvictCallback[K, V]) {
	l.onEvictCallback = onEvict
}

func (l *LRU[K, V]) Cost() uint {
	return l.currentCost
}

func (l *LRU[K, V]) MaxCost() uint {
	return l.maxCost
}
