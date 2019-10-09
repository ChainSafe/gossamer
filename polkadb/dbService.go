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
}

// NewDbService opens and returns a new DB object
func NewDbService(path string) (*DbService, error) {
	return &DbService{
		path: path,
		StateDB: nil,
		BlockDB: nil,
	}, nil
}

// Start...
func (s *DbService) Start() error {

	stateDataDir := filepath.Join(s.path, "state")
	blockDataDir := filepath.Join(s.path, "block")

	stateDb, err := NewStateDB(stateDataDir)
	if err != nil {
		return err
	}

	blockDb, err := NewBlockDB(blockDataDir)
	if err != nil {
		return err
	}

	s.BlockDB = blockDb
	s.StateDB = stateDb

	return nil
}

// Stop kills running BlockDB and StateDB instances
func (s *DbService) Stop() error {
	// Closing Badger Databases
	err := s.StateDB.Db.Close()
	if err != nil {
		return err
	}

	err = s.BlockDB.Db.Close()
	if err != nil {
		return err
	}
	return nil
}

