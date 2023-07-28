// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package proof

import (
	"github.com/ChainSafe/gossamer/internal/trie/hashdb"
	"github.com/ChainSafe/gossamer/internal/trie/memorydb"
)

var EmptyPrefix = hashdb.Prefix{}

type StorageProof struct {
	//TODO: Improve it using sets
	trieNodes [][]byte
}

func (sp *StorageProof) toMemoryDB() hashdb.HashDB {
	db := memorydb.NewMemoryDB()

	for _, proof := range sp.trieNodes {
		db.Insert(proof)
	}

	return db
}

func NewStorageProof(proof [][]byte) *StorageProof {
	return &StorageProof{
		trieNodes: proof,
	}
}
