// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package trie

import (
	"fmt"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/trie/tracking"
)

// EmptyHash is the empty trie hash.
var EmptyHash = common.MustBlake2bHash([]byte{0})

type ReadChildTries interface {
	GetChild(keyToChild []byte) (Trie, error)
	GetFromChild(keyToChild, key []byte) ([]byte, error)
	GetChildTries() map[common.Hash]Trie
}

type WriteChildTries interface {
	PutIntoChild(keyToChild, key, value []byte) error
	DeleteChild(keyToChild []byte) (err error)
	ClearFromChild(keyToChild, key []byte) error
}

type KVSRead interface {
	Get(key []byte) []byte
}

type KVSWrite interface {
	Put(key, value []byte) error
	Delete(key []byte) error
}

type TrieIterator interface {
	Entries() (keyValueMap map[string][]byte)
	NextKey(key []byte) []byte
}

type PrefixTrieRead interface {
	GetKeysWithPrefix(prefix []byte) (keysLE [][]byte)
}

type PrefixTrieWrite interface {
	ClearPrefix(prefix []byte) (err error)
	ClearPrefixLimit(prefix []byte, limit uint32) (
		deleted uint32, allDeleted bool, err error)
}

type TrieDeltas interface {
	GetChangedNodeHashes() (inserted, deleted map[common.Hash]struct{}, err error)
	HandleTrackedDeltas(success bool, pendingDeltas tracking.Getter)
}

type Versioned interface {
	SetVersion(TrieLayout)
}

type Hashable interface {
	MustHash() common.Hash
	Hash() (common.Hash, error)
	// TODO: remove this method, it is not directly related with tries
	GenesisBlock() (genesisHeader types.Header, err error)
}

type ReadOnlyTrie interface {
	fmt.Stringer

	KVSRead
	Hashable
	ReadChildTries
	PrefixTrieRead
	TrieIterator
	TrieDeltas
}

type Trie interface {
	ReadOnlyTrie
	WriteChildTries
	PrefixTrieWrite
	KVSWrite
	Versioned
}
