// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package availabilitystore

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/internal/database"
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

// AvailabilityStoreSubsystem is the struct that holds subsystem data for the availability store
type AvailabilityStoreSubsystem struct {
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	SubSystemToOverseer chan<- any
	OverseerToSubSystem <-chan any
	availabilityStore   AvailabilityStore
	//TODO: pruningConfig PruningConfig
	//TODO: clock         Clock
	//TODO: metrics       Metrics
}

// AvailabilityStore is the struct that holds data for the availability store
type AvailabilityStore struct {
	availableTable database.Table
	chunkTable     database.Table
	metaTable      database.Table
	//TODO: unfinalizedTable database.Table
	//TODO: pruneByTimeTable database.Table
}

// NewAvailabilityStore creates a new instance of AvailabilityStore
func NewAvailabilityStore(db database.Database) *AvailabilityStore {
	return &AvailabilityStore{
		availableTable: database.NewTable(db, avaliableDataPrefix),
		chunkTable:     database.NewTable(db, chunkPrefix),
		metaTable:      database.NewTable(db, metaPrefix),
	}
}

// loadAvailableData loads available data from the availability store
func (as *AvailabilityStore) loadAvailableData(candidate common.Hash) (*AvailableData, error) {
	resultBytes, err := as.availableTable.Get(candidate[:])
	if err != nil {
		return nil, fmt.Errorf("getting candidate %v from available table: %w", candidate, err)
	}
	result := AvailableData{}
	err = json.Unmarshal(resultBytes, &result)
	if err != nil {
		return nil, fmt.Errorf("unmarshalling available data: %w", err)
	}
	return &result, nil
}

// loadMetaData loads metadata from the availability store
func (as *AvailabilityStore) loadMetaData(candidate common.Hash) (*CandidateMeta, error) {
	resultBytes, err := as.metaTable.Get(candidate[:])
	if err != nil {
		return nil, fmt.Errorf("getting candidate %v from available table: %w", candidate, err)
	}
	result := CandidateMeta{}
	err = json.Unmarshal(resultBytes, &result)
	if err != nil {
		return nil, fmt.Errorf("unmarshalling candidate meta: %w", err)
	}
	return &result, nil
}

// storeMetaData stores metadata in the availability store
func (as *AvailabilityStore) storeMetaData(candidate common.Hash, meta CandidateMeta) error {
	dataBytes, err := json.Marshal(meta)
	if err != nil {
		return fmt.Errorf("marshalling meta for candidate: %w", err)
	}
	err = as.metaTable.Put(candidate[:], dataBytes)
	if err != nil {
		return fmt.Errorf("storing metadata for candidate %v: %w", candidate, err)
	}
	return nil
}

// loadChunk loads a chunk from the availability store
func (as *AvailabilityStore) loadChunk(candidate common.Hash, validatorIndex uint32) (*ErasureChunk, error) {
	resultBytes, err := as.chunkTable.Get(append(candidate[:], uint32ToBytes(validatorIndex)...))
	if err != nil {
		return nil, fmt.Errorf("getting candidate %v, index %d from chunk table: %w", candidate, validatorIndex, err)
	}
	result := ErasureChunk{}
	err = json.Unmarshal(resultBytes, &result)
	if err != nil {
		return nil, fmt.Errorf("unmarshalling chunk: %w", err)
	}
	return &result, nil
}

// storeChunk stores a chunk in the availability store
func (as *AvailabilityStore) storeChunk(candidate common.Hash, chunk ErasureChunk) error {
	meta, err := as.loadMetaData(candidate)
	if err != nil {

		if errors.Is(err, database.ErrNotFound) {
			// TODO: were creating metadata here, but we should be doing it in the parachain block import?
			// TODO: also we need to determine how many chunks we need to store
			meta = &CandidateMeta{
				ChunksStored: make([]bool, 16),
			}
		} else {
			return fmt.Errorf("load metadata: %w", err)
		}
	}

	if meta.ChunksStored[chunk.Index] {
		logger.Debugf("Chunk %d already stored", chunk.Index)
		return nil // already stored
	} else {
		dataBytes, err := json.Marshal(chunk)
		if err != nil {
			return fmt.Errorf("marshalling chunk: %w", err)
		}
		err = as.chunkTable.Put(append(candidate[:], uint32ToBytes(chunk.Index)...), dataBytes)
		if err != nil {
			return fmt.Errorf("storing chunk for candidate %v, index %d: %w", candidate, chunk.Index, err)
		}

		meta.ChunksStored[chunk.Index] = true
		err = as.storeMetaData(candidate, *meta)
		if err != nil {
			return fmt.Errorf("storing metadata for candidate %v: %w", candidate, err)
		}
	}
	logger.Debugf("stored chuck %d for %v", chunk.Index, candidate)
	return nil
}

// storeAvailableData stores available data in the availability store
func (as *AvailabilityStore) storeAvailableData(candidate common.Hash, data AvailableData) error {
	dataBytes, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshalling available data: %w", err)
	}
	err = as.availableTable.Put(candidate[:], dataBytes)
	if err != nil {
		return fmt.Errorf("storing available data for candidate %v: %w", candidate, err)
	}
	return nil
}

func uint32ToBytes(value uint32) []byte {
	result := make([]byte, 4)
	binary.LittleEndian.PutUint32(result, value)
	return result
}

// Run runs the availability store subsystem
func (av *AvailabilityStoreSubsystem) Run(ctx context.Context, OverseerToSubsystem chan any,
	SubsystemToOverseer chan any) error {

	av.wg.Add(2)
	go av.processMessages()

	return nil
}

// Name returns the name of the availability store subsystem
func (*AvailabilityStoreSubsystem) Name() parachaintypes.SubSystemName {
	return parachaintypes.AvailabilityStore
}

func (av *AvailabilityStoreSubsystem) processMessages() {
	for {
		select {
		case msg := <-av.OverseerToSubSystem:
			logger.Debugf("received message %v", msg)
			switch msg := msg.(type) {
			case QueryAvailableData:
				err := av.handleQueryAvailableData(msg)
				if err != nil {
					logger.Errorf("failed to handle available data: %w", err)
				}
			case QueryDataAvailability:
				err := av.handleQueryDataAvailability(msg)
				if err != nil {
					logger.Errorf("failed to handle query data availability: %w", err)
				}
			case QueryChunk:
				err := av.handleQueryChunk(msg)
				if err != nil {
					logger.Errorf("failed to handle query chunk: %w", err)
				}
			case QueryChunkSize:
				err := av.handleQueryChunkSize(msg)
				if err != nil {
					logger.Errorf("failed to handle query chunk size: %w", err)
				}
			case QueryAllChunks:
				err := av.handleQueryAllChunks(msg)
				if err != nil {
					logger.Errorf("failed to handle query all chunks: %w", err)
				}
			case QueryChunkAvailability:
				err := av.handleQueryChunkAvailability(msg)
				if err != nil {
					logger.Errorf("failed to handle query chunk availability: %w", err)
				}
			case StoreChunk:
				err := av.handleStoreChunk(msg)
				if err != nil {
					logger.Errorf("failed to handle store chunk: %w", err)
				}
			case StoreAvailableData:
				err := av.handleStoreAvailableData(msg)
				if err != nil {
					logger.Errorf("failed to handle store available data: %w", err)
				}

			case parachaintypes.ActiveLeavesUpdateSignal:
				av.ProcessActiveLeavesUpdateSignal()

			case parachaintypes.BlockFinalizedSignal:
				av.ProcessBlockFinalizedSignal()

			default:
				logger.Error(parachaintypes.ErrUnknownOverseerMessage.Error())
			}

		case <-av.ctx.Done():
			if err := av.ctx.Err(); err != nil {
				logger.Errorf("ctx error: %v\n", err)
			}
			av.wg.Done()
			return
		}

	}
}

func (av *AvailabilityStoreSubsystem) ProcessActiveLeavesUpdateSignal() {
	// TODO: #3630
}

func (av *AvailabilityStoreSubsystem) ProcessBlockFinalizedSignal() {
	// TODO: #3630
}

func (av *AvailabilityStoreSubsystem) handleQueryAvailableData(msg QueryAvailableData) error {
	result, err := av.availabilityStore.loadAvailableData(msg.CandidateHash)
	if err != nil {
		msg.Sender <- AvailableData{}
		return fmt.Errorf("load available data: %w", err)
	}
	msg.Sender <- *result
	return nil
}

func (av *AvailabilityStoreSubsystem) handleQueryDataAvailability(msg QueryDataAvailability) error {
	_, err := av.availabilityStore.loadMetaData(msg.CandidateHash)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			msg.Sender <- false
			return nil
		} else {
			return fmt.Errorf("load metadata: %w", err)
		}
	}
	msg.Sender <- err == nil
	return nil
}

func (av *AvailabilityStoreSubsystem) handleQueryChunk(msg QueryChunk) error {
	result, err := av.availabilityStore.loadChunk(msg.CandidateHash, msg.ValidatorIndex)
	if err != nil {
		msg.Sender <- ErasureChunk{}
		return fmt.Errorf("load chunk: %w", err)
	}
	msg.Sender <- *result
	return nil
}

func (av *AvailabilityStoreSubsystem) handleQueryChunkSize(msg QueryChunkSize) error {
	meta, err := av.availabilityStore.loadMetaData(msg.CandidateHash)
	if err != nil {
		return fmt.Errorf("load metadata: %w", err)
	}
	var validatorIndex uint32
	for i, v := range meta.ChunksStored {
		if v {
			validatorIndex = uint32(i)
			break
		}
	}

	chunk, err := av.availabilityStore.loadChunk(msg.CandidateHash, validatorIndex)
	if err != nil {
		return fmt.Errorf("load chunk: %w", err)
	}
	msg.Sender <- uint32(len(chunk.Chunk))
	return nil
}

func (av *AvailabilityStoreSubsystem) handleQueryAllChunks(msg QueryAllChunks) error {
	meta, err := av.availabilityStore.loadMetaData(msg.CandidateHash)
	if err != nil {
		msg.Sender <- []ErasureChunk{}
		return fmt.Errorf("load metadata: %w", err)
	}
	chunks := []ErasureChunk{}
	for i, v := range meta.ChunksStored {
		if v {
			chunk, err := av.availabilityStore.loadChunk(msg.CandidateHash, uint32(i))
			if err != nil {
				logger.Errorf("load chunk: %w", err)
			}
			chunks = append(chunks, *chunk)
		} else {
			logger.Warnf("chunk %d not stored for %v", i, msg.CandidateHash)
		}
	}
	msg.Sender <- chunks
	return nil
}

func (av *AvailabilityStoreSubsystem) handleQueryChunkAvailability(msg QueryChunkAvailability) error {
	meta, err := av.availabilityStore.loadMetaData(msg.CandidateHash)
	if err != nil {
		msg.Sender <- false
		return fmt.Errorf("load metadata: %w", err)
	}
	msg.Sender <- meta.ChunksStored[msg.ValidatorIndex]
	return nil
}

func (av *AvailabilityStoreSubsystem) handleStoreChunk(msg StoreChunk) error {
	err := av.availabilityStore.storeChunk(msg.CandidateHash, msg.Chunk)
	if err != nil {
		msg.Sender <- err
		return fmt.Errorf("store chunk: %w", err)
	}
	msg.Sender <- nil
	return nil
}

func (av *AvailabilityStoreSubsystem) handleStoreAvailableData(msg StoreAvailableData) error {
	err := av.availabilityStore.storeAvailableData(msg.CandidateHash, msg.AvailableData)
	if err != nil {
		msg.Sender <- err
		return fmt.Errorf("store available data: %w", err)
	}
	msg.Sender <- err // TODO: determine how to replicate Rust's Result type
	return nil
}

func (av *AvailabilityStoreSubsystem) Stop() {
	av.cancel()
	av.wg.Wait()
}
