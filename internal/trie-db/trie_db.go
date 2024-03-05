package triedb

import (
	hashdb "github.com/ChainSafe/gossamer/internal/hash-db"
	"golang.org/x/exp/constraints"
)

// / Database value
type DBValue []byte

// / A trie recorder that can be used to record all kind of [`TrieAccess`]'s.
// /
// / To build a trie proof a recorder is required that records all trie accesses. These recorded trie
// / accesses can then be used to create the proof.
type TrieRecorder interface {
	/// Record the given [`TrieAccess`].
	///
	/// Depending on the [`TrieAccess`] a call of [`Self::trie_nodes_recorded_for_key`] afterwards
	/// must return the correct recorded state.
	// fn record<'a>(&mut self, access: TrieAccess<'a, H>);

	/// Check if we have recorded any trie nodes for the given `key`.
	///
	/// Returns [`RecordedForKey`] to express the state of the recorded trie nodes.
	// fn trie_nodes_recorded_for_key(&self, key: &[u8]) -> RecordedForKey;
}

// / A key-value datastore implemented as a database-backed modified Merkle tree.
// pub trait Trie<L: TrieLayout> {
type Trie[Hash any] interface {
	// /// Return the root of the trie.
	// fn root(&self) -> &TrieHash<L>;

	// /// Is the trie empty?
	// fn is_empty(&self) -> bool {
	// 	*self.root() == L::Codec::hashed_null_node()
	// }

	// /// Does the trie contain a given key?
	// fn contains(&self, key: &[u8]) -> Result<bool, TrieHash<L>, CError<L>> {
	// 	self.get(key).map(|x| x.is_some())
	// }

	/// Returns the hash of the value for `key`.
	// fn get_hash(&self, key: &[u8]) -> Result<Option<TrieHash<L>>, TrieHash<L>, CError<L>>;
	GetHash(key []byte) (*Hash, error)

	/// What is the value of the given key in this trie?
	// fn get(&self, key: &[u8]) -> Result<Option<DBValue>, TrieHash<L>, CError<L>> {
	// 	self.get_with(key, |v: &[u8]| v.to_vec())
	// }
	Get(key []byte) (*DBValue, error)

	// /// Search for the key with the given query parameter. See the docs of the `Query`
	// /// trait for more details.
	// fn get_with<Q: Query<L::Hash>>(
	// 	&self,
	// 	key: &[u8],
	// 	query: Q,
	// ) -> Result<Option<Q::Item>, TrieHash<L>, CError<L>>;

	// /// Look up the [`MerkleValue`] of the node that is the closest descendant for the provided
	// /// key.
	// ///
	// /// When the provided key leads to a node, then the merkle value of that node
	// /// is returned. However, if the key does not lead to a node, then the merkle value
	// /// of the closest descendant is returned. `None` if no such descendant exists.
	// fn lookup_first_descendant(
	// 	&self,
	// 	key: &[u8],
	// ) -> Result<Option<MerkleValue<TrieHash<L>>>, TrieHash<L>, CError<L>>;

	// /// Returns a depth-first iterator over the elements of trie.
	// fn iter<'a>(
	// 	&'a self,
	// ) -> Result<
	// 	Box<dyn TrieIterator<L, Item = TrieItem<TrieHash<L>, CError<L>>> + 'a>,
	// 	TrieHash<L>,
	// 	CError<L>,
	// >;

	// /// Returns a depth-first iterator over the keys of elemets of trie.
	// fn key_iter<'a>(
	// 	&'a self,
	// ) -> Result<
	// 	Box<dyn TrieIterator<L, Item = TrieKeyItem<TrieHash<L>, CError<L>>> + 'a>,
	// 	TrieHash<L>,
	// 	CError<L>,
	// >;
}

type TrieMut[Hash any] interface {
	/// Return the root of the trie.
	// fn root(&mut self) -> &TrieHash<L>;

	/// Is the trie empty?
	// fn is_empty(&self) -> bool;

	/// Does the trie contain a given key?
	// fn contains(&self, key: &[u8]) -> Result<bool, TrieHash<L>, CError<L>> {
	// 	self.get(key).map(|x| x.is_some())
	// }

	/// What is the value of the given key in this trie?
	// fn get<'a, 'key>(&'a self, key: &'key [u8]) -> Result<Option<DBValue>, TrieHash<L>, CError<L>>
	// where
	// 	'a: 'key;

	/// Insert a `key`/`value` pair into the trie. An empty value is equivalent to removing
	/// `key` from the trie. Returns the old value associated with this key, if it existed.
	// fn insert(
	// 	&mut self,
	// 	key: &[u8],
	// 	value: &[u8],
	// ) -> Result<Option<Value<L>>, TrieHash<L>, CError<L>>;
	Insert(key []byte, value []byte) (*Value, error)

	/// Remove a `key` from the trie. Equivalent to making it equal to the empty
	/// value. Returns the old value associated with this key, if it existed.
	// fn remove(&mut self, key: &[u8]) -> Result<Option<Value<L>>, TrieHash<L>, CError<L>>;
	Remove(key []byte) (*Value, error)
}

// / Trait with definition of trie layout.
// / Contains all associated trait needed for
// / a trie definition or implementation.
// pub trait TrieLayout {
type TrieLayout[H constraints.Ordered, Hasher hashdb.Hasher[H]] struct {
	/// If true, the trie will use extension nodes and
	/// no partial in branch, if false the trie will only
	/// use branch and node with partials in both.
	UseExtension bool
	/// If true, the trie will allow empty values into `TrieDBMut`
	AllowEmpty bool
	/// Threshold above which an external node should be
	/// use to store a node value.
	MaxInlineValue *uint32

	/// Hasher to use for this trie.
	// type Hash: Hasher;
	/// Codec to use (needs to match hasher and nibble ops).
	// type Codec: NodeCodec<HashOut = <Self::Hash as Hasher>::Out>;
	Codec NodeCodec[H]
}

// / A cache that can be used to speed-up certain operations when accessing the trie.
// /
// / The [`TrieDB`]/[`TrieDBMut`] by default are working with the internal hash-db in a non-owning
// / mode. This means that for every lookup in the trie, every node is always fetched and decoded on
// / the fly. Fetching and decoding a node always takes some time and can kill the performance of any
// / application that is doing quite a lot of trie lookups. To circumvent this performance
// / degradation, a cache can be used when looking up something in the trie. Any cache that should be
// / used with the [`TrieDB`]/[`TrieDBMut`] needs to implement this trait.
// /
// / The trait is laying out a two level cache, first the trie nodes cache and then the value cache.
// / The trie nodes cache, as the name indicates, is for caching trie nodes as [`NodeOwned`]. These
// / trie nodes are referenced by their hash. The value cache is caching [`CachedValue`]'s and these
// / are referenced by the key to look them up in the trie. As multiple different tries can have
// / different values under the same key, it up to the cache implementation to ensure that the
// / correct value is returned. As each trie has a different root, this root can be used to
// / differentiate values under the same key.
// pub trait TrieCache<NC: NodeCodec> {
type TrieCache interface {
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
	// fn lookup_value_for_key(&mut self, key: &[u8]) -> Option<&CachedValue<NC::HashOut>>;

	/// Cache the given `value` for the given `key`.
	///
	/// # Attention
	///
	/// The cache can be used for different tries, aka with different roots. This means
	/// that the cache implementation needs to take care of caching `value` for the current
	/// trie root.
	// fn cache_value_for_key(&mut self, key: &[u8], value: CachedValue<NC::HashOut>);

	/// Get or insert a [`NodeOwned`].
	///
	/// The cache implementation should look up based on the given `hash` if the node is already
	/// known. If the node is not yet known, the given `fetch_node` function can be used to fetch
	/// the particular node.
	///
	/// Returns the [`NodeOwned`] or an error that happened on fetching the node.
	// fn get_or_insert_node(
	// 	&mut self,
	// 	hash: NC::HashOut,
	// 	fetch_node: &mut dyn FnMut() -> Result<NodeOwned<NC::HashOut>, NC::HashOut, NC::Error>,
	// ) -> Result<&NodeOwned<NC::HashOut>, NC::HashOut, NC::Error>;

	/// Get the [`NodeOwned`] that corresponds to the given `hash`.
	// fn get_node(&mut self, hash: &NC::HashOut) -> Option<&NodeOwned<NC::HashOut>>;
}
