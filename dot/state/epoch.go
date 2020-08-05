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

 	"github.com/ChainSafe/gossamer/dot/types"
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

// newEpochDB instantiates a badgerDB instance for stssoring relevant epoch info
func newEpochDB(db chaindb.Database) *EpochDB {
	return &EpochDB{
		db,
	}
}

type EpochState struct {
	db *EpochDB
	lock sync.RWMutex
}

// NewEpochStateFromGenesis
func NewEpochStateFromGenesis(db chaindb.Database, info *types.EpochInfo) (*EpochState, error) {
	epochDB := newEpochDB(db)
	err := epochDB.Put(currentEpochKey, []byte{0, 0, 0, 0, 0, 0, 0, 0})
	if err != nil {
		return nil, err
	}

	s := &EpochState{
		db: epochDB,
	}

	err = s.SetEpochInfo(0, info)
	if err != nil {
		return nil, err
	}

	return s, nil
}

func NewEpochState(db chaindb.Database) (*EpochState) {
	epochDB := newEpochDB(db)
	return &EpochState{
		db: epochDB,
	}
}

func (s *EpochState) SetCurrentEpoch(epoch uint64) error {
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, epoch)
	return s.db.Put(currentEpochKey, buf)
}

func (s *EpochState) GetCurrentEpoch() (uint64, error) {
	b, err := s.db.Get(currentEpochKey)
	if err != nil {
		return 0, err
	}

	return binary.LittleEndian.Uint64(b), nil
} 

func (s *EpochState) SetEpochInfo(epoch uint64, info *types.EpochInfo) error {
	return nil
}

func (s *EpochState) GetEpochInfo(epoch uint64) (*types.EpochInfo, error) {
	return nil, nil
}