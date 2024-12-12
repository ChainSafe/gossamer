// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package offchain

import (
	"bytes"
	"log"
	"sync"

	"github.com/ChainSafe/gossamer/internal/client/db/columns"
	"github.com/ChainSafe/gossamer/internal/primitives/core/hash"
	"github.com/ChainSafe/gossamer/internal/primitives/database"
)

// / Offchain local storage
type LocalStorage struct {
	// db: Arc<dyn Database<DbHash>>,
	db database.Database[hash.H256]
	// locks: Arc<Mutex<HashMap<Vec<u8>, Arc<Mutex<()>>>>>,
	locks    map[string]*sync.Mutex
	locksMtx sync.Mutex
}

// / Create offchain local storage with given `KeyValueDB` backend.
func NewLocalStorage(db database.Database[hash.H256]) *LocalStorage {
	return &LocalStorage{
		db:    db,
		locks: make(map[string]*sync.Mutex),
	}
}

func (ls *LocalStorage) Set(prefix, key, value []byte) {
	var tx database.Transaction[hash.H256]
	tx.Set(columns.Offchain, ConcatenatePrefixAndKey(prefix, key), value)

	err := ls.db.Commit(tx)
	if err != nil {
		log.Printf("ERROR: error setting on local storage: %v", err)
	}
}

func (ls *LocalStorage) Remove(prefix, key []byte) {
	var tx database.Transaction[hash.H256]
	tx.Remove(columns.Offchain, ConcatenatePrefixAndKey(prefix, key))

	err := ls.db.Commit(tx)
	if err != nil {
		log.Printf("ERROR: error removing on local storage: %v", err)
	}
}

func (ls *LocalStorage) Get(prefix, key []byte) []byte {
	return ls.db.Get(columns.Offchain, ConcatenatePrefixAndKey(prefix, key))
}

func (ls *LocalStorage) CompareAndSet(prefix, itemKey, oldValue, newValue []byte) bool {
	key := ConcatenatePrefixAndKey(prefix, itemKey)

	ls.locksMtx.Lock()
	_, ok := ls.locks[string(key)]
	if !ok {
		ls.locks[string(key)] = &sync.Mutex{}
	}
	keyLock := ls.locks[string(key)]
	ls.locksMtx.Unlock()

	var isSet bool
	{
		keyLock.Lock()
		val := ls.db.Get(columns.Offchain, key)
		isSet = bytes.Equal(val, oldValue)

		if isSet {
			ls.Set(prefix, itemKey, newValue)
		}
	}

	// clean the lock map if we're the only entry
	ls.locksMtx.Lock()
	{
		keyLock.Unlock()
		_, ok := ls.locks[string(key)]
		if ok {
			delete(ls.locks, string(key))
		}
	}
	ls.locksMtx.Unlock()

	return isSet
}

// / Concatenate the prefix and key to create an offchain key in the db.
func ConcatenatePrefixAndKey(prefix, key []byte) []byte {
	return append(prefix, key...)
}
