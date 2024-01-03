package triedb

type Trie[Hash HashOut] interface {
	Root() Hash
	IsEmpty() bool
	Contains(key []byte) (bool, error)
	GetHash(key []byte) (*Hash, error)
	Get(key []byte) (*DBValue, error)
	//TODO:
	//get_with
	//lookup_first_descendant
}
