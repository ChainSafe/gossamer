// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package db

import (
	"bytes"
	"fmt"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/trie"
)

type KeyFunction func(key common.Hash, prefix trie.Prefix) common.Hash

type MemoryDBItem struct {
	data []byte
	//Reference count
	rc int32
}

type MemoryDB struct {
	data           map[common.Hash]MemoryDBItem
	hashedNullNode common.Hash
	nullNodeData   []byte
	keyFunction    KeyFunction
}

func NewMemoryDB() *MemoryDB {
	return newMemoryDBFromNullNode([]byte{0}, []byte{0})
}

func newMemoryDBFromNullNode(nullKey []byte, nullNodeData []byte) *MemoryDB {
	hashedKey := common.MustBlake2bHash(nullKey)

	return &MemoryDB{
		data:           make(map[common.Hash]MemoryDBItem),
		hashedNullNode: hashedKey,
		nullNodeData:   nullNodeData,
		keyFunction:    hashKey,
	}
}

func (mdb *MemoryDB) Get(key []byte) (value []byte, err error) {
	if len(key) < common.HashLength {
		return nil, fmt.Errorf("expected %d bytes length key, given %d (%x)", common.HashLength, len(key), value)
	}
	var hash common.Hash
	copy(hash[:], key)

	if value, found := mdb.data[hash]; found {
		return value.data, nil
	}

	return nil, nil
}

func (mdb *MemoryDB) GetWithPrefix(key []byte, prefix trie.Prefix) (value []byte, err error) {
	if bytes.Equal(key, mdb.hashedNullNode[:]) {
		return mdb.nullNodeData, nil
	}

	computatedKey := mdb.keyFunction(common.Hash(key), prefix)
	return mdb.Get(computatedKey[:])
}

func (mdb *MemoryDB) Insert(prefix trie.Prefix, value []byte) common.Hash {
	if bytes.Equal(value, mdb.nullNodeData) {
		return mdb.hashedNullNode
	}

	key := common.MustBlake2bHash(value)
	mdb.emplace(key, prefix, value)
	return key
}

func (mdb *MemoryDB) emplace(key common.Hash, prefix trie.Prefix, value []byte) {
	if bytes.Equal(value, mdb.nullNodeData) {
		return
	}

	key = mdb.keyFunction(key, prefix)
	data, ok := mdb.data[key]
	if !ok {
		mdb.data[key] = MemoryDBItem{value, 0}
		return
	}

	if data.rc <= 0 {
		data.data = value
	}
	data.rc++
}

func hashKey(key common.Hash, prefix trie.Prefix) common.Hash {
	return key
}
