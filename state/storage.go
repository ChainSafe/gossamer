package state

import (
	"github.com/ChainSafe/gossamer/common"
	"github.com/ChainSafe/gossamer/polkadb"
	"github.com/ChainSafe/gossamer/trie"
)

type storageState struct {
	trie trie.Trie
	db polkadb.StateDB
}

func newStorageState() *storageState {
	return &storageState{
		trie: trie.Trie{},
		db:   polkadb.StateDB{},
	}
}


func (s *storageState) ExistsStorage(key []byte) (bool, error) {
	return s.trie.Exists(key)
}

func (s *storageState) GetStorage(key []byte) ([]byte, error) {
	return s.trie.Get(key)
}

func (s *storageState) StorageRoot() (common.Hash, error) {
	return s.trie.Root()
}

func (s *storageState) EnumeratedTrieRoot(values [][]byte) {
	//TODO
}

func (s *storageState) SetStorage(key []byte, value []byte) error {
	return s.trie.Put(key, value)
}

func (s *storageState) ClearPrefix(prefix []byte) {

}

func (s *storageState) ClearStorage(key []byte) error {
	return s.trie.Delete(key)
}


//TODO: add child storage funcs
