package statemachine

import "golang.org/x/exp/constraints"

// / Trait that allows consolidate two transactions together.
type Consolidate interface {
	Consolidate(other Consolidate)
}

type Transaction interface {
	Consolidate
}

// / The output type of the `Hasher`
//
//	type Out: AsRef<[u8]>
//		+ AsMut<[u8]>
//		+ Default
//		+ MaybeDebug
//		+ core::cmp::Ord
//		+ PartialEq
//		+ Eq
//		+ hash::Hash
//		+ Send
//		+ Sync
//		+ Clone
//		+ Copy;
type HasherOut interface {
	constraints.Ordered
}

// / Trait describing an object that can hash a slice of bytes. Used to abstract
// / other types over the hashing algorithm. Defines a single `hash` method and an
// / `Out` associated type with the necessary bounds.
type Hasher[H HasherOut] interface {
	Hash(x []byte) H
}

// pub trait Hasher: Sync + Send {
//
// 	/// What to use to build `HashMap`s with this `Hasher`.
// 	type StdHasher: Sync + Send + Default + hash::Hasher;
// 	/// The length in bytes of the `Hasher` output.
// 	const LENGTH: usize;

// 	/// Compute the hash of the provided slice of bytes returning the `Out` type of the `Hasher`.
// 	fn hash(x: &[u8]) -> Self::Out;
// }

// / Trait modelling datastore keyed by a hash defined by the `Hasher`.
// pub trait HashDB<H: Hasher, T>: Send + Sync + AsHashDB<H, T> {
type HashDB[H HasherOut, T any] interface {
	// 	/// Look up a given hash into the bytes that hash to it, returning None if the
	// 	/// hash is not known.
	// 	fn get(&self, key: &H::Out, prefix: Prefix) -> Option<T>;
	Get(key H, prefix Prefix) *T
	// 	/// Check for the existence of a hash-key.
	// 	fn contains(&self, key: &H::Out, prefix: Prefix) -> bool;
	Contains(key H, prefix Prefix) bool
	// 	/// Insert a datum item into the DB and return the datum's hash for a later lookup. Insertions
	// 	/// are counted and the equivalent number of `remove()`s must be performed before the data
	// 	/// is considered dead.
	// 	fn insert(&mut self, prefix: Prefix, value: &[u8]) -> H::Out;
	Insert(prefix Prefix, value []byte) H
	// 	/// Like `insert()`, except you provide the key and the data is all moved.
	// 	fn emplace(&mut self, key: H::Out, prefix: Prefix, value: T);
	Emplace(key H, prefix Prefix, value T)
	// 	/// Remove a datum previously inserted. Insertions can be "owed" such that the same number of
	// 	/// `insert()`s may happen without the data being eventually being inserted into the DB.
	// 	/// It can be "owed" more than once.
	// 	fn remove(&mut self, key: &H::Out, prefix: Prefix);
	Remove(key H, prefix Prefix)
}

// / Type of in-memory overlay.
// type Overlay: HashDB<H, DBValue> + Default + Consolidate;
type Overlay[H HasherOut] interface {
	HashDB[H, []byte]
	Consolidate
}

type Prefix struct {
	Key    []byte
	Padded *byte
}

// / Key-value pairs storage that is used by trie backend essence.
type TrieBackendStorage[H HasherOut] interface {
	Get(key H, prefix Prefix) (*[]byte, error)
}

// / A state backend is used to read state data and can have changes committed
// / to it.
// /
// / The clone operation (if implemented) should be cheap.
type Backend interface{}
