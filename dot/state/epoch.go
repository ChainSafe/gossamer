// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package state

import (
	"encoding/binary"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/internal/database"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

var (
	ErrConfigNotFound   = errors.New("config data not found")
	ErrEpochNotInMemory = errors.New("epoch not found in memory map")
	errEpochLengthCannotBeZero = errors.New("epoch length cannot be zero")
	errHashNotInMemory         = errors.New("hash not found in memory map")
	errEpochNotInDatabase      = errors.New("epoch data not found in the database")
	errHashNotPersisted        = errors.New("hash with next epoch not found in database")
	errNoFirstNonOriginBlock   = errors.New("no first non origin block")
)

var (
	epochPrefix         = "epoch"
	currentEpochKey     = []byte("current")
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

// GenesisEpochDescriptor is the informations provided by calling
// the genesis WASM runtime exported function `BabeAPIConfiguration`
type GenesisEpochDescriptor struct {
	EpochData  *types.EpochDataRaw
	ConfigData *types.ConfigData
}

// EpochState tracks information related to each epoch
type EpochState struct {
	db           GetPutter
	baseState    *BaseState
	blockState   *BlockState
	epochLength  uint64 // measured in slots
	slotDuration uint64
	skipToEpoch  uint64

	nextEpochDataLock sync.RWMutex
	// nextEpochData follows the format map[epoch]map[block hash]next epoch data
	nextEpochData nextEpochMap[types.NextEpochData]

	nextConfigDataLock sync.RWMutex
	// nextConfigData follows the format map[epoch]map[block hash]next config data
	nextConfigData nextEpochMap[types.NextConfigDataV1]

	genesisEpochDescriptor *GenesisEpochDescriptor
}

// NewEpochStateFromGenesis returns a new EpochState given information for the first epoch, fetched from the runtime
func NewEpochStateFromGenesis(db database.Database, blockState *BlockState,
	genesisConfig *types.BabeConfiguration) (*EpochState, error) {
	if genesisConfig.EpochLength == 0 {
		return nil, errEpochLengthCannotBeZero
	}

	s := &EpochState{
		baseState:      NewBaseState(db),
		blockState:     blockState,
		db:             database.NewTable(db, epochPrefix),
		epochLength:    genesisConfig.EpochLength,
		slotDuration:   genesisConfig.SlotDuration,
		nextEpochData:  make(nextEpochMap[types.NextEpochData]),
		nextConfigData: make(nextEpochMap[types.NextConfigDataV1]),

		genesisEpochDescriptor: &GenesisEpochDescriptor{
			EpochData: &types.EpochDataRaw{
				Authorities: genesisConfig.GenesisAuthorities,
				Randomness:  genesisConfig.Randomness,
			},
			ConfigData: &types.ConfigData{
				C1:             genesisConfig.C1,
				C2:             genesisConfig.C2,
				SecondarySlots: genesisConfig.SecondarySlots,
			},
		},
	}

	err := s.StoreCurrentEpoch(0)
	if err != nil {
		return nil, fmt.Errorf("storing current epoch")
	}

	err = s.setLatestConfigData(0)
	if err != nil {
		return nil, err
	}

	if err := s.baseState.storeSkipToEpoch(0); err != nil {
		return nil, err
	}

	return s, nil
}

// NewEpochState returns a new EpochState
func NewEpochState(db database.Database, blockState *BlockState,
	genesisConfig *types.BabeConfiguration) (*EpochState, error) {
	if genesisConfig.EpochLength == 0 {
		return nil, errEpochLengthCannotBeZero
	}

	baseState := NewBaseState(db)
	skipToEpoch, err := baseState.loadSkipToEpoch()
	if err != nil {
		return nil, err
	}

	return &EpochState{
		baseState:      baseState,
		blockState:     blockState,
		db:             database.NewTable(db, epochPrefix),
		epochLength:    genesisConfig.EpochLength,
		slotDuration:   genesisConfig.SlotDuration,
		skipToEpoch:    skipToEpoch,
		nextEpochData:  make(nextEpochMap[types.NextEpochData]),
		nextConfigData: make(nextEpochMap[types.NextConfigDataV1]),
		genesisEpochDescriptor: &GenesisEpochDescriptor{
			EpochData: &types.EpochDataRaw{
				Authorities: genesisConfig.GenesisAuthorities,
				Randomness:  genesisConfig.Randomness,
			},
			ConfigData: &types.ConfigData{
				C1:             genesisConfig.C1,
				C2:             genesisConfig.C2,
				SecondarySlots: genesisConfig.SecondarySlots,
			},
		},
	}, nil
}

// GetEpochLength returns the length of an epoch in slots
func (s *EpochState) GetEpochLength() uint64 {
	return s.epochLength
}

// GetSlotDuration returns the duration of a slot
func (s *EpochState) GetSlotDuration() (time.Duration, error) {
	return time.ParseDuration(fmt.Sprintf("%dms", s.slotDuration))
}

// StoreCurrentEpoch sets the current epoch
func (s *EpochState) StoreCurrentEpoch(epoch uint64) error {
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

	//  actually the epoch number for block number #1 is epoch 0,
	// epochs start from 0 and are incremented (almost, given that epochs might be skipped)
	// sequentially 0...1...2, so the block number #1 belongs to epoch 0
	if header.Number == 1 {
		return 0, nil
	}

	chainFirstSlotNumber, err := s.retrieveFirstNonOriginBlockSlot(header.Hash())
	if err != nil {
		return 0, fmt.Errorf("retrieving very first slot number: %w", err)
	}

	slotNumber, err := header.SlotNumber()
	if err != nil {
		return 0, fmt.Errorf("getting slot number: %w", err)
	}

	return (slotNumber - chainFirstSlotNumber) / s.epochLength, nil
}

// SetEpochDataRaw sets the epoch data raw for a given epoch
func (s *EpochState) SetEpochDataRaw(epoch uint64, raw *types.EpochDataRaw) error {
	enc, err := scale.Marshal(*raw)
	if err != nil {
		return err
	}

	return s.db.Put(epochDataKey(epoch), enc)
}

// GetEpochDataRaw returns the raw epoch data for a given epoch persisted in database
// otherwise will try to get the data from the in-memory map using the header
// if the header params is nil then it will search only in database
func (s *EpochState) GetEpochDataRaw(epoch uint64, header *types.Header) (*types.EpochDataRaw, error) {
	if epoch == 0 {
		return s.genesisEpochDescriptor.EpochData, nil
	}

	epochDataRaw, err := s.getEpochDataRawFromDatabase(epoch)
	if err != nil && !errors.Is(err, database.ErrNotFound) {
		return nil, fmt.Errorf("failed to retrieve epoch data from database: %w", err)
	}

	if epochDataRaw != nil {
		return epochDataRaw, nil
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

	return inMemoryEpochData.ToEpochDataRaw(), nil
}

// getEpochDataRawFromDatabase returns the epoch data for a given epoch persisted in database
func (s *EpochState) getEpochDataRawFromDatabase(epoch uint64) (*types.EpochDataRaw, error) {
	enc, err := s.db.Get(epochDataKey(epoch))
	if err != nil {
		return nil, err
	}

	raw := new(types.EpochDataRaw)
	err = scale.Unmarshal(enc, raw)
	if err != nil {
		return nil, fmt.Errorf("unmarshaling into epoch data raw: %w", err)
	}

	return raw, nil
}

// GetLatestEpochDataRaw returns the EpochData for the current epoch
func (s *EpochState) GetLatestEpochDataRaw() (*types.EpochDataRaw, error) {
	curr, err := s.GetCurrentEpoch()
	if err != nil {
		return nil, err
	}

	return s.GetEpochDataRaw(curr, nil)
}

// StoreConfigData sets the BABE config data for a given epoch
func (s *EpochState) StoreConfigData(epoch uint64, info *types.ConfigData) error {
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
		if tryEpoch == 0 {
			return s.genesisEpochDescriptor.ConfigData, nil
		}

		configData, err = s.getConfigDataFromDatabase(uint64(tryEpoch))
		if err != nil && !errors.Is(err, database.ErrNotFound) {
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

func (s *EpochState) HandleBABEDigest(header *types.Header, digest types.BabeConsensusDigest) error {
	headerHash := header.Hash()

	digestValue, err := digest.Value()
	if err != nil {
		return fmt.Errorf("getting digest value: %w", err)
	}
	switch val := digestValue.(type) {
	case types.NextEpochData:
		currEpoch, err := s.GetEpochForBlock(header)
		if err != nil {
			return fmt.Errorf("getting epoch for block %d (%s): %w",
				header.Number, headerHash, err)
		}

		nextEpoch := currEpoch + 1
		s.storeBABENextEpochData(nextEpoch, headerHash, val)
		logger.Debugf("stored BABENextEpochData data: %v for hash: %s to epoch: %d", digest, headerHash, nextEpoch)
		return nil

	case types.BABEOnDisabled:
		return nil

	case types.VersionedNextConfigData:
		nextConfigDataVersion, err := val.Value()
		if err != nil {
			return fmt.Errorf("getting digest value: %w", err)
		}

		switch nextConfigData := nextConfigDataVersion.(type) {
		case types.NextConfigDataV1:
			currEpoch, err := s.GetEpochForBlock(header)
			if err != nil {
				return fmt.Errorf("getting epoch for block %d (%s): %w", header.Number, headerHash, err)
			}
			nextEpoch := currEpoch + 1
			s.storeBABENextConfigData(nextEpoch, headerHash, nextConfigData)
			logger.Debugf("stored BABENextConfigData data: %v for hash: %s to epoch: %d", digest, headerHash, nextEpoch)
			return nil
		default:
			return fmt.Errorf("next config data version not supported: %T", nextConfigDataVersion)
		}
	}

	return errors.New("invalid consensus digest data")
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
		if errors.Is(err, database.ErrNotFound) {
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

// GetStartSlotForEpoch returns the first slot in the given epoch, this method receives
// the best block hash in order to discover the correct block
func (s *EpochState) GetStartSlotForEpoch(epoch uint64, bestBlockHash common.Hash) (uint64, error) {
	chainFirstSlotNumber, err := s.retrieveFirstNonOriginBlockSlot(bestBlockHash)
	if err != nil {
		if errors.Is(err, errNoFirstNonOriginBlock) {
			if epoch == 0 {
				slotDuration, err := s.GetSlotDuration()
				if err != nil {
					return 0, fmt.Errorf("getting slot duration: %w", err)
				}
				return uint64(time.Now().UnixNano()) / uint64(slotDuration.Nanoseconds()), nil
			}

			return 0, fmt.Errorf(
				"%w: first non origin block is needed for epoch %d",
				errNoFirstNonOriginBlock,
				epoch)
		}
		return 0, fmt.Errorf("retrieving first non origin block slot: %w", err)
	}
	return s.epochLength*epoch + chainFirstSlotNumber, nil
}

// retrieveFirstNonOriginBlockSlot returns the slot number of the very first non origin block
// if there is more than one first non origin block then it uses the block hash to check ancestry
// e.g to return the correct slot number for a specific fork
func (s *EpochState) retrieveFirstNonOriginBlockSlot(blockHash common.Hash) (uint64, error) {
	firstNonOriginHashes, err := s.blockState.GetHashesByNumber(1)
	if err != nil {
		return 0, fmt.Errorf("getting hashes using number 1: %w", err)
	}

	if len(firstNonOriginHashes) == 0 {
		return 0, errNoFirstNonOriginBlock
	}

	var firstNonOriginBlockHash common.Hash
	if len(firstNonOriginHashes) == 1 {
		firstNonOriginBlockHash = firstNonOriginHashes[0]
	} else {
		blockHeader, err := s.blockState.GetHeader(blockHash)
		if err != nil {
			return 0, fmt.Errorf("getting block by header: %w", err)
		}

		if blockHeader.Number == 1 {
			return blockHeader.SlotNumber()
		}

		for _, hash := range firstNonOriginHashes {
			isDescendant, err := s.blockState.IsDescendantOf(hash, blockHash)
			if err != nil {
				return 0, fmt.Errorf("while checking ancestry: %w", err)
			}

			if isDescendant {
				firstNonOriginBlockHash = hash
				break
			}
		}
	}

	firstNonGenesisHeader, err := s.blockState.GetHeader(firstNonOriginBlockHash)
	if err != nil {
		return 0, fmt.Errorf("getting first non genesis block by hash: %w", err)
	}

	chainFirstSlotNumber, err := firstNonGenesisHeader.SlotNumber()
	if err != nil {
		return 0, fmt.Errorf("getting slot number: %w", err)
	}

	return chainFirstSlotNumber, nil
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
func (s *EpochState) storeBABENextEpochData(epoch uint64, hash common.Hash, nextEpochData types.NextEpochData) {
	s.nextEpochDataLock.Lock()
	defer s.nextEpochDataLock.Unlock()

	_, has := s.nextEpochData[epoch]
	if !has {
		s.nextEpochData[epoch] = make(map[common.Hash]types.NextEpochData)
	}
	s.nextEpochData[epoch][hash] = nextEpochData
}

// StoreBABENextConfigData stores the types.NextConfigData under epoch and hash keys
func (s *EpochState) storeBABENextConfigData(epoch uint64, hash common.Hash, nextConfigData types.NextConfigDataV1) {
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

	epochRawInDatabase, err := s.getEpochDataRawFromDatabase(nextEpoch)

	// if an error occurs and the error is database.ErrNotFound we ignore
	// since this error is what we will handle in the next lines
	if err != nil && !errors.Is(err, database.ErrNotFound) {
		return fmt.Errorf("cannot check if next epoch data is already defined for epoch %d: %w", nextEpoch, err)
	}

	// epoch data already defined we don't need to lookup in the map
	if epochRawInDatabase != nil {
		return nil
	}

	finalizedNextEpochData, err := findFinalizedHeaderForEpoch(s.nextEpochData, s, nextEpoch)
	if err != nil {
		return fmt.Errorf("cannot find next epoch data: %w", err)
	}

	err = s.SetEpochDataRaw(nextEpoch, finalizedNextEpochData.ToEpochDataRaw())
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

	// if an error occurs and the error is database.ErrNotFound we ignore
	// since this error is what we will handle in the next lines
	if err != nil && !errors.Is(err, database.ErrNotFound) {
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
	err = s.StoreConfigData(nextEpoch, cd)
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
