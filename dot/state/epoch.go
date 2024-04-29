// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package state

import (
	"bytes"
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
	ErrConfigNotFound          = errors.New("config data not found")
	ErrEpochNotInMemory        = errors.New("epoch not found in memory map")
	errEpochLengthCannotBeZero = errors.New("epoch length cannot be zero")
	errHashNotInMemory         = errors.New("hash not found in memory map")
	errEpochNotInDatabase      = errors.New("epoch data not found in the database")
	errHashNotPersisted        = errors.New("hash with next epoch not found in database")
	errNoFirstNonOriginBlock   = errors.New("no first non origin block")
)

var (
	epochPrefix      = "epoch"
	currentEpochKey  = []byte("current")
	epochDataPrefix  = []byte("epochinfo")
	configDataPrefix = []byte("configinfo")
	skipToKey        = []byte("skipto")
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
	db           GetterPutterNewBatcher
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

	searchOnDatabase := func() (*types.EpochDataRaw, error) {
		epochDataRaw, err := getEpochDefinitionFromDatabase[types.EpochDataRaw](s.db, epoch, epochDataKey)
		if err != nil && !errors.Is(err, database.ErrNotFound) {
			return nil, fmt.Errorf("failed to retrieve epoch data from database: %w", err)
		}

		return epochDataRaw, nil
	}

	searchOnMemory := func() (*types.EpochDataRaw, error) {
		s.nextEpochDataLock.RLock()
		defer s.nextEpochDataLock.RUnlock()

		inMemoryEpochData, err := s.nextEpochData.Retrieve(s.blockState, epoch, header)
		if err != nil {
			return nil, fmt.Errorf("failed to get epoch data from memory: %w", err)
		}

		return inMemoryEpochData.ToEpochDataRaw(), nil
	}

	return retrieveEpochDefinitions(searchOnDatabase, searchOnMemory)
}

// GetSkippedEpochDataRaw returns the raw epoch data for a skipped epoch that is stored in advance
// of the start of the given epoch, also this method will update the epoch number from the
// skipped epoch to the current epoch
func (s *EpochState) GetSkippedEpochDataRaw(skippedEpoch, currentEpoch uint64,
	header *types.Header) (*types.EpochDataRaw, error) {
	if skippedEpoch == 0 {
		return s.genesisEpochDescriptor.EpochData, nil
	}

	searchOnDatabase := func() (*types.EpochDataRaw, error) {
		epochDataRaw, err := getAndUpdateEpochDefinitionKey[types.EpochDataRaw](s.db,
			skippedEpoch, currentEpoch, epochDataKey)
		if err != nil && !errors.Is(err, database.ErrNotFound) {
			return nil, fmt.Errorf("failed to retrieve epoch data from database: %w", err)
		}

		return epochDataRaw, nil
	}

	searchOnMemory := func() (*types.EpochDataRaw, error) {
		s.nextEpochDataLock.RLock()
		defer s.nextEpochDataLock.RUnlock()

		inMemoryEpochData, err := s.nextEpochData.RetrieveAndUpdate(s.blockState,
			skippedEpoch, currentEpoch, header)
		if err != nil {
			return nil, fmt.Errorf("failed to get epoch data from memory: %w", err)
		}

		return inMemoryEpochData.ToEpochDataRaw(), nil
	}

	return retrieveEpochDefinitions(searchOnDatabase, searchOnMemory)
}

// UpdateSkippedEpochDefinitions updates the skipped epoch definitions by changing the
// changing the key from skipped epoch to current epoch on each epoch data raw storage
// and on config data storage, it returns an error if the skipped epoch number does not
// exists in the database.
func (s *EpochState) UpdateSkippedEpochDefinitions(skippedEpoch, currentEpoch uint64,
	header *types.Header) error {
	if skippedEpoch == 0 {
		return nil
	}

	err := s.updateSkippedEpochDataRaw(skippedEpoch, currentEpoch, header)
	if err != nil {
		return fmt.Errorf("updatting skipped epoch data raw: %w", err)
	}

	err = s.updateSkippedConfigData(skippedEpoch, currentEpoch, header)
	if err != nil {
		return fmt.Errorf("updatting skipped config data: %w", err)
	}

	return nil
}

// updateSkippedEpochDataRaw only updates the key from `skippedEpoch` to `currentEpoch`
// returns an error if `skippedEpoch` does not exists on database or in memory
func (s *EpochState) updateSkippedEpochDataRaw(skippedEpoch, currentEpoch uint64,
	header *types.Header) error {
	if skippedEpoch == 0 {
		return nil
	}

	_, err := updateEpochDefinitionKey(s.db,
		skippedEpoch, currentEpoch, epochDataKey)
	if err != nil && !errors.Is(err, database.ErrNotFound) {
		return fmt.Errorf("getting and updating epoch definition key: %w", err)
	}

	if err == nil {
		return nil
	}

	s.nextEpochDataLock.RLock()
	defer s.nextConfigDataLock.RUnlock()

	_, err = s.nextEpochData.RetrieveAndUpdate(s.blockState,
		skippedEpoch, currentEpoch, header)
	if err != nil {
		return fmt.Errorf("updating in memory epoch data definition: %w", err)
	}

	return nil
}

// StoreConfigData sets the BABE config data for a given epoch
func (s *EpochState) StoreConfigData(epoch uint64, info *types.ConfigData) error {
	enc, err := scale.Marshal(*info)
	if err != nil {
		return err
	}

	return s.db.Put(configDataKey(epoch), enc)
}

// GetConfigData returns the newest config data for a given epoch persisted in database
// otherwise tries to get the data from the in-memory map using the header. If we don't
// find any config data for the current epoch we lookup in the previous epochs, as the spec says:
// - The supplied configuration data are intended to be used from the next epoch onwards.
// If the header params is nil then it will search only in the database.
func (s *EpochState) GetConfigData(epoch uint64, header *types.Header) (configData *types.ConfigData, err error) {
	if epoch == 0 {
		return s.genesisEpochDescriptor.ConfigData, nil
	}

	searchOnDatabase := func(epoch uint64) retrieveFrom[types.ConfigData] {
		return func() (*types.ConfigData, error) {
			configData, err = getEpochDefinitionFromDatabase[types.ConfigData](
				s.db, epoch, configDataKey)
			if err != nil && !errors.Is(err, database.ErrNotFound) {
				return nil, err
			}

			return configData, nil
		}
	}

	searchOnMemory := func(epoch uint64) retrieveFrom[types.ConfigData] {
		return func() (*types.ConfigData, error) {
			// we will check in the memory map and if we don't find the data
			// then we continue searching through the previous epoch
			s.nextConfigDataLock.RLock()
			defer s.nextConfigDataLock.RUnlock()
			inMemoryConfigData, err := s.nextConfigData.Retrieve(s.blockState, epoch, header)
			if err != nil {
				return nil, err
			}

			return inMemoryConfigData.ToConfigData(), nil
		}
	}

	for tryEpoch := int(epoch); tryEpoch >= 0; tryEpoch-- {
		if tryEpoch == 0 {
			return s.genesisEpochDescriptor.ConfigData, nil
		}

		configData, err := retrieveEpochDefinitions(
			searchOnDatabase(uint64(tryEpoch)),
			searchOnMemory(uint64(tryEpoch)),
		)

		if err != nil {
			if errors.Is(err, errEpochNotInDatabase) || errors.Is(err, ErrEpochNotInMemory) {
				continue
			}

			return nil, fmt.Errorf("while iterating on epoch %d: %w", tryEpoch, err)
		}

		return configData, err
	}

	return nil, fmt.Errorf("%w: epoch %d", ErrConfigNotFound, epoch)
}

func (s *EpochState) GetSkippedConfigData(skippedEpoch, currentEpoch uint64,
	header *types.Header) (*types.ConfigData, error) {
	if skippedEpoch == 0 {
		return s.genesisEpochDescriptor.ConfigData, nil
	}

	searchOnDatabase := func() (*types.ConfigData, error) {
		configData, err := getAndUpdateEpochDefinitionKey[types.ConfigData](
			s.db, skippedEpoch, currentEpoch, configDataKey)
		if err != nil && !errors.Is(err, database.ErrNotFound) {
			return nil, fmt.Errorf("getting and updating epoch definition key: %w", err)
		}
		return configData, nil
	}

	searchOnMemory := func() (*types.ConfigData, error) {
		s.nextConfigDataLock.RLock()
		defer s.nextConfigDataLock.RUnlock()

		inMemoryConfigData, err := s.nextConfigData.RetrieveAndUpdate(s.blockState,
			skippedEpoch, currentEpoch, header)
		if err != nil {
			return nil, fmt.Errorf("retrieving and updating in memory epoch definition: %w", err)
		}

		return inMemoryConfigData.ToConfigData(), nil
	}

	skippedConfigData, err := retrieveEpochDefinitions(searchOnDatabase, searchOnMemory)
	if err != nil {
		if errors.Is(err, ErrEpochNotInMemory) || errors.Is(err, errEpochNotInDatabase) {
			// if there is no config data for the skipped epoch them
			// we keep searching using previous epochs
			return s.GetConfigData(skippedEpoch-1, header)
		}

		return nil, fmt.Errorf("retrieving epoch definitions: %w", err)
	}

	return skippedConfigData, nil
}

// updateSkippedConfigData only updates the key from `skippedEpoch` to `currentEpoch`
func (s *EpochState) updateSkippedConfigData(skippedEpoch, currentEpoch uint64,
	header *types.Header) error {
	if skippedEpoch == 0 {
		return nil
	}

	_, err := updateEpochDefinitionKey(s.db,
		skippedEpoch, currentEpoch, configDataKey)
	if err != nil && !errors.Is(err, database.ErrNotFound) {
		return fmt.Errorf("getting and updating epoch definition key: %w", err)
	}

	if err == nil {
		return nil
	}

	s.nextConfigDataLock.RLock()
	defer s.nextConfigDataLock.RUnlock()

	_, err = s.nextConfigData.RetrieveAndUpdate(s.blockState,
		skippedEpoch, currentEpoch, header)
	if err != nil {
		if errors.Is(err, ErrEpochNotInMemory) || errors.Is(err, errEpochNotInDatabase) {
			// if there is no config data for the skipped epoch them
			// that just mean for this skipped epoch the runtime didn't
			// issue any config data, but we can still use prev epochs config data
			return nil
		}

		return fmt.Errorf("updating in memory epoch config data definition: %w", err)
	}

	return nil
}

// retrieveFrom type annotation makes it generic to query the database
// or memory in order to find some data
type retrieveFrom[T types.EpochDataRaw | types.ConfigData] func() (*T, error)

func retrieveEpochDefinitions[T types.EpochDataRaw | types.ConfigData](
	fromDatabase retrieveFrom[T], fromMemory retrieveFrom[T]) (
	epochDataRaw *T, err error) {

	epochDataRaw, err = fromDatabase()
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve epoch data from database: %w", err)
	}

	if epochDataRaw != nil {
		return epochDataRaw, nil
	}

	if fromMemory == nil {
		return nil, errEpochNotInDatabase
	}

	return fromMemory()
}

type prefixedKeyBuilder func(epoch uint64) []byte

// updateEpochDefinitionKey updates the informations from database
// by querying the raw bytes from prefix + oldEpoch and inserting
// at prefix + newEpoch and return the values stored at prefix + oldEpoch
func updateEpochDefinitionKey(db GetterPutterNewBatcher,
	oldEpoch, newEpoch uint64, usePrefix prefixedKeyBuilder) ([]byte, error) {
	rawBytes, err := db.Get(usePrefix(oldEpoch))
	if err != nil {
		return nil, fmt.Errorf("getting epoch data: %w", err)
	}

	updateKeyBatcher := db.NewBatch()
	defer func() {
		if err := updateKeyBatcher.Close(); err != nil {
			logger.Criticalf("cannot close epoch data raw batcher: %w", err)
		}
	}()

	err = updateKeyBatcher.Del(usePrefix(oldEpoch))
	if err != nil {
		return nil, fmt.Errorf("deleting old epoch key: %w", err)
	}

	err = updateKeyBatcher.Put(usePrefix(newEpoch), rawBytes)
	if err != nil {
		return nil, fmt.Errorf("storing new epoch key: %w", err)
	}

	if err := updateKeyBatcher.Flush(); err != nil {
		return nil, fmt.Errorf("flushing batcher: %w", err)
	}

	return rawBytes, nil
}

func getAndUpdateEpochDefinitionKey[T types.ConfigData | types.EpochDataRaw](
	db GetterPutterNewBatcher, oldEpoch, newEpoch uint64, usePrefix prefixedKeyBuilder) (*T, error) {
	rawBytes, err := updateEpochDefinitionKey(db, oldEpoch, newEpoch, usePrefix)
	if err != nil {
		return nil, fmt.Errorf("updating epoch key definition: %w", err)
	}

	raw := new(T)
	err = scale.Unmarshal(rawBytes, raw)
	if err != nil {
		return nil, fmt.Errorf("unmarshaling into epoch data raw: %w", err)
	}

	return raw, nil
}

func getEpochDefinitionFromDatabase[T types.ConfigData | types.EpochDataRaw](
	db Getter, epoch uint64, usePrefix prefixedKeyBuilder) (*T, error) {
	rawBytes, err := db.Get(usePrefix(epoch))
	if err != nil {
		return nil, err
	}

	info := new(T)
	err = scale.Unmarshal(rawBytes, info)
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

func (nem nextEpochMap[T]) RetrieveAndUpdate(blockState *BlockState,
	oldEpoch, newEpoch uint64, header *types.Header) (*T, error) {
	oldEpochHashes, has := nem[oldEpoch]
	if !has {
		return nil, fmt.Errorf("%w: %d", ErrEpochNotInMemory, oldEpoch)
	}

	hashToMove, value, err := findAncestor(blockState, oldEpochHashes, header)
	if err != nil {
		return nil, err
	}

	// just remove the HASH -> Next Epoch Data from the old epoch
	// and introduce the HASH -> Next Epoch Data into the new epoch
	delete(oldEpochHashes, hashToMove)
	nem[oldEpoch] = oldEpochHashes

	hashes, ok := nem[newEpoch]
	if !ok {
		hashes = make(map[common.Hash]T)
	}

	hashes[hashToMove] = *value
	nem[newEpoch] = hashes
	return value, nil
}

func (nem nextEpochMap[T]) Retrieve(blockState *BlockState, epoch uint64, header *types.Header) (*T, error) {
	atEpoch, has := nem[epoch]
	if !has {
		return nil, fmt.Errorf("%w: %d", ErrEpochNotInMemory, epoch)
	}

	_, value, err := findAncestor(blockState, atEpoch, header)
	return value, err
}

func findAncestor[T types.NextEpochData | types.NextConfigDataV1](blockState *BlockState,
	hashesAtEpoch map[common.Hash]T, header *types.Header) (common.Hash, *T, error) {

	currentHeader := header

	for {
		for hash, value := range hashesAtEpoch {
			if bytes.Equal(hash[:], currentHeader.Hash().ToBytes()) {
				return hash, &value, nil
			}

			isDescendant, err := blockState.IsDescendantOf(hash, currentHeader.Hash())
			if err != nil {
				if errors.Is(err, database.ErrNotFound) {
					continue
				}

				return common.Hash{}, nil, fmt.Errorf("cannot verify the ancestry: %w", err)
			}

			if isDescendant {
				return hash, &value, nil
			}
		}

		// if there is no more ancestors then return
		if bytes.Equal(currentHeader.ParentHash.ToBytes(), common.EmptyHash.ToBytes()) {
			return common.Hash{}, nil, fmt.Errorf("%w: could not found config data for hash %s",
				errHashNotInMemory, currentHeader.Hash())
		}

		// sometimes while moving to the next epoch is possible the header
		// is not fully imported by the blocktree, in this case we will use
		// its parent header which migth be already imported.
		parentHeader, err := blockState.GetHeader(header.ParentHash)
		if err != nil {
			return common.Hash{}, nil, fmt.Errorf("cannot get parent header: %w", err)
		}

		currentHeader = parentHeader
	}
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
	chainFirstSlotNumber, err := s.blockState.getFirstNonOriginSlotNumber()
	if err != nil {
		return 0, fmt.Errorf("retrieving first non origin block slot: %w", err)

	}
	if chainFirstSlotNumber != 0 {
		return chainFirstSlotNumber, nil
	}

	// if the chain first slot number is not set in the database then we will
	// try to find the first non origin block by checking the ancestry
	// of the block hash
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

	chainFirstSlotNumber, err = firstNonGenesisHeader.SlotNumber()
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

	fmt.Printf("storing next config data for epoch %d, hash: %s\n", epoch, hash.String())

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

	epochRawInDatabase, err := getEpochDefinitionFromDatabase[types.EpochDataRaw](
		s.db, nextEpoch, epochDataKey)

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

	configInDatabase, err := getEpochDefinitionFromDatabase[types.ConfigData](
		s.db, nextEpoch, epochDataKey)

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
