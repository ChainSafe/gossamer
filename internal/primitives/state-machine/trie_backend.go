package statemachine

import (
	"github.com/ChainSafe/gossamer/internal/primitives/runtime"
	triedb "github.com/ChainSafe/gossamer/internal/trie-db"
)

// pub trait TrieCacheProvider<H: Hasher> {
type TrieCacheProvider[H any] interface {
	// 	/// Cache type that implements [`trie_db::TrieCache`].
	// 	type Cache<'a>: TrieCacheT<sp_trie::NodeCodec<H>> + 'a
	// 	where
	// 		Self: 'a;

	// 	/// Return a [`trie_db::TrieDB`] compatible cache.
	// 	///
	// 	/// The `storage_root` parameter *must* be the storage root of the trie this cache is used for.
	// 	///
	// 	/// NOTE: Implementors should use the `storage_root` to differentiate between storage keys that
	// 	/// may belong to different tries.
	// 	fn as_trie_db_cache(&self, storage_root: H::Out) -> Self::Cache<'_>;
	AsTrieDBCache(storageRoot H) triedb.TrieCache

	// 	/// Returns a cache that can be used with a [`trie_db::TrieDBMut`].
	// 	///
	// 	/// When finished with the operation on the trie, it is required to call [`Self::merge`] to
	// 	/// merge the cached items for the correct `storage_root`.
	// 	fn as_trie_db_mut_cache(&self) -> Self::Cache<'_>;

	// /// Merge the cached data in `other` into the provider using the given `new_root`.
	// ///
	// /// This must be used for the cache returned by [`Self::as_trie_db_mut_cache`] as otherwise the
	// /// cached data is just thrown away.
	// fn merge<'a>(&'a self, other: Self::Cache<'a>, new_root: H::Out);
}

// / Patricia trie-based backend. Transaction type is an overlay of changes to commit.
// pub struct TrieBackend<S: TrieBackendStorage<H>, H: Hasher, C = DefaultCache<H>> {
type TrieBackend[H runtime.Hash, Hasher runtime.Hasher[H]] struct {
	// pub(crate) essence: TrieBackendEssence<S, H, C>,
	Essence TrieBackendEssence[H, Hasher]
	// next_storage_key_cache: CacheCell<Option<CachedIter<S, H, C>>>,
}
