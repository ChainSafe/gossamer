// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package state

import (
	"encoding/binary"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/ChainSafe/chaindb"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

var (
	ErrConfigNotFound     = errors.New("config data not found")
	ErrEpochNotInMemory   = errors.New("epoch not found in memory map")
	errHashNotInMemory    = errors.New("hash not found in memory map")
	errEpochNotInDatabase = errors.New("epoch data not found in the database")
	errHashNotPersisted   = errors.New("hash with next epoch not found in database")
	errNoPreRuntimeDigest = errors.New("header does not contain pre-runtime digest")
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
	db          GetPutter
	baseState   *BaseState
	blockState  *BlockState
	epochLength uint64 // measured in slots
	skipToEpoch uint64

	nextEpochDataLock sync.RWMutex
	// nextEpochData follows the format map[epoch]map[block hash]next epoch data
	nextEpochData nextEpochMap[types.NextEpochData]

	nextConfigDataLock sync.RWMutex
	// nextConfigData follows the format map[epoch]map[block hash]next config data
	nextConfigData nextEpochMap[types.NextConfigDataV1]
}

// NewEpochStateFromGenesis returns a new EpochState given information for the first epoch, fetched from the runtime
func NewEpochStateFromGenesis(db *chaindb.BadgerDB, blockState *BlockState,
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
		nextEpochData:  make(nextEpochMap[types.NextEpochData]),
		nextConfigData: make(nextEpochMap[types.NextConfigDataV1]),
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
func NewEpochState(db *chaindb.BadgerDB, blockState *BlockState) (*EpochState, error) {
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
		nextEpochData:  make(nextEpochMap[types.NextEpochData]),
		nextConfigData: make(nextEpochMap[types.NextConfigDataV1]),
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
		digestValue, err := d.Value()
		if err != nil {
			continue
		}
		predigest, ok := digestValue.(types.PreRuntimeDigest)
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

	return 0, errNoPreRuntimeDigest
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

// GetEpochData returns the epoch data for a given epoch persisted in database
// otherwise will try to get the data from the in-memory map using the header
// if the header params is nil then it will search only in database
func (s *EpochState) GetEpochData(epoch uint64, header *types.Header) (*types.EpochData, error) {
	epochData, err := s.getEpochDataFromDatabase(epoch)
	if err != nil && !errors.Is(err, chaindb.ErrKeyNotFound) {
		return nil, fmt.Errorf("failed to retrieve epoch data from database: %w", err)
	}

	if epochData != nil {
		return epochData, nil
	}

	if header == nil {
		return nil, errEpochNotInDatabase
	}

	s.nextEpochDataLock.RLock()
	defer s.nextEpochDataLock.RUnlock()

	inMemoryEpochData, err := s.nextEpochData.Retrieve(s.blockState, epoch, header)
	if err != nil {
		return nil, fmt.Errorf("failed to get epoch data from memory: %w", err)
	}

	epochData, err = inMemoryEpochData.ToEpochData()
	if err != nil {
		return nil, fmt.Errorf("cannot transform into epoch data: %w", err)
	}

	return epochData, nil
}

// getEpochDataFromDatabase returns the epoch data for a given epoch persisted in database
func (s *EpochState) getEpochDataFromDatabase(epoch uint64) (*types.EpochData, error) {
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

// GetLatestEpochData returns the EpochData for the current epoch
func (s *EpochState) GetLatestEpochData() (*types.EpochData, error) {
	curr, err := s.GetCurrentEpoch()
	if err != nil {
		return nil, err
	}

	return s.GetEpochData(curr, nil)
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

// GetConfigData returns the newest config data for a given epoch persisted in database
// otherwise tries to get the data from the in-memory map using the header. If we don't
// find any config data for the current epoch we lookup in the previous epochs, as the spec says:
// - The supplied configuration data are intended to be used from the next epoch onwards.
// If the header params is nil then it will search only in the database.
func (s *EpochState) GetConfigData(epoch uint64, header *types.Header) (configData *types.ConfigData, err error) {
	for tryEpoch := int(epoch); tryEpoch >= 0; tryEpoch-- {
		configData, err = s.getConfigDataFromDatabase(uint64(tryEpoch))
		if err != nil && !errors.Is(err, chaindb.ErrKeyNotFound) {
			return nil, fmt.Errorf("failed to retrieve config epoch from database: %w", err)
		}

		if configData != nil {
			return configData, nil
		}

		// there is no config data for the `tryEpoch` on database and we don't have a
		// header to lookup in the memory map, so let's go retrieve the previous epoch
		if header == nil {
			continue
		}

		// we will check in the memory map and if we don't find the data
		// then we continue searching through the previous epoch
		s.nextConfigDataLock.RLock()
		inMemoryConfigData, err := s.nextConfigData.Retrieve(s.blockState, uint64(tryEpoch), header)
		s.nextConfigDataLock.RUnlock()

		if errors.Is(err, ErrEpochNotInMemory) {
			continue
		} else if err != nil {
			return nil, fmt.Errorf("failed to get config data from memory: %w", err)
		}

		return inMemoryConfigData.ToConfigData(), err
	}

	return nil, fmt.Errorf("%w: epoch %d", ErrConfigNotFound, epoch)
}

// getConfigDataFromDatabase returns the BABE config data for a given epoch persisted in database
func (s *EpochState) getConfigDataFromDatabase(epoch uint64) (*types.ConfigData, error) {
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

type nextEpochMap[T types.NextEpochData | types.NextConfigDataV1] map[uint64]map[common.Hash]T

func (nem nextEpochMap[T]) Retrieve(blockState *BlockState, epoch uint64, header *types.Header) (*T, error) {
	atEpoch, has := nem[epoch]
	if !has {
		return nil, fmt.Errorf("%w: %d", ErrEpochNotInMemory, epoch)
	}

	headerHash := header.Hash()
	for hash, value := range atEpoch {
		isDescendant, err := blockState.IsDescendantOf(hash, headerHash)

		// sometimes while moving to the next epoch is possible the header
		// is not fully imported by the blocktree, in this case we will use
		// its parent header which migth be already imported.
		if errors.Is(err, chaindb.ErrKeyNotFound) {
			parentHeader, err := blockState.GetHeader(header.ParentHash)
			if err != nil {
				return nil, fmt.Errorf("cannot get parent header: %w", err)
			}

			return nem.Retrieve(blockState, epoch, parentHeader)
		}

		if err != nil {
			return nil, fmt.Errorf("cannot verify the ancestry: %w", err)
		}

		if isDescendant {
			return &value, nil
		}
	}

	return nil, fmt.Errorf("%w: %s", errHashNotInMemory, headerHash)
}

// GetLatestConfigData returns the most recently set ConfigData
func (s *EpochState) GetLatestConfigData() (*types.ConfigData, error) {
	b, err := s.db.Get(latestConfigDataKey)
	if err != nil {
		return nil, err
	}

	epoch := binary.LittleEndian.Uint64(b)
	return s.GetConfigData(epoch, nil)
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

	if header.Number >= 1 {
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

	if epoch < s.skipToEpoch {
		return true, nil
	}

	return false, nil
}

// StoreBABENextEpochData stores the types.NextEpochData under epoch and hash keys
func (s *EpochState) StoreBABENextEpochData(epoch uint64, hash common.Hash, nextEpochData types.NextEpochData) {
	s.nextEpochDataLock.Lock()
	defer s.nextEpochDataLock.Unlock()

	_, has := s.nextEpochData[epoch]
	if !has {
		s.nextEpochData[epoch] = make(map[common.Hash]types.NextEpochData)
	}
	s.nextEpochData[epoch][hash] = nextEpochData
}

// StoreBABENextConfigData stores the types.NextConfigData under epoch and hash keys
func (s *EpochState) StoreBABENextConfigData(epoch uint64, hash common.Hash, nextConfigData types.NextConfigDataV1) {
	s.nextConfigDataLock.Lock()
	defer s.nextConfigDataLock.Unlock()

	_, has := s.nextConfigData[epoch]
	if !has {
		s.nextConfigData[epoch] = make(map[common.Hash]types.NextConfigDataV1)
	}
	s.nextConfigData[epoch][hash] = nextConfigData
}

// FinalizeBABENextEpochData stores the right types.NextEpochData by
// getting the set of hashes from the received epoch and for each hash
// check if the header is in the database then it's been finalized and
// thus we can also set the corresponding EpochData in the database
func (s *EpochState) FinalizeBABENextEpochData(finalizedHeader *types.Header) error {
	if finalizedHeader.Number == 0 {
		return nil
	}

	s.nextEpochDataLock.Lock()
	defer s.nextEpochDataLock.Unlock()

	var nextEpoch uint64 = 1
	if finalizedHeader.Number != 0 {
		finalizedBlockEpoch, err := s.GetEpochForBlock(finalizedHeader)
		if err != nil {
			return fmt.Errorf("cannot get epoch for block %d (%s): %w",
				finalizedHeader.Number, finalizedHeader.Hash(), err)
		}

		nextEpoch = finalizedBlockEpoch + 1
	}

	epochInDatabase, err := s.getEpochDataFromDatabase(nextEpoch)

	// if an error occurs and the error is chaindb.ErrKeyNotFound we ignore
	// since this error is what we will handle in the next lines
	if err != nil && !errors.Is(err, chaindb.ErrKeyNotFound) {
		return fmt.Errorf("cannot check if next epoch data is already defined for epoch %d: %w", nextEpoch, err)
	}

	// epoch data already defined we don't need to lookup in the map
	if epochInDatabase != nil {
		return nil
	}

	finalizedNextEpochData, err := findFinalizedHeaderForEpoch(s.nextEpochData, s, nextEpoch)
	if err != nil {
		return fmt.Errorf("cannot find next epoch data: %w", err)
	}

	ed, err := finalizedNextEpochData.ToEpochData()
	if err != nil {
		return fmt.Errorf("cannot transform epoch data: %w", err)
	}

	err = s.SetEpochData(nextEpoch, ed)
	if err != nil {
		return fmt.Errorf("cannot set epoch data: %w", err)
	}

	// remove previous epochs from the memory
	for e := range s.nextEpochData {
		if e <= nextEpoch {
			delete(s.nextEpochData, e)
		}
	}

	return nil
}

// FinalizeBABENextConfigData stores the right types.NextConfigData by
// getting the set of hashes from the received epoch and for each hash
// check if the header is in the database then it's been finalized and
// thus we can also set the corresponding NextConfigData in the database
func (s *EpochState) FinalizeBABENextConfigData(finalizedHeader *types.Header) error {
	if finalizedHeader.Number == 0 {
		return nil
	}

	s.nextConfigDataLock.Lock()
	defer s.nextConfigDataLock.Unlock()

	var nextEpoch uint64 = 1
	if finalizedHeader.Number != 0 {
		finalizedBlockEpoch, err := s.GetEpochForBlock(finalizedHeader)
		if err != nil {
			return fmt.Errorf("cannot get epoch for block %d (%s): %w",
				finalizedHeader.Number, finalizedHeader.Hash(), err)
		}

		nextEpoch = finalizedBlockEpoch + 1
	}

	configInDatabase, err := s.getConfigDataFromDatabase(nextEpoch)

	// if an error occurs and the error is chaindb.ErrKeyNotFound we ignore
	// since this error is what we will handle in the next lines
	if err != nil && !errors.Is(err, chaindb.ErrKeyNotFound) {
		return fmt.Errorf("cannot check if next epoch config is already defined for epoch %d: %w", nextEpoch, err)
	}

	// config data already defined we don't need to lookup in the map
	if configInDatabase != nil {
		return nil
	}

	// not every epoch will have `ConfigData`
	finalizedNextConfigData, err := findFinalizedHeaderForEpoch(s.nextConfigData, s, nextEpoch)
	if errors.Is(err, ErrEpochNotInMemory) {
		logger.Debugf("config data for epoch %d not found in memory", nextEpoch)
		return nil
	} else if err != nil {
		return fmt.Errorf("cannot find next config data: %w", err)
	}

	cd := finalizedNextConfigData.ToConfigData()
	err = s.SetConfigData(nextEpoch, cd)
	if err != nil {
		return fmt.Errorf("cannot set config data: %w", err)
	}

	// remove previous epochs from the memory
	for e := range s.nextConfigData {
		if e <= nextEpoch {
			delete(s.nextConfigData, e)
		}
	}

	return nil
}

// findFinalizedHeaderForEpoch given a specific epoch (the key) will go through the hashes looking
// for a database persisted hash (belonging to the finalized chain)
// which contains the right configuration or data to be persisted and safely used
func findFinalizedHeaderForEpoch[T types.NextConfigDataV1 | types.NextEpochData](
	nextEpochMap map[uint64]map[common.Hash]T, es *EpochState, epoch uint64) (next *T, err error) {
	hashes, has := nextEpochMap[epoch]
	if !has {
		return nil, ErrEpochNotInMemory
	}

	for hash, inMemory := range hashes {
		persisted, err := es.blockState.HasHeaderInDatabase(hash)
		if err != nil {
			return nil, fmt.Errorf("failed to check header exists in database: %w", err)
		}

		if !persisted {
			continue
		}

		return &inMemory, nil
	}

	return nil, errHashNotPersisted
}
