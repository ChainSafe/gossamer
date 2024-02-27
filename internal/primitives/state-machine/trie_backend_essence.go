package statemachine

import (
	"sync"

	hashdb "github.com/ChainSafe/gossamer/internal/hash-db"
	"github.com/ChainSafe/gossamer/internal/primitives/trie"
	"github.com/ChainSafe/gossamer/internal/primitives/trie/recorder"
	triedb "github.com/ChainSafe/gossamer/internal/trie-db"
)

// / Local cache for child root.
type Cache[H any] struct {
	// pub child_root: HashMap<Vec<u8>, Option<H>>,
	ChildRoot map[string]H
}

// / Patricia trie-based pairs storage essence.
// pub struct TrieBackendEssence<S: TrieBackendStorage<H>, H: Hasher, C> {
type TrieBackendEssence[H HasherOut] struct {
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

func newTrieBackendEssence[H HasherOut](
	storage TrieBackendStorage[H],
	root H,
	cache TrieCacheProvider[H],
	recorder *recorder.Recorder[H],
) TrieBackendEssence[H] {
	return TrieBackendEssence[H]{
		storage:       storage,
		root:          root,
		cache:         Cache[H]{ChildRoot: make(map[string]H)},
		trieNodeCache: cache,
		recorder:      recorder,
	}
}

// / Get backend storage reference.
func (tbe *TrieBackendEssence[H]) BackendStorage() TrieBackendStorage[H] {
	return tbe.storage
}

// /// Get backend storage mutable reference.
// pub fn backend_storage_mut(&mut self) -> &mut S {
// 	&mut self.storage
// }

// / Get trie root.
func (tbe *TrieBackendEssence[H]) Root() H {
	return tbe.root
}

// / Set trie root. This is useful for testing.
func (tbe *TrieBackendEssence[H]) SetRoot(root H) {
	tbe.resetCache()
	tbe.root = root
}

func (tbe *TrieBackendEssence[H]) resetCache() {
	tbe.cache = Cache[H]{ChildRoot: make(map[string]H)}
}

// /// Consumes self and returns underlying storage.
// pub fn into_storage(self) -> S {
// 	self.storage
// }

func withRecorderAndCache[H HasherOut, R any](
	tbe *TrieBackendEssence[H],
	storageRoot *H,
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
func withRecorderAndCacheForStorageRoot[H HasherOut, R any](
	tbe *TrieBackendEssence[H],
	storageRoot *H,
	callback func(triedb.TrieRecorder, triedb.TrieCache) (*H, R, error),
) (R, error) {
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
	_, res, err := callback(nil, nil)
	return res, err
}

func (tbe *TrieBackendEssence[H]) StorageHash(key []byte) (*H, error) {
	return withRecorderAndCache[H, *H](tbe, nil, func(recorder triedb.TrieRecorder, cache triedb.TrieCache) (*H, error) {
		return triedb.NewTrieDBBuilder[H](tbe.root).Build().GetHash(key)
	})
}

func (tbe *TrieBackendEssence[H]) Storage(key []byte) (*StorageValue, error) {
	return withRecorderAndCache[H, *StorageValue](tbe, nil, func(recorder triedb.TrieRecorder, cache triedb.TrieCache) (*StorageValue, error) {
		val, err := trie.ReadTrieValue[H](tbe, tbe.root, key, recorder, cache)
		if val == nil {
			return nil, err
		}
		storageValue := StorageValue(*val)
		return &storageValue, err
	})
}

// /// Get the value of storage at given key.
// pub fn storage(&self, key: &[u8]) -> Result<Option<StorageValue>> {
// 	let map_e = |e| format!("Trie lookup error: {}", e);

// 	self.with_recorder_and_cache(None, |recorder, cache| {
// 		read_trie_value::<Layout<H>, _>(self, &self.root, key, recorder, cache).map_err(map_e)
// 	})
// }

func (tbe *TrieBackendEssence[H]) Get(key H, prefix hashdb.Prefix) *triedb.DBValue {
	panic("unimplemented")
}

func (tbe *TrieBackendEssence[H]) Contains(key H, prefix hashdb.Prefix) bool {
	panic("unimplemented")
}

// // / Key-value pairs storage that is used by trie backend essence.
type TrieBackendStorage[H HasherOut] interface {
	Get(key H, prefix hashdb.Prefix) (*[]byte, error)
}
