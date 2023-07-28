// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package memorydb

import (
	"bytes"
	"fmt"

	"github.com/ChainSafe/gossamer/internal/trie/hashdb"
	"github.com/ChainSafe/gossamer/lib/common"
)

type MemoryDBItem struct {
	data []byte
	//Reference count
	rc int32
}

type MemoryDB struct {
	data           map[common.Hash]MemoryDBItem
	hashedNullNode common.Hash
	nullNodeData   []byte
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

func (mdb *MemoryDB) Insert(prefix hashdb.Prefix, value []byte) common.Hash {
	if bytes.Equal(value, mdb.nullNodeData) {
		return mdb.hashedNullNode
	}

	key := common.MustBlake2bHash(value)
	mdb.emplace(key, prefix, value)
	return key
}

func (mdb *MemoryDB) emplace(key common.Hash, prefix hashdb.Prefix, value []byte) {
	if bytes.Equal(value, mdb.nullNodeData) {
		return
	}

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
