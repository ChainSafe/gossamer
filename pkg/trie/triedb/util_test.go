// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package triedb

import (
	"bytes"
	"strings"

	"github.com/ChainSafe/gossamer/internal/database"
	"github.com/ChainSafe/gossamer/pkg/trie/db"
	"github.com/ChainSafe/gossamer/pkg/trie/triedb/hash"
	"github.com/stretchr/testify/assert"
)

// MemoryDB is an in-memory implementation of the Database interface backed by a
// map. It uses blake2b as hashing algorithm
type MemoryDB struct {
	data           map[string][]byte
	hashedNullNode string
	nullNodeData   []byte
}

func NewMemoryDB[H hash.Hash, Hasher hash.Hasher[H]](data []byte) *MemoryDB {
	return &MemoryDB{
		data:           make(map[string][]byte),
		hashedNullNode: string((*new(Hasher)).Hash(data).Bytes()),
		nullNodeData:   data,
	}
}

func (db *MemoryDB) emplace(key []byte, value []byte) {
	if bytes.Equal(value, db.nullNodeData) {
		return
	}

	db.data[string(key)] = value
}

func (db *MemoryDB) Get(key []byte) ([]byte, error) {
	dbKey := string(key)
	if strings.Contains(dbKey, db.hashedNullNode) {
		return db.nullNodeData, nil
	}
	if value, has := db.data[dbKey]; has {
		return value, nil
	}

	return nil, nil
}

func (db *MemoryDB) Put(key []byte, value []byte) error {
	db.emplace(key, value)
	return nil
}

func (db *MemoryDB) Del(key []byte) error {
	dbKey := string(key)
	delete(db.data, dbKey)
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

func newTestDB(t assert.TestingT) database.Table {
	db, err := database.NewPebble("", true)
	assert.NoError(t, err)
	return database.NewTable(db, "trie")
}
