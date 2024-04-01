package triedb

import (
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/trie"
	"github.com/ChainSafe/gossamer/pkg/trie/db"
)

type TrieDB struct {
	rootHash common.Hash
	db       db.DBGetter
}

// NewTrieDB creates a new TrieDB using the given root and db
func NewTrieDB(rootHash common.Hash, db db.DBGetter) *TrieDB {
	return &TrieDB{
		rootHash: rootHash,
		db:       db,
	}
}

func (t *TrieDB) Hash() (common.Hash, error) {
	return t.rootHash, nil
}

func (t *TrieDB) MustHash() common.Hash {
	h, err := t.Hash()
	if err != nil {
		panic(err)
	}

	return h
}

func (t *TrieDB) Get(key []byte) []byte {
	panic("implement me")
}

func (t *TrieDB) GetKeysWithPrefix(prefix []byte) (keysLE [][]byte) {
	panic("implement me")
}

var _ trie.ReadOnlyTrie = (*TrieDB)(nil)
