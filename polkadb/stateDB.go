package polkadb

import (
	log "github.com/ChainSafe/log15"
)

// StateDB stores trie structure in an underlying Database
type StateDB struct {
	Db Database
}

// NewStateDB instantiates badgerDB instance for storing trie structure
func NewStateDB(dataDir string) (*StateDB, error) {
	db, err := NewBadgerDB(dataDir)
	if err != nil {
		log.Crit("error instantiating StateDB", "error", err)
		return nil, err
	}

	return &StateDB{
		db,
	}, nil
}

func (s *StateDB) Close() error {
	return s.Db.Close()
}

func (s *StateDB) Del(key []byte) error {
	return s.Db.Del(key)
}

func (s *StateDB) Get(key []byte) ([]byte, error) {
	return s.Db.Get(key)
}

func (s *StateDB) Has(key []byte) (bool, error) {
	return s.Db.Has(key)
}

func (s *StateDB) NewBatch() Batch {
	return &batchWriter{
		db: s.Db.(*BadgerDB),
		b:  make(map[string][]byte),
	}
}

func (s *StateDB) NewIterator() Iterable {
	return Iterable{}
}

func (s *StateDB) Path() string {
	return s.Db.(*BadgerDB).config.DataDir
}

func (s *StateDB) Put(key, value []byte) error {
	return s.Db.Put(key, value)
}
