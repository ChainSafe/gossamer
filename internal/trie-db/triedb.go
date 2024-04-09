package triedb

import (
	hashdb "github.com/ChainSafe/gossamer/internal/hash-db"
)

// / A builder for creating a [`TrieDB`].
// pub struct TrieDBBuilder<'db, 'cache, L: TrieLayout> {
type TrieDBBuilder[Hash any] struct {
	// db: &'db dyn HashDBRef<L::Hash, DBValue>,
	// root: &'db TrieHash<L>,
	// cache: Option<&'cache mut dyn TrieCache<L::Codec>>,
	// recorder: Option<&'cache mut dyn TrieRecorder<TrieHash<L>>>,
}

//	impl<'db, 'cache, L: TrieLayout> TrieDBBuilder<'db, 'cache, L> {
//		/// Create a new trie-db builder with the backing database `db` and `root`.
//		///
//		/// This doesn't check if `root` exists in the given `db`. If `root` doesn't exist it will fail
//		/// when trying to lookup any key.
//		#[inline]
//		pub fn new(db: &'db dyn HashDBRef<L::Hash, DBValue>, root: &'db TrieHash<L>) -> Self {
//			Self { db, root, cache: None, recorder: None }
//		}
func NewTrieDBBuilder[Hash comparable](db hashdb.HashDBRef[Hash, DBValue], root Hash) TrieDBBuilder[Hash] {
	return TrieDBBuilder[Hash]{}
}

// 	/// Use the given `cache` for the db.
// 	#[inline]
// 	pub fn with_cache(mut self, cache: &'cache mut dyn TrieCache<L::Codec>) -> Self {
// 		self.cache = Some(cache);
// 		self
// 	}

// 	/// Use the given optional `cache` for the db.
// 	#[inline]
// 	pub fn with_optional_cache<'ocache: 'cache>(
// 		mut self,
// 		cache: Option<&'ocache mut dyn TrieCache<L::Codec>>,
// 	) -> Self {
// 		// Make the compiler happy by "converting" the lifetime
// 		self.cache = cache.map(|c| c as _);
// 		self
// 	}

// 	/// Use the given `recorder` to record trie accesses.
// 	#[inline]
// 	pub fn with_recorder(mut self, recorder: &'cache mut dyn TrieRecorder<TrieHash<L>>) -> Self {
// 		self.recorder = Some(recorder);
// 		self
// 	}

// 	/// Use the given optional `recorder` to record trie accesses.
// 	#[inline]
// 	pub fn with_optional_recorder<'recorder: 'cache>(
// 		mut self,
// 		recorder: Option<&'recorder mut dyn TrieRecorder<TrieHash<L>>>,
// 	) -> Self {
// 		// Make the compiler happy by "converting" the lifetime
// 		self.recorder = recorder.map(|r| r as _);
// 		self
// 	}

// /// Build the [`TrieDB`].
// #[inline]
// pub fn build(self) -> TrieDB<'db, 'cache, L> {
func (tdbb TrieDBBuilder[Hash]) Build() TrieDB[Hash] {
	//			TrieDB {
	//				db: self.db,
	//				root: self.root,
	//				cache: self.cache.map(core::cell::RefCell::new),
	//				recorder: self.recorder.map(core::cell::RefCell::new),
	//			}
	//		}
	//	}
	// TODO: recreate dot/state/tries.go in here
	panic("unimplemented")
}

// pub struct TrieDB<'db, 'cache, L>
type TrieDB[Hash any] struct {
	// where
	//
	//	L: TrieLayout,
	//
	//	{
	//		db: &'db dyn HashDBRef<L::Hash, DBValue>,
	//		root: &'db TrieHash<L>,
	//		cache: Option<core::cell::RefCell<&'cache mut dyn TrieCache<L::Codec>>>,
	//		recorder: Option<core::cell::RefCell<&'cache mut dyn TrieRecorder<TrieHash<L>>>>,
}

// / Returns the hash of the value for `key`.
// fn get_hash(&self, key: &[u8]) -> Result<Option<TrieHash<L>>, TrieHash<L>, CError<L>>;
func (tdb TrieDB[Hash]) GetHash(key []byte) (*Hash, error) {
	panic("unimpl")
}

// / What is the value of the given key in this trie?
//
//	fn get(&self, key: &[u8]) -> Result<Option<DBValue>, TrieHash<L>, CError<L>> {
//		self.get_with(key, |v: &[u8]| v.to_vec())
//	}
func (tdb TrieDB[Hash]) Get(key []byte) (*DBValue, error) {
	panic("unimpl")
}
