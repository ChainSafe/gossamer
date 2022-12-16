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
	Put(key []byte, value []byte)
	Get(key []byte) []byte
	Root() (common.Hash, error)
	SetChild(keyToChild []byte, child *trie.Trie) error
	SetChildStorage(keyToChild, key, value []byte) error
	GetChildStorage(keyToChild, key []byte) ([]byte, error)
	Delete(key []byte)
	DeleteChild(keyToChild []byte)
	DeleteChildLimit(keyToChild []byte, limit *[]byte) (uint32, bool, error)
	ClearChildStorage(keyToChild, key []byte) error
	NextKey([]byte) []byte
	ClearPrefixInChild(keyToChild, prefix []byte) error
	GetChildNextKey(keyToChild, key []byte) ([]byte, error)
	GetChild(keyToChild []byte) (*trie.Trie, error)
	ClearPrefix(prefix []byte)
	ClearPrefixLimit(prefix []byte, limit uint32) (uint32, bool)
	BeginStorageTransaction()
	CommitStorageTransaction()
	RollbackStorageTransaction()
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

//go:generate mockery --name TransactionState --structname TransactionState --case underscore --keeptree

// TransactionState interface for adding transactions to pool
type TransactionState interface {
	AddToPool(vt *transaction.ValidTransaction) common.Hash
}
