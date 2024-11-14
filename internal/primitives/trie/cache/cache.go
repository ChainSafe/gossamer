// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package cache

import (
	"log"
	"sync"

	costlru "github.com/ChainSafe/gossamer/internal/cost-lru"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime"
	"github.com/ChainSafe/gossamer/pkg/trie/triedb"
	"github.com/elastic/go-freelru"
)

// The maximum number of existing keys in the shared cache that a single local cache
// can promote to the front of the LRU cache in one go.
//
// If we have a big shared cache and the local cache hits all of those keys we don't
// want to spend forever bumping all of them.
const sharedNodeCacheMaxPromotedKeys = uint(1792)

// Same as sharedNodeCacheMaxPromotedKeys for value cache.
const sharedValueCacheMaxPromotedKeys = uint(1792)

// The maximum portion of the shared cache (in percent) that a single local
// cache can replace in one go.
//
// We don't want a single local cache instance to have the ability to replace
// everything in the shared cache.
const sharedNodeCacheMaxReplacePercent = uint(33)

// Same as sharedNodeCacheMaxReplacePercent for value cache.
const sharedValueCacheMaxReplacePercent = uint(33)

// The maximum size of the memory allocated on the heap by the local cache, in bytes.
//
// The size of the node cache should always be bigger than the value cache. The value
// cache is only holding weak references to the actual values found in the nodes and
// we account for the size of the node as part of the node cache.
const localNodeCacheMaxSize = uint(8 * 1024 * 1024)

// Same as localNodeCacheMaxSize for value cache.
const localValueCacheMaxSize = uint(2 * 1024 * 1024)

// An internal struct to store the cached trie nodes.
type nodeCached[H runtime.Hash] struct {
	// The cached node.
	Node triedb.CachedNode[H]
	// Whether this node was fetched from the shared cache or not.
	FromSharedCache bool
}

// Returns the number of bytes allocated on the heap by this node.
func (nc nodeCached[H]) ByteSize() uint {
	return nc.Node.ByteSize()
}

// The local trie cache.
//
// This cache should be used per state instance created by the backend. One state instance is
// referring to the state of one block. It will cache all the accesses that are done to the state
// which could not be fulfilled by the [SharedTrieCache]. These locally cached items are merged
// back to the shared trie cache when this instance is dropped.
//
// When using [LocalTrieCache.TrieCache] or [LocalTrieCache.TrieCacheMut] it will lock mutexes.
// So, it is important that these methods are not called multiple times, because they otherwise
// deadlock.
type LocalTrieCache[H runtime.Hash] struct {
	// The shared trie cache that created this instance.
	shared *SharedTrieCache[H]

	// The local cache for the trie nodes.
	nodeCache    *costlru.LRU[H, nodeCached[H]]
	nodeCacheMtx sync.Mutex

	// The local cache for the values.
	valueCache    *costlru.LRU[ValueCacheKeyComparable[H], triedb.CachedValue[H]]
	valueCacheMtx sync.Mutex

	// Keeps track of all values accessed in the shared cache.
	//
	// This will be used to ensure that these nodes are brought to the front of the LRU when this
	// local instance is merged back to the shared cache. This can actually lead to collision when
	// two [ValueCacheKey] instances with different storage roots and keys map to the same hash. However,
	// as we only use this set to update the LRU position it is fine, even if we bring the wrong
	// value to the top. The important part is that we always get the correct value from the value
	// cache for a given key.
	sharedValueCacheAccess    *freelru.LRU[ValueCacheKeyComparable[H], any]
	sharedValueCacheAccessMtx sync.Mutex
}

func (ltc *LocalTrieCache[H]) Commit() {
	ltc.shared.Lock()
	defer ltc.shared.Unlock()

	sharedInner := &ltc.shared.inner

	updateItems := make([]updateItem[H], 0)
	ltc.nodeCacheMtx.Lock()
	for _, hash := range ltc.nodeCache.Keys() {
		node, ok := ltc.nodeCache.Peek(hash)
		if !ok {
			panic("node should be found")
		}
		updateItems = append(updateItems, updateItem[H]{
			Hash:       hash,
			nodeCached: node,
		})
	}
	ltc.nodeCache.Purge()
	ltc.nodeCacheMtx.Unlock()
	sharedInner.nodeCache.Update(updateItems)

	added := make([]sharedValueCacheAdded[H], 0)
	ltc.valueCacheMtx.Lock()
	for _, key := range ltc.valueCache.Keys() {
		value, ok := ltc.valueCache.Get(key)
		if !ok {
			panic("value should be found")
		}
		added = append(added, sharedValueCacheAdded[H]{
			ValueCacheKey: key.ValueCacheKey(),
			CachedValue:   value,
		})
	}
	ltc.valueCache.Purge()
	ltc.valueCacheMtx.Unlock()

	ltc.sharedValueCacheAccessMtx.Lock()
	accessed := ltc.sharedValueCacheAccess.Keys()
	ltc.sharedValueCacheAccess.Purge()
	ltc.sharedValueCacheAccessMtx.Unlock()

	sharedInner.valueCache.Update(added, accessed)
}

// Returns a [triedb.TrieDB] compatible [triedb.TrieCache].
//
// The given storageRoot needs to be the storage root of the trie this cache is used for.
func (ltc *LocalTrieCache[H]) TrieCache(storageRoot H) (cache *TrieCache[H], unlock func()) {
	ltc.valueCacheMtx.Lock()
	ltc.sharedValueCacheAccessMtx.Lock()
	ltc.nodeCacheMtx.Lock()

	unlock = func() {
		ltc.valueCacheMtx.Unlock()
		ltc.sharedValueCacheAccessMtx.Unlock()
		ltc.nodeCacheMtx.Unlock()
	}

	return &TrieCache[H]{
		sharedCache: ltc.shared,
		localCache:  ltc.nodeCache,
		valueCache: &forStorageRootValueCache[H]{
			storageRoot:            storageRoot,
			localValueCache:        ltc.valueCache,
			sharedValueCacheAccess: ltc.sharedValueCacheAccess,
		},
	}, unlock
}

// Returns a [triedb.TrieDB] compatible [triedb.TrieCache].
//
// After finishing all operations with [triedb.TrieDB] and having obtained
// the new storage root, [TrieCache.MergeInto] should be called to update this local
// cache instance. If the function is not called, cached data is just thrown away and not
// propagated to the shared cache.
func (ltc *LocalTrieCache[H]) TrieCacheMut() (cache *TrieCache[H], unlock func()) {
	ltc.nodeCacheMtx.Lock()
	return &TrieCache[H]{
		sharedCache: ltc.shared,
		localCache:  ltc.nodeCache,
		valueCache:  &freshValueCache[H]{},
	}, ltc.nodeCacheMtx.Unlock
}

// Merge the cached data in other into self.
//
// This must be used for the cache returned by [LocalTrieCache.TrieCacheMut] as otherwise the
// cached data is just thrown away.
func (ltc *LocalTrieCache[H]) Merge(other *TrieCache[H], newRoot H) {
	other.MergeInto(ltc, newRoot)
}

// The abstraction of the value cache for the [TrieCache].
type valueCache[H runtime.Hash] interface {
	get(key []byte, sharedCache *SharedTrieCache[H]) triedb.CachedValue[H]
	insert(key []byte, value triedb.CachedValue[H])
}

// The value cache is fresh, aka not yet associated to any storage root.
// This is used for example when a new trie is being build, to cache new values.
type freshValueCache[H runtime.Hash] map[string]triedb.CachedValue[H]

func (fvc *freshValueCache[H]) get(key []byte, sharedCache *SharedTrieCache[H]) triedb.CachedValue[H] { //nolint:unused
	val, ok := (*fvc)[string(key)]
	if ok {
		return val
	} else {
		return nil
	}
}
func (fvc *freshValueCache[H]) insert(key []byte, value triedb.CachedValue[H]) { //nolint:unused
	(*fvc)[string(key)] = value
}

// The value cache is already bound to a specific storage root.
type forStorageRootValueCache[H runtime.Hash] struct {
	sharedValueCacheAccess *freelru.LRU[ValueCacheKeyComparable[H], any]
	localValueCache        *costlru.LRU[ValueCacheKeyComparable[H], triedb.CachedValue[H]]
	storageRoot            H
}

func (fsrvc forStorageRootValueCache[H]) get( //nolint:unused
	key []byte, sharedCache *SharedTrieCache[H]) triedb.CachedValue[H] {
	// We first need to look up in the local cache and then the shared cache.
	// It can happen that some value is cached in the shared cache, but the
	// weak reference of the data can not be upgraded anymore. This for example
	// happens when the node is dropped that contains the strong reference to the data.
	//
	// So, the logic of the trie would lookup the data and the node and store both
	// in our local caches.
	vck := ValueCacheKey[H]{
		StorageKey:  key,
		StorageRoot: fsrvc.storageRoot,
	}
	val, ok := fsrvc.localValueCache.Peek(vck.ValueCacheKeyComparable())
	if ok {
		return val
	}

	sharedVal := sharedCache.PeekValueByHash(vck.ValueCacheKeyComparable(), fsrvc.storageRoot, key)
	if sharedVal != nil {
		fsrvc.sharedValueCacheAccess.Add(vck.ValueCacheKeyComparable(), any(nil))
		return sharedVal
	}

	return nil
}
func (fsrvc forStorageRootValueCache[H]) insert(key []byte, value triedb.CachedValue[H]) { //nolint:unused
	vck := ValueCacheKey[H]{
		StorageKey:  key,
		StorageRoot: fsrvc.storageRoot,
	}
	fsrvc.localValueCache.Add(vck.ValueCacheKeyComparable(), value)
}

// The [triedb.TrieCache] implementation.
//
// If this instance was created using [LocalTrieCache.TrieCacheMut], it needs to
// be merged back into the [LocalTrieCache] with [LocalTrieCache.MergeInto] after all operations are
// done.
type TrieCache[H runtime.Hash] struct {
	sharedCache *SharedTrieCache[H]
	localCache  *costlru.LRU[H, nodeCached[H]]
	valueCache  valueCache[H]
}

// Merge this cache into the given [LocalTrieCache].
//
// This function is only required to be called when this instance was created through
// [LocalTrieCache.TrieCacheMut], otherwise this method is a no-op. The given
// storageRoot is the new storage root that was obtained after finishing all operations
// using the [triedb.TrieDB].
func (tc *TrieCache[H]) MergeInto(local *LocalTrieCache[H], storageRoot H) {
	cache, ok := tc.valueCache.(*freshValueCache[H])
	if !ok {
		return
	}

	if len(*cache) != 0 {
		local.valueCacheMtx.Lock()
		defer local.valueCacheMtx.Unlock()

		vck := ValueCacheKey[H]{
			StorageRoot: storageRoot,
		}
		for k, v := range *cache {
			vck.StorageKey = []byte(k)
			ok, _ := local.valueCache.Add(vck.ValueCacheKeyComparable(), v)
			if !ok {
				panic("huh?")
			}
		}
	}
}

func (tc *TrieCache[H]) GetOrInsertNode(
	hash H, fetchNode func() (triedb.CachedNode[H], error),
) (triedb.CachedNode[H], error) {
	var isLocalCacheHit bool = true

	// First try to grab the node from the local cache.
	var err error
	node, ok := tc.localCache.Get(hash)
	if !ok {
		isLocalCacheHit = false

		// It was not in the local cache; try the shared cache.
		shared := tc.sharedCache.PeekNode(hash)
		if shared != nil {
			log.Printf("TRACE: Serving node from shared cache: %s\n", hash)
			node = nodeCached[H]{Node: shared, FromSharedCache: true}
		} else {
			// It was not in the shared cache; try fetching it from the database.
			var fetched triedb.CachedNode[H]
			fetched, err = fetchNode()
			if err != nil {
				log.Printf("TRACE: Serving node from database failed: %s\n", hash)
				return nil, err
			} else {
				log.Printf("TRACE: Serving node from database: %s\n", hash)
				node = nodeCached[H]{Node: fetched, FromSharedCache: false}
			}
		}
		tc.localCache.Add(hash, node)
	}

	if isLocalCacheHit {
		log.Printf("TRACE: Serving node from local cache: %s\n", hash)
	}

	return node.Node, nil
}

func (tc *TrieCache[H]) GetNode(hash H) triedb.CachedNode[H] {
	var isLocalCacheHit bool = true

	// First try to grab the node from the local cache.
	var node *nodeCached[H]
	nc, ok := tc.localCache.Get(hash)
	if !ok {
		isLocalCacheHit = false

		// It was not in the local cache; try the shared cache.
		peeked := tc.sharedCache.PeekNode(hash)
		if peeked != nil {
			log.Printf("TRACE: Serving node from shared cache: %s\n", hash)
			node = &nodeCached[H]{Node: peeked, FromSharedCache: true}
		} else {
			log.Printf("TRACE: Serving node from cahe failed: %s\n", hash)
			return nil
		}
	} else {
		node = &nc
	}

	if isLocalCacheHit {
		log.Printf("TRACE: Serving node from local cache: %s\n", hash)
	}

	return node.Node
}

func (tc *TrieCache[H]) GetValue(key []byte) triedb.CachedValue[H] {
	cached := tc.valueCache.get(key, tc.sharedCache)
	log.Printf("TRACE: Looked up value for key: %x\n", key)
	return cached
}

func (tc *TrieCache[H]) SetValue(key []byte, value triedb.CachedValue[H]) {
	log.Printf("TRACE: Caching value for key: %x\n", key)
	tc.valueCache.insert(key, value)
}
