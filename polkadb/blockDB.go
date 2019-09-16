package polkadb

import (
	log "github.com/ChainSafe/log15"
)

// BlockDB contains badger.DB instance
type BlockDB struct {
	Db Database
}

// NewBlockDB instantiates BlockDB for storing relevant BlockData
func NewBlockDB(dataDir string) (*BlockDB, error) {
	db, err := NewBadgerService(dataDir)
	if err != nil {
		log.Crit("error instantiating BlockDB", "error", err)
		return nil, err
	}

	return &BlockDB{
		db,
	}, nil
}
