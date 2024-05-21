// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package db

import (
	"bytes"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/trie/db"
)

type memoryDBEntry struct {
	value []byte
	rc    int32
}

// MemoryDB is an in-memory implementation of the Database interface backed by a
// map. It uses blake2b as hashing algorithm
type MemoryDB struct {
	data           map[common.Hash]memoryDBEntry
	hashedNullNode common.Hash
	nullNodeData   []byte
}

func memoryDBFromNullNode(nullKey, nullNodeData []byte) *MemoryDB {
	return &MemoryDB{
		data:           make(map[common.Hash]memoryDBEntry),
		hashedNullNode: common.MustBlake2bHash(nullKey),
		nullNodeData:   nullNodeData,
	}
}

func NewMemoryDB(data []byte) *MemoryDB {
	return memoryDBFromNullNode(data, data)
}

func (db *MemoryDB) emplace(key common.Hash, value []byte) {
	if bytes.Equal(value, db.nullNodeData) {
		return
	}

	var (
		entry memoryDBEntry
		has   bool
	)

	if entry, has = db.data[key]; has {
		if entry.rc <= 0 {
			entry.value = value
		}
		entry.rc++
	} else {
		entry = memoryDBEntry{
			value: value,
			rc:    1,
		}
	}
	db.data[key] = entry
}

func (db *MemoryDB) Get(key []byte) ([]byte, error) {
	dbKey := common.NewHash(key)
	if dbKey == db.hashedNullNode {
		return db.nullNodeData, nil
	}
	if entry, has := db.data[dbKey]; has {
		if entry.rc > 0 {
			return entry.value, nil
		}
	}

	return nil, nil
}

func (db *MemoryDB) Put(key []byte, value []byte) error {
	dbKey := common.NewHash(key)
	db.emplace(dbKey, value)
	return nil
}

var _ db.Database = &MemoryDB{}
