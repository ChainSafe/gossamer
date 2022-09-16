// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package state

import (
	"errors"
	"fmt"
	"sync"

	"github.com/ChainSafe/chaindb"
	"github.com/ChainSafe/gossamer/dot/state/pruner"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	rtstorage "github.com/ChainSafe/gossamer/lib/runtime/storage"
	"github.com/ChainSafe/gossamer/lib/trie"
	"github.com/ChainSafe/gossamer/lib/trie/proof"
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
	tries      *Tries

	db chaindb.Database
	sync.RWMutex

	// change notifiers
	observerListMutex sync.RWMutex
	observerList      []Observer
	pruner            pruner.Pruner
}

// NewStorageState creates a new StorageState backed by the given block state
// and database located at basePath.
func NewStorageState(db chaindb.Database, blockState *BlockState,
	tries *Tries, onlinePruner pruner.Config) (*StorageState, error) {
	if db == nil {
		return nil, fmt.Errorf("cannot have nil database")
	}

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

// StoreTrie stores the given trie in the StorageState and writes it to the database
func (s *StorageState) StoreTrie(ts *rtstorage.TrieState, header *types.Header,
	stateVersion trie.Version) error {
	root := ts.MustRoot(stateVersion)

	s.tries.softSet(root, ts.Trie())

	if header == nil {
		if _, ok := s.pruner.(*pruner.FullNode); ok {
			panic("block header cannot be empty for Full node pruner")
		}
	}

	if header != nil {
		insertedNodeHashes, err := ts.GetInsertedNodeHashes()
		if err != nil {
			return fmt.Errorf("failed to get state trie inserted keys: block %s %w", header.Hash(), err)
		}

		deletedNodeHashes := ts.GetDeletedNodeHashes()
		err = s.pruner.StoreJournalRecord(deletedNodeHashes, insertedNodeHashes, header.Hash(), int64(header.Number))
		if err != nil {
			return err
		}
	}

	logger.Tracef("cached trie in storage state: %s", root)

	if err := ts.Trie().WriteDirty(s.db); err != nil {
		logger.Warnf("failed to write trie with root %s to database: %s", root, err)
		return err
	}

	go s.notifyAll(root, stateVersion)
	return nil
}

// TrieState returns the TrieState for a given state root.
// If no state root is provided, it returns the TrieState for the current chain head.
func (s *StorageState) TrieState(root *common.Hash, version trie.Version) (*rtstorage.TrieState, error) {
	if root == nil {
		sr, err := s.blockState.BestBlockStateRoot()
		if err != nil {
			return nil, err
		}
		root = &sr
	}

	t := s.tries.get(*root)
	if t == nil {
		var err error
		t, err = s.LoadFromDB(*root, version)
		if err != nil {
			return nil, err
		}

		s.tries.softSet(*root, t)
	} else if t.MustHash(version) != *root {
		panic("trie does not have expected root")
	}

	nextTrie := t.Snapshot()
	next := rtstorage.NewTrieState(nextTrie)

	logger.Tracef("returning trie with root %s to be modified", root)
	return next, nil
}

// LoadFromDB loads an encoded trie from the DB where the key is `root`
func (s *StorageState) LoadFromDB(root common.Hash, version trie.Version) (*trie.Trie, error) {
	t := trie.NewEmptyTrie()
	err := t.Load(s.db, root, version)
	if err != nil {
		return nil, err
	}

	s.tries.softSet(t.MustHash(version), t)
	return t, nil
}

func (s *StorageState) loadTrie(root *common.Hash, version trie.Version) (*trie.Trie, error) {
	if root == nil {
		sr, err := s.blockState.BestBlockStateRoot()
		if err != nil {
			return nil, err
		}
		root = &sr
	}

	t := s.tries.get(*root)
	if t != nil {
		return t, nil
	}

	tr, err := s.LoadFromDB(*root, version)
	if err != nil {
		return nil, fmt.Errorf("trie does not exist at root %s: %w", *root, err)
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

	t := s.tries.get(*root)
	if t != nil {
		val := t.Get(key)
		return val, nil
	}

	return trie.GetFromDB(s.db, *root, key)
}

// GetStorageByBlockHash returns the value at the given key at the given block hash
func (s *StorageState) GetStorageByBlockHash(bhash *common.Hash, key []byte) ([]byte, error) {
	var (
		root common.Hash
		err  error
	)

	if bhash != nil {
		header, err := s.blockState.GetHeader(*bhash)
		if err != nil {
			return nil, err
		}

		root = header.StateRoot
	} else {
		root, err = s.StorageRoot()
		if err != nil {
			return nil, err
		}
	}

	return s.GetStorage(&root, key)
}

// GetStateRootFromBlock returns the state root hash of a given block hash
func (s *StorageState) GetStateRootFromBlock(bhash *common.Hash) (*common.Hash, error) {
	if bhash == nil {
		b := s.blockState.BestBlockHash()
		bhash = &b
	}

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

// Entries returns the entries from the trie corresponding to the given state
// root as a map of key (string of LE encoded bytes) to value byte slice.
func (s *StorageState) Entries(root *common.Hash, version trie.Version) (
	entries map[string][]byte, err error) {
	tr, err := s.loadTrie(root, version)
	if err != nil {
		return nil, err
	}

	return tr.Entries(), nil
}

// GetKeysWithPrefix returns all that match the given prefix for the given hash
// (or best block state root if hash is nil) in lexicographic order
func (s *StorageState) GetKeysWithPrefix(root *common.Hash, prefix []byte,
	version trie.Version) ([][]byte, error) {
	tr, err := s.loadTrie(root, version)
	if err != nil {
		return nil, err
	}

	return tr.GetKeysWithPrefix(prefix), nil
}

// GetStorageChild returns a child trie, if it exists
func (s *StorageState) GetStorageChild(root *common.Hash, keyToChild []byte,
	version trie.Version) (*trie.Trie, error) {
	tr, err := s.loadTrie(root, version)
	if err != nil {
		return nil, err
	}

	return tr.GetChild(keyToChild)
}

// GetStorageFromChild get a value from a child trie
func (s *StorageState) GetStorageFromChild(root *common.Hash, keyToChild,
	key []byte, version trie.Version) (value []byte, err error) {
	tr, err := s.loadTrie(root, version)
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

// GenerateTrieProof returns the proofs related to the keys on the state root trie
func (s *StorageState) GenerateTrieProof(stateRoot common.Hash, keys [][]byte,
	version trie.Version) (encodedProofNodes [][]byte, err error) {
	return proof.Generate(stateRoot[:], keys, s.db, version)
}
