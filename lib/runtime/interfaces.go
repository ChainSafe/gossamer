// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package runtime

import (
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/transaction"
	"github.com/ChainSafe/gossamer/pkg/trie"
)

// Trie storage interface.
type Trie interface {
	Root() (common.Hash, error)
	Put(key []byte, value []byte) (err error)
	Get(key []byte) []byte
	Delete(key []byte) (err error)
	NextKey([]byte) []byte
	ClearPrefix(prefix []byte) (err error)
	ClearPrefixLimit(prefix []byte, limit uint32) (
		deleted uint32, allDeleted bool, err error)
}

// ChildTrie storage interface.S
type ChildTrie interface {
	GetChildRoot(keyToChild []byte) (common.Hash, error)
	SetChildStorage(keyToChild, key, value []byte) error
	GetChildStorage(keyToChild, key []byte) ([]byte, error)
	DeleteChild(keyToChild []byte) (err error)
	DeleteChildLimit(keyToChild []byte, limit *[]byte) (
		deleted uint32, allDeleted bool, err error)
	ClearChildStorage(keyToChild, key []byte) error
	ClearPrefixInChild(keyToChild, prefix []byte) error
	ClearPrefixInChildWithLimit(keyToChild, prefix []byte, limit uint32) (uint32, bool, error)
	GetChildNextKey(keyToChild, key []byte) ([]byte, error)
}

// Transactional storage interface.
type Transactional interface {
	StartTransaction()
	CommitTransaction()
	RollbackTransaction()
}

// Runtime storage interface.
type Runtime interface {
	LoadCode() []byte
	SetVersion(v trie.TrieLayout)
}

// Storage runtime interface.
type Storage interface {
	Trie
	ChildTrie
	Transactional
	Runtime
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
