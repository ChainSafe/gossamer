// Copyright 2019 ChainSafe Systems (ON) Corp.
// This file is part of gossamer.
//
// The gossamer library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The gossamer library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the gossamer library. If not, see <http://www.gnu.org/licenses/>.

package state

import (
	"encoding/binary"
	"errors"
	"fmt"
	"sync"

	"github.com/ChainSafe/chaindb"
	"github.com/ChainSafe/gossamer/dot/state/pruner"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	rtstorage "github.com/ChainSafe/gossamer/lib/runtime/storage"
	"github.com/ChainSafe/gossamer/lib/trie"
)

// storagePrefix storage key prefix.
var storagePrefix = "storage"
var codeKey = common.CodeKey

// ErrTrieDoesNotExist is returned when attempting to interact with a trie that is not stored in the StorageState
var ErrTrieDoesNotExist = errors.New("trie with given root does not exist")

func errTrieDoesNotExist(hash common.Hash) error {
	return fmt.Errorf("%w: %s", ErrTrieDoesNotExist, hash)
}

// StorageState is the struct that holds the trie, db and lock
type StorageState struct {
	blockState *BlockState
	tries      *sync.Map // map[common.Hash]*trie.Trie // map of root -> trie

	db chaindb.Database
	sync.RWMutex

	// change notifiers
	changedLock  sync.RWMutex
	observerList []Observer
	pruner       pruner.Pruner
	syncing      bool
}

// NewStorageState creates a new StorageState backed by the given trie and database located at basePath.
func NewStorageState(db chaindb.Database, blockState *BlockState, t *trie.Trie, onlinePruner pruner.Config) (*StorageState, error) {
	if db == nil {
		return nil, fmt.Errorf("cannot have nil database")
	}

	if t == nil {
		return nil, fmt.Errorf("cannot have nil trie")
	}

	tries := new(sync.Map)
	tries.Store(t.MustHash(), t)

	storageTable := chaindb.NewTable(db, storagePrefix)

	var p pruner.Pruner
	if onlinePruner.Mode == pruner.Full {
		var err error
		p, err = pruner.NewFullNode(db, storageTable, onlinePruner.RetainedBlocks, logger)
		if err != nil {
			return nil, err
		}
	} else {
		p = &pruner.ArchiveNode{}
	}

	return &StorageState{
		blockState:   blockState,
		tries:        tries,
		db:           storageTable,
		observerList: []Observer{},
		pruner:       p,
	}, nil
}

// SetSyncing sets whether the node is currently syncing or not
func (s *StorageState) SetSyncing(syncing bool) {
	s.syncing = syncing
}

func (s *StorageState) pruneKey(keyHeader *types.Header) {
	s.tries.Delete(keyHeader.StateRoot)
}

// StoreTrie stores the given trie in the StorageState and writes it to the database
func (s *StorageState) StoreTrie(ts *rtstorage.TrieState, header *types.Header) error {
	root := ts.MustRoot()

	if s.syncing {
		// keep only the trie at the head of the chain when syncing
		// TODO: probably remove this when memory usage improves
		s.tries.Range(func(k, _ interface{}) bool {
			s.tries.Delete(k)
			return true
		})
	}

	_, _ = s.tries.LoadOrStore(root, ts.Trie())

	if _, ok := s.pruner.(*pruner.FullNode); header == nil && ok {
		return fmt.Errorf("block cannot be empty for Full node pruner")
	}

	if header != nil {
		insKeys, err := ts.GetInsertedNodeHashes()
		if err != nil {
			return fmt.Errorf("failed to get state trie inserted keys: block %s %w", header.Hash(), err)
		}

		delKeys := ts.GetDeletedNodeHashes()
		err = s.pruner.StoreJournalRecord(delKeys, insKeys, header.Hash(), header.Number.Int64())
		if err != nil {
			return err
		}
	}

	logger.Trace("cached trie in storage state", "root", root)

	if err := ts.Trie().WriteDirty(s.db); err != nil {
		logger.Warn("failed to write trie to database", "root", root, "error", err)
		return err
	}

	go s.notifyAll(root)
	return nil
}

// TrieState returns the TrieState for a given state root.
// If no state root is provided, it returns the TrieState for the current chain head.
func (s *StorageState) TrieState(root *common.Hash) (*rtstorage.TrieState, error) {
	if root == nil {
		sr, err := s.blockState.BestBlockStateRoot()
		if err != nil {
			return nil, err
		}
		root = &sr
	}

	st, has := s.tries.Load(*root)
	if !has {
		var err error
		st, err = s.LoadFromDB(*root)
		if err != nil {
			return nil, err
		}

		_, _ = s.tries.LoadOrStore(*root, st)
	}

	t := st.(*trie.Trie)

	if has && t.MustHash() != *root {
		panic("trie does not have expected root")
	}

	nextTrie := t.Snapshot()
	next, err := rtstorage.NewTrieState(nextTrie)
	if err != nil {
		return nil, err
	}

	logger.Trace("returning trie to be modified", "root", root)
	return next, nil
}

// LoadFromDB loads an encoded trie from the DB where the key is `root`
func (s *StorageState) LoadFromDB(root common.Hash) (*trie.Trie, error) {
	t := trie.NewEmptyTrie()
	err := t.Load(s.db, root)
	if err != nil {
		return nil, err
	}

	_, _ = s.tries.LoadOrStore(t.MustHash(), t)
	return t, nil
}

func (s *StorageState) loadTrie(root *common.Hash) (*trie.Trie, error) {
	if root == nil {
		sr, err := s.blockState.BestBlockStateRoot()
		if err != nil {
			return nil, err
		}
		root = &sr
	}

	if t, has := s.tries.Load(*root); has && t != nil {
		return t.(*trie.Trie), nil
	}

	tr, err := s.LoadFromDB(*root)
	if err != nil {
		return nil, errTrieDoesNotExist(*root)
	}

	return tr, nil
}

// ExistsStorage check if the key exists in the storage trie with the given storage hash
// If no hash is provided, the current chain head is used
func (s *StorageState) ExistsStorage(root *common.Hash, key []byte) (bool, error) {
	val, err := s.GetStorage(root, key)
	return val != nil, err
}

// GetStorage gets the object from the trie using the given key and storage hash
// If no hash is provided, the current chain head is used
func (s *StorageState) GetStorage(root *common.Hash, key []byte) ([]byte, error) {
	if root == nil {
		sr, err := s.blockState.BestBlockStateRoot()
		if err != nil {
			return nil, err
		}
		root = &sr
	}

	if t, has := s.tries.Load(*root); has {
		val := t.(*trie.Trie).Get(key)
		return val, nil
	}

	return trie.GetFromDB(s.db, *root, key)
}

// GetStorageByBlockHash returns the value at the given key at the given block hash
func (s *StorageState) GetStorageByBlockHash(bhash common.Hash, key []byte) ([]byte, error) {
	header, err := s.blockState.GetHeader(bhash)
	if err != nil {
		return nil, err
	}

	return s.GetStorage(&header.StateRoot, key)
}

// GetStateRootFromBlock returns the state root hash of a given block hash
func (s *StorageState) GetStateRootFromBlock(bhash *common.Hash) (*common.Hash, error) {
	header, err := s.blockState.GetHeader(*bhash)
	if err != nil {
		return nil, err
	}
	return &header.StateRoot, nil
}

// StorageRoot returns the root hash of the current storage trie
func (s *StorageState) StorageRoot() (common.Hash, error) {
	return s.blockState.BestBlockStateRoot()
}

// EnumeratedTrieRoot not implemented
func (s *StorageState) EnumeratedTrieRoot(values [][]byte) {
	//TODO
	panic("not implemented")
}

// Entries returns Entries from the trie with the given state root
func (s *StorageState) Entries(root *common.Hash) (map[string][]byte, error) {
	tr, err := s.loadTrie(root)
	if err != nil {
		return nil, err
	}

	return tr.Entries(), nil
}

// GetKeysWithPrefix returns all that match the given prefix for the given hash (or best block state root if hash is nil) in lexicographic order
func (s *StorageState) GetKeysWithPrefix(root *common.Hash, prefix []byte) ([][]byte, error) {
	tr, err := s.loadTrie(root)
	if err != nil {
		return nil, err
	}

	return tr.GetKeysWithPrefix(prefix), nil
}

// GetStorageChild returns a child trie, if it exists
func (s *StorageState) GetStorageChild(root *common.Hash, keyToChild []byte) (*trie.Trie, error) {
	tr, err := s.loadTrie(root)
	if err != nil {
		return nil, err
	}

	return tr.GetChild(keyToChild)
}

// GetStorageFromChild get a value from a child trie
func (s *StorageState) GetStorageFromChild(root *common.Hash, keyToChild, key []byte) ([]byte, error) {
	tr, err := s.loadTrie(root)
	if err != nil {
		return nil, err
	}

	return tr.GetFromChild(keyToChild, key)
}

// LoadCode returns the runtime code (located at :code)
func (s *StorageState) LoadCode(hash *common.Hash) ([]byte, error) {
	return s.GetStorage(hash, codeKey)
}

// LoadCodeHash returns the hash of the runtime code (located at :code)
func (s *StorageState) LoadCodeHash(hash *common.Hash) (common.Hash, error) {
	code, err := s.LoadCode(hash)
	if err != nil {
		return common.NewHash([]byte{}), err
	}

	return common.Blake2bHash(code)
}

// GetBalance gets the balance for an account with the given public key
func (s *StorageState) GetBalance(hash *common.Hash, key [32]byte) (uint64, error) {
	skey, err := common.BalanceKey(key)
	if err != nil {
		return 0, err
	}

	bal, err := s.GetStorage(hash, skey)
	if err != nil {
		return 0, err
	}

	if len(bal) != 8 {
		return 0, nil
	}

	return binary.LittleEndian.Uint64(bal), nil
}

func (s *StorageState) pruneStorage(closeCh chan interface{}) {
	for {
		select {
		case key := <-s.blockState.pruneKeyCh:
			s.pruneKey(key)
		case <-closeCh:
			return
		}
	}
}
