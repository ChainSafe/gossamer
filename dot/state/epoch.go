// Copyright 2019 ChainSafe Systems (ON) Corp.
// This file is part of gossamer.
//
// The gossamer library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The gossamer library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the gossamer library. If not, see <http://www.gnu.org/licenses/>.

package state

import (
	"encoding/binary"
	"sync"

	"github.com/ChainSafe/chaindb"
)

var (
	epochPrefix = []byte("epoch")
	currentEpochKey = []byte("current")
	epochInfoPrefix = []byte("epochinfo")
)

func epochInfoKey(epoch uint64) []byte {
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, epoch)
	return append(epochInfoPrefix, buf...)
}
 
// EpochDB stores epoch info in an underlying Database
type EpochDB struct {
	db chaindb.Database
}

// Put appends `epoch` to the key and sets the key-value pair in the db
func (db *EpochDB) Put(key, value []byte) error {
	key = append(epochPrefix, key...)
	return db.db.Put(key, value)
}

// Get appends `epoch` to the key and retrieves the value from the db
func (db *EpochDB) Get(key []byte) ([]byte, error) {
	key = append(epochPrefix, key...)
	return db.db.Get(key)
}

// Delete deletes a key from the db
func (db *EpochDB) Delete(key []byte) error {
	key = append(epochPrefix, key...)
	return db.db.Del(key)
}

// Has appends `epoch` to the key and checks for existence in the db
func (db *EpochDB) Has(key []byte) (bool, error) {
	key = append(epochPrefix, key...)
	return db.db.Has(key)
}

// newEpochDB instantiates a badgerDB instance for storing relevant epoch info
func newEpochDB(db chaindb.Database) *EpochDB {
	return &EpochDB{
		db,
	}
}

type EpochState struct {
	db *EpochDB
	lock sync.RWMutex
}

func NewEpochStateFromGenesis(db chaindb.Database) *EpochState {
	epochDB := newEpochDB(db)
	epochDB.Put(currentEpochKey, []byte{0, 0, 0, 0, 0, 0, 0, 0})
	return &EpochState{
		db: epochDB,
	}
}