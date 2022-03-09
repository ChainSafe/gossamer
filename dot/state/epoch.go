// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package state

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/ChainSafe/chaindb"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

var (
	epochPrefix         = "epoch"
	epochLengthKey      = []byte("epochlength")
	currentEpochKey     = []byte("current")
	firstSlotKey        = []byte("firstslot")
	slotDurationKey     = []byte("slotduration")
	epochDataPrefix     = []byte("epochinfo")
	configDataPrefix    = []byte("configinfo")
	latestConfigDataKey = []byte("lcfginfo")
	skipToKey           = []byte("skipto")
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
	baseState   *BaseState
	blockState  *BlockState
	epochLength uint64 // measured in slots
	skipToEpoch uint64

	nextEpochLock  sync.RWMutex
	nextEpochData  map[uint64]map[common.Hash]types.NextEpochData
	nextConfigData map[uint64]map[common.Hash]types.NextConfigData
}

// NewEpochStateFromGenesis returns a new EpochState given information for the first epoch, fetched from the runtime
func NewEpochStateFromGenesis(db chaindb.Database, blockState *BlockState,
	genesisConfig *types.BabeConfiguration) (*EpochState, error) {
	baseState := NewBaseState(db)

	err := baseState.storeFirstSlot(1) // this may change once the first block is imported
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
		baseState:      NewBaseState(db),
		blockState:     blockState,
		db:             epochDB,
		epochLength:    genesisConfig.EpochLength,
		nextEpochData:  make(map[uint64]map[common.Hash]types.NextEpochData),
		nextConfigData: make(map[uint64]map[common.Hash]types.NextConfigData),
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

	if err = s.baseState.storeEpochLength(genesisConfig.EpochLength); err != nil {
		return nil, err
	}

	if err = s.baseState.storeSlotDuration(genesisConfig.SlotDuration); err != nil {
		return nil, err
	}

	if err := s.baseState.storeSkipToEpoch(0); err != nil {
		return nil, err
	}

	return s, nil
}

// NewEpochState returns a new EpochState
func NewEpochState(db chaindb.Database, blockState *BlockState) (*EpochState, error) {
	baseState := NewBaseState(db)

	epochLength, err := baseState.loadEpochLength()
	if err != nil {
		return nil, err
	}

	skipToEpoch, err := baseState.loadSkipToEpoch()
	if err != nil {
		return nil, err
	}

	return &EpochState{
		baseState:      baseState,
		blockState:     blockState,
		db:             chaindb.NewTable(db, epochPrefix),
		epochLength:    epochLength,
		skipToEpoch:    skipToEpoch,
		nextEpochData:  make(map[uint64]map[common.Hash]types.NextEpochData),
		nextConfigData: make(map[uint64]map[common.Hash]types.NextConfigData),
	}, nil
}

// GetEpochLength returns the length of an epoch in slots
func (s *EpochState) GetEpochLength() (uint64, error) {
	return s.baseState.loadEpochLength()
}

// GetSlotDuration returns the duration of a slot
func (s *EpochState) GetSlotDuration() (time.Duration, error) {
	d, err := s.baseState.loadSlotDuration()
	if err != nil {
		return 0, err
	}

	return time.ParseDuration(fmt.Sprintf("%dms", d))
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

	firstSlot, err := s.baseState.loadFirstSlot()
	if err != nil {
		return 0, err
	}

	for _, d := range header.Digest.Types {
		predigest, ok := d.Value().(types.PreRuntimeDigest)
		if !ok {
			continue
		}

		digest, err := types.DecodeBabePreDigest(predigest.Data)
		if err != nil {
			return 0, fmt.Errorf("failed to decode babe header: %w", err)
		}

		var slotNumber uint64
		switch d := digest.(type) {
		case types.BabePrimaryPreDigest:
			slotNumber = d.SlotNumber
		case types.BabeSecondaryVRFPreDigest:
			slotNumber = d.SlotNumber
		case types.BabeSecondaryPlainPreDigest:
			slotNumber = d.SlotNumber
		}

		if slotNumber < firstSlot {
			return 0, nil
		}

		return (slotNumber - firstSlot) / s.epochLength, nil
	}

	return 0, errors.New("header does not contain pre-runtime digest")
}

// SetEpochData sets the epoch data for a given epoch
func (s *EpochState) SetEpochData(epoch uint64, info *types.EpochData) error {
	raw := info.ToEpochDataRaw()

	enc, err := scale.Marshal(*raw)
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

	raw := &types.EpochDataRaw{}
	err = scale.Unmarshal(enc, raw)
	if err != nil {
		return nil, err
	}

	return raw.ToEpochData()
}

func (s *EpochState) GetEpochDataForHeader(epoch uint64, header *types.Header) (*types.EpochData, error) {
	s.nextEpochLock.RLock()
	defer s.nextEpochLock.RUnlock()

	atEpoch, has := s.nextEpochData[epoch]
	if !has {
		return nil, fmt.Errorf("epoch %d not found in memory stored epoch data", epoch)
	}

	headerHash := header.Hash()

	for hash, value := range atEpoch {
		isDescendant, err := s.blockState.IsDescendantOf(hash, headerHash)
		if err != nil {
			return nil, fmt.Errorf("cannot verify the ancestry: %w", err)
		}

		if isDescendant {
			return value.ToEpochData()
		}
	}

	return nil, fmt.Errorf("cannot find epoch data for header %s", headerHash)
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
	has, err := s.db.Has(epochDataKey(epoch))
	if errors.Is(chaindb.ErrKeyNotFound, err) {
		s.nextEpochLock.Lock()
		defer s.nextEpochLock.Unlock()

		_, has = s.nextEpochData[epoch]
		return has, nil
	}

	return has, err
}

// SetConfigData sets the BABE config data for a given epoch
func (s *EpochState) SetConfigData(epoch uint64, info *types.ConfigData) error {
	enc, err := scale.Marshal(*info)
	if err != nil {
		return err
	}

	// this assumes the most recently set config data is the highest on the chain
	if err = s.setLatestConfigData(epoch); err != nil {
		return err
	}

	return s.db.Put(configDataKey(epoch), enc)
}

func (s *EpochState) setLatestConfigData(epoch uint64) error {
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, epoch)
	return s.db.Put(latestConfigDataKey, buf)
}

// GetConfigData returns the BABE config data for a given epoch
func (s *EpochState) GetConfigData(epoch uint64) (*types.ConfigData, error) {
	enc, err := s.db.Get(configDataKey(epoch))
	if err != nil {
		return nil, err
	}

	info := &types.ConfigData{}
	err = scale.Unmarshal(enc, info)
	if err != nil {
		return nil, err
	}

	return info, nil
}

func (s *EpochState) GetConfigDataForHeader(epoch uint64, header *types.Header) (*types.ConfigData, error) {
	s.nextEpochLock.RLock()
	defer s.nextEpochLock.RUnlock()

	atEpoch, has := s.nextConfigData[epoch]
	if !has {
		return nil, fmt.Errorf("epoch %d not found in memory stored config data", epoch)
	}

	headerHash := header.Hash()

	for hash, value := range atEpoch {
		isDescendant, err := s.blockState.IsDescendantOf(hash, headerHash)
		if err != nil {
			return nil, fmt.Errorf("cannot verify the ancestry: %w", err)
		}

		if isDescendant {
			return value.ToConfigData(), nil
		}
	}

	return nil, fmt.Errorf("cannot find config data for header %s", headerHash)
}

// GetLatestConfigData returns the most recently set ConfigData
func (s *EpochState) GetLatestConfigData() (*types.ConfigData, error) {
	b, err := s.db.Get(latestConfigDataKey)
	if err != nil {
		return nil, err
	}

	epoch := binary.LittleEndian.Uint64(b)
	return s.GetConfigData(epoch)
}

// HasConfigData returns whether config data exists for a given epoch
func (s *EpochState) HasConfigData(epoch uint64) (bool, error) {
	has, err := s.db.Has(configDataKey(epoch))
	if errors.Is(chaindb.ErrKeyNotFound, err) {
		s.nextEpochLock.Lock()
		defer s.nextEpochLock.Unlock()

		_, has := s.nextConfigData[epoch]
		return has, nil
	}

	return has, err
}

// GetStartSlotForEpoch returns the first slot in the given epoch.
// If 0 is passed as the epoch, it returns the start slot for the current epoch.
func (s *EpochState) GetStartSlotForEpoch(epoch uint64) (uint64, error) {
	firstSlot, err := s.baseState.loadFirstSlot()
	if err != nil {
		return 0, err
	}

	return s.epochLength*epoch + firstSlot, nil
}

// GetEpochFromTime returns the epoch for a given time
func (s *EpochState) GetEpochFromTime(t time.Time) (uint64, error) {
	slotDuration, err := s.GetSlotDuration()
	if err != nil {
		return 0, err
	}

	firstSlot, err := s.baseState.loadFirstSlot()
	if err != nil {
		return 0, err
	}

	slot := uint64(t.UnixNano()) / uint64(slotDuration.Nanoseconds())

	if slot < firstSlot {
		return 0, errors.New("given time is before network start")
	}

	return (slot - firstSlot) / s.epochLength, nil
}

// SetFirstSlot sets the first slot number of the network
func (s *EpochState) SetFirstSlot(slot uint64) error {
	// check if block 1 was finalised already; if it has, don't set first slot again
	header, err := s.blockState.GetHighestFinalisedHeader()
	if err != nil {
		return err
	}

	if header.Number.Cmp(big.NewInt(1)) > -1 {
		return errors.New("first slot has already been set")
	}

	return s.baseState.storeFirstSlot(slot)
}

// SkipVerify returns whether verification for the given header should be skipped or not.
// Only used in the case of imported state.
func (s *EpochState) SkipVerify(header *types.Header) (bool, error) {
	epoch, err := s.GetEpochForBlock(header)
	if err != nil {
		return false, err
	}

	if epoch <= s.skipToEpoch {
		return true, nil
	}

	return false, nil
}

func (b *EpochState) StoreBABENextEpochData(epoch uint64, hash common.Hash, val types.NextEpochData) {
	b.nextEpochLock.Lock()
	defer b.nextEpochLock.Unlock()

	_, has := b.nextEpochData[epoch]
	if !has {
		b.nextEpochData[epoch] = make(map[common.Hash]types.NextEpochData)
	}
	b.nextEpochData[epoch][hash] = val
}

func (b *EpochState) StoreBABENextConfigData(epoch uint64, hash common.Hash, val types.NextConfigData) {
	b.nextEpochLock.Lock()
	defer b.nextEpochLock.Unlock()

	_, has := b.nextConfigData[epoch]
	if !has {
		b.nextConfigData[epoch] = make(map[common.Hash]types.NextConfigData)
	}
	b.nextConfigData[epoch][hash] = val
}

func (b *EpochState) GetBABENextEpochDataToFinalize(epoch uint64, hash common.Hash) (types.NextEpochData, bool) {
	b.nextEpochLock.RLock()
	defer b.nextEpochLock.RUnlock()

	epochData, has := b.nextEpochData[epoch]
	if !has {
		return types.NextEpochData{}, false
	}

	nextEpochData, has := epochData[hash]
	if has {
		delete(b.nextConfigData, epoch)
	}
	return nextEpochData, has
}

func (b *EpochState) GetBABENextConfigDataToFinalize(epoch uint64, hash common.Hash) (types.NextConfigData, bool) {
	b.nextEpochLock.RLock()
	defer b.nextEpochLock.RUnlock()

	epochData, has := b.nextConfigData[epoch]
	if !has {
		return types.NextConfigData{}, false
	}

	nextConfigData, has := epochData[hash]
	if has {
		delete(b.nextConfigData, epoch)
	}

	return nextConfigData, has
}
