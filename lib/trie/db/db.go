// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only
package db

import (
	"fmt"
	"sync"

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

type MemoryDB struct {
	data  map[common.Hash][]byte
	mutex sync.RWMutex
}

func NewEmptyMemoryDB() *MemoryDB {
	return &MemoryDB{
		data: make(map[common.Hash][]byte),
	}
}

func NewMemoryDBFromProof(encodedNodes [][]byte) (*MemoryDB, error) {
	data := make(map[common.Hash][]byte, len(encodedNodes))

	for _, encodedProofNode := range encodedNodes {
		nodeHash, err := common.Blake2bHash(encodedProofNode)
		if err != nil {
			return nil, err
		}

		data[nodeHash] = encodedProofNode
	}

	return &MemoryDB{
		data: data,
	}, nil

}

func (mdb *MemoryDB) Copy() Database {
	newDB := NewEmptyMemoryDB()
	copyData := make(map[common.Hash][]byte, len(mdb.data))

	for k, v := range mdb.data {
		copyData[k] = v
	}

	newDB.data = copyData
	return newDB
}

func (mdb *MemoryDB) Get(key []byte) ([]byte, error) {
	if len(key) != common.HashLength {
		return nil, fmt.Errorf("expected %d bytes length key, given %d (%x)", common.HashLength, len(key), key)
	}
	var hash common.Hash
	copy(hash[:], key)

	mdb.mutex.RLock()
	defer mdb.mutex.RUnlock()

	if value, found := mdb.data[hash]; found {
		return value, nil
	}

	return nil, nil
}

func (mdb *MemoryDB) Put(key, value []byte) error {
	if len(key) != common.HashLength {
		return fmt.Errorf("expected %d bytes length key, given %d (%x)", common.HashLength, len(key), key)
	}

	var hash common.Hash
	copy(hash[:], key)

	mdb.mutex.Lock()
	defer mdb.mutex.Unlock()

	mdb.data[hash] = value
	return nil
}
