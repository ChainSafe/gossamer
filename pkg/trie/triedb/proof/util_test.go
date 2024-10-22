// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package proof

import (
	"bytes"

	"github.com/ChainSafe/gossamer/internal/database"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime"
	"github.com/ChainSafe/gossamer/pkg/trie/db"
)

// MemoryDB is an in-memory implementation of the Database interface backed by a
// map. It uses blake2b as hashing algorithm
type MemoryDB struct {
	data           map[string][]byte
	hashedNullNode []byte
	nullNodeData   []byte
}

func memoryDBFromNullNode(nullKey, nullNodeData []byte) *MemoryDB {
	return &MemoryDB{
		data:           make(map[string][]byte),
		hashedNullNode: runtime.BlakeTwo256{}.Hash(nullKey).Bytes(),
		nullNodeData:   nullNodeData,
	}
}

func NewMemoryDB(data []byte) *MemoryDB {
	return memoryDBFromNullNode(data, data)
}

func (db *MemoryDB) emplace(key []byte, value []byte) {
	if bytes.Equal(value, db.nullNodeData) {
		return
	}

	db.data[string(key)] = value
}

func (db *MemoryDB) Get(key []byte) ([]byte, error) {
	dbKey := key
	if bytes.Equal(dbKey, db.hashedNullNode) {
		return db.nullNodeData, nil
	}
	if value, has := db.data[string(dbKey)]; has {
		return value, nil
	}

	return nil, nil
}

func (db *MemoryDB) Put(key []byte, value []byte) error {
	dbKey := key
	db.emplace(dbKey, value)
	return nil
}

func (db *MemoryDB) Del(key []byte) error {
	dbKey := key
	delete(db.data, string(dbKey))
	return nil
}

func (db *MemoryDB) Flush() error {
	return nil
}

func (db *MemoryDB) NewBatch() database.Batch {
	return &MemoryBatch{db}
}

var _ db.RWDatabase = &MemoryDB{}

type MemoryBatch struct {
	*MemoryDB
}

func (b *MemoryBatch) Close() error {
	return nil
}

func (*MemoryBatch) Reset() {}

func (b *MemoryBatch) ValueSize() int {
	return 1
}

var _ database.Batch = &MemoryBatch{}
