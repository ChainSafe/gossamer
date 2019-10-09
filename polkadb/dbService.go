package polkadb

import (
	"path/filepath"

	"github.com/ChainSafe/gossamer/internal/services"
)

var _ services.Service = &DbService{}

// DbService contains both databases for service registry
type DbService struct {
	path string
	StateDB *StateDB
	BlockDB *BlockDB

	err chan<- error
}

// NewDatabaseService opens and returns a new DB object
func NewDatabaseService(path string) (*DbService, error) {
	return &DbService{
		path: path,
		StateDB: nil,
		BlockDB: nil,
		err:     nil,
	}, nil
}

// Start...
func (s *DbService) Start() <-chan error {
	ch := make(chan error)
	s.err = ch
	stateDataDir := filepath.Join(s.path, "state")
	blockDataDir := filepath.Join(s.path, "block")

	stateDb, err := NewStateDB(stateDataDir)
	if err != nil {
		s.err <- err
	}

	blockDb, err := NewBlockDB(blockDataDir)
	if err != nil {
		s.err <- err
	}

	s.BlockDB = blockDb
	s.StateDB = stateDb

	return ch
}

// Stop kills running BlockDB and StateDB instances
func (s *DbService) Stop() <-chan error {
	e := make(chan error)
	// Closing Badger Databases
	err := s.StateDB.Db.Close()
	if err != nil {
		e <- err
	}

	err = s.BlockDB.Db.Close()
	if err != nil {
		e <- err
	}
	return e
}

