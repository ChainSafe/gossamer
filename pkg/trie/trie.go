// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package trie

import (
	"fmt"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/trie/db"
	"github.com/ChainSafe/gossamer/pkg/trie/node"
	"github.com/ChainSafe/gossamer/pkg/trie/tracking"
)

type ChildTrieGetter interface {
	GetChild(keyToChild []byte) (Trie, error)
	GetFromChild(keyToChild, key []byte) ([]byte, error)
	GetChildTries() map[common.Hash]Trie
}

type ChildTrieSetter interface {
	PutIntoChild(keyToChild, key, value []byte) error
}

type ChildTrieDeleter interface {
	DeleteChild(keyToChild []byte) (err error)
	ClearFromChild(keyToChild, key []byte) error
}

type KVStore interface {
	Get(key []byte) []byte
	Put(key, value []byte) error
	Delete(key []byte) error
}

type TrieIterator interface {
	Entries() (keyValueMap map[string][]byte)
	NextKey(key []byte) []byte
}

type PrefixTrie interface {
	GetKeysWithPrefix(prefix []byte) (keysLE [][]byte)
	ClearPrefix(prefix []byte) (err error)
	ClearPrefixLimit(prefix []byte, limit uint32) (
		deleted uint32, allDeleted bool, err error)
}

type TrieDeltas interface {
	GetChangedNodeHashes() (inserted, deleted map[common.Hash]struct{}, err error)
	HandleTrackedDeltas(success bool, pendingDeltas tracking.Getter)
}

type DBBackedTrie interface {
	Load(db db.DBGetter, rootHash common.Hash) error
	WriteDirty(db db.NewBatcher) error
}

type Versioned interface {
	SetVersion(TrieLayout)
}

type Hashable interface {
	MustHash() common.Hash
	Hash() (common.Hash, error)
	GenesisBlock() (genesisHeader types.Header, err error)
}

type Trie interface {
	KVStore
	Hashable
	ChildTrieGetter
	ChildTrieSetter
	ChildTrieDeleter
	TrieIterator
	TrieDeltas
	PrefixTrie
	DBBackedTrie
	Versioned
	fmt.Stringer

	RootNode() *node.Node

	//TODO:this method should not be part of the API, find a way to remove it
	InsertKeyLE(key, value []byte,
		pendingDeltas tracking.DeltaRecorder) (err error)
}
