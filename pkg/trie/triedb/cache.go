// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package triedb

import "github.com/ChainSafe/gossamer/pkg/trie/triedb/node"

// CachedValue a value as cached by TrieCache
type CachedValue[H comparable] interface {
	Type() string
}
type (
	// The value doesn't exists in ithe trie
	NonExisting struct{}
	// We cached the hash, because we did not yet accessed the data
	ExistingHash[H comparable] struct {
		hash H
	}
	// The value xists in the trie
	Existing[H comparable] struct {
		hash H      // The hash of the value
		data []byte // The actual data of the value
	}
)

func (v NonExisting) Type() string     { return "NonExisting" }
func (v ExistingHash[H]) Type() string { return "ExistingHash" }
func (v Existing[H]) Type() string     { return "Existing" }

type TrieCache[Out node.HashOut] interface {
	LookupValueForKey(key []byte) *CachedValue[Out]
	CacheValueForKey(key []byte, value CachedValue[Out])
	GetOrInsertNode(hash Out, fetchNode func() (node.Node[Out], error))
	GetNode(hash Out) node.Node[Out]
}
