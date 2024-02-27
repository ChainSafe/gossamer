package trie

import (
	hashdb "github.com/ChainSafe/gossamer/internal/hash-db"
	triedb "github.com/ChainSafe/gossamer/internal/trie-db"
)

// / Reexport from `hash_db`, with genericity set for `Hasher` trait.
// / This uses a noops `KeyFunction` (key addressing must be hashed or using
// / an encoding scheme that avoid key conflict).
// pub type MemoryDB<H> = memory_db::MemoryDB<H, memory_db::HashKey<H>, trie_db::DBValue>;
type PrefixedMemoryDB[Hasher any] struct{}

// pub type TrieHash<L> = <<L as TrieLayout>::Hash as Hasher>::Out;

// / Builder for creating a [`TrieDB`].
// pub type TrieDBBuilder<'a, 'cache, L> = trie_db::TrieDBBuilder<'a, 'cache, L>;
// type TrieDBBuilder[Hash, DB, Cache, Layout any] triedb.TrieDBBuilder[Hash, DB, Cache, Layout]

// / Read a value from the trie.
// pub fn read_trie_value<L: TrieLayout, DB: hash_db::HashDBRef<L::Hash, trie_db::DBValue>>(
//
//	db: &DB,
//	root: &TrieHash<L>,
//	key: &[u8],
//	recorder: Option<&mut dyn TrieRecorder<TrieHash<L>>>,
//	cache: Option<&mut dyn TrieCache<L::Codec>>,
//
// ) -> Result<Option<Vec<u8>>, Box<TrieError<L>>> {
func ReadTrieValue[TrieHash comparable](
	db hashdb.HashDBRef[TrieHash, triedb.DBValue],
	root TrieHash,
	key []byte,
	recorder triedb.TrieRecorder,
	cache triedb.TrieCache,
) (*[]byte, error) {
	//		TrieDBBuilder::<L>::new(db, root)
	//			.with_optional_cache(cache)
	//			.with_optional_recorder(recorder)
	//			.build()
	//			.get(key)
	//	}
	dbVal, err := triedb.NewTrieDBBuilder[TrieHash](root).Build().Get(key)
	if dbVal == nil {
		return nil, err
	}
	val := []byte(*dbVal)
	return &val, err
}
