// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package trie

import (
	hashdb "github.com/ChainSafe/gossamer/internal/hash-db"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime"
	"github.com/tidwall/btree"
)

// A proof that some set of key-value pairs are included in the storage trie. The proof contains
// the storage values so that the partial storage backend can be reconstructed by a verifier that
// does not already have access to the key-value pairs.
//
// The proof consists of the set of serialised nodes in the storage trie accessed when looking up
// the keys covered by the proof. Verifying the proof requires constructing the partial trie from
// the serialised nodes and performing the key lookups.
type StorageProof struct {
	trieNodes btree.Set[string]
}

// Constructs a [StorageProof] from a subset of encoded trie nodes.
func NewStorageProof(trieNodes [][]byte) StorageProof {
	set := btree.Set[string]{}
	for _, trieNode := range trieNodes {
		set.Insert(string(trieNode))
	}
	return StorageProof{
		trieNodes: set,
	}
}

// Returns whether this is an empty proof.
func (sp *StorageProof) Empty() bool {
	return sp.trieNodes.Len() == 0
}

// Returns all the encoded trie ndoes in lexigraphical order from the proof.
func (sp *StorageProof) Nodes() [][]byte {
	var ret [][]byte
	sp.trieNodes.Scan(func(v string) bool {
		ret = append(ret, []byte(v))
		return true
	})
	return ret
}

// Constructs a [MemoryDB] from a [StorageProof]
func NewMemoryDBFromStorageProof[H runtime.Hash, Hasher runtime.Hasher[H]](sp StorageProof) *MemoryDB[H, Hasher] {
	db := NewMemoryDB[H, Hasher]()
	sp.trieNodes.Scan(func(v string) bool {
		db.Insert(hashdb.EmptyPrefix, []byte(v))
		return true
	})
	return db
}
