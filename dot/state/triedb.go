package state

import (
	"github.com/ChainSafe/gossamer/internal/database"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/trie"
)

type TrieDB struct {
	db    database.Table
	tries *Tries
}

// NewTrieDB creates a new TrieDB
// db is expected to be a table of the database, see `database.NewTable` for more details
func NewTrieDB(db database.Table, tries *Tries) *TrieDB {
	return &TrieDB{
		db:    db,
		tries: tries,
	}
}

func (tdb *TrieDB) deleteCached(root common.Hash) {
	tdb.tries.delete(root)
}

func (tdb *TrieDB) Delete(root common.Hash) error {
	tdb.tries.delete(root)
	return tdb.db.Del(root.ToBytes())
}

func (tdb *TrieDB) Put(t *trie.Trie) error {
	return t.WriteDirty(tdb.db)
}

func (tdb *TrieDB) Get(root common.Hash) (*trie.Trie, error) {
	// Get trie from memory
	t := tdb.tries.get(root)

	// If it doesn't exist, get it from the database and set it in memory
	if t == nil {
		var err error
		t, err = tdb.getFromDB(root)
		if err != nil {
			return nil, err
		}

		tdb.tries.softSet(root, t)
	}

	return t, nil
}

func (tdb *TrieDB) getFromDB(root common.Hash) (*trie.Trie, error) {
	t := trie.NewEmptyTrie()
	err := t.Load(tdb.db, root)
	if err != nil {
		return nil, err
	}

	return t, nil
}

func (tdb *TrieDB) GetKey(root common.Hash, key []byte) ([]byte, error) {
	t, err := tdb.Get(root)
	if err != nil {
		return nil, err
	}

	return t.Get(key), nil
}
