// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package statemachine

import (
	"bytes"
	"errors"
	"log"
	"slices"
	"sync"

	hashdb "github.com/ChainSafe/gossamer/internal/hash-db"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime"
	"github.com/ChainSafe/gossamer/internal/primitives/storage"
	"github.com/ChainSafe/gossamer/internal/primitives/trie"
	"github.com/ChainSafe/gossamer/internal/primitives/trie/cache"
	"github.com/ChainSafe/gossamer/internal/primitives/trie/recorder"
	ptrie "github.com/ChainSafe/gossamer/pkg/trie"
	triedb "github.com/ChainSafe/gossamer/pkg/trie/triedb"
	"golang.org/x/exp/constraints"
)

type IterState uint

const (
	Pending IterState = iota
	FinishedComplete
	FinishedIncomplete
)

// A raw iterator over the storage.
type rawIter[H runtime.Hash, Hasher runtime.Hasher[H]] struct {
	stopOnImcompleteDatabase bool
	skipIfFirst              StorageKey
	root                     H
	childInfo                storage.ChildInfo
	trieIter                 triedb.TrieDBRawIterator[H, Hasher]
	state                    IterState
}

func prepare[H runtime.Hash, Hasher runtime.Hasher[H], R any](
	ri *rawIter[H, Hasher],
	backend *trieBackendEssence[H, Hasher],
	callback func(*triedb.TrieDB[H, Hasher], *triedb.TrieDBRawIterator[H, Hasher]) (*R, error),
) (*R, error) {
	if ri.state != Pending {
		return nil, nil
	}

	var result *R
	var err error
	withTrieDB[H, Hasher](backend, ri.root, ri.childInfo, func(db *triedb.TrieDB[H, Hasher]) {
		result, err = callback(db, &ri.trieIter)
	})
	if err != nil {
		ri.state = FinishedIncomplete
		if errors.Is(err, triedb.ErrIncompleteDB) && ri.stopOnImcompleteDatabase {
			return nil, nil
		}
		return nil, err
	}
	if result != nil {
		return result, nil
	}
	ri.state = FinishedComplete
	return nil, nil
}

func (ri *rawIter[H, Hasher]) NextKey(backend *TrieBackend[H, Hasher]) (StorageKey, error) {
	skipIfFirst := ri.skipIfFirst
	ri.skipIfFirst = nil

	key, err := prepare[H, Hasher, []byte](
		ri,
		&backend.essence,
		func(trie *triedb.TrieDB[H, Hasher], trieIter *triedb.TrieDBRawIterator[H, Hasher]) (*[]byte, error) {
			result, err := trieIter.NextKey()
			if err != nil {
				return nil, err
			}
			if skipIfFirst != nil {
				if result != nil {
					if slices.Equal(result, skipIfFirst) {
						result, err = trieIter.NextKey()
						if err != nil {
							return nil, err
						}
					}
				}
			}
			return &result, nil
		})

	if key == nil {
		return nil, err
	}
	storageKey := StorageKey(*key)
	return storageKey, nil
}

func (ri *rawIter[H, Hasher]) NextKeyValue(backend *TrieBackend[H, Hasher]) (*StorageKeyValue, error) {
	skipIfFirst := ri.skipIfFirst
	ri.skipIfFirst = nil

	pair, err := prepare[H, Hasher, StorageKeyValue](
		ri,
		&backend.essence,
		func(trie *triedb.TrieDB[H, Hasher], trieIter *triedb.TrieDBRawIterator[H, Hasher]) (*StorageKeyValue, error) {
			result, err := trieIter.NextItem()
			if err != nil {
				return nil, err
			}
			if skipIfFirst != nil {
				if result != nil {
					if bytes.Equal(result.Key, skipIfFirst) {
						result, err = trieIter.NextItem()
						if err != nil {
							return nil, err
						}
					}
				}
			}
			if result != nil {
				return &StorageKeyValue{result.Key, result.Value}, nil
			}
			return nil, nil
		})
	return pair, err
}

func (ri *rawIter[H, Hasher]) Complete() bool {
	return ri.state == FinishedComplete
}

// Patricia trie-based pairs storage essence.
type trieBackendEssence[H runtime.Hash, Hasher runtime.Hasher[H]] struct {
	storage TrieBackendStorage[H]
	root    H
	empty   H
	cache   struct {
		childRoot map[string]*H
		sync.RWMutex
	}
	trieNodeCache TrieCacheProvider[H, *cache.TrieCache[H]]
	recorder      *recorder.Recorder[H]
}

func newTrieBackendEssence[H runtime.Hash, Hasher runtime.Hasher[H]](
	storage TrieBackendStorage[H],
	root H,
	cache TrieCacheProvider[H, *cache.TrieCache[H]],
	recorder *recorder.Recorder[H],
) trieBackendEssence[H, Hasher] {
	return trieBackendEssence[H, Hasher]{
		storage: storage,
		root:    root,
		cache: struct {
			childRoot map[string]*H
			sync.RWMutex
		}{childRoot: make(map[string]*H)},
		trieNodeCache: cache,
		recorder:      recorder,
	}
}

// Get backend storage reference.
func (tbe *trieBackendEssence[H, Hasher]) BackendStorage() TrieBackendStorage[H] {
	return tbe.storage
}

// Get trie root.
func (tbe *trieBackendEssence[H, Hasher]) Root() H {
	return tbe.root
}

// Set trie root. This is useful for testing.
func (tbe *trieBackendEssence[H, Hasher]) SetRoot(root H) {
	tbe.resetCache()
	tbe.root = root
}

func (tbe *trieBackendEssence[H, Hasher]) resetCache() {
	tbe.cache = struct {
		childRoot map[string]*H
		sync.RWMutex
	}{childRoot: make(map[string]*H)}
}

func withRecorderAndCache[H runtime.Hash, Hasher runtime.Hasher[H]](
	tbe *trieBackendEssence[H, Hasher],
	storageRoot *H,
	callback func(triedb.TrieRecorder, triedb.TrieCache[H]),
) {
	root := tbe.root
	if storageRoot != nil {
		root = *storageRoot
	}

	var cache triedb.TrieCache[H]
	var unlock func()
	if tbe.trieNodeCache != nil {
		cache, unlock = tbe.trieNodeCache.TrieCache(root)
	}

	var recorder triedb.TrieRecorder
	if tbe.recorder != nil {
		recorder = tbe.recorder.TrieRecorder(root)
	}
	callback(recorder, cache)
	if unlock != nil {
		unlock()
	}
}

// Call the given closure passing it the recorder and the cache.
//
// This function must only be used when the operation in callback is
// calculating a storageRoot. It is expected that callback returns
// the new storage root. This is required to register the changes in the cache
// for the correct storage root. The given storageRoot corresponds to the root of the "old"
// trie. If the value is not given, tbe.root is used.
func withRecorderAndCacheForStorageRoot[H runtime.Hash, Hasher runtime.Hasher[H], R any](
	tbe *trieBackendEssence[H, Hasher],
	storageRoot *H,
	callback func(triedb.TrieRecorder, triedb.TrieCache[H]) (*H, R),
) R {
	root := tbe.root
	if storageRoot != nil {
		root = *storageRoot
	}

	var recorder triedb.TrieRecorder
	if tbe.recorder != nil {
		recorder = tbe.recorder.TrieRecorder(root)
	}

	if tbe.trieNodeCache != nil {
		cache, unlock := tbe.trieNodeCache.TrieCacheMut()
		newRoot, r := callback(recorder, cache)
		if newRoot != nil {
			tbe.trieNodeCache.Merge(cache, *newRoot)
		}
		unlock()
		return r
	} else {
		_, r := callback(recorder, nil)
		return r
	}
}

// Calls the given closure with a [triedb.TrieDB] constructed for the given
// storage root and (optionally) child trie.
func withTrieDB[H runtime.Hash, Hasher runtime.Hasher[H]](
	tbe *trieBackendEssence[H, Hasher],
	root H,
	childInfo storage.ChildInfo,
	callback func(*triedb.TrieDB[H, Hasher]),
) {
	var backend hashdb.HashDB[H] = tbe
	var db hashdb.HashDB[H] = tbe
	if childInfo != nil {
		db = trie.NewKeyspacedDB[H](backend, childInfo.Keyspace())
	}

	withRecorderAndCache(tbe, &root, func(recorder triedb.TrieRecorder, cache triedb.TrieCache[H]) {
		trieDB := triedb.NewTrieDB(
			root, db, triedb.WithCache[H, Hasher](cache), triedb.WithRecorder[H, Hasher](recorder))
		trieDB.SetVersion(ptrie.V1)
		callback(trieDB)
	})
}

// Access the root of the child storage in its parent trie
func (tbe *trieBackendEssence[H, Hasher]) childRoot(childInfo storage.ChildInfo) (*H, error) {
	tbe.cache.RLock()
	{
		result, ok := tbe.cache.childRoot[string(childInfo.StorageKey())]
		tbe.cache.RUnlock()
		if ok {
			return result, nil
		}
	}

	r, err := tbe.Storage(childInfo.PrefixedStorageKey())
	if err != nil {
		return nil, err
	}
	var result *H
	if r != nil {
		h := (*(new(Hasher))).NewHash(r)
		result = &h
	}
	tbe.cache.Lock()
	defer tbe.cache.Unlock()
	tbe.cache.childRoot[string(childInfo.StorageKey())] = result

	return result, nil
}

// Return the next key in the child trie i.e. the minimum key that is strictly superior to
// key in lexicographic order.
func (tbe *trieBackendEssence[H, Hasher]) NextChildStorageKey(
	childInfo storage.ChildInfo, key []byte,
) (StorageKey, error) {
	childRoot, err := tbe.childRoot(childInfo)
	if err != nil {
		return nil, err
	}
	if childRoot == nil {
		return nil, nil
	}

	return tbe.NextStorageKeyFromRoot(*childRoot, childInfo, key)
}

// Return next key from main trie or child trie by providing corresponding root.
func (tbe *trieBackendEssence[H, Hasher]) NextStorageKeyFromRoot(
	root H, childInfo storage.ChildInfo, key []byte,
) (StorageKey, error) {
	var err error
	var nextKey []byte
	withTrieDB(tbe, root, childInfo, func(trie *triedb.TrieDB[H, Hasher]) {
		var iter triedb.TrieIterator[H, []byte]
		iter, err = trie.KeyIterator()
		if err != nil {
			return
		}

		// The key just after the one given in input, basically key++0.
		// Note: We are sure this is the next key if:
		// * size of key has no limit (i.e. we can always add 0 to the path),
		// * and no keys can be inserted between key and key++0
		potentialNextKey := key
		potentialNextKey = append(potentialNextKey, 0)

		err = iter.Seek(potentialNextKey)
		if err != nil {
			return
		}

		var nextElement []byte
		nextElement, err = iter.Next()
		if err != nil {
			return
		}
		nextKey = nextElement
	})

	if err != nil {
		return nil, err
	}
	return nextKey, err
}

// Get the value of storage at given key.
func (tbe *trieBackendEssence[H, Hasher]) Storage(key []byte) (val StorageValue, err error) {
	withRecorderAndCache(tbe, nil, func(recorder triedb.TrieRecorder, cache triedb.TrieCache[H]) {
		val, err = trie.ReadTrieValue[H, Hasher](tbe, tbe.root, key, recorder, cache, triedb.V1)
	})
	return
}

// Returns the hash value
func (tbe *trieBackendEssence[H, Hasher]) StorageHash(key []byte) (hash *H, err error) {
	withRecorderAndCache[H, Hasher](tbe, nil, func(recorder triedb.TrieRecorder, cache triedb.TrieCache[H]) {
		trieDB := triedb.NewTrieDB(
			tbe.root, tbe, triedb.WithCache[H, Hasher](cache), triedb.WithRecorder[H, Hasher](recorder))
		trieDB.SetVersion(ptrie.V1)
		hash, err = trieDB.GetHash(key)
	})
	return
}

// Get the value of child storage at given key.
func (tbe *trieBackendEssence[H, Hasher]) ChildStorage(childInfo storage.ChildInfo, key []byte) (StorageValue, error) {
	var childRoot H
	root, err := tbe.childRoot(childInfo)
	if err != nil {
		return nil, err
	}
	if root == nil {
		return nil, nil
	}
	childRoot = *root

	var val StorageValue
	withRecorderAndCache(tbe, &childRoot, func(recorder triedb.TrieRecorder, cache triedb.TrieCache[H]) {
		val, err = trie.ReadChildTrieValue[H, Hasher](
			childInfo.Keyspace(),
			tbe,
			childRoot,
			key,
			recorder,
			cache,
			triedb.V1,
		)
	})
	return val, err
}

// Returns the hash value
func (tbe *trieBackendEssence[H, Hasher]) ChildStorageHash(childInfo storage.ChildInfo, key []byte) (*H, error) {
	var childRoot H
	root, err := tbe.childRoot(childInfo)
	if err != nil {
		return nil, err
	}
	if root == nil {
		return nil, nil
	}
	childRoot = *root

	var hash *H
	withRecorderAndCache(tbe, &childRoot, func(recorder triedb.TrieRecorder, cache triedb.TrieCache[H]) {
		hash, err = trie.ReadChildTrieHash[H, Hasher](
			childInfo.Keyspace(),
			tbe,
			childRoot,
			key,
			recorder,
			cache,
			triedb.V1,
		)
	})
	return hash, err
}

// Get the closest merkle value at given key.
func (tbe *trieBackendEssence[H, Hasher]) ClosestMerkleValue(key []byte) (val triedb.MerkleValue[H], err error) {
	withRecorderAndCache(tbe, nil, func(recorder triedb.TrieRecorder, cache triedb.TrieCache[H]) {
		val, err = trie.ReadTrieFirstDescendantValue[H, Hasher](tbe, tbe.root, key, recorder, cache, triedb.V1)
	})
	return
}

// Get the child closest merkle value at given key.
func (tbe *trieBackendEssence[H, Hasher]) ChildClosestMerkleValue(
	childInfo storage.ChildInfo, key []byte,
) (val triedb.MerkleValue[H], err error) {
	var childRoot H
	root, err := tbe.childRoot(childInfo)
	if err != nil {
		return nil, err
	}
	if root == nil {
		return nil, nil //nolint:nilnil
	}
	childRoot = *root

	withRecorderAndCache(tbe, &childRoot, func(recorder triedb.TrieRecorder, cache triedb.TrieCache[H]) {
		val, err = trie.ReadChildTrieFirstDescendantValue[H, Hasher](
			childInfo.Keyspace(), tbe, tbe.root, key, recorder, cache, triedb.V1)
	})
	return
}

// Create a raw iterator over the storage.
func (tbe *trieBackendEssence[H, Hasher]) RawIter(args IterArgs) (*rawIter[H, Hasher], error) {
	var root H = tbe.root
	if args.ChildInfo != nil {
		childRoot, err := tbe.childRoot(args.ChildInfo)
		if err != nil {
			return &rawIter[H, Hasher]{}, err
		}
		if childRoot != nil {
			root = *childRoot
		} else {
			return &rawIter[H, Hasher]{
				state: FinishedComplete,
			}, nil
		}
	}

	if root == (*new(H)) {
		return &rawIter[H, Hasher]{
			state: FinishedComplete,
		}, nil
	}

	var trieIter *triedb.TrieDBRawIterator[H, Hasher]
	var err error
	withTrieDB[H, Hasher](tbe, root, args.ChildInfo, func(db *triedb.TrieDB[H, Hasher]) {
		var prefix []byte
		if args.Prefix != nil {
			prefix = args.Prefix
		}

		if args.StartAt != nil {
			trieIter, err = triedb.NewPrefixedTrieDBRawIteratorThenSeek[H, Hasher](db, prefix, args.StartAt)
		} else {
			trieIter, err = triedb.NewPrefixedTrieDBRawIterator[H, Hasher](db, prefix)
		}
	})
	if err != nil {
		return &rawIter[H, Hasher]{}, err
	}

	var skipIfFirst StorageKey
	if args.StartAtExclusive {
		if args.StartAt != nil {
			storageKey := StorageKey(args.StartAt)
			skipIfFirst = storageKey
		}
	}

	return &rawIter[H, Hasher]{
		stopOnImcompleteDatabase: args.StopOnIncompleteDatabase,
		skipIfFirst:              skipIfFirst,
		childInfo:                args.ChildInfo,
		root:                     root,
		trieIter:                 *trieIter,
		state:                    Pending,
	}, nil
}

// Return the storage root after applying the given delta.
func (tbe *trieBackendEssence[H, Hasher]) StorageRoot(
	delta []Delta, stateVersion storage.StateVersion,
) (H, *trie.PrefixedMemoryDB[H, Hasher]) {
	writeOverlay := trie.NewPrefixedMemoryDB[H, Hasher]()

	root := withRecorderAndCacheForStorageRoot(
		tbe, nil, func(recorder triedb.TrieRecorder, cache triedb.TrieCache[H]) (*H, H) {
			eph := newEphemeral[H, Hasher](tbe.BackendStorage(), writeOverlay)
			root, err := trie.DeltaTrieRoot[H, Hasher](eph, tbe.root, delta, recorder, cache, stateVersion.TrieLayout())
			if err != nil {
				log.Printf("WARN: failed to write to trie: %v", err)
				return nil, tbe.root
			}
			return &root, root
		},
	)

	return root, writeOverlay
}

func (tbe *trieBackendEssence[H, Hasher]) ChildStorageRoot(
	childInfo storage.ChildInfo,
	delta []Delta,
	stateVersion storage.StateVersion,
) (H, bool, *trie.PrefixedMemoryDB[H, Hasher]) {
	defaultRoot := trie.EmptyChildTrieRoot[H, Hasher]()
	writeOverlay := trie.NewPrefixedMemoryDB[H, Hasher]()
	var childRoot H
	hash, err := tbe.childRoot(childInfo)
	if err != nil {
		log.Printf("WARN: Failed to read child storage root: %v", childInfo)
	}
	if hash == nil {
		childRoot = defaultRoot
	} else {
		childRoot = *hash
	}

	newChildRoot := withRecorderAndCacheForStorageRoot(
		tbe, &childRoot, func(recorder triedb.TrieRecorder, cache triedb.TrieCache[H]) (*H, H) {
			eph := newEphemeral(tbe.BackendStorage(), writeOverlay)
			root, err := trie.ChildDeltaTrieRoot[H, Hasher](
				childInfo.Keyspace(), eph, childRoot, delta, recorder, cache, stateVersion.TrieLayout())
			if err != nil {
				log.Printf("WARN: Failed to write to trie: %v", err)
				return nil, childRoot
			}
			return &root, root
		})

	isDefault := newChildRoot == defaultRoot
	return newChildRoot, isDefault, writeOverlay
}

func (tbe *trieBackendEssence[H, Hasher]) Get(key H, prefix hashdb.Prefix) []byte {
	if key == tbe.empty {
		val := []byte{0}
		return val
	}
	val, err := tbe.storage.Get(key, prefix)
	if err != nil {
		log.Printf("WARN: failed to write to trie: %v\n", err)
		return nil
	}
	return val
}

func (tbe *trieBackendEssence[H, Hasher]) Contains(key H, prefix hashdb.Prefix) bool {
	return tbe.Get(key, prefix) != nil
}

func (tbe *trieBackendEssence[H, Hasher]) Insert(prefix hashdb.Prefix, value []byte) H {
	panic("unimplemented")
}

func (tbe *trieBackendEssence[H, Hasher]) Emplace(key H, prefix hashdb.Prefix, value []byte) {
	panic("unimplemented")
}

func (tbe *trieBackendEssence[H, Hasher]) Remove(key H, prefix hashdb.Prefix) {
	panic("unimplemented")
}

type ephemeral[H runtime.Hash, Hasher runtime.Hasher[H]] struct {
	storage TrieBackendStorage[H]
	overlay *trie.PrefixedMemoryDB[H, Hasher]
}

func newEphemeral[H runtime.Hash, Hasher runtime.Hasher[H]](
	storage TrieBackendStorage[H],
	overlay *trie.PrefixedMemoryDB[H, Hasher],
) *ephemeral[H, Hasher] {
	return &ephemeral[H, Hasher]{
		storage, overlay,
	}
}

func (e *ephemeral[H, Hasher]) Get(key H, prefix hashdb.Prefix) []byte {
	val := e.overlay.Get(key, prefix)
	if val == nil {
		val, err := e.storage.Get(key, prefix)
		if err != nil {
			log.Printf("WARN: failed to read from DB: %v\n", err)
			return nil
		}
		return val
	}
	return val
}

func (e *ephemeral[H, Hasher]) Contains(key H, prefix hashdb.Prefix) bool {
	return e.Get(key, prefix) != nil
}

func (e *ephemeral[H, Hasher]) Insert(prefix hashdb.Prefix, value []byte) H {
	return e.overlay.Insert(prefix, value)
}

func (e *ephemeral[H, Hasher]) Emplace(key H, prefix hashdb.Prefix, value []byte) {
	e.overlay.Emplace(key, prefix, value)
}

func (e *ephemeral[H, Hasher]) Remove(key H, prefix hashdb.Prefix) {
	e.overlay.Remove(key, prefix)
}

// Key-value pairs storage that is used by trie backend essence.
type TrieBackendStorage[H constraints.Ordered] interface {
	Get(key H, prefix hashdb.Prefix) ([]byte, error)
}

type HashDBTrieBackendStorage[H runtime.Hash] struct {
	hashdb.HashDB[H]
}

func (hdbtbs HashDBTrieBackendStorage[H]) Get(key H, prefix hashdb.Prefix) ([]byte, error) {
	return hdbtbs.HashDB.Get(key, prefix), nil
}
