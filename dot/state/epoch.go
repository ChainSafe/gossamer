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
	"errors"

	"github.com/ChainSafe/chaindb"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/scale"
)

var (
	epochPrefix      = "epoch"
	epochLengthKey   = []byte("epochlength")
	currentEpochKey  = []byte("current")
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
	db          chaindb.Database
	epochLength uint64 // measured in slots
}

// NewEpochStateFromGenesis returns a new EpochState given information for the first epoch, fetched from the runtime
func NewEpochStateFromGenesis(db chaindb.Database, genesisConfig *types.BabeConfiguration) (*EpochState, error) {
	epochDB := chaindb.NewTable(db, epochPrefix)
	err := epochDB.Put(currentEpochKey, []byte{1, 0, 0, 0, 0, 0, 0, 0})
	if err != nil {
		return nil, err
	}

	if genesisConfig.EpochLength == 0 {
		return nil, errors.New("epoch length is 0")
	}

	s := &EpochState{
		db:          epochDB,
		epochLength: genesisConfig.EpochLength,
	}

	auths, err := types.BABEAuthorityRawToAuthority(genesisConfig.GenesisAuthorities)
	if err != nil {
		return nil, err
	}

	err = s.SetEpochData(1, &types.EpochData{
		Authorities: auths,
		Randomness:  genesisConfig.Randomness,
	})
	if err != nil {
		return nil, err
	}

	err = s.SetConfigData(1, &types.ConfigData{
		C1:             genesisConfig.C1,
		C2:             genesisConfig.C2,
		SecondarySlots: genesisConfig.SecondarySlots,
	})
	if err != nil {
		return nil, err
	}

	err = storeEpochLength(db, genesisConfig.EpochLength)
	logger.Crit("NewEpochStateFromGenesis", "epochLength", s.epochLength)

	return s, nil
}

// NewEpochState returns a new EpochState
func NewEpochState(db chaindb.Database) (*EpochState, error) {
	epochLength, err := loadEpochLength(db)
	if err != nil {
		return nil, err
	}
	return &EpochState{
		db:          chaindb.NewTable(db, epochPrefix),
		epochLength: epochLength,
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

func (s *EpochState) GetEpochLength() uint64 {
	return s.epochLength
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
		if len(d) == 0 {
			continue
		}

		if d[0] != types.PreRuntimeDigestType {
			continue
		}

		di, err := types.DecodeDigestItem(d)
		if err != nil {
			return 0, err
		}

		predigest := di.(*types.PreRuntimeDigest)

		babeHeader := new(types.BabeHeader)
		err = babeHeader.Decode(predigest.Data)
		if err != nil {
			return 0, err
		}

		logger.Crit("GetEpochForBlock", "epochLength", s.epochLength)
		return (babeHeader.SlotNumber / s.epochLength) + 1, nil
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

// GetEpochInfo returns the epoch data for a given epoch
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

// HasEpochData returns whether config data exists for a given epoch
func (s *EpochState) HasConfigData(epoch uint64) (bool, error) {
	return s.db.Has(configDataKey(epoch))
}

// GetStartSlotForEpoch returns the first slot in the given epoch.
// If 0 is passed as the epoch, it returns the start slot for the current epoch.
func (s *EpochState) GetStartSlotForEpoch(epoch uint64) (uint64, error) {
	curr, err := s.GetCurrentEpoch()
	if err != nil {
		return 0, nil
	}

	if epoch == 0 {
		// epoch 0 doesn't exist, use 0 for latest epoch
		epoch = curr
	}

	if epoch == 1 {
		return 1, nil
	}

	return s.epochLength*(epoch-1) + 1, nil
}
