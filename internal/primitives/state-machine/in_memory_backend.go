// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package statemachine

import (
	"github.com/ChainSafe/gossamer/internal/primitives/runtime"
	"github.com/ChainSafe/gossamer/internal/primitives/storage"
	"github.com/ChainSafe/gossamer/internal/primitives/trie"
)

// MemoryDBTrieBackend is [TrieBackend] fulfilled with [trie.PrefixedMemoryDB]
type MemoryDBTrieBackend[H runtime.Hash, Hasher runtime.Hasher[H]] struct {
	*TrieBackend[H, Hasher]
}

// Create a new empty instance of in-memory backend.
func NewMemoryDBTrieBackend[H runtime.Hash, Hasher runtime.Hasher[H]]() MemoryDBTrieBackend[H, Hasher] {
	mdb := trie.NewPrefixedMemoryDB[H, Hasher]()
	root := (*new(Hasher)).Hash([]byte{0})
	return MemoryDBTrieBackend[H, Hasher]{
		TrieBackend: NewTrieBackend[H, Hasher](HashDBTrieBackendStorage[H]{mdb}, root, nil, nil),
	}
}

type change struct {
	storage.ChildInfo // can be nil
	StorageCollection
}

func (tb *MemoryDBTrieBackend[H, Hasher]) clone() MemoryDBTrieBackend[H, Hasher] {
	return MemoryDBTrieBackend[H, Hasher]{
		NewTrieBackend[H, Hasher](tb.essence.BackendStorage(), tb.essence.root, nil, nil),
	}
}

// Copy the state, with applied updates
func (tb *MemoryDBTrieBackend[H, Hasher]) update(
	changes []change, stateVersion storage.StateVersion,
) MemoryDBTrieBackend[H, Hasher] {
	clone := tb.clone()
	clone.insert(changes, stateVersion)
	return clone
}

// Insert values into backend trie.
func (tb *MemoryDBTrieBackend[H, Hasher]) insert(changes []change, stateVersion storage.StateVersion) {
	top := make([]change, 0)
	child := make([]change, 0)
	for _, change := range changes {
		if change.ChildInfo == nil {
			top = append(top, change)
		} else {
			child = append(child, change)
		}
	}
	var deltas []Delta
	for _, change := range top {
		for _, skv := range change.StorageCollection {
			deltas = append(deltas, Delta{Key: skv.StorageKey, Value: skv.StorageValue})
		}
	}

	var childDeltas []ChildDelta
	for _, change := range child {
		var delta []Delta
		for _, skv := range change.StorageCollection {
			delta = append(delta, Delta{Key: skv.StorageKey, Value: skv.StorageValue})
		}
		childDeltas = append(childDeltas, ChildDelta{
			ChildInfo: change.ChildInfo,
			Deltas:    delta,
		})
	}
	root, tx := tb.FullStorageRoot(deltas, childDeltas, stateVersion)

	tb.applyTransaction(root, tx)
}

// Apply the given transaction to this backend and set the root to the given value.
func (tb *MemoryDBTrieBackend[H, Hasher]) applyTransaction(root H, transaction BackendTransaction[H, Hasher]) {
	hdbtbs := tb.essence.storage.(HashDBTrieBackendStorage[H])
	storage := hdbtbs.HashDB.(*trie.PrefixedMemoryDB[H, Hasher])

	storage.Consolidate(&transaction.MemoryDB)
	new := NewTrieBackend[H, Hasher](hdbtbs, root, nil, nil)
	tb.TrieBackend = new
}
