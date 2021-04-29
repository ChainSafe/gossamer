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
	"encoding/json"
	"fmt"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/genesis"

	"github.com/ChainSafe/chaindb"
)

// SetupDatabase will return an instance of database based on basepath
func SetupDatabase(basepath string) (chaindb.Database, error) {
	return chaindb.NewBadgerDB(&chaindb.Config{
		DataDir: basepath,
	})
}

// BaseState is a wrapper for the chaindb.Database, without any prefixes
type BaseState struct {
	db chaindb.Database
}

// NewBaseState returns a new BaseState
func NewBaseState(db chaindb.Database) *BaseState {
	return &BaseState{
		db: db,
	}
}

// StoreNodeGlobalName stores the current node name to avoid create new ones after each initialization
func (s *BaseState) StoreNodeGlobalName(nodeName string) error {
	return s.db.Put(common.NodeNameKey, []byte(nodeName))
}

// LoadNodeGlobalName loads the latest stored node global name
func (s *BaseState) LoadNodeGlobalName() (string, error) {
	nodeName, err := s.db.Get(common.NodeNameKey)
	if err != nil {
		return "", err
	}

	return string(nodeName), nil
}

// StoreBestBlockHash stores the hash at the BestBlockHashKey
func (s *BaseState) StoreBestBlockHash(hash common.Hash) error {
	return s.db.Put(common.BestBlockHashKey, hash[:])
}

// LoadBestBlockHash loads the hash stored at BestBlockHashKey
func (s *BaseState) LoadBestBlockHash() (common.Hash, error) {
	hash, err := s.db.Get(common.BestBlockHashKey)
	if err != nil {
		return common.Hash{}, err
	}

	return common.NewHash(hash), nil
}

// StoreGenesisData stores the given genesis data at the known GenesisDataKey.
func (s *BaseState) StoreGenesisData(gen *genesis.Data) error {
	enc, err := json.Marshal(gen)
	if err != nil {
		return fmt.Errorf("cannot scale encode genesis data: %s", err)
	}

	return s.db.Put(common.GenesisDataKey, enc)
}

// LoadGenesisData retrieves the genesis data stored at the known GenesisDataKey.
func (s *BaseState) LoadGenesisData() (*genesis.Data, error) {
	enc, err := s.db.Get(common.GenesisDataKey)
	if err != nil {
		return nil, err
	}

	data := &genesis.Data{}
	err = json.Unmarshal(enc, data)
	if err != nil {
		return nil, err
	}

	return data, nil
}

// StoreLatestStorageHash stores the current root hash in the database at LatestStorageHashKey
func (s *BaseState) StoreLatestStorageHash(root common.Hash) error {
	return s.db.Put(common.LatestStorageHashKey, root[:])
}

// LoadLatestStorageHash retrieves the hash stored at LatestStorageHashKey from the DB
func (s *BaseState) LoadLatestStorageHash() (common.Hash, error) {
	hashbytes, err := s.db.Get(common.LatestStorageHashKey)
	if err != nil {
		return common.Hash{}, err
	}

	return common.NewHash(hashbytes), nil
}

func (s *BaseState) storeSkipToEpoch(epoch uint64) error {
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, epoch)
	return s.db.Put(skipToKey, buf)
}

func (s *BaseState) loadSkipToEpoch() (uint64, error) {
	data, err := s.db.Get(skipToKey)
	if err != nil {
		return 0, err
	}

	return binary.LittleEndian.Uint64(data), nil
}

func (s *BaseState) storeFirstSlot(slot uint64) error {
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, slot)
	return s.db.Put(firstSlotKey, buf)
}

func (s *BaseState) loadFirstSlot() (uint64, error) {
	data, err := s.db.Get(firstSlotKey)
	if err != nil {
		return 0, err
	}

	return binary.LittleEndian.Uint64(data), nil
}
