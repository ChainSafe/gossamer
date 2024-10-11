// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package triedb

import (
	"bytes"
	"strings"

	"github.com/ChainSafe/gossamer/internal/database"
	chash "github.com/ChainSafe/gossamer/internal/primitives/core/hash"
	"github.com/ChainSafe/gossamer/pkg/trie/db"
	"github.com/ChainSafe/gossamer/pkg/trie/triedb/hash"
	"github.com/stretchr/testify/assert"
	"golang.org/x/exp/maps"
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

func (db *MemoryDB) Clone() *MemoryDB {
	return &MemoryDB{
		data:           maps.Clone(db.data),
		hashedNullNode: db.hashedNullNode,
		nullNodeData:   db.nullNodeData,
	}
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

type TestTrieCache[H hash.Hash] struct {
	valueCache map[string]CachedValue[H]
	nodeCache  map[H]CachedNode[H]
}

func NewTestTrieCache[H hash.Hash]() *TestTrieCache[H] {
	return &TestTrieCache[H]{
		valueCache: make(map[string]CachedValue[H]),
		nodeCache:  make(map[H]CachedNode[H]),
	}
}

func (ttc *TestTrieCache[H]) GetValue(key []byte) CachedValue[H] {
	cv, ok := ttc.valueCache[string(key)]
	if !ok {
		return nil
	}
	return cv
}

func (ttc *TestTrieCache[H]) SetValue(key []byte, value CachedValue[H]) {
	ttc.valueCache[string(key)] = value
}

func (ttc *TestTrieCache[H]) GetOrInsertNode(hash H, fetchNode func() (CachedNode[H], error)) (CachedNode[H], error) {
	node, ok := ttc.nodeCache[hash]
	if !ok {
		var err error
		node, err = fetchNode()
		if err != nil {
			return nil, err
		}
		ttc.nodeCache[hash] = node
	}
	return node, nil
}

func (ttc *TestTrieCache[H]) GetNode(hash H) CachedNode[H] {
	node, ok := ttc.nodeCache[hash]
	if !ok {
		return nil
	}
	return node
}

var _ TrieCache[chash.H256] = &TestTrieCache[chash.H256]{}
