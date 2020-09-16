// Copyright 2020 ChainSafe Systems (ON) Corp.
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

package runtime

import "github.com/ChainSafe/chaindb"

// NodeStorageTypePersistent flog to identify offchain storage as persistent (db)
const NodeStorageTypePersistent int32 = 1

// NodeStorageTypeLocal flog to identify offchain storage as local (memory)
const NodeStorageTypeLocal int32 = 2

var storagePrefix = []byte("storage")

// NodeStorage struct for storage of runtime offchain worker data
type NodeStorage struct {
	LocalStorage      BasicStorage
	PersistentStorage BasicStorage
}

// NodeStorageDB stores runtime persistent data
type NodeStorageDB struct {
	db chaindb.Database
}

// NewNodeStorageDB instantiates badgerDB instance for storing runtime persistent data
func NewNodeStorageDB(db chaindb.Database) *NodeStorageDB {
	return &NodeStorageDB{
		db,
	}
}

// Put appends `storage` to the key and sets the key-value pair in the db
func (storageDB *NodeStorageDB) Put(key, value []byte) error {
	key = append(storagePrefix, key...)
	return storageDB.db.Put(key, value)
}

// Get appends `storage` to the key and retrieves the value from the db
func (storageDB *NodeStorageDB) Get(key []byte) ([]byte, error) {
	key = append(storagePrefix, key...)
	return storageDB.db.Get(key)
}
