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
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/ChainSafe/chaindb"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/scale"
)

var (
	epochPrefix      = "epoch"
	epochLengthKey   = []byte("epochlength")
	currentEpochKey  = []byte("current")
	firstSlotKey     = []byte("firstslot")
	epochDataPrefix  = []byte("epochinfo")
	configDataPrefix = []byte("configinfo")
)

func epochDataKey(epoch uint64) []byte {
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, epoch)
	return append(epochDataPrefix, buf...)
}

func configDataKey(epoch uint64) []byte {
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, epoch)
	return append(configDataPrefix, buf...)
}

// EpochState tracks information related to each epoch
type EpochState struct {
	baseDB      chaindb.Database
	db          chaindb.Database
	epochLength uint64 // measured in slots
	firstSlot   uint64
}

// NewEpochStateFromGenesis returns a new EpochState given information for the first epoch, fetched from the runtime
func NewEpochStateFromGenesis(db chaindb.Database, genesisConfig *types.BabeConfiguration) (*EpochState, error) {
	err := storeFirstSlot(db, 1) // this may change once the first block is imported
	if err != nil {
		return nil, err
	}

	epochDB := chaindb.NewTable(db, epochPrefix)
	err = epochDB.Put(currentEpochKey, []byte{0, 0, 0, 0, 0, 0, 0, 0})
	if err != nil {
		return nil, err
	}

	if genesisConfig.EpochLength == 0 {
		return nil, errors.New("epoch length is 0")
	}

	s := &EpochState{
		baseDB:      db,
		db:          epochDB,
		epochLength: genesisConfig.EpochLength,
		firstSlot:   1,
	}

	auths, err := types.BABEAuthorityRawToAuthority(genesisConfig.GenesisAuthorities)
	if err != nil {
		return nil, err
	}

	err = s.SetEpochData(0, &types.EpochData{
		Authorities: auths,
		Randomness:  genesisConfig.Randomness,
	})
	if err != nil {
		return nil, err
	}

	err = s.SetConfigData(0, &types.ConfigData{
		C1:             genesisConfig.C1,
		C2:             genesisConfig.C2,
		SecondarySlots: genesisConfig.SecondarySlots,
	})
	if err != nil {
		return nil, err
	}

	err = storeEpochLength(db, genesisConfig.EpochLength)
	if err != nil {
		return nil, err
	}

	return s, nil
}

// NewEpochState returns a new EpochState
func NewEpochState(db chaindb.Database) (*EpochState, error) {
	epochLength, err := loadEpochLength(db)
	if err != nil {
		return nil, err
	}

	firstSlot, err := loadFirstSlot(db)
	if err != nil {
		return nil, err
	}

	return &EpochState{
		baseDB:      db,
		db:          chaindb.NewTable(db, epochPrefix),
		epochLength: epochLength,
		firstSlot:   firstSlot,
	}, nil
}

func storeEpochLength(db chaindb.Database, l uint64) error {
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, l)
	return db.Put(epochLengthKey, buf)
}

func loadEpochLength(db chaindb.Database) (uint64, error) {
	data, err := db.Get(epochLengthKey)
	if err != nil {
		return 0, err
	}

	return binary.LittleEndian.Uint64(data), nil
}

func storeFirstSlot(db chaindb.Database, slot uint64) error {
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, slot)
	return db.Put(firstSlotKey, buf)
}

func loadFirstSlot(db chaindb.Database) (uint64, error) {
	data, err := db.Get(firstSlotKey)
	if err != nil {
		return 0, err
	}

	return binary.LittleEndian.Uint64(data), nil
}

// SetCurrentEpoch sets the current epoch
func (s *EpochState) SetCurrentEpoch(epoch uint64) error {
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, epoch)
	return s.db.Put(currentEpochKey, buf)
}

// GetCurrentEpoch returns the current epoch
func (s *EpochState) GetCurrentEpoch() (uint64, error) {
	b, err := s.db.Get(currentEpochKey)
	if err != nil {
		return 0, err
	}

	return binary.LittleEndian.Uint64(b), nil
}

// GetEpochForBlock checks the pre-runtime digest to determine what epoch the block was formed in.
func (s *EpochState) GetEpochForBlock(header *types.Header) (uint64, error) {
	if header == nil {
		return 0, errors.New("header is nil")
	}

	for _, d := range header.Digest {
		if d.Type() != types.PreRuntimeDigestType {
			continue
		}

		predigest := d.(*types.PreRuntimeDigest)

		r := &bytes.Buffer{}
		_, _ = r.Write(predigest.Data)
		digest, err := types.DecodeBabePreDigest(r)
		if err != nil {
			return 0, fmt.Errorf("failed to decode babe header: %w", err)
		}

		return (digest.SlotNumber() - s.firstSlot) / s.epochLength, nil
	}

	return 0, errors.New("header does not contain pre-runtime digest")
}

// SetEpochData sets the epoch data for a given epoch
func (s *EpochState) SetEpochData(epoch uint64, info *types.EpochData) error {
	raw := info.ToEpochDataRaw()

	enc, err := scale.Encode(raw)
	if err != nil {
		return err
	}

	return s.db.Put(epochDataKey(epoch), enc)
}

// GetEpochData returns the epoch data for a given epoch
func (s *EpochState) GetEpochData(epoch uint64) (*types.EpochData, error) {
	enc, err := s.db.Get(epochDataKey(epoch))
	if err != nil {
		return nil, err
	}

	info, err := scale.Decode(enc, &types.EpochDataRaw{})
	if err != nil {
		return nil, err
	}

	raw, ok := info.(*types.EpochDataRaw)
	if !ok {
		return nil, errors.New("failed to decode raw epoch data")
	}

	return raw.ToEpochData()
}

// GetLatestEpochData returns the EpochData for the current epoch
func (s *EpochState) GetLatestEpochData() (*types.EpochData, error) {
	curr, err := s.GetCurrentEpoch()
	if err != nil {
		return nil, err
	}

	return s.GetEpochData(curr)
}

// HasEpochData returns whether epoch data exists for a given epoch
func (s *EpochState) HasEpochData(epoch uint64) (bool, error) {
	return s.db.Has(epochDataKey(epoch))
}

// SetConfigData sets the BABE config data for a given epoch
func (s *EpochState) SetConfigData(epoch uint64, info *types.ConfigData) error {
	enc, err := scale.Encode(info)
	if err != nil {
		return err
	}

	return s.db.Put(configDataKey(epoch), enc)
}

// GetConfigData returns the BABE config data for a given epoch
func (s *EpochState) GetConfigData(epoch uint64) (*types.ConfigData, error) {
	enc, err := s.db.Get(configDataKey(epoch))
	if err != nil {
		return nil, err
	}

	info, err := scale.Decode(enc, new(types.ConfigData))
	if err != nil {
		return nil, err
	}

	return info.(*types.ConfigData), nil
}

// HasConfigData returns whether config data exists for a given epoch
func (s *EpochState) HasConfigData(epoch uint64) (bool, error) {
	return s.db.Has(configDataKey(epoch))
}

// GetStartSlotForEpoch returns the first slot in the given epoch.
// If 0 is passed as the epoch, it returns the start slot for the current epoch.
func (s *EpochState) GetStartSlotForEpoch(epoch uint64) (uint64, error) {
	return s.epochLength*epoch + s.firstSlot, nil
}

// SetFirstSlot sets the first slot number of the network
func (s *EpochState) SetFirstSlot(slot uint64) error {
	s.firstSlot = slot
	return storeFirstSlot(s.baseDB, slot)
}
