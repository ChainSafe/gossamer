package hashdb

// / A trie node prefix, it is the nibble path from the trie root
// / to the trie node.
// / For a node containing no partial key value it is the full key.
// / For a value node or node containing a partial key, it is the full key minus its node partial
// / nibbles (the node key can be split into prefix and node partial).
// / Therefore it is always the leftmost portion of the node key, so its internal representation
// / is a non expanded byte slice followed by a last padded byte representation.
// / The padded byte is an optional padded value.
type Prefix struct {
	Key    []byte
	Padded *byte
}

/// Trait modelling datastore keyed by a hash defined by the `Hasher`.
// pub trait HashDB<H: Hasher, T>: Send + Sync + AsHashDB<H, T> {
// 	/// Look up a given hash into the bytes that hash to it, returning None if the
// 	/// hash is not known.
// 	fn get(&self, key: &H::Out, prefix: Prefix) -> Option<T>;

// 	/// Check for the existence of a hash-key.
// 	fn contains(&self, key: &H::Out, prefix: Prefix) -> bool;

// 	/// Insert a datum item into the DB and return the datum's hash for a later lookup. Insertions
// 	/// are counted and the equivalent number of `remove()`s must be performed before the data
// 	/// is considered dead.
// 	fn insert(&mut self, prefix: Prefix, value: &[u8]) -> H::Out;

// 	/// Like `insert()`, except you provide the key and the data is all moved.
// 	fn emplace(&mut self, key: H::Out, prefix: Prefix, value: T);

// 	/// Remove a datum previously inserted. Insertions can be "owed" such that the same number of
// 	/// `insert()`s may happen without the data being eventually being inserted into the DB.
// 	/// It can be "owed" more than once.
// 	fn remove(&mut self, key: &H::Out, prefix: Prefix);
// }

// / Trait for immutable reference of HashDB.
// pub trait HashDBRef<H: Hasher, T> {
type HashDBRef[Hash comparable, T any] interface {
	/// Look up a given hash into the bytes that hash to it, returning None if the
	/// hash is not known.
	// 	fn get(&self, key: &H::Out, prefix: Prefix) -> Option<T>;
	Get(key Hash, prefix Prefix) *T

	/// Check for the existance of a hash-key.
	// fn contains(&self, key: &H::Out, prefix: Prefix) -> bool;
	Contains(key Hash, prefix Prefix) bool
}
