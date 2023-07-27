// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package proof

import (
	"github.com/ChainSafe/gossamer/lib/trie"
	"github.com/ChainSafe/gossamer/lib/trie/db"
)

var EmptyPrefix = trie.Prefix{}

type StorageProof struct {
	//TODO: Improve it using sets
	trieNodes [][]byte
}

func (sp *StorageProof) toMemoryDB() trie.HashDB {
	db := db.NewMemoryDB()

	for _, proof := range sp.trieNodes {
		db.Insert(EmptyPrefix, proof)
	}

	return db
}

func NewStorageProof(proof [][]byte) *StorageProof {
	return &StorageProof{
		trieNodes: proof,
	}
}
