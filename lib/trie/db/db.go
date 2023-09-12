// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only
package db

import (
	"github.com/ChainSafe/gossamer/internal/database"
	"github.com/ChainSafe/gossamer/lib/common"
)

type Database interface {
	DBGetter
	DBPutter
}

// DBGetter gets a value corresponding to the given key.
type DBGetter interface {
	Get(key []byte) (value []byte, err error)
}

// DBPutter puts a value at the given key and returns an error.
type DBPutter interface {
	Put(key []byte, value []byte) error
}

func NewEmptyInMemoryDB() Database {
	db, _ := database.LoadDatabase("", true)
	return db
}

func NewInMemoryDBFromProof(encodedNodes [][]byte) (Database, error) {
	db := NewEmptyInMemoryDB()
	for _, encodedProofNode := range encodedNodes {
		nodeHash, err := common.Blake2bHash(encodedProofNode)
		if err != nil {
			return nil, err
		}

		db.Put(nodeHash.ToBytes(), encodedProofNode)
	}

	return db, nil

}
