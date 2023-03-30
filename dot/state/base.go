// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package state

import (
	"encoding/binary"
	"encoding/json"
	"fmt"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/genesis"
)

// BaseState is a wrapper for a database, without any prefixes
type BaseState struct {
	db GetSetDeleter
}

// NewBaseState returns a new BaseState
func NewBaseState(db GetSetDeleter) *BaseState {
	return &BaseState{
		db: db,
	}
}

// StoreNodeGlobalName stores the current node name to avoid create new ones after each initialization
func (s *BaseState) StoreNodeGlobalName(nodeName string) error {
	return s.db.Set(common.NodeNameKey, []byte(nodeName))
}

// LoadNodeGlobalName loads the latest stored node global name
func (s *BaseState) LoadNodeGlobalName() (string, error) {
	nodeName, err := s.db.Get(common.NodeNameKey)
	if err != nil {
		return "", err
	}

	return string(nodeName), nil
}

// StoreGenesisData stores the given genesis data at the known GenesisDataKey.
func (s *BaseState) StoreGenesisData(gen *genesis.Data) error {
	enc, err := json.Marshal(gen)
	if err != nil {
		return fmt.Errorf("cannot scale encode genesis data: %s", err)
	}

	return s.db.Set(common.GenesisDataKey, enc)
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

// StoreCodeSubstitutedBlockHash stores the hash at the CodeSubstitutedBlock key
func (s *BaseState) StoreCodeSubstitutedBlockHash(hash common.Hash) error {
	return s.db.Set(common.CodeSubstitutedBlock, hash[:])
}

// LoadCodeSubstitutedBlockHash loads the hash stored at CodeSubstitutedBlock key
func (s *BaseState) LoadCodeSubstitutedBlockHash() common.Hash {
	hash, err := s.db.Get(common.CodeSubstitutedBlock)
	if err != nil {
		return common.Hash{}
	}

	return common.NewHash(hash)
}

// Set sets a value at the given key in the database.
func (s *BaseState) Set(key, value []byte) error {
	return s.db.Set(key, value)
}

// Get retrieves value by key from database
func (s *BaseState) Get(key []byte) ([]byte, error) {
	return s.db.Get(key)
}

// Delete deletes the given key from the database.
func (s *BaseState) Delete(key []byte) error {
	return s.db.Delete(key)
}

func (s *BaseState) storeSkipToEpoch(epoch uint64) error {
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, epoch)
	return s.db.Set(skipToKey, buf)
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
	return s.db.Set(firstSlotKey, buf)
}

func (s *BaseState) loadFirstSlot() (uint64, error) {
	data, err := s.db.Get(firstSlotKey)
	if err != nil {
		return 0, err
	}

	return binary.LittleEndian.Uint64(data), nil
}

func (s *BaseState) storeEpochLength(l uint64) error {
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, l)
	return s.db.Set(epochLengthKey, buf)
}

func (s *BaseState) loadEpochLength() (uint64, error) {
	data, err := s.db.Get(epochLengthKey)
	if err != nil {
		return 0, err
	}

	return binary.LittleEndian.Uint64(data), nil
}

func (s *BaseState) storeSlotDuration(duration uint64) error {
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, duration)
	return s.db.Set(slotDurationKey, buf)
}

func (s *BaseState) loadSlotDuration() (uint64, error) {
	data, err := s.db.Get(slotDurationKey)
	if err != nil {
		return 0, err
	}

	return binary.LittleEndian.Uint64(data), nil
}
