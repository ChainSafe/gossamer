// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package database

import (
	"sync"

	"github.com/ChainSafe/gossamer/internal/primitives/core/hash"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime"
)

type countValue struct {
	Count uint32
	Value []byte
}

// MemDB implements `Database` as an in-memory hash map. `Commit` is not atomic.
type MemDB[H runtime.Hash] struct {
	inner map[ColumnID]map[string]countValue
	sync.RWMutex
}

// NewMemDB is constructor for MemDB
func NewMemDB[H runtime.Hash]() *MemDB[H] {
	return &MemDB[H]{
		inner: make(map[ColumnID]map[string]countValue),
	}
}

// Commit the `transaction` to the database atomically. Any further calls to `get` or `lookup`
// will reflect the new state.
func (mdb *MemDB[H]) Commit(transaction Transaction[H]) error {
	mdb.Lock()
	defer mdb.Unlock()
	for _, change := range transaction {
		switch change := change.(type) {
		case Set:
			_, ok := mdb.inner[change.ColumnID]
			if !ok {
				mdb.inner[change.ColumnID] = make(map[string]countValue)
			}
			mdb.inner[change.ColumnID][string(change.Key)] = countValue{1, change.Value}
		case Remove:
			_, ok := mdb.inner[change.ColumnID]
			if !ok {
				mdb.inner[change.ColumnID] = make(map[string]countValue)
			}
			delete(mdb.inner[change.ColumnID], string(change.Key))
		case Store[H]:
			_, ok := mdb.inner[change.ColumnID]
			if !ok {
				mdb.inner[change.ColumnID] = make(map[string]countValue)
			}
			cv, ok := mdb.inner[change.ColumnID][change.Hash.String()]
			if ok {
				cv.Count += 1
				mdb.inner[change.ColumnID][change.Hash.String()] = cv
			} else {
				mdb.inner[change.ColumnID][change.Hash.String()] = countValue{1, change.Preimage}
			}
		case Reference[H]:
			_, ok := mdb.inner[change.ColumnID]
			if !ok {
				mdb.inner[change.ColumnID] = make(map[string]countValue)
			}
			cv, ok := mdb.inner[change.ColumnID][change.Hash.String()]
			if ok {
				cv.Count += 1
				mdb.inner[change.ColumnID][change.Hash.String()] = cv
			}
		case Release[H]:
			_, ok := mdb.inner[change.ColumnID]
			if !ok {
				mdb.inner[change.ColumnID] = make(map[string]countValue)
			}
			cv, ok := mdb.inner[change.ColumnID][change.Hash.String()]
			if ok {
				cv.Count -= 1
				if cv.Count == 0 {
					delete(mdb.inner[change.ColumnID], change.Hash.String())
				} else {
					mdb.inner[change.ColumnID][change.Hash.String()] = cv
				}
			}
		}
	}
	return nil
}

// Retrieve the value previously stored against `key` or `nil` if `key` is not currently in the database.
func (mdb *MemDB[H]) Get(col ColumnID, key []byte) []byte {
	mdb.RLock()
	defer mdb.RUnlock()
	_, ok := mdb.inner[col]
	if !ok {
		return nil
	}
	cv, ok := mdb.inner[col][string(key)]
	if ok {
		return cv.Value
	}
	return nil
}

// Check if the value exists in the database without retrieving it.
func (mdb *MemDB[H]) Contains(col ColumnID, key []byte) bool {
	return mdb.Get(col, key) != nil
}

var _ Database[hash.H256] = &MemDB[hash.H256]{}
