package triedb

import "github.com/ChainSafe/gossamer/pkg/trie/triedb/hash"

type CachedValues[H any] interface {
	NonExistingCachedValue[H] | ExistingHashCachedValue[H] | ExistingCachedValue[H]
	CachedValue[H]
}

type CachedValue[H any] interface {
	data() []byte
	hash() *H
}

func NewCachedValue[H any, CV CachedValues[H]](cv CV) CachedValue[H] {
	return cv
}

// The value doesn't exist in the trie.
type NonExistingCachedValue[H any] struct{}

func (NonExistingCachedValue[H]) data() []byte { return nil }
func (NonExistingCachedValue[H]) hash() *H     { return nil }

// We cached the hash, because we did not yet accessed the data.
type ExistingHashCachedValue[H any] struct {
	Hash H
}

func (ExistingHashCachedValue[H]) data() []byte  { return nil }
func (ehcv ExistingHashCachedValue[H]) hash() *H { return &ehcv.Hash }

// The value exists in the trie.
type ExistingCachedValue[H any] struct {
	/// The hash of the value.
	Hash H
	/// The actual data of the value stored as [`BytesWeak`].
	///
	/// The original data [`Bytes`] is stored in the trie node
	/// that is also cached by the [`TrieCache`]. If this node is dropped,
	/// this data will also not be "upgradeable" anymore.
	Data []byte
}

func (ecv ExistingCachedValue[H]) data() []byte { return ecv.Data }
func (ecv ExistingCachedValue[H]) hash() *H     { return &ecv.Hash }

// A cache that can be used to speed-up certain operations when accessing the trie.
//
// The [`TrieDB`]/[`TrieDBMut`] by default are working with the internal hash-db in a non-owning
// mode. This means that for every lookup in the trie, every node is always fetched and decoded on
// the fly. Fetching and decoding a node always takes some time and can kill the performance of any
// application that is doing quite a lot of trie lookups. To circumvent this performance
// degradation, a cache can be used when looking up something in the trie. Any cache that should be
// used with the [`TrieDB`]/[`TrieDBMut`] needs to implement this trait.
//
// The trait is laying out a two level cache, first the trie nodes cache and then the value cache.
// The trie nodes cache, as the name indicates, is for caching trie nodes as [`NodeOwned`]. These
// trie nodes are referenced by their hash. The value cache is caching [`CachedValue`]'s and these
// are referenced by the key to look them up in the trie. As multiple different tries can have
// different values under the same key, it up to the cache implementation to ensure that the
// correct value is returned. As each trie has a different root, this root can be used to
// differentiate values under the same key.
type TrieCache[H hash.Hash] interface {
	/// Lookup value for the given `key`.
	///
	/// Returns the `None` if the `key` is unknown or otherwise `Some(_)` with the associated
	/// value.
	///
	/// [`Self::cache_data_for_key`] is used to make the cache aware of data that is associated
	/// to a `key`.
	///
	/// # Attention
	///
	/// The cache can be used for different tries, aka with different roots. This means
	/// that the cache implementation needs to take care of always returning the correct value
	/// for the current trie root.
	GetValue(key []byte) CachedValue[H]
	/// Cache the given `value` for the given `key`.
	///
	/// # Attention
	///
	/// The cache can be used for different tries, aka with different roots. This means
	/// that the cache implementation needs to take care of caching `value` for the current
	/// trie root.
	SetValue(key []byte, value CachedValue[H])

	/// Get or insert a [`NodeOwned`].
	///
	/// The cache implementation should look up based on the given `hash` if the node is already
	/// known. If the node is not yet known, the given `fetch_node` function can be used to fetch
	/// the particular node.
	///
	/// Returns the [`NodeOwned`] or an error that happened on fetching the node.
	GetOrInsertNode(hash H, fetchNode func() (NodeOwned[H], error)) (NodeOwned[H], error)

	/// Get the [`NodeOwned`] that corresponds to the given `hash`.
	GetNode(hash H) NodeOwned[H]
}
