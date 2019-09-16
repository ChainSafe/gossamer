package polkadb

import (
	log "github.com/ChainSafe/log15"
)

// StateDB contains badger.DB instance
type StateDB struct {
	Db Database
}

// NewStateDB instantiates StateDB for trie structure
func NewStateDB(dataDir string) (*StateDB, error) {
	db, err := NewBadgerService(dataDir)
	if err != nil {
		log.Crit("error instantiating StateDB", "error", err)
		return nil, err
	}

	return &StateDB{
		db,
	}, nil
}
