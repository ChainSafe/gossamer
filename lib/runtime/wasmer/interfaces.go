// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package wasmer

import (
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/transaction"
	"github.com/ChainSafe/gossamer/lib/trie"
)

// Storage runtime interface.
type Storage interface {
	GetSetter
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
	LoadCode() []byte
}

// GetSetter gets and sets key values.
type GetSetter interface {
	Getter
	Putter
}

// Getter gets a value from a key.
type Getter interface {
	Get(key []byte) []byte
}

// Putter puts a value for a key.
type Putter interface {
	Put(key []byte, value []byte)
}

// BasicNetwork interface for functions used by runtime network state function
type BasicNetwork interface {
	NetworkState() common.NetworkState
}

// TransactionState is the interface for the transaction state.
type TransactionState interface {
	AddToPool(vt *transaction.ValidTransaction) common.Hash
}
