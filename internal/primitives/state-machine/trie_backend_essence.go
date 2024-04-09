package statemachine

import (
	"log"
	"slices"
	"sync"

	hashdb "github.com/ChainSafe/gossamer/internal/hash-db"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime"
	"github.com/ChainSafe/gossamer/internal/primitives/storage"
	"github.com/ChainSafe/gossamer/internal/primitives/trie"
	"github.com/ChainSafe/gossamer/internal/primitives/trie/recorder"
	triedb "github.com/ChainSafe/gossamer/internal/trie-db"
	"golang.org/x/exp/constraints"
)

// / Local cache for child root.
type Cache[H any] struct {
	// pub child_root: HashMap<Vec<u8>, Option<H>>,
	ChildRoot map[string]H
}

type IterState uint

const (
	Pending IterState = iota
	FinishedComplete
	FinishedIncomplete
)

// / A raw iterator over the storage.
type RawIter[H runtime.Hash, Hasher runtime.Hasher[H]] struct {
	stopOnImcompleteDatabase bool
	skipIfFirst              *StorageKey
	root                     H
	childInfo                *storage.ChildInfo
	trieIter                 triedb.TrieDBRawIterator[H]
	state                    IterState
}

func prepare[H runtime.Hash, Hasher runtime.Hasher[H], R any](
	ri *RawIter[H, Hasher],
	backend *TrieBackendEssence[H, Hasher],
	callback func(triedb.TrieDB[H], triedb.TrieDBRawIterator[H]) (*R, error),
) (*R, error) {
	if ri.state != Pending {
		return nil, nil
	}

	var result *R
	var err error
	withTrieDB[H, Hasher, R](backend, ri.root, ri.childInfo, func(db triedb.TrieDB[H]) {
		result, err = callback(db, ri.trieIter)
	})
	if err != nil {
		ri.state = FinishedIncomplete
		// if matches!(*error, TrieError::IncompleteDatabase(_)) &&
		// 			self.stop_on_incomplete_database
		// 		{
		// 			None
		// 		} else {
		// 			Some(Err(format!("TrieDB iteration error: {}", error)))
		// 		}
		return nil, err
	}
	if result != nil {
		return result, err
	}
	ri.state = FinishedComplete
	return nil, nil
}

func (ri *RawIter[H, Hasher]) NextKey(backend *TrieBackend[H, Hasher]) (*StorageKey, error) {
	skipIfFirst := ri.skipIfFirst
	ri.skipIfFirst = nil

	key, err := prepare[H, Hasher, []byte](ri, &backend.Essence, func(trie triedb.TrieDB[H], trieIter triedb.TrieDBRawIterator[H]) (*[]byte, error) {
		result, err := trieIter.NextKey(trie)
		if skipIfFirst != nil {
			if result != nil {
				if slices.Equal(*result, *skipIfFirst) {
					result, err = trieIter.NextKey(trie)
				}
			}
		}
		return result, err
	})

	if key == nil {
		return nil, err
	}
	var storageKey StorageKey
	storageKey = StorageKey(*key)
	return &storageKey, nil
}

func (ri *RawIter[H, Hasher]) NextPair(backend *TrieBackend[H, Hasher]) (*StorageKey, *StorageValue, error) {
	panic("unimpl")
}

// / Patricia trie-based pairs storage essence.
// pub struct TrieBackendEssence<S: TrieBackendStorage<H>, H: Hasher, C> {
type TrieBackendEssence[H runtime.Hash, Hasher runtime.Hasher[H]] struct {
	storage TrieBackendStorage[H]
	// storage: S,
	root H
	// root: H::Out,
	empty H
	// empty: H::Out,
	// #[cfg(feature = "std")]
	// pub(crate) cache: Arc<RwLock<Cache<H::Out>>>,
	cache    Cache[H]
	cacheMtx sync.RWMutex
	// pub(crate) trie_node_cache: Option<C>,
	trieNodeCache TrieCacheProvider[H]
	// #[cfg(feature = "std")]
	// pub(crate) recorder: Option<Recorder<H>>,
	recorder *recorder.Recorder[H]
}

func newTrieBackendEssence[H runtime.Hash, Hasher runtime.Hasher[H]](
	storage TrieBackendStorage[H],
	root H,
	cache TrieCacheProvider[H],
	recorder *recorder.Recorder[H],
) TrieBackendEssence[H, Hasher] {
	return TrieBackendEssence[H, Hasher]{
		storage:       storage,
		root:          root,
		cache:         Cache[H]{ChildRoot: make(map[string]H)},
		trieNodeCache: cache,
		recorder:      recorder,
	}
}

// / Get backend storage reference.
func (tbe *TrieBackendEssence[H, Hasher]) BackendStorage() TrieBackendStorage[H] {
	return tbe.storage
}

// /// Get backend storage mutable reference.
// pub fn backend_storage_mut(&mut self) -> &mut S {
// 	&mut self.storage
// }

// / Get trie root.
func (tbe *TrieBackendEssence[H, Hasher]) Root() H {
	return tbe.root
}

// / Set trie root. This is useful for testing.
func (tbe *TrieBackendEssence[H, Hasher]) SetRoot(root H) {
	tbe.resetCache()
	tbe.root = root
}

func (tbe *TrieBackendEssence[H, Hasher]) resetCache() {
	tbe.cache = Cache[H]{ChildRoot: make(map[string]H)}
}

// /// Consumes self and returns underlying storage.
// pub fn into_storage(self) -> S {
// 	self.storage
// }

func withRecorderAndCache[H runtime.Hash, Hasher runtime.Hasher[H], R any](
	tbe *TrieBackendEssence[H, Hasher],
	storageRoot *H,
	// TODO: try and remove return params on callback
	callback func(triedb.TrieRecorder, triedb.TrieCache) (R, error),
) (R, error) {
	root := tbe.root
	if storageRoot != nil {
		root = *storageRoot
	}
	var cache triedb.TrieCache
	if tbe.trieNodeCache != nil {
		cache = tbe.trieNodeCache.AsTrieDBCache(root)
	}

	// TODO: implement recorder and cache
	// #[cfg(feature = "std")]
	// {
	// 	let mut recorder = self.recorder.as_ref().map(|r| r.as_trie_recorder(storage_root));
	// 	let recorder = match recorder.as_mut() {
	// 		Some(recorder) => Some(recorder as &mut dyn TrieRecorder<H::Out>),
	// 		None => None,
	// 	};
	// 	callback(recorder, cache)
	// }
	return callback(nil, cache)
}

// / Call the given closure passing it the recorder and the cache.
// /
// / This function must only be used when the operation in `callback` is
// / calculating a `storage_root`. It is expected that `callback` returns
// / the new storage root. This is required to register the changes in the cache
// / for the correct storage root. The given `storage_root` corresponds to the root of the "old"
// / trie. If the value is not given, `self.root` is used.
func withRecorderAndCacheForStorageRoot[H runtime.Hash, Hasher runtime.Hasher[H], R any](
	tbe *TrieBackendEssence[H, Hasher],
	storageRoot *H,
	// TODO: try and remove return params on callback
	callback func(triedb.TrieRecorder, triedb.TrieCache) (*H, R),
) R {
	// let storage_root = storage_root.unwrap_or_else(|| self.root);
	// let mut recorder = self.recorder.as_ref().map(|r| r.as_trie_recorder(storage_root));
	// let recorder = match recorder.as_mut() {
	// 	Some(recorder) => Some(recorder as &mut dyn TrieRecorder<H::Out>),
	// 	None => None,
	// };

	// let result = if let Some(local_cache) = self.trie_node_cache.as_ref() {
	// 	let mut cache = local_cache.as_trie_db_mut_cache();

	// 	let (new_root, r) = callback(recorder, Some(&mut cache));

	// 	if let Some(new_root) = new_root {
	// 		local_cache.merge(cache, new_root);
	// 	}

	// 	r
	// } else {
	// 	callback(recorder, None).1
	// };

	// result
	_, res := callback(nil, nil)
	return res
}

// / Calls the given closure with a [`TrieDb`] constructed for the given
// / storage root and (optionally) child trie.
func withTrieDB[H runtime.Hash, Hasher runtime.Hasher[H], R any](
	tbe *TrieBackendEssence[H, Hasher],
	root H,
	childInfo *storage.ChildInfo,
	callback func(triedb.TrieDB[H]),
) {
	backend := tbe
	var dbOpt *trie.KeySpacedDB[H, triedb.DBValue]
	if childInfo != nil {
		db := trie.NewKeySpacedDB[H, triedb.DBValue](backend, (*childInfo).Keyspace())
		dbOpt = &db
	}
	var db hashdb.HashDB[H, triedb.DBValue]
	if dbOpt != nil {
		db = dbOpt
	} else {
		db = backend
	}

	// withRecorderAndCache(tbe, &root, func(triedb.TrieRecorder, triedb.TrieCache) (R, error) {
	trie := triedb.NewTrieDBBuilder[H](db, root).Build()
	callback(trie)
	// })

	// panic("unimpl")
}

// / Returns the hash value
func (tbe *TrieBackendEssence[H, Hasher]) StorageHash(key []byte) (*H, error) {
	return withRecorderAndCache[H, Hasher, *H](tbe, nil, func(recorder triedb.TrieRecorder, cache triedb.TrieCache) (*H, error) {
		return triedb.NewTrieDBBuilder[H](tbe, tbe.root).Build().GetHash(key)
	})
}

// / Get the value of storage at given key.
func (tbe *TrieBackendEssence[H, Hasher]) Storage(key []byte) (*StorageValue, error) {
	return withRecorderAndCache[H, Hasher, *StorageValue](tbe, nil, func(recorder triedb.TrieRecorder, cache triedb.TrieCache) (*StorageValue, error) {
		val, err := trie.ReadTrieValue[H](tbe, tbe.root, key, recorder, cache)
		if val == nil {
			return nil, err
		}
		storageValue := StorageValue(*val)
		return &storageValue, err
	})
}

// / Create a raw iterator over the storage.
func (tbe *TrieBackendEssence[H, Hasher]) RawIter(args IterArgs) (RawIter[H, Hasher], error) {
	panic("unimpl")
}

func (tbe *TrieBackendEssence[H, Hasher]) StorageRoot(delta []struct {
	Key   []byte
	Value *[]byte
}, stateVersion storage.StateVersion) (H, trie.PrefixedMemoryDB[H, Hasher]) {
	writeOverlay := trie.NewPrefixedMemoryDB[H, Hasher]()

	root := withRecorderAndCacheForStorageRoot[H, Hasher, H](tbe, nil, func(_ triedb.TrieRecorder, _ triedb.TrieCache) (*H, H) {
		eph := newEphemeral[H, Hasher](tbe.BackendStorage(), writeOverlay)
		var (
			root H
			err  error
		)
		switch stateVersion {
		case storage.StateVersionV0:
			root, err = trie.DeltaTrieRoot[H](eph, tbe.root, delta, nil, nil)
		case storage.StateVersionV1:
			root, err = trie.DeltaTrieRoot[H](eph, tbe.root, delta, nil, nil)
		}
		if err != nil {
			log.Printf("WARN: failed to write to trie: %w", err)
			return nil, tbe.root
		}
		return &root, root
	})

	return root, writeOverlay
}

func (tbe *TrieBackendEssence[H, Hasher]) Get(key H, prefix hashdb.Prefix) *triedb.DBValue {
	if key == tbe.empty {
		val := triedb.DBValue{0}
		return &val
	}
	val, err := tbe.storage.Get(key, prefix)
	if err != nil {
		log.Printf("WARN: failed to write to trie: %v\n", err)
		return nil
	}
	return val
}

func (tbe *TrieBackendEssence[H, Hasher]) Contains(key H, prefix hashdb.Prefix) bool {
	return tbe.Get(key, prefix) != nil
}

func (tbe *TrieBackendEssence[H, Hasher]) Insert(prefix hashdb.Prefix, value []byte) H {
	panic("unimplemented")
}

func (tbe *TrieBackendEssence[H, Hasher]) Emplace(key H, prefix hashdb.Prefix, value triedb.DBValue) {
	panic("unimplemented")
}

func (tbe *TrieBackendEssence[H, Hasher]) Remove(key H, prefix hashdb.Prefix) {
	panic("unimplemented")
}

type ephemeral[H runtime.Hash, Hasher runtime.Hasher[H]] struct {
	storage TrieBackendStorage[H]
	overlay trie.PrefixedMemoryDB[H, Hasher]
}

func newEphemeral[H runtime.Hash, Hasher runtime.Hasher[H]](
	storage TrieBackendStorage[H],
	overlay trie.PrefixedMemoryDB[H, Hasher],
) *ephemeral[H, Hasher] {
	return &ephemeral[H, Hasher]{
		storage, overlay,
	}
}

func (e *ephemeral[H, Hasher]) Get(key H, prefix hashdb.Prefix) *triedb.DBValue {
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

func (e *ephemeral[H, Hasher]) Emplace(key H, prefix hashdb.Prefix, value triedb.DBValue) {
	e.overlay.Emplace(key, prefix, value)
}

func (e *ephemeral[H, Hasher]) Remove(key H, prefix hashdb.Prefix) {
	e.overlay.Remove(key, prefix)
}

// / Key-value pairs storage that is used by trie backend essence.
type TrieBackendStorage[H constraints.Ordered] interface {
	Get(key H, prefix hashdb.Prefix) (*triedb.DBValue, error)
}
