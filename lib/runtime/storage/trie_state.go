// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package storage

import (
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/trie"
)

type TrieState interface {
	SetVersion(v trie.TrieLayout)
	StartTransaction()
	RollbackTransaction() error
	CommitTransaction() error
	Put(key, value []byte) error
	Get(key []byte) []byte
	Root() (common.Hash, error)
	Has(key []byte) bool
	Delete(key []byte) error
	NextKey(key []byte) []byte
	ClearPrefix(prefix []byte) error
	ClearPrefixLimit(prefix []byte, limit uint32) (loops uint32, deleted uint32, allDeleted bool, err error)
	SetChildStorage(keyToChild, key, value []byte) error
	GetChildRoot(keyToChild []byte) (common.Hash, error)
	GetChildStorage(keyToChild, key []byte) ([]byte, error)
	DeleteChild(keyToChild []byte) error
	// TODO: use limit as *uint32
	DeleteChildLimit(key []byte, limit *[]byte) (deleted uint32, allDeleted bool, err error)
	ClearChildStorage(keyToChild, key []byte) error
	ClearPrefixInChild(keyToChild, prefix []byte) error
	ClearPrefixInChildWithLimit(keyToChild, prefix []byte, limit uint32) (uint32, uint32, bool, error)
	GetChildNextKey(keyToChild, key []byte) ([]byte, error)
	LoadCode() []byte
	LoadCodeHash() (common.Hash, error)
	GetChangedNodeHashes() (inserted, deleted map[common.Hash]struct{}, err error)
}
