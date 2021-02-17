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
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/genesis"
	rtstorage "github.com/ChainSafe/gossamer/lib/runtime/storage"
	"github.com/ChainSafe/gossamer/lib/trie"
)

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
	tries      map[common.Hash]*trie.Trie

	baseDB chaindb.Database
	db     chaindb.Database
	lock   sync.RWMutex

	// change notifiers
	changedLock   sync.RWMutex
	subscriptions map[byte]*StorageSubscription
}

// NewStorageState creates a new StorageState backed by the given trie and database located at basePath.
func NewStorageState(db chaindb.Database, blockState *BlockState, t *trie.Trie) (*StorageState, error) {
	if db == nil {
		return nil, fmt.Errorf("cannot have nil database")
	}

	if t == nil {
		return nil, fmt.Errorf("cannot have nil trie")
	}

	tries := make(map[common.Hash]*trie.Trie)
	tries[t.MustHash()] = t

	return &StorageState{
		blockState:    blockState,
		tries:         tries,
		baseDB:        db,
		db:            chaindb.NewTable(db, storagePrefix),
		subscriptions: make(map[byte]*StorageSubscription),
	}, nil
}

func (s *StorageState) pruneKey(keyHeader *types.Header) {
	s.lock.Lock()
	defer s.lock.Unlock()

	_, ok := s.tries[keyHeader.StateRoot]
	if !ok {
		return
	}

	delete(s.tries, keyHeader.StateRoot)
	// TODO: database pruning needs to be refactored since the trie is now stored by nodes
}

// StoreTrie stores the given trie in the StorageState and writes it to the database
func (s *StorageState) StoreTrie(ts *rtstorage.TrieState) error {
	s.lock.Lock()
	root := ts.MustRoot()
	s.tries[root] = ts.Trie()
	s.lock.Unlock()

	logger.Trace("cached trie in storage state", "root", root)

	if err := ts.Trie().WriteDirty(s.db); err != nil {
		logger.Warn("failed to write trie to database", "root", root, "error", err)
		return err
	}

	go func() {
		if err := s.notifyStorageSubscriptions(root); err != nil {
			logger.Warn("failed to notify storage subscriptions", "error", err)
		}
	}()
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

	s.lock.RLock()
	defer s.lock.RUnlock()

	if s.tries[*root] != nil {
		return rtstorage.NewTrieState(s.tries[*root])
	}

	tr, err := s.LoadFromDB(*root)
	if err != nil {
		return nil, err
	}

	return rtstorage.NewTrieState(tr)
}

// StoreInDB encodes the entire trie and writes it to the DB
// The key to the DB entry is the root hash of the trie
func (s *StorageState) notifyStorageSubscriptions(root common.Hash) error {
	s.lock.RLock()
	t := s.tries[root]
	s.lock.RUnlock()

	if t == nil {
		return errTrieDoesNotExist(root)
	}

	// notify subscribers of database changes
	s.changedLock.Lock()
	defer s.changedLock.Unlock()

	for _, sub := range s.subscriptions {
		subRes := &SubscriptionResult{
			Hash: root,
		}
		if len(sub.Filter) == 0 {
			// no filter, so send all changes
			ent := t.Entries()
			for k, v := range ent {
				if k != ":code" {
					// todo, currently we're ignoring :code since this is a lot of data
					kv := &KeyValue{
						Key:   common.MustHexToBytes(fmt.Sprintf("0x%x", k)),
						Value: v,
					}
					subRes.Changes = append(subRes.Changes, *kv)
				}
			}
		} else {
			// filter result to include only interested keys
			for k := range sub.Filter {
				value, err := t.Get(common.MustHexToBytes(k))
				if err != nil {
					logger.Error("Error retrieving value from state tries")
					continue
				}
				kv := &KeyValue{
					Key:   common.MustHexToBytes(k),
					Value: value,
				}
				subRes.Changes = append(subRes.Changes, *kv)
			}
		}
		s.notifyChanged(subRes)
	}

	return nil
}

// LoadFromDB loads an encoded trie from the DB where the key is `root`
func (s *StorageState) LoadFromDB(root common.Hash) (*trie.Trie, error) {
	t := trie.NewEmptyTrie()
	err := LoadTrie(s.db, t, root)
	if err != nil {
		return nil, err
	}

	s.lock.Lock()
	defer s.lock.Unlock()

	s.tries[t.MustHash()] = t
	return t, nil
}

// ExistsStorage check if the key exists in the storage trie with the given storage hash
// If no hash is provided, the current chain head is used
func (s *StorageState) ExistsStorage(hash *common.Hash, key []byte) (bool, error) {
	if hash == nil {
		sr, err := s.blockState.BestBlockStateRoot()
		if err != nil {
			return false, err
		}
		hash = &sr
	}

	s.lock.RLock()
	defer s.lock.RUnlock()
	val, err := s.tries[*hash].Get(key)
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

	s.lock.RLock()
	defer s.lock.RUnlock()

	if s.tries[*root] != nil {
		return s.tries[*root].Get(key)
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
	sr, err := s.blockState.BestBlockStateRoot()
	if err != nil {
		return common.Hash{}, err
	}

	s.lock.RLock()
	defer s.lock.RUnlock()

	if s.tries[sr] == nil {
		return common.Hash{}, errTrieDoesNotExist(sr)
	}

	return s.tries[sr].Hash()
}

// EnumeratedTrieRoot not implemented
func (s *StorageState) EnumeratedTrieRoot(values [][]byte) {
	//TODO
	panic("not implemented")
}

// Entries returns Entries from the trie with the given state root
func (s *StorageState) Entries(root *common.Hash) (map[string][]byte, error) {
	if root == nil {
		head, err := s.blockState.BestBlockStateRoot()
		if err != nil {
			return nil, err
		}
		root = &head
	}

	s.lock.RLock()
	defer s.lock.RUnlock()

	if s.tries[*root] != nil {
		return s.tries[*root].Entries(), nil
	}

	tr, err := s.LoadFromDB(*root)
	if err != nil {
		return nil, err
	}

	return tr.Entries(), nil
}

// GetKeysWithPrefix returns all that match the given prefix for the given hash (or best block state root if hash is nil) in lexicographic order
func (s *StorageState) GetKeysWithPrefix(hash *common.Hash, prefix []byte) ([][]byte, error) {
	if hash == nil {
		sr, err := s.blockState.BestBlockStateRoot()
		if err != nil {
			return nil, err
		}
		hash = &sr
	}
	t := s.tries[*hash]
	if t == nil {
		return nil, fmt.Errorf("unable to retrieve trie with hash %x", *hash)
	}
	return t.GetKeysWithPrefix(prefix), nil
}

// GetStorageChild return GetChild from the trie
func (s *StorageState) GetStorageChild(hash *common.Hash, keyToChild []byte) (*trie.Trie, error) {
	if hash == nil {
		sr, err := s.blockState.BestBlockStateRoot()
		if err != nil {
			return nil, err
		}
		hash = &sr
	}

	s.lock.RLock()
	defer s.lock.RUnlock()

	if s.tries[*hash] == nil {
		return nil, errTrieDoesNotExist(*hash)
	}

	return s.tries[*hash].GetChild(keyToChild)
}

// GetStorageFromChild return GetFromChild from the trie
func (s *StorageState) GetStorageFromChild(hash *common.Hash, keyToChild, key []byte) ([]byte, error) {
	if hash == nil {
		sr, err := s.blockState.BestBlockStateRoot()
		if err != nil {
			return nil, err
		}
		hash = &sr
	}

	s.lock.RLock()
	defer s.lock.RUnlock()

	if s.tries[*hash] == nil {
		return nil, errTrieDoesNotExist(*hash)
	}
	return s.tries[*hash].GetFromChild(keyToChild, key)
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

// GetGenesisData retrieves current genesis data from database
func (s *StorageState) GetGenesisData() (*genesis.Data, error) {
	return LoadGenesisData(s.baseDB)
}
