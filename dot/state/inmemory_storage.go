// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package state

import (
	"errors"
	"fmt"
	"sync"

	"github.com/ChainSafe/gossamer/dot/state/pruner"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/internal/database"
	"github.com/ChainSafe/gossamer/lib/common"
	inmemory_storage "github.com/ChainSafe/gossamer/lib/runtime/storage/inmemory"
	"github.com/ChainSafe/gossamer/pkg/trie"
	"github.com/ChainSafe/gossamer/pkg/trie/proof"
)

// storagePrefix storage key prefix.
var storagePrefix = "storage"
var codeKey = common.CodeKey

// ErrTrieDoesNotExist is returned when attempting to interact with a trie that is not stored in the StorageState
var ErrTrieDoesNotExist = errors.New("trie with given root does not exist")

func errTrieDoesNotExist(hash common.Hash) error {
	return fmt.Errorf("%w: %s", ErrTrieDoesNotExist, hash)
}

// InmemoryStorageState is the struct that holds the trie, db and lock
type InmemoryStorageState struct {
	blockState *BlockState
	tries      *Tries

	db GetterPutterNewBatcher
	sync.RWMutex

	// change notifiers
	observerListMutex sync.RWMutex
	observerList      []Observer
	pruner            pruner.Pruner
}

// NewStorageState creates a new StorageState backed by the given block state
// and database located at basePath.
func NewStorageState(db database.Database, blockState *BlockState,
	tries *Tries) (*InmemoryStorageState, error) {
	storageTable := database.NewTable(db, storagePrefix)

	return &InmemoryStorageState{
		blockState:   blockState,
		tries:        tries,
		db:           storageTable,
		observerList: []Observer{},
		pruner:       &pruner.ArchiveNode{},
	}, nil
}

// StoreTrie stores the given trie in the StorageState and writes it to the database
func (s *InmemoryStorageState) StoreTrie(ts *inmemory_storage.InMemoryTrieState, header *types.Header) error {
	root := ts.MustRoot()
	s.tries.softSet(root, ts.Trie())

	if header != nil {
		insertedNodeHashes, deletedNodeHashes, err := ts.GetChangedNodeHashes()
		if err != nil {
			return fmt.Errorf("getting trie changed node hashes for block hash %s: %w", header.Hash(), err)
		}

		err = s.pruner.StoreJournalRecord(deletedNodeHashes, insertedNodeHashes, header.Hash(), int64(header.Number))
		if err != nil {
			return fmt.Errorf("storing journal record: %w", err)
		}
	}

	logger.Tracef("cached trie in storage state: %s", root)

	if err := ts.Trie().WriteDirty(s.db); err != nil {
		logger.Warnf("failed to write trie with root %s to database: %s", root, err)
		return err
	}

	go s.notifyAll(root)
	return nil
}

// TrieState returns the TrieState for a given state root.
// If no state root is provided, it returns the TrieState for the current chain head.
func (s *InmemoryStorageState) TrieState(root *common.Hash) (*inmemory_storage.InMemoryTrieState, error) {
	if root == nil {
		sr, err := s.blockState.BestBlockStateRoot()
		if err != nil {
			return nil, fmt.Errorf("while getting best block state root: %w", err)
		}
		root = &sr
	}

	t := s.tries.get(*root)
	if t == nil {
		var err error
		t, err = s.LoadFromDB(*root)
		if err != nil {
			return nil, fmt.Errorf("while loading from database: %w", err)
		}

		s.tries.softSet(*root, t)
	} else if t.MustHash() != *root {
		panic("trie does not have expected root")
	}

	nextTrie := t.Snapshot()
	next := inmemory_storage.NewTrieState(nextTrie)

	logger.Tracef("returning trie with root %s to be modified", root)
	return next, nil
}

// LoadFromDB loads an encoded trie from the DB where the key is `root`
func (s *InmemoryStorageState) LoadFromDB(root common.Hash) (*trie.InMemoryTrie, error) {
	t := trie.NewInMemoryTrie(nil, s.db)
	err := t.Load(s.db, root)
	if err != nil {
		return nil, err
	}

	s.tries.softSet(t.MustHash(), t)
	return t, nil
}

func (s *InmemoryStorageState) loadTrie(root *common.Hash) (*trie.InMemoryTrie, error) {
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

	tr, err := s.LoadFromDB(*root)
	if err != nil {
		return nil, fmt.Errorf("trie does not exist at root %s: %w", *root, err)
	}

	return tr, nil
}

// ExistsStorage check if the key exists in the storage trie with the given storage hash
// If no hash is provided, the current chain head is used
func (s *InmemoryStorageState) ExistsStorage(root *common.Hash, key []byte) (bool, error) {
	val, err := s.GetStorage(root, key)
	return val != nil, err
}

// GetStorage gets the object from the trie using the given key and storage hash
// If no hash is provided, the current chain head is used
func (s *InmemoryStorageState) GetStorage(root *common.Hash, key []byte) ([]byte, error) {
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
func (s *InmemoryStorageState) GetStorageByBlockHash(bhash *common.Hash, key []byte) ([]byte, error) {
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
func (s *InmemoryStorageState) GetStateRootFromBlock(bhash *common.Hash) (*common.Hash, error) {
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
func (s *InmemoryStorageState) StorageRoot() (common.Hash, error) {
	return s.blockState.BestBlockStateRoot()
}

// Entries returns Entries from the trie with the given state root
func (s *InmemoryStorageState) Entries(root *common.Hash) (map[string][]byte, error) {
	tr, err := s.loadTrie(root)
	if err != nil {
		return nil, err
	}

	return tr.Entries(), nil
}

// GetKeysWithPrefix returns all that match the given prefix for the given hash
// (or best block state root if hash is nil) in lexicographic order
func (s *InmemoryStorageState) GetKeysWithPrefix(root *common.Hash, prefix []byte) ([][]byte, error) {
	tr, err := s.loadTrie(root)
	if err != nil {
		return nil, err
	}

	return tr.GetKeysWithPrefix(prefix), nil
}

// GetStorageChild returns a child trie, if it exists
func (s *InmemoryStorageState) GetStorageChild(root *common.Hash, keyToChild []byte) (trie.Trie, error) {
	tr, err := s.loadTrie(root)
	if err != nil {
		return nil, err
	}

	return tr.GetChild(keyToChild)
}

// GetStorageFromChild get a value from a child trie
func (s *InmemoryStorageState) GetStorageFromChild(root *common.Hash, keyToChild, key []byte) ([]byte, error) {
	tr, err := s.loadTrie(root)
	if err != nil {
		return nil, err
	}

	return tr.GetFromChild(keyToChild, key)
}

// LoadCode returns the runtime code (located at :code)
func (s *InmemoryStorageState) LoadCode(hash *common.Hash) ([]byte, error) {
	return s.GetStorage(hash, codeKey)
}

// LoadCodeHash returns the hash of the runtime code (located at :code)
func (s *InmemoryStorageState) LoadCodeHash(hash *common.Hash) (common.Hash, error) {
	code, err := s.LoadCode(hash)
	if err != nil {
		return common.NewHash([]byte{}), err
	}

	return common.Blake2bHash(code)
}

// GenerateTrieProof returns the proofs related to the keys on the state root trie
func (s *InmemoryStorageState) GenerateTrieProof(stateRoot common.Hash, keys [][]byte) (
	encodedProofNodes [][]byte, err error) {
	return proof.Generate(stateRoot[:], keys, s.db)
}
