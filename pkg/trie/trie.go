// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package trie

import (
	"fmt"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/trie/tracking"
)

// EmptyHash is the empty trie hash.
var EmptyHash = common.MustBlake2bHash([]byte{0})

type ChildTriesRead interface {
	GetChild(keyToChild []byte) (Trie, error)
	GetFromChild(keyToChild, key []byte) ([]byte, error)
	GetChildTries() map[common.Hash]Trie
}

type ChildTriesWrite interface {
	PutIntoChild(keyToChild, key, value []byte) error
	DeleteChild(keyToChild []byte) (err error)
	ClearFromChild(keyToChild, key []byte) error
}

type KVStoreRead interface {
	Get(key []byte) []byte
}

type KVStoreWrite interface {
	Put(key, value []byte) error
	Delete(key []byte) error
}

type TrieIterator interface {
	// NextKey performs a depth-first search on the trie and returns the next key
	// and value based on the current state of the iterator.
	NextEntry() (entry *Entry)

	// NextKey performs a depth-first search on the trie and returns the next key
	// based on the current state of the iterator.
	NextKey() (nextKey []byte)

	NextKeyFunc(func(nextKey []byte) bool) (nextKey []byte)

	// Seek moves the iterator to the first key that is greater than the target key.
	Seek(targetKey []byte)
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
}

type TrieRead interface {
	fmt.Stringer

	KVStoreRead
	Hashable
	ChildTriesRead

	Iter() TrieIterator
	PrefixedIter(prefix []byte) TrieIterator
	Entries() (keyValueMap map[string][]byte)
	NextKey(key []byte) []byte
	GetKeysWithPrefix(prefix []byte) (keysLE [][]byte)
}

type Trie interface {
	TrieRead
	ChildTriesWrite
	PrefixTrieWrite
	KVStoreWrite
	Versioned
	TrieDeltas
}
