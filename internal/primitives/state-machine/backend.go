package statemachine

import (
	"github.com/ChainSafe/gossamer/internal/primitives/state-machine/stats"
	"github.com/ChainSafe/gossamer/internal/primitives/storage"
	"github.com/ChainSafe/gossamer/internal/primitives/trie"
	"golang.org/x/exp/constraints"
)

// // / Trait that allows consolidate two transactions together.
// type Consolidate interface {
// 	Consolidate(other Consolidate)
// }

// type Transaction interface {
// 	// Consolidate
// }

// // / The output type of the `Hasher`
// //
// //	type Out: AsRef<[u8]>
// //		+ AsMut<[u8]>
// //		+ Default
// //		+ MaybeDebug
// //		+ core::cmp::Ord
// //		+ PartialEq
// //		+ Eq
// //		+ hash::Hash
// //		+ Send
// //		+ Sync
// //		+ Clone
// //		+ Copy;
type HasherOut interface {
	constraints.Ordered
}

// / Trait describing an object that can hash a slice of bytes. Used to abstract
// / other types over the hashing algorithm. Defines a single `hash` method and an
// / `Out` associated type with the necessary bounds.
type Hasher[H HasherOut] interface {
	Hash(x []byte) H
}

// // pub trait Hasher: Sync + Send {
// //
// // 	/// What to use to build `HashMap`s with this `Hasher`.
// // 	type StdHasher: Sync + Send + Default + hash::Hasher;
// // 	/// The length in bytes of the `Hasher` output.
// // 	const LENGTH: usize;

// // 	/// Compute the hash of the provided slice of bytes returning the `Out` type of the `Hasher`.
// // 	fn hash(x: &[u8]) -> Self::Out;
// // }

// // / Trait modelling datastore keyed by a hash defined by the `Hasher`.
// // pub trait HashDB<H: Hasher, T>: Send + Sync + AsHashDB<H, T> {
// type HashDB[H HasherOut, T any] interface {
// 	// 	/// Look up a given hash into the bytes that hash to it, returning None if the
// 	// 	/// hash is not known.
// 	// 	fn get(&self, key: &H::Out, prefix: Prefix) -> Option<T>;
// 	Get(key H, prefix Prefix) *T
// 	// 	/// Check for the existence of a hash-key.
// 	// 	fn contains(&self, key: &H::Out, prefix: Prefix) -> bool;
// 	Contains(key H, prefix Prefix) bool
// 	// 	/// Insert a datum item into the DB and return the datum's hash for a later lookup. Insertions
// 	// 	/// are counted and the equivalent number of `remove()`s must be performed before the data
// 	// 	/// is considered dead.
// 	// 	fn insert(&mut self, prefix: Prefix, value: &[u8]) -> H::Out;
// 	Insert(prefix Prefix, value []byte) H
// 	// 	/// Like `insert()`, except you provide the key and the data is all moved.
// 	// 	fn emplace(&mut self, key: H::Out, prefix: Prefix, value: T);
// 	Emplace(key H, prefix Prefix, value T)
// 	// 	/// Remove a datum previously inserted. Insertions can be "owed" such that the same number of
// 	// 	/// `insert()`s may happen without the data being eventually being inserted into the DB.
// 	// 	/// It can be "owed" more than once.
// 	// 	fn remove(&mut self, key: &H::Out, prefix: Prefix);
// 	Remove(key H, prefix Prefix)
// }

// // / Type of in-memory overlay.
// // type Overlay: HashDB<H, DBValue> + Default + Consolidate;
// type Overlay[H HasherOut] interface {
// 	HashDB[H, []byte]
// 	Consolidate
// }

// type Prefix struct {
// 	Key    []byte
// 	Padded *byte
// }

// / A struct containing arguments for iterating over the storage.
type IterArgs struct {
	/// The prefix of the keys over which to iterate.
	// 	pub prefix: Option<&'a [u8]>,
	Prefix *[]byte

	/// The prefix from which to start the iteration from.
	///
	/// This is inclusive and the iteration will include the key which is specified here.
	StartAt *[]byte

	/// If this is `true` then the iteration will *not* include
	/// the key specified in `start_at`, if there is such a key.
	// 	pub start_at_exclusive: bool,
	StartAtExclusive bool

	/// The info of the child trie over which to iterate over.
	// 	pub child_info: Option<ChildInfo>,
	ChildInfo *storage.ChildInfo

	/// Whether to stop iteration when a missing trie node is reached.
	///
	/// When a missing trie node is reached the iterator will:
	///   - return an error if this is set to `false` (default)
	///   - return `None` if this is set to `true`
	// pub stop_on_incomplete_database: bool,
	StopOnIncompleteDatabase bool
}

// / A trait for a raw storage iterator.
// type StorageIterator[H HasherOut, Backend any] interface {
// 	// 	/// Fetches the next key from the storage.
// 	// 	fn next_key(
// 	// 		&mut self,
// 	// 		backend: &Self::Backend,
// 	// 	) -> Option<core::result::Result<StorageKey, Self::Error>>;
// 	NextKey(backend Backend[H, T]) (overlayedchanges.StorageKey, error)

// 	// 	/// Fetches the next key and value from the storage.
// 	// 	fn next_pair(
// 	// 		&mut self,
// 	// 		backend: &Self::Backend,
// 	// 	) -> Option<core::result::Result<(StorageKey, StorageValue), Self::Error>>;
// 	NextPair(backend Backend[H, T]) (struct {
// 		overlayedchanges.StorageKey
// 		overlayedchanges.StorageValue
// 	}, error)

// 	/// Returns whether the end of iteration was reached without an error.
// 	// fn was_complete(&self) -> bool;
// 	WasComplete() bool
// }

// / An iterator over storage keys and values.
// pub struct PairsIter<'a, H, I>
// where
//
//	H: Hasher,
//	I: StorageIterator<H>,
//
//	{
//		backend: Option<&'a I::Backend>,
//		raw_iter: I,
//		_phantom: PhantomData<H>,
//	}
// type PairsIter[H HasherOut, T Transaction] struct {
// 	backend *Backend[H, T]
// 	rawIter StorageIterator[H, T]
// }

// / An iterator over storage keys.
// pub struct KeysIter<'a, H, I>
// where
//
//	H: Hasher,
//	I: StorageIterator<H>,
//
//	{
//		backend: Option<&'a I::Backend>,
//		raw_iter: I,
//		_phantom: PhantomData<H>,
//	}
// type KeysIter[H HasherOut, T Transaction] struct {
// 	backend *Backend[H, T]
// 	rawIter StorageIterator[H, T]
// }

type BackendTransaction[Hasher any] trie.PrefixedMemoryDB[Hasher]

// / A state backend is used to read state data and can have changes committed
// / to it.
// /
// / The clone operation (if implemented) should be cheap.
type Backend[Hash HasherOut, H Hasher[Hash]] interface {
	/// Get keyed storage or None if there is nothing associated.
	// fn storage(&self, key: &[u8]) -> Result<Option<StorageValue>, Self::Error>;
	Storage(key []byte) (*StorageValue, error)

	/// Get keyed storage value hash or None if there is nothing associated.
	// fn storage_hash(&self, key: &[u8]) -> Result<Option<H::Out>, Self::Error>;
	StorageHash(key []byte) (*Hash, error)

	/// Get keyed child storage or None if there is nothing associated.
	// fn child_storage(
	// 	&self,
	// 	child_info: &ChildInfo,
	// 	key: &[u8],
	// ) -> Result<Option<StorageValue>, Self::Error>;
	ChildStorage(childInfo storage.ChildInfo, key []byte) (*StorageValue, error)

	/// Get child keyed storage value hash or None if there is nothing associated.
	// fn child_storage_hash(
	// 	&self,
	// 	child_info: &ChildInfo,
	// 	key: &[u8],
	// ) -> Result<Option<H::Out>, Self::Error>;
	ChildStorageHash(childInfo storage.ChildInfo, key []byte) (*Hash, error)

	/// true if a key exists in storage.
	// fn exists_storage(&self, key: &[u8]) -> Result<bool, Self::Error> {
	// 	Ok(self.storage_hash(key)?.is_some())
	// }
	ExistsStorage(key []byte) (bool, error)

	/// true if a key exists in child storage.
	// fn exists_child_storage(
	// 	&self,
	// 	child_info: &ChildInfo,
	// 	key: &[u8],
	// ) -> Result<bool, Self::Error> {
	// 	Ok(self.child_storage_hash(child_info, key)?.is_some())
	// }
	ExistsChildStorage(childInfo storage.ChildInfo, key []byte) (bool, error)

	/// Return the next key in storage in lexicographic order or `None` if there is no value.
	// fn next_storage_key(&self, key: &[u8]) -> Result<Option<StorageKey>, Self::Error>;
	NextStorageKey(key []byte) (*StorageKey, error)

	/// Return the next key in child storage in lexicographic order or `None` if there is no value.
	// fn next_child_storage_key(
	// 	&self,
	// 	child_info: &ChildInfo,
	// 	key: &[u8],
	// ) -> Result<Option<StorageKey>, Self::Error>;
	NextChildStorageKey(childInfo storage.ChildInfo, key []byte) (*StorageKey, error)

	/// Calculate the storage root, with given delta over what is already stored in
	/// the backend, and produce a "transaction" that can be used to commit.
	/// Does not include child storage updates.
	// fn storage_root<'a>(
	// 	&self,
	// 	delta: impl Iterator<Item = (&'a [u8], Option<&'a [u8]>)>,
	// 	state_version: StateVersion,
	// ) -> (H::Out, Self::Transaction)
	// where
	// 	H::Out: Ord;
	StorageRoot(delta []struct {
		Key   []byte
		Value []byte
	}, stateVersion storage.StateVersion) (Hash, BackendTransaction[H])

	/// Calculate the child storage root, with given delta over what is already stored in
	/// the backend, and produce a "transaction" that can be used to commit. The second argument
	/// is true if child storage root equals default storage root.
	// fn child_storage_root<'a>(
	// 	&self,
	// 	child_info: &ChildInfo,
	// 	delta: impl Iterator<Item = (&'a [u8], Option<&'a [u8]>)>,
	// 	state_version: StateVersion,
	// ) -> (H::Out, bool, Self::Transaction)
	// where
	// 	H::Out: Ord;
	ChildStorageRoot(childInfo storage.ChildInfo, delta []struct {
		Key   []byte
		Value []byte
	}, stateVersion storage.StateVersion) (H, bool, BackendTransaction[H])

	// /// Returns a lifetimeless raw storage iterator.
	// fn raw_iter(&self, args: IterArgs) -> Result<Self::RawIter, Self::Error>;
	// RawIter(args IterArgs) (StorageIterator[H, T], error)

	// /// Get an iterator over key/value pairs.
	// fn pairs<'a>(&'a self, args: IterArgs) -> Result<PairsIter<'a, H, Self::RawIter>, Self::Error> {
	// 	Ok(PairsIter {
	// 		backend: Some(self),
	// 		raw_iter: self.raw_iter(args)?,
	// 		_phantom: Default::default(),
	// 	})
	// }
	// Pairs(args IterArgs) (PairsIter[H, T], error)

	// /// Get an iterator over keys.
	// fn keys<'a>(&'a self, args: IterArgs) -> Result<KeysIter<'a, H, Self::RawIter>, Self::Error> {
	// 	Ok(KeysIter {
	// 		backend: Some(self),
	// 		raw_iter: self.raw_iter(args)?,
	// 		_phantom: Default::default(),
	// 	})
	// }
	// Keys(args IterArgs) (KeysIter[H, T], error)

	// /// Calculate the storage root, with given delta over what is already stored
	// /// in the backend, and produce a "transaction" that can be used to commit.
	// /// Does include child storage updates.
	// fn full_storage_root<'a>(
	// 	&self,
	// 	delta: impl Iterator<Item = (&'a [u8], Option<&'a [u8]>)>,
	// 	child_deltas: impl Iterator<
	// 		Item = (&'a ChildInfo, impl Iterator<Item = (&'a [u8], Option<&'a [u8]>)>),
	// 	>,
	// 	state_version: StateVersion,
	// ) -> (H::Out, Self::Transaction)
	// where
	// 	H::Out: Ord + Encode,
	FullStorageRoot(delta []struct {
		Key   []byte
		Value []byte
	}, childDeltas []struct {
		storage.ChildInfo
		Values []struct {
			Key   []byte
			Value []byte
		}
	}, stateVersion storage.StateVersion) (H, BackendTransaction[H])

	/// Register stats from overlay of state machine.
	///
	/// By default nothing is registered.
	// fn register_overlay_stats(&self, _stats: &crate::stats::StateMachineStats);
	RegisterOverlayStats(stats stats.StateMachineStats)

	/// Query backend usage statistics (i/o, memory)
	///
	/// Not all implementations are expected to be able to do this. In the
	/// case when they don't, empty statistics is returned.
	// fn usage_info(&self) -> UsageInfo;
	UsageInfo() stats.UsageInfo

	/// Wipe the state database.
	// fn wipe(&self) -> Result<(), Self::Error> {
	// 	unimplemented!()
	// }
	Wipe() error

	/// Commit given transaction to storage.
	// fn commit(
	// 	&self,
	// 	_: H::Out,
	// 	_: Self::Transaction,
	// 	_: StorageCollection,
	// 	_: ChildStorageCollection,
	// ) -> Result<(), Self::Error> {
	// 	unimplemented!()
	// }
	Commit(H, BackendTransaction[H], StorageCollection, ChildStorageCollection) error

	/// Get the read/write count of the db
	// fn read_write_count(&self) -> (u32, u32, u32, u32) {
	// 	unimplemented!()
	// }
	ReadWriteCount() (uint32, uint32, uint32, uint32)

	// /// Get the read/write count of the db
	// fn reset_read_write_count(&self) {
	// 	unimplemented!()
	// }
	ResetReadWriteCount()

	// /// Get the whitelist for tracking db reads/writes
	// fn get_whitelist(&self) -> Vec<TrackedStorageKey> {
	// 	Default::default()
	// }
	GetWhitelist() []storage.TrackedStorageKey

	// /// Update the whitelist for tracking db reads/writes
	// fn set_whitelist(&self, _: Vec<TrackedStorageKey>) {}
	SetWhitelist([]storage.TrackedStorageKey)

	// /// Estimate proof size
	// fn proof_size(&self) -> Option<u32> {
	// 	unimplemented!()
	// }
	ProofSize() *uint32

	// /// Extend storage info for benchmarking db
	// fn get_read_and_written_keys(&self) -> Vec<(Vec<u8>, u32, u32, bool)> {
	// 	unimplemented!()
	// }
	GetReadAndWrittenKeys() []struct {
		Key         []byte
		Read        uint32
		Write       uint32
		Whitelisted bool
	}
}
