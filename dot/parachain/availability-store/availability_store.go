// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package availabilitystore

import (
	"bytes"
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
	availableDataPrefix = "available"
	chunkPrefix         = "chunk"
	metaPrefix          = "meta"
	unfinalizedPrefix   = "unfinalized"
	pruneByTimePrefix   = "prune_by_time"

	tombstoneValue = " "
)

type BETimestamp uint64

func (b BETimestamp) ToBytes() []byte {
	res := make([]byte, 8)
	binary.LittleEndian.PutUint64(res, uint64(b))
	return res
}

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
	availableTable   database.Table
	chunkTable       database.Table
	metaTable        database.Table
	unfinalizedTable database.Table
	pruneByTimeTable database.Table
}

type AvailabilityStoreBatch struct {
	availableBatch   database.Batch
	chunkBatch       database.Batch
	metaBatch        database.Batch
	unfinalizedBatch database.Batch
	pruneByTimeBatch database.Batch
}

func NewAvailabilityStoreBatch(as *AvailabilityStore) *AvailabilityStoreBatch {
	return &AvailabilityStoreBatch{
		availableBatch:   as.availableTable.NewBatch(),
		chunkBatch:       as.chunkTable.NewBatch(),
		metaBatch:        as.metaTable.NewBatch(),
		unfinalizedBatch: as.unfinalizedTable.NewBatch(),
		pruneByTimeBatch: as.pruneByTimeTable.NewBatch(),
	}
}

func (asb *AvailabilityStoreBatch) Write() error {
	err := asb.availableBatch.Flush()
	if err != nil {
		return asb.Reset(fmt.Errorf("writing available batch: %w", err))
	}
	err = asb.chunkBatch.Flush()
	if err != nil {
		return asb.Reset(fmt.Errorf("writing chunk batch: %w", err))
	}
	err = asb.metaBatch.Flush()
	if err != nil {
		return asb.Reset(fmt.Errorf("writing meta batch: %w", err))
	}
	err = asb.unfinalizedBatch.Flush()
	if err != nil {
		return asb.Reset(fmt.Errorf("writing unfinalized batch: %w", err))
	}
	err = asb.pruneByTimeBatch.Flush()
	if err != nil {
		return asb.Reset(fmt.Errorf("writing prune by time batch: %w", err))
	}
	return nil
}

// Reset resets the batch and returns the error
func (asb *AvailabilityStoreBatch) Reset(err error) error {
	asb.availableBatch.Reset()
	asb.chunkBatch.Reset()
	asb.metaBatch.Reset()
	asb.unfinalizedBatch.Reset()
	asb.pruneByTimeBatch.Reset()
	return err
}

// NewAvailabilityStore creates a new instance of AvailabilityStore
func NewAvailabilityStore(db database.Database) *AvailabilityStore {
	return &AvailabilityStore{
		availableTable:   database.NewTable(db, availableDataPrefix),
		chunkTable:       database.NewTable(db, chunkPrefix),
		metaTable:        database.NewTable(db, metaPrefix),
		unfinalizedTable: database.NewTable(db, unfinalizedPrefix),
		pruneByTimeTable: database.NewTable(db, pruneByTimePrefix),
	}
}

// loadAvailableData loads available data from the availability store
func (as *AvailabilityStore) loadAvailableData(candidate parachaintypes.CandidateHash) (*AvailableData, error) {
	resultBytes, err := as.availableTable.Get(candidate.Value[:])
	if err != nil {
		return nil, fmt.Errorf("getting candidate %v from available table: %w", candidate.Value, err)
	}
	result := AvailableData{}
	err = json.Unmarshal(resultBytes, &result)
	if err != nil {
		return nil, fmt.Errorf("unmarshalling available data: %w", err)
	}
	return &result, nil
}

// writeAvailableData put available data in the availabilityBatch of the given batch
func (as *AvailabilityStore) writeAvailableData(batch *AvailabilityStoreBatch,
	candidate parachaintypes.CandidateHash, data AvailableData) error {
	dataBytes, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshalling available data: %w", err)
	}
	err = batch.availableBatch.Put(candidate.Value[:], dataBytes)
	if err != nil {
		return fmt.Errorf("storing available data for candidate %v: %w", candidate, err)
	}
	return nil
}

func (as *AvailabilityStore) deleteAvailableData(batch *AvailabilityStoreBatch,
	candidate parachaintypes.CandidateHash) error {
	err := batch.availableBatch.Del(candidate.Value[:])
	if err != nil {
		return fmt.Errorf("deleting available data for candidate %v: %w", candidate, err)
	}
	return nil
}

// loadMetaData loads metadata from the availability store
func (as *AvailabilityStore) loadMeta(candidate parachaintypes.CandidateHash) (*CandidateMeta, error) {
	resultBytes, err := as.metaTable.Get(candidate.Value[:])
	if err != nil {
		return nil, fmt.Errorf("getting candidate %v from meta table: %w", candidate.Value, err)
	}
	result := CandidateMeta{}
	err = json.Unmarshal(resultBytes, &result)
	if err != nil {
		return nil, fmt.Errorf("unmarshalling candidate meta: %w", err)
	}
	return &result, nil
}

// storeMetaData stores metadata in the availability store
func (as *AvailabilityStore) writeMeta(batch *AvailabilityStoreBatch, candidate parachaintypes.CandidateHash,
	meta CandidateMeta) error {
	dataBytes, err := json.Marshal(meta)
	if err != nil {
		return fmt.Errorf("marshalling meta for candidate: %w", err)
	}
	err = batch.metaBatch.Put(candidate.Value[:], dataBytes)
	if err != nil {
		return fmt.Errorf("storing metadata for candidate %v: %w", candidate, err)
	}
	return nil
}

func (as *AvailabilityStore) deleteMeta(batch *AvailabilityStoreBatch, candidate parachaintypes.CandidateHash) error {
	err := batch.metaBatch.Del(candidate.Value[:])
	if err != nil {
		return fmt.Errorf("deleting meta for candidate %v: %w", candidate, err)
	}
	return nil
}

// loadChunk loads a chunk from the availability store
func (as *AvailabilityStore) loadChunk(candidate parachaintypes.CandidateHash, validatorIndex uint32) (*ErasureChunk,
	error) {
	resultBytes, err := as.chunkTable.Get(append(candidate.Value[:], uint32ToBytes(validatorIndex)...))
	if err != nil {
		return nil, fmt.Errorf("getting candidate %v, index %d from chunk table: %w", candidate.Value, validatorIndex, err)
	}
	result := ErasureChunk{}
	err = json.Unmarshal(resultBytes, &result)
	if err != nil {
		return nil, fmt.Errorf("unmarshalling chunk: %w", err)
	}
	return &result, nil
}

func (as *AvailabilityStore) writeChunk(batch *AvailabilityStoreBatch, candidate parachaintypes.CandidateHash,
	chunk ErasureChunk) error {
	dataBytes, err := json.Marshal(chunk)
	if err != nil {
		return fmt.Errorf("marshalling chunk for candidate %v, index %d: %w", candidate, chunk.Index, err)
	}
	err = batch.chunkBatch.Put(append(candidate.Value[:], uint32ToBytes(chunk.Index)...), dataBytes)
	if err != nil {
		return fmt.Errorf("writing chunk for candidate %v, index %d: %w", candidate, chunk.Index, err)
	}
	return nil
}

func (as *AvailabilityStore) deleteChunk(batch *AvailabilityStoreBatch, candidate parachaintypes.CandidateHash,
	chunkIndex uint32) error {
	err := batch.chunkBatch.Del(append(candidate.Value[:], uint32ToBytes(chunkIndex)...))
	if err != nil {
		return fmt.Errorf("deleting chunk for candidate %v, index %d: %w", candidate, chunkIndex, err)
	}
	return nil
}

func (as *AvailabilityStore) writeUnfinalizedBlockContains(batch *AvailabilityStoreBatch,
	blockNumber parachaintypes.BlockNumber, hash common.Hash, candidateHash parachaintypes.CandidateHash) error {
	key := append(uint32ToBytesBigEndian(uint32(blockNumber)), hash[:]...)
	key = append(key, candidateHash.Value[:]...)

	err := batch.unfinalizedBatch.Put(key, []byte(tombstoneValue))
	if err != nil {
		return fmt.Errorf("writing unfinalized block contains, block: %v hash: %v candidate hash: %v: %w",
			blockNumber, hash, candidateHash, err)
	}
	return nil
}

func (as *AvailabilityStore) deleteUnfinalizedInclusion(batch *AvailabilityStoreBatch,
	blockNumber parachaintypes.BlockNumber, hash common.Hash, candidateHash parachaintypes.CandidateHash) error {
	key := append(uint32ToBytesBigEndian(uint32(blockNumber)), hash[:]...)
	key = append(key, candidateHash.Value[:]...)

	err := batch.unfinalizedBatch.Del(key)
	if err != nil {
		return fmt.Errorf("deleting unfinalized inclusion, block: %v hash: %v: candidate hash: %v: %w", blockNumber,
			hash, candidateHash, err)
	}
	return nil
}

func (as *AvailabilityStore) deleteUnfinalizedHeight(batch *AvailabilityStoreBatch,
	blockNumber parachaintypes.BlockNumber) error {
	keyPrefix := append([]byte(unfinalizedPrefix), uint32ToBytesBigEndian(uint32(blockNumber))...)
	itr := as.unfinalizedTable.NewIterator()
	for itr.First(); itr.Valid(); itr.Next() {
		comp := bytes.Compare(itr.Key()[0:len(keyPrefix)], keyPrefix)
		if comp < 0 {
			continue
		} else if comp > 0 {
			break
		}
		err := batch.unfinalizedBatch.Del(itr.Key()[len(unfinalizedPrefix):])
		if err != nil {
			return fmt.Errorf("deleting unfinalized height %v: %w", blockNumber, err)
		}
	}
	itr.Release()

	return nil
}

func (as *AvailabilityStore) writePruningKey(batch *AvailabilityStoreBatch, pruneAt BETimestamp,
	candidate parachaintypes.CandidateHash) error {
	pruneKey := append(pruneAt.ToBytes(), candidate.Value[:]...)
	err := batch.pruneByTimeBatch.Put(pruneKey, []byte(tombstoneValue))
	if err != nil {
		return fmt.Errorf("writing pruning key: %w", err)
	}
	return nil
}

func (as *AvailabilityStore) deletePruningKey(batch *AvailabilityStoreBatch, pruneAt BETimestamp,
	candidate parachaintypes.CandidateHash) error {
	pruneKey := append(pruneAt.ToBytes(), candidate.Value[:]...)
	err := batch.pruneByTimeBatch.Del(pruneKey)
	if err != nil {
		return fmt.Errorf("deleting pruning key: %w", err)
	}
	return nil
}

// storeChunk stores a chunk in the availability store, returns true on success, false on failure,
// and error on internal error.
func (as *AvailabilityStore) storeChunk(candidate parachaintypes.CandidateHash, chunk ErasureChunk) (bool,
	error) {
	batch := NewAvailabilityStoreBatch(as)

	meta, err := as.loadMeta(candidate)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			// we weren't informed of this candidate by import events
			return false, nil
		} else {
			return false, fmt.Errorf("load metadata: %w", err)
		}
	}

	if meta.ChunksStored[chunk.Index] {
		logger.Debugf("Chunk %d already stored", chunk.Index)
		return true, nil // already stored
	} else {
		err = as.writeChunk(batch, candidate, chunk)
		if err != nil {
			return false, fmt.Errorf("storing chunk for candidate %v, index %d: %w", candidate, chunk.Index, err)
		}

		meta.ChunksStored[chunk.Index] = true
		err = as.writeMeta(batch, candidate, *meta)
		if err != nil {
			return false, fmt.Errorf("storing metadata for candidate %v: %w", candidate, err)
		}
	}

	err = batch.Write()
	if err != nil {
		return false, fmt.Errorf("writing batch: %w", err)
	}

	logger.Debugf("stored chuck %d for %v", chunk.Index, candidate)
	return true, nil
}

//func (as *AvailabilityStore) storeAvailableData(candidate parachaintypes.CandidateHash,
//	nValidators uint, data AvailableData, expectedErasureRoot common.Hash) (bool, error) {
//	batch := NewAvailabilityStoreBatch(as)
//	meta, err := as.loadMeta(candidate)
//	if err != nil {
//		if errors.Is(err, database.ErrNotFound) {
//			// we weren't informed of this candidate by import events
//			return false, nil
//		} else {
//			return false, fmt.Errorf("load metadata: %w", err)
//		}
//	}
//}

// todo(ed) determin if this should be LittleEndian or BigEndian
func uint32ToBytes(value uint32) []byte {
	result := make([]byte, 4)
	binary.LittleEndian.PutUint32(result, value)
	return result
}

func uint32ToBytesBigEndian(value uint32) []byte {
	result := make([]byte, 4)
	binary.BigEndian.PutUint32(result, value)
	return result
}

// Run runs the availability store subsystem
func (av *AvailabilityStoreSubsystem) Run(ctx context.Context, OverseerToSubsystem chan any,
	SubsystemToOverseer chan any) error {

	av.wg.Add(1)
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
	_, err := av.availabilityStore.loadMeta(msg.CandidateHash)
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
	meta, err := av.availabilityStore.loadMeta(msg.CandidateHash)
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
	meta, err := av.availabilityStore.loadMeta(msg.CandidateHash)
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
	meta, err := av.availabilityStore.loadMeta(msg.CandidateHash)
	if err != nil {
		msg.Sender <- false
		return fmt.Errorf("load metadata: %w", err)
	}
	msg.Sender <- meta.ChunksStored[msg.ValidatorIndex]
	return nil
}

func (av *AvailabilityStoreSubsystem) handleStoreChunk(msg StoreChunk) error {
	_, err := av.availabilityStore.storeChunk(msg.CandidateHash, msg.Chunk)
	if err != nil {
		msg.Sender <- err
		return fmt.Errorf("store chunk: %w", err)
	}
	msg.Sender <- nil
	return nil
}

func (av *AvailabilityStoreSubsystem) handleStoreAvailableData(msg StoreAvailableData) error {
	// TODO: implement store available data
	//err := av.availabilityStore.writeAvailableData(msg.CandidateHash, msg.AvailableData)
	//if err != nil {
	//	msg.Sender <- err
	//	return fmt.Errorf("store available data: %w", err)
	//}
	//msg.Sender <- err // TODO: determine how to replicate Rust's Result type
	return nil
}

func (av *AvailabilityStoreSubsystem) Stop() {
	av.cancel()
	av.wg.Wait()
}
