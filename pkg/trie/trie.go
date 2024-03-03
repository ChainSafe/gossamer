// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package trie

import (
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/trie/db"
	"github.com/ChainSafe/gossamer/pkg/trie/tracking"
)

type ChildTrieManager interface {
	GetChild(keyToChild []byte) (Trie, error)
	GetFromChild(keyToChild, key []byte) ([]byte, error)
	PutIntoChild(keyToChild, key, value []byte) error
	DeleteChild(keyToChild []byte) (err error)
	ClearFromChild(keyToChild, key []byte) error
	GetChildTries() map[common.Hash]Trie
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
	handleTrackedDeltas(success bool, pendingDeltas tracking.Getter)
}

type DBBackedTrie interface {
	Load(db db.DBGetter, rootHash common.Hash) error
	WriteDirty(db NewBatcher) error
	writeDirtyNode(db db.DBPutter, n *Node) (err error)
}

type Printable interface {
	String() string
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
	Printable
	Hashable
	ChildTrieManager
	TrieIterator
	TrieDeltas
	PrefixTrie
	DBBackedTrie
	Versioned

	RootNode() *Node

	//TODO:this method should not be part of the API, find a way to remove it
	insertKeyLE(key, value []byte,
		pendingDeltas DeltaRecorder) (err error)
}
