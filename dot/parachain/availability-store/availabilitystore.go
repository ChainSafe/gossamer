// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package availability_store

import (
	"context"
	"encoding/binary"
	"encoding/json"

	"github.com/ChainSafe/gossamer/internal/database"
	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/lib/common"
)

var logger = log.NewFromGlobal(log.AddContext("pkg", "parachain-availability-store"))

const (
	avaliableDataPrefix = "available"
	chunkPrefix         = "chunk"
	metaPrefix          = "meta"
	unfinalizedPrefix   = "unfinalized"
	pruneByTimePrefix   = "prune_by_time"
)

type AvailabilityStoreSubsystem struct {
	SubSystemToOverseer chan<- any
	OverseerToSubSystem <-chan any
	availabilityStore   AvailabilityStore
	//pruningConfig PruningConfig
	//clock         Clock
	//metrics       Metrics
}

type AvailabilityStore struct {
	availableTable database.Table
	chunkTable     database.Table
	metaTable      database.Table
	//unfinalizedTable database.Table
	//pruneByTimeTable database.Table
}

func NewAvailabilityStore(db database.Database) (*AvailabilityStore, error) {
	return &AvailabilityStore{
		availableTable: database.NewTable(db, avaliableDataPrefix),
		chunkTable:     database.NewTable(db, chunkPrefix),
		metaTable:      database.NewTable(db, metaPrefix),
	}, nil
}

func (as *AvailabilityStore) LoadAvailableData(candidate common.Hash) (AvailableData, error) {
	resultBytes, err := as.availableTable.Get(candidate[:])
	if err != nil {
		return AvailableData{}, err
	}
	result := AvailableData{}
	err = json.Unmarshal(resultBytes, &result)
	return result, err
}

func (as *AvailabilityStore) LoadMetaData(candidate common.Hash) (CandidateMeta, error) {
	resultBytes, err := as.metaTable.Get(candidate[:])
	if err != nil {
		return CandidateMeta{}, err
	}
	result := CandidateMeta{}
	err = json.Unmarshal(resultBytes, &result)
	return result, err
}

func (as *AvailabilityStore) LoadChunk(candidate common.Hash, validatorIndex uint32) (ErasureChunk, error) {
	validatorIndexBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(validatorIndexBytes, validatorIndex)
	resultBytes, err := as.chunkTable.Get(append(candidate[:], validatorIndexBytes...))
	if err != nil {
		return ErasureChunk{}, err
	}
	result := ErasureChunk{}
	err = json.Unmarshal(resultBytes, &result)
	return result, err
}

func (as *AvailabilityStore) StoreChunk(candidate common.Hash, chunk ErasureChunk) error {
	dataBytes, err := json.Marshal(chunk)
	if err != nil {
		return err
	}
	meta, err := as.metaTable.Get(candidate[:])
	if err != nil {
		return err
	}
	metaData := CandidateMeta{}
	err = json.Unmarshal(meta, &metaData)
	if err != nil {
		return err
	}
	// TODO: determine how to update and store meta data
	return as.chunkTable.Put(candidate[:], dataBytes)
}

func (as *AvailabilityStore) StoreAvailableData(candidate common.Hash, data AvailableData) error {
	dataBytes, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return as.availableTable.Put(candidate[:], dataBytes)
}

func (av *AvailabilityStoreSubsystem) Run(ctx context.Context, OverseerToSubsystem chan any,
	SubsystemToOverseer chan any) error {
	av.processMessages()
	return nil
}

func (*AvailabilityStoreSubsystem) Name() parachaintypes.SubSystemName {
	return parachaintypes.AvailabilityStore
}

func (av *AvailabilityStoreSubsystem) processMessages() {
	for msg := range av.OverseerToSubSystem {
		logger.Debugf("received message %v", msg)
		switch msg := msg.(type) {
		case QueryAvailableData:
			av.handleQueryAvailableData(msg)
		case QueryDataAvailability:
			av.handleQueryDataAvailability(msg)
		case QueryChunk:
			av.handleQueryChunk(msg)
		case QueryChunkSize:
			av.handleQueryChunkSize(msg)
		case QueryAllChunks:
			av.handleQueryAllChunks(msg)
		case QueryChunkAvailability:
			av.handleQueryChunkAvailability(msg)
		case StoreChunk:
			av.handleStoreChunk(msg)
		case StoreAvailableData:
			av.handleStoreAvailableData(msg)
		}
	}
}

func (av *AvailabilityStoreSubsystem) handleQueryAvailableData(msg QueryAvailableData) {
	result, err := av.availabilityStore.LoadAvailableData(msg.CandidateHash)
	if err != nil {
		logger.Errorf("failed to load available data: %w", err)
	}
	msg.Sender <- result
}

func (av *AvailabilityStoreSubsystem) handleQueryDataAvailability(msg QueryDataAvailability) {
	_, err := av.availabilityStore.LoadMetaData(msg.CandidateHash)
	if err != nil {
		//TODO: add check to see if error is not found
		msg.Sender <- false
	} else {
		msg.Sender <- true
	}
}

func (av *AvailabilityStoreSubsystem) handleQueryChunk(msg QueryChunk) {
	result, err := av.availabilityStore.LoadChunk(msg.CandidateHash, msg.ValidatorIndex)
	if err != nil {
		logger.Errorf("failed to load chunk: %w", err)
	}
	msg.Sender <- result
}

func (av *AvailabilityStoreSubsystem) handleQueryChunkSize(msg QueryChunkSize) {
	_, err := av.availabilityStore.LoadMetaData(msg.CandidateHash)
	if err != nil {
		logger.Errorf("failed to load meta data: %w", err)
	}
	validatorIndex := 0 // TODO: determine how to get validator index
	chunk, err := av.availabilityStore.LoadChunk(msg.CandidateHash, uint32(validatorIndex))
	if err != nil {
		logger.Errorf("failed to load chunk: %w", err)
	}
	msg.Sender <- uint32(len(chunk.Chunk))
}

func (av *AvailabilityStoreSubsystem) handleQueryAllChunks(msg QueryAllChunks) {
	// TODO: handle query all chunks
}

func (av *AvailabilityStoreSubsystem) handleQueryChunkAvailability(msg QueryChunkAvailability) {
	// TODO: handle query chunk availability
}

func (av *AvailabilityStoreSubsystem) handleStoreChunk(msg StoreChunk) {
	err := av.availabilityStore.StoreChunk(msg.CandidateHash, msg.Chunk)
	if err != nil {
		msg.Sender <- err
	}
	msg.Sender <- nil
}

func (av *AvailabilityStoreSubsystem) handleStoreAvailableData(msg StoreAvailableData) {
	err := av.availabilityStore.StoreAvailableData(msg.CandidateHash, msg.AvailableData)
	if err != nil {
		logger.Errorf("failed to load available data: %w", err)
	}
	msg.Sender <- err // TODO: determine how to replicate Rust's Result type
}
