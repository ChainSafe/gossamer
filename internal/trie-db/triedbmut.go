package triedb

import hashdb "github.com/ChainSafe/gossamer/internal/hash-db"

// / Value representation for Node.
// #[derive(Clone, Eq)]
// pub enum Value<L: TrieLayout> {
type Value any

// type ValueOptions interface {
// 	Inline | Node | NewNode
// }
// 	/// Value bytes inlined in a trie node.
// 	Inline(Bytes),
// 	/// Hash of the value.
// 	Node(TrieHash<L>),
// 	/// Hash of value bytes if calculated and value bytes.
// 	/// The hash may be undefined until it node is added
// 	/// to the db.
// 	NewNode(Option<TrieHash<L>>, Bytes),
// }

// / A builder for creating a [`TrieDBMut`].
// pub struct TrieDBMutBuilder<'db, L: TrieLayout> {
type TrieDBMutBuilder[Hash any] struct {
	// db: &'db mut dyn HashDB<L::Hash, DBValue>,
	// root: &'db mut TrieHash<L>,
	// cache: Option<&'db mut dyn TrieCache<L::Codec>>,
	// recorder: Option<&'db mut dyn TrieRecorder<TrieHash<L>>>,
}

// / Create a builder for constructing a new trie with the backing database `db` and empty
// / `root`.
func NewTrieDBMutBuilder[Hash comparable](
	db hashdb.HashDB[Hash, DBValue], root Hash,
) TrieDBMutBuilder[Hash] {
	return TrieDBMutBuilder[Hash]{}
}

// / Create a builder for constructing a new trie with the backing database `db` and `root`.
// /
// / This doesn't check if `root` exists in the given `db`. If `root` doesn't exist it will fail
// / when trying to lookup any key.
func NewTrieDBMutBuilderFromExisting[Hash comparable](
	db hashdb.HashDB[Hash, DBValue], root Hash,
) TrieDBMutBuilder[Hash] {
	return TrieDBMutBuilder[Hash]{}
}

func (tdbb TrieDBMutBuilder[Hash]) Build() TrieMut[Hash] {
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
	return nil
}
