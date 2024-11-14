// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package cache

import (
	"log"
	"sync"

	costlru "github.com/ChainSafe/gossamer/internal/cost-lru"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime"
	"github.com/ChainSafe/gossamer/pkg/trie/triedb"
	"github.com/dolthub/maphash"
	"github.com/elastic/go-freelru"
)

// The shared node cache.
//
// Internally this stores all cached nodes in a [costlru.LRU]. It ensures that when updating the
// cache, that the cache stays within its allowed bounds.
type sharedNodeCache[H runtime.Hash] struct {
	// The cached nodes, ordered by least recently used.
	lru          *costlru.LRU[H, triedb.CachedNode[H]]
	itemsEvicted uint
}

type hasher[K comparable] struct {
	maphash.Hasher[K]
}

func (h hasher[K]) Hash(key K) uint32 {
	return uint32(h.Hasher.Hash(key)) //nolint:gosec
}

// Constructor for [sharedNodeCache] with fixed size in number of bytes.
func newSharedNodeCache[H runtime.Hash](sizeBytes uint) *sharedNodeCache[H] {
	snc := sharedNodeCache[H]{}
	itemsEvictedPtr := &snc.itemsEvicted
	h := hasher[H]{maphash.NewHasher[H]()}
	var err error

	snc.lru, err = costlru.New(sizeBytes, h.Hash, func(hash H, node triedb.CachedNode[H]) uint32 {
		return uint32(node.ByteSize()) //nolint:gosec
	})
	if err != nil {
		panic(err)
	}
	snc.lru.SetOnEvict(func(h H, cn triedb.CachedNode[H]) {
		(*itemsEvictedPtr)++
	})
	return &snc
}

// updateItem contains a hash and a [nodeCached] instance.
type updateItem[H runtime.Hash] struct {
	Hash H
	nodeCached[H]
}

// Update the cache with the list of nodes which were either newly added or accessed.
func (snc *sharedNodeCache[H]) Update(list []updateItem[H]) {
	accessCount := uint(0)
	addCount := uint(0)

	snc.itemsEvicted = 0
	maxItemsEvicted := uint(snc.lru.Len()*100) / sharedNodeCacheMaxReplacePercent //nolint:gosec
	for _, ui := range list {
		if ui.nodeCached.FromSharedCache {
			_, ok := snc.lru.Get(ui.Hash)
			if ok {
				accessCount++
				if accessCount >= sharedNodeCacheMaxPromotedKeys {
					// Stop when we've promoted a large enough number of items.
					break
				}
				continue
			}
		}

		added, _ := snc.lru.Add(ui.Hash, ui.nodeCached.Node)
		if added {
			addCount++
		}

		if snc.itemsEvicted > maxItemsEvicted {
			// Stop when we've evicted a big enough chunk of the shared cache.
			break
		}
	}

	log.Printf(
		"DEBUG: Updated the shared node cache: %d accesses, %d new values, %d/%d evicted (length = %d, size=%d/%d)\n",
		accessCount, addCount, snc.itemsEvicted, maxItemsEvicted, snc.lru.Len(), snc.lru.Cost(), snc.lru.MaxCost(),
	)
}

// Reset clears the cache
func (snc *sharedNodeCache[H]) Reset() {
	snc.lru.Purge()
}

// The comparable type that identifies this instance of storage root and storage key, used in [sharedValueCache] LRU.
type ValueCacheKeyComparable[H runtime.Hash] struct {
	StorageRoot H
	StorageKey  string
}

func (vckh ValueCacheKeyComparable[H]) ValueCacheKey() ValueCacheKey[H] {
	return ValueCacheKey[H]{
		StorageRoot: vckh.StorageRoot,
		StorageKey:  []byte(vckh.StorageKey),
	}
}

// The key type that is being used to address a [CachedValue].
type ValueCacheKey[H runtime.Hash] struct {
	// The storage root of the trie this key belongs to.
	StorageRoot H
	// The key to access the value in the storage.
	StorageKey []byte
}

func (vck ValueCacheKey[H]) ValueCacheKeyComparable() ValueCacheKeyComparable[H] {
	return ValueCacheKeyComparable[H]{
		StorageRoot: vck.StorageRoot,
		StorageKey:  string(vck.StorageKey),
	}
}

// The shared value cache.
//
// The cache ensures that it stays in the configured size bounds.
type sharedValueCache[H runtime.Hash] struct {
	lru          *costlru.LRU[ValueCacheKeyComparable[H], triedb.CachedValue[H]]
	itemsEvicted uint
}

// Constructor for [sharedValueCache].
func newSharedValueCache[H runtime.Hash](size uint) *sharedValueCache[H] {
	var svc sharedValueCache[H]
	itemsEvictedPtr := &svc.itemsEvicted
	var err error
	h := hasher[ValueCacheKeyComparable[H]]{maphash.NewHasher[ValueCacheKeyComparable[H]]()}

	svc.lru, err = costlru.New(size, h.Hash, func(key ValueCacheKeyComparable[H], value triedb.CachedValue[H]) uint32 {
		keyCost := uint32(len(key.StorageKey)) //nolint:gosec
		switch value := value.(type) {
		case triedb.NonExistingCachedValue[H]:
			return keyCost + 1
		case triedb.ExistingHashCachedValue[H]:
			return keyCost + uint32(value.Hash.Length()) //nolint:gosec
		case triedb.ExistingCachedValue[H]:
			return keyCost + uint32(value.Hash.Length()+len(value.Data)) //nolint:gosec
		default:
			panic("unreachable")
		}
	})
	if err != nil {
		panic(err)
	}
	svc.lru.SetOnEvict(func(key ValueCacheKeyComparable[H], value triedb.CachedValue[H]) {
		(*itemsEvictedPtr)++
	})
	return &svc
}

// sharedValueCacheAdded contains a [ValueCacheKey] and [triedb.CachedValue]. Used in [sharedValueCache.Update].
type sharedValueCacheAdded[H runtime.Hash] struct {
	ValueCacheKey[H]
	triedb.CachedValue[H]
}

// Update the cache with the added values and the accessed values.
//
// The added values are the ones that have been collected by doing operations on the trie and
// now should be stored in the shared cache. The accessed values are only referenced by the
// [ValueCacheKeyComparable] and represent the values that were retrieved from this shared cache.
// These accessed values are being put to the front of the internal LRU like the added ones.
func (svc *sharedValueCache[H]) Update(added []sharedValueCacheAdded[H], accessed []ValueCacheKeyComparable[H]) {
	accessCount := uint(0)
	addCount := uint(0)

	for _, hash := range accessed {
		// Access every node in the map to put it to the front.
		//
		// Since we are only comparing the hashes here it may lead us to promoting the wrong
		// values as the most recently accessed ones. However this is harmless as the only
		// consequence is that we may accidentally prune a recently used value too early.
		_, ok := svc.lru.Get(hash)
		if ok {
			accessCount++
		}
	}

	// Insert all of the new items which were *not* found in the shared cache.
	//
	// Limit how many items we'll replace in the shared cache in one go so that
	// we don't evict the whole shared cache nor we keep spinning our wheels
	// evicting items which we've added ourselves in previous iterations of this loop.
	svc.itemsEvicted = 0
	maxItemsEvicted := uint(svc.lru.Len()) * 100 / sharedValueCacheMaxReplacePercent //nolint:gosec

	for _, svca := range added {
		added, _ := svc.lru.Add(svca.ValueCacheKey.ValueCacheKeyComparable(), svca.CachedValue)
		if added {
			addCount++
		}

		if svc.itemsEvicted > maxItemsEvicted {
			// Stop when we've evicted a big enough chunk of the shared cache.
			break
		}
	}

	log.Printf(
		"DEBUG: Updated the shared value cache: %d accesses, %d new values, %d/%d evicted (length = %d, size=%d/%d)\n",
		accessCount, addCount, svc.itemsEvicted, maxItemsEvicted, svc.lru.Len(), svc.lru.Cost(), svc.lru.MaxCost(),
	)
}

func (snc *sharedValueCache[H]) Reset() {
	snc.lru.Purge()
}

// The inner of [SharedTrieCache].
type sharedTrieCacheInner[H runtime.Hash] struct {
	nodeCache  *sharedNodeCache[H]
	valueCache *sharedValueCache[H]
}

// The shared trie cache.
//
// It should be instantiated once per node. It will hold the trie nodes and values of all
// operations to the state. To not use all available memory it will ensure to stay in the
// bounds given via the size in [NewSharedTrieCache].
//
// The instance of this object can be shared.
type SharedTrieCache[H runtime.Hash] struct {
	inner sharedTrieCacheInner[H]
	mtx   sync.RWMutex
}

// Create a new [SharedTrieCache].
func NewSharedTrieCache[H runtime.Hash](size uint) *SharedTrieCache[H] {
	totalBudget := size

	// Split our memory budget between the two types of caches.
	valueCacheBudget := uint(float32(totalBudget) * 0.20) // 20% for the value cache
	nodeCacheBudget := totalBudget - valueCacheBudget     // 80% for the node cache

	return &SharedTrieCache[H]{
		inner: sharedTrieCacheInner[H]{
			nodeCache:  newSharedNodeCache[H](nodeCacheBudget),
			valueCache: newSharedValueCache[H](valueCacheBudget),
		},
	}
}

// Create a new [LocalTrieCache] instance from this shared cache.
func (stc *SharedTrieCache[H]) LocalTrieCache() LocalTrieCache[H] {
	h := hasher[H]{maphash.NewHasher[H]()}
	nodeCache, err := costlru.New(localNodeCacheMaxSize, h.Hash, func(hash H, node nodeCached[H]) uint32 {
		return uint32(node.ByteSize()) //nolint:gosec
	})
	if err != nil {
		panic(err)
	}

	valueCache, err := costlru.New(
		localValueCacheMaxSize,
		hasher[ValueCacheKeyComparable[H]]{maphash.NewHasher[ValueCacheKeyComparable[H]]()}.Hash,
		func(key ValueCacheKeyComparable[H], value triedb.CachedValue[H]) uint32 {
			keyCost := uint32(len(key.StorageKey)) //nolint:gosec
			switch value := value.(type) {
			case triedb.NonExistingCachedValue[H]:
				return keyCost + 1
			case triedb.ExistingHashCachedValue[H]:
				return keyCost + uint32(value.Hash.Length()) //nolint:gosec
			case triedb.ExistingCachedValue[H]:
				return keyCost + uint32(value.Hash.Length()+len(value.Data)) //nolint:gosec
			default:
				panic("unreachable")
			}
		})
	if err != nil {
		panic(err)
	}

	sharedValueCacheAccess, err := freelru.New[ValueCacheKeyComparable[H], any](
		uint32(sharedValueCacheMaxPromotedKeys),
		hasher[ValueCacheKeyComparable[H]]{maphash.NewHasher[ValueCacheKeyComparable[H]]()}.Hash,
	)
	if err != nil {
		panic(err)
	}

	return LocalTrieCache[H]{
		shared:                 stc,
		nodeCache:              nodeCache,
		valueCache:             valueCache,
		sharedValueCacheAccess: sharedValueCacheAccess,
	}
}

func (stc *SharedTrieCache[H]) Lock() {
	stc.mtx.Lock()
}

func (stc *SharedTrieCache[H]) Unlock() {
	stc.mtx.Unlock()
}

// Get a copy of the node for key.
//
// This will temporarily lock the shared cache for reading.
//
// This doesn't change the least recently order in the internal LRU.
func (stc *SharedTrieCache[H]) PeekNode(key H) triedb.CachedNode[H] {
	stc.mtx.RLock()
	defer stc.mtx.RUnlock()
	node, ok := stc.inner.nodeCache.lru.Peek(key)
	if ok {
		return node
	}
	return nil
}

// Get a copy of the [triedb.CachedValue] for key.
//
// This will temporarily lock the shared cache for reading.
//
// This doesn't reorder any of the elements in the internal LRU.
func (stc *SharedTrieCache[H]) PeekValueByHash(
	hash ValueCacheKeyComparable[H], storageRoot H, storageKey []byte,
) triedb.CachedValue[H] {
	stc.mtx.RLock()
	defer stc.mtx.RUnlock()
	val, ok := stc.inner.valueCache.lru.Peek(hash)
	if ok {
		return val
	}
	return nil
}

// Reset the node cache.
func (stc *SharedTrieCache[H]) ResetNodeCache() {
	stc.mtx.Lock()
	defer stc.mtx.Unlock()
	stc.inner.nodeCache.Reset()
}

// Reset the value cache.
func (stc *SharedTrieCache[H]) ResetValueCache() {
	stc.mtx.Lock()
	defer stc.mtx.Unlock()
	stc.inner.valueCache.Reset()
}

func (stc *SharedTrieCache[H]) usedMemorySize() uint {
	stc.mtx.RLock()
	defer stc.mtx.RUnlock()
	return stc.inner.nodeCache.lru.Cost() + stc.inner.valueCache.lru.Cost()
}
