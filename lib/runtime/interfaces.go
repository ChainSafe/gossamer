// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package runtime

import (
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/transaction"
	"github.com/ChainSafe/gossamer/lib/trie"
)

// Storage runtime interface.
type Storage interface {
	Put(key []byte, value []byte, version trie.Version) (err error)
	Get(key []byte) []byte
	Root() (common.Hash, error)
	SetChild(keyToChild []byte, child *trie.Trie, version trie.Version) error
	SetChildStorage(keyToChild, key, value []byte, version trie.Version) error
	GetChildStorage(keyToChild, key []byte) ([]byte, error)
	Delete(key []byte) (err error)
	DeleteChild(keyToChild []byte) (err error)
	DeleteChildLimit(keyToChild []byte, limit *[]byte) (
		deleted uint32, allDeleted bool, err error)
	ClearChildStorage(keyToChild, key []byte, version trie.Version) error
	NextKey([]byte) []byte
	ClearPrefixInChild(keyToChild, prefix []byte) error
	ClearPrefixInChildWithLimit(keyToChild, prefix []byte, limit uint32) (uint32, bool, error)
	GetChildNextKey(keyToChild, key []byte) ([]byte, error)
	GetChild(keyToChild []byte) (*trie.Trie, error)
	ClearPrefix(prefix []byte) (err error)
	ClearPrefixLimit(prefix []byte, limit uint32) (
		deleted uint32, allDeleted bool, err error)
	BeginStorageTransaction()
	CommitStorageTransaction()
	RollbackStorageTransaction()
	LoadCode() []byte
}

// BasicNetwork interface for functions used by runtime network state function
type BasicNetwork interface {
	NetworkState() common.NetworkState
}

// BasicStorage interface for functions used by runtime offchain workers
type BasicStorage interface {
	Put(key []byte, value []byte) error
	Get(key []byte) ([]byte, error)
	Del(key []byte) error
}

// TransactionState interface for adding transactions to pool
type TransactionState interface {
	AddToPool(vt *transaction.ValidTransaction) common.Hash
}
