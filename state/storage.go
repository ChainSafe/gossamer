package state

import (
	"github.com/ChainSafe/gossamer/common"
	"github.com/ChainSafe/gossamer/polkadb"
	"github.com/ChainSafe/gossamer/trie"
)

type StorageState struct {
	trie *trie.Trie
	Db   *polkadb.StateDB
}

func NewStorageState(dataDir string) (*StorageState, error) {
	stateDb, err := polkadb.NewStateDB(dataDir)
	if err != nil {
		return nil, err
	}
	return &StorageState{
		trie: &trie.Trie{},
		Db:   stateDb,
	}, nil
}

func (s *StorageState) ExistsStorage(key []byte) (bool, error) {
	val, err := s.trie.Get(key)
	return (val != nil), err
}

func (s *StorageState) GetStorage(key []byte) ([]byte, error) {
	return s.trie.Get(key)
}

func (s *StorageState) StorageRoot() (common.Hash, error) {
	return s.trie.Hash()
}

func (s *StorageState) EnumeratedTrieRoot(values [][]byte) {
	//TODO
}

func (s *StorageState) SetStorage(key []byte, value []byte) error {
	return s.trie.Put(key, value)
}

func (s *StorageState) ClearPrefix(prefix []byte) {
	// Implemented in ext_clear_prefix
}

func (s *StorageState) ClearStorage(key []byte) error {
	return s.trie.Delete(key)
}

//TODO: add child storage funcs
