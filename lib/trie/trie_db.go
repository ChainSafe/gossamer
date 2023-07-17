package trie

import "github.com/ChainSafe/gossamer/lib/common"

type Prefix struct {
	data   []byte
	padded *[]byte
}

type HashDB interface {
	DBGetter
	GetWithPrefix(key []byte, prefix Prefix) (value []byte, err error)
	Insert(prefix Prefix, value []byte) common.Hash
}

type TrieDBBuilder struct {
	db     HashDB
	root   []byte
	layout TrieLayout
	//TODO: implement cache and recorder
}

func NewTrieDBBuilder(db HashDB, root []byte, layout TrieLayout) *TrieDBBuilder {
	return &TrieDBBuilder{db, root, layout}
}

func (tdbb TrieDBBuilder) build() *TrieDB {
	return &TrieDB{tdbb.db, tdbb.root, tdbb.layout}
}

type TrieDB struct {
	db     HashDB
	root   []byte
	layout TrieLayout
	//TODO: implement cache and recorder
}

func (tdb TrieDB) GetValue(key []byte) (*[]byte, error) {
	return NewLookup(tdb.db, tdb.root).Lookup(key, NewNibbleSlice(key))
}
