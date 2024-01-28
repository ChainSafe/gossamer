// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package availabilitystore

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"sync"
	"time"

	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/internal/database"
	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/erasure"
	"github.com/ChainSafe/gossamer/lib/trie"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

var logger = log.NewFromGlobal(log.AddContext("pkg", "parachain-availability-store"))

const (
	availableDataPrefix = "available"
	chunkPrefix         = "chunk"
	metaPrefix          = "meta"
	unfinalizedPrefix   = "unfinalized"
	pruneByTimePrefix   = "prune_by_time"

	// Unavailable blocks are kept for 1 hour.
	keepUnavilableFor = time.Hour

	// Finalized data is kept for 25 hours.
	keepFinalizedFor = time.Hour * 25

	// The pruning interval.
	pruningInterval = time.Minute * 5
)

// BETimestamp is a unix time wrapper with big-endian encoding
type BETimestamp uint64

func timeNow() BETimestamp {
	return BETimestamp(time.Now().Unix())
}

// ToBigEndianBytes returns the big-endian encoding of the timestamp
func (b BETimestamp) ToBigEndianBytes() []byte {
	res := make([]byte, 8)
	binary.BigEndian.PutUint64(res, uint64(b))
	return res
}

// pruningConfig Struct holding pruning timing configuration.
// The only purpose of this structure is to use different timing
// configurations in production and in testing.
type pruningConfig struct {
	keepUnavailableFor time.Duration
	keepFinalizedFor   time.Duration
	pruningInterval    time.Duration
}

var defaultPruningConfig = pruningConfig{
	keepUnavailableFor: keepUnavilableFor,
	keepFinalizedFor:   keepFinalizedFor,
	pruningInterval:    pruningInterval,
}

// AvailabilityStoreSubsystem is the struct that holds subsystem data for the availability store
type AvailabilityStoreSubsystem struct {
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	SubSystemToOverseer    chan<- any
	OverseerToSubSystem    <-chan any
	availabilityStore      availabilityStore
	finalizedBlockNumber   parachaintypes.BlockNumber
	knownUnfinalizedBlocks knownUnfinalizedBlocks
	pruningConfig          pruningConfig
}

// NewAvailabilityStoreSubsystem creates a new instance of AvailabilityStoreSubsystem
func NewAvailabilityStoreSubsystem(db database.Database) *AvailabilityStoreSubsystem {
	availabilityStore := NewAvailabilityStore(db)

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)

	availabilityStoreSubsystem := &AvailabilityStoreSubsystem{
		ctx:                    ctx,
		cancel:                 cancel,
		pruningConfig:          defaultPruningConfig,
		availabilityStore:      *availabilityStore,
		knownUnfinalizedBlocks: *newKnownUnfinalizedBlock(),
	}

	return availabilityStoreSubsystem
}

type knownUnfinalizedBlocks struct {
	byHash   map[common.Hash]struct{}
	byNumber map[BlockEntry]struct{}
}

func newKnownUnfinalizedBlock() *knownUnfinalizedBlocks {
	return &knownUnfinalizedBlocks{
		byHash:   make(map[common.Hash]struct{}),
		byNumber: make(map[BlockEntry]struct{}),
	}
}

func (kud *knownUnfinalizedBlocks) isKnown(hash common.Hash) bool {
	_, ok := kud.byHash[hash]
	return ok
}

func (kud *knownUnfinalizedBlocks) insert(hash common.Hash, number parachaintypes.BlockNumber) {
	kud.byHash[hash] = struct{}{}
	kud.byNumber[BlockEntry{number, hash}] = struct{}{}
}

// availabilityStore is the struct that holds data for the availability store
type availabilityStore struct {
	available   database.Table
	chunk       database.Table
	meta        database.Table
	unfinalized database.Table
	pruneByTime database.Table
}

type availabilityStoreBatch struct {
	available   database.Batch
	chunk       database.Batch
	meta        database.Batch
	unfinalized database.Batch
	pruneByTime database.Batch
}

func newAvailabilityStoreBatch(as *availabilityStore) *availabilityStoreBatch {
	return &availabilityStoreBatch{
		available:   as.available.NewBatch(),
		chunk:       as.chunk.NewBatch(),
		meta:        as.meta.NewBatch(),
		unfinalized: as.unfinalized.NewBatch(),
		pruneByTime: as.pruneByTime.NewBatch(),
	}
}

// flush flushes the batch and resets the batch if error during flushing
func (asb *availabilityStoreBatch) flush() error {
	err := asb.flushAll()
	if err != nil {
		asb.reset()
	}
	return err
}

// flushAll flushes all the batches and returns the error
func (asb *availabilityStoreBatch) flushAll() error {
	err := asb.available.Flush()
	if err != nil {
		return fmt.Errorf("writing available batch: %w", err)
	}
	err = asb.chunk.Flush()
	if err != nil {
		return fmt.Errorf("writing chunk batch: %w", err)
	}
	err = asb.meta.Flush()
	if err != nil {
		return fmt.Errorf("writing meta batch: %w", err)
	}
	err = asb.unfinalized.Flush()
	if err != nil {
		return fmt.Errorf("writing unfinalized batch: %w", err)
	}
	err = asb.pruneByTime.Flush()
	if err != nil {
		return fmt.Errorf("writing prune by time batch: %w", err)
	}
	return nil
}

// reset resets the batch and returns the error
func (asb *availabilityStoreBatch) reset() {
	asb.available.Reset()
	asb.chunk.Reset()
	asb.meta.Reset()
	asb.unfinalized.Reset()
	asb.pruneByTime.Reset()
}

// NewAvailabilityStore creates a new instance of AvailabilityStore
func NewAvailabilityStore(db database.Database) *availabilityStore {
	return &availabilityStore{
		available:   database.NewTable(db, availableDataPrefix),
		chunk:       database.NewTable(db, chunkPrefix),
		meta:        database.NewTable(db, metaPrefix),
		unfinalized: database.NewTable(db, unfinalizedPrefix),
		pruneByTime: database.NewTable(db, pruneByTimePrefix),
	}
}

// loadAvailableData loads available data from the availability store
func (as *availabilityStore) loadAvailableData(candidate parachaintypes.CandidateHash) (*AvailableData, error) {
	resultBytes, err := as.available.Get(candidate.Value[:])
	if err != nil {
		return nil, fmt.Errorf("getting candidate %v from available table: %w", candidate.Value, err)
	}
	result := AvailableData{}
	err = scale.Unmarshal(resultBytes, &result)
	if err != nil {
		return nil, fmt.Errorf("unmarshalling available data: %w", err)
	}
	return &result, nil
}

// loadMeta loads meta data from the availability store
func (as *availabilityStore) loadMeta(candidate parachaintypes.CandidateHash) (*CandidateMeta, error) {
	resultBytes, err := as.meta.Get(candidate.Value[:])
	if err != nil {
		return nil, fmt.Errorf("getting candidate %v from meta table: %w", candidate.Value, err)
	}
	result := CandidateMeta{}
	err = scale.Unmarshal(resultBytes, &result)
	if err != nil {
		return nil, fmt.Errorf("unmarshalling candidate meta: %w", err)
	}
	return &result, nil
}

// loadChunk loads a chunk from the availability store
func (as *availabilityStore) loadChunk(candidate parachaintypes.CandidateHash, validatorIndex uint32) (*ErasureChunk,
	error) {
	resultBytes, err := as.chunk.Get(append(candidate.Value[:], uint32ToBytes(validatorIndex)...))
	if err != nil {
		return nil, fmt.Errorf("getting candidate %v, index %d from chunk table: %w", candidate.Value, validatorIndex, err)
	}
	result := ErasureChunk{}
	err = scale.Unmarshal(resultBytes, &result)
	if err != nil {
		return nil, fmt.Errorf("unmarshalling chunk: %w", err)
	}
	return &result, nil
}

// storeChunk stores a chunk in the availability store, returns true on success, false on failure,
// and error on internal error.
func (as *availabilityStore) storeChunk(candidate parachaintypes.CandidateHash, chunk ErasureChunk) (bool,
	error) {
	batch := newAvailabilityStoreBatch(as)

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
	}

	dataBytes, err := scale.Marshal(chunk)
	if err != nil {
		return false, fmt.Errorf("marshalling chunk for candidate %v, index %d: %w", candidate, chunk.Index, err)
	}
	err = batch.chunk.Put(append(candidate.Value[:], uint32ToBytes(chunk.Index)...), dataBytes)
	if err != nil {
		return false, fmt.Errorf("writing chunk for candidate %v, index %d: %w", candidate, chunk.Index, err)
	}

	meta.ChunksStored[chunk.Index] = true

	err = writeMeta(batch.meta, candidate, meta)
	if err != nil {
		return false, fmt.Errorf("storing metadata for candidate %v: %w", candidate, err)
	}

	err = batch.flush()
	if err != nil {
		return false, fmt.Errorf("writing batch: %w", err)
	}

	logger.Debugf("stored chuck %d for %v", chunk.Index, candidate)
	return true, nil
}

func (as *availabilityStore) storeAvailableData(subsystem *AvailabilityStoreSubsystem,
	candidate parachaintypes.CandidateHash, nValidators uint32, data AvailableData,
	expectedErasureRoot common.Hash) (bool, error) {
	batch := newAvailabilityStoreBatch(as)
	meta, err := as.loadMeta(candidate)
	if err != nil && !errors.Is(err, database.ErrNotFound) {
		return false, fmt.Errorf("load metadata: %w", err)
	}
	if meta != nil && meta.DataAvailable {
		return true, nil // already stored
	}

	candidateMeta := CandidateMeta{}

	now := timeNow()
	pruneAt := now + BETimestamp(subsystem.pruningConfig.KeepUnavailableFor.Seconds())
	err = subsystem.writePruningKey(batch.pruneByTime, pruneAt, candidate)
	if err != nil {
		return false, fmt.Errorf("writing pruning key: %w", err)
	}

	candidateMeta.State = NewStateVDT()
	err = candidateMeta.State.Set(Unavailable{Timestamp: now})
	if err != nil {
		return false, fmt.Errorf("setting state to unavailable: %w", err)
	}
	candidateMeta.DataAvailable = false
	candidateMeta.ChunksStored = make([]bool, nValidators)

	dataEncoded, err := scale.Marshal(data)
	if err != nil {
		return false, fmt.Errorf("encoding data: %w", err)
	}

	chunks, err := erasure.ObtainChunks(uint(nValidators), dataEncoded)
	if err != nil {
		return false, fmt.Errorf("obtaining chunks: %w", err)
	}

	branches, err := branchesFromChunks(chunks)
	if err != nil {
		return false, fmt.Errorf("creating branches from chunks: %w", err)
	}
	if branches.root != expectedErasureRoot {
		return false, errInvalidErasureRoot
	}

	for i, chunk := range chunks {
		erasureChunk := ErasureChunk{
			Index: uint32(i),
			Chunk: chunk,
		}

		dataBytes, err := scale.Marshal(erasureChunk)
		if err != nil {
			return false, fmt.Errorf("marshalling chunk for candidate %v, index %d: %w", candidate, erasureChunk.Index, err)
		}
		err = batch.chunk.Put(append(candidate.Value[:], uint32ToBytes(erasureChunk.Index)...), dataBytes)
		if err != nil {
			return false, fmt.Errorf("writing chunk for candidate %v, index %d: %w", candidate, erasureChunk.Index, err)
		}

		candidateMeta.ChunksStored[i] = true
	}

	candidateMeta.DataAvailable = true
	candidateMeta.ChunksStored = make([]bool, nValidators)
	for i := range candidateMeta.ChunksStored {
		candidateMeta.ChunksStored[i] = true
	}

	err = writeMeta(batch.meta, candidate, &candidateMeta)
	if err != nil {
		return false, fmt.Errorf("storing metadata for candidate %v: %w", candidate, err)
	}

	dataBytes, err := scale.Marshal(data)
	if err != nil {
		return false, fmt.Errorf("marshalling available data: %w", err)
	}
	err = batch.available.Put(candidate.Value[:], dataBytes)
	if err != nil {
		return false, fmt.Errorf("storing available data for candidate %v: %w", candidate, err)
	}

	err = batch.flush()
	if err != nil {
		return false, fmt.Errorf("writing batch: %w", err)
	}

	logger.Debugf("stored data and chunks for %v", candidate.Value)
	return true, nil
}

func uint32ToBytes(value uint32) []byte {
	result := make([]byte, 4)
	binary.BigEndian.PutUint32(result, value)
	return result
}

func branchesFromChunks(chunks [][]byte) (branches, error) {
	tr := trie.NewEmptyTrie()

	for i, chunk := range chunks {
		err := tr.Put(uint32ToBytes(uint32(i)), common.MustBlake2bHash(chunk).ToBytes())
		if err != nil {
			return branches{}, fmt.Errorf("putting chunk %d in trie: %w", i, err)
		}
	}
	branchHash, err := trie.V1.Hash(tr)
	if err != nil {
		return branches{}, fmt.Errorf("hashing trie: %w", err)
	}
	b := branches{
		trieStorage: tr,
		root:        branchHash,
		chunks:      chunks,
		currentPos:  0,
	}
	return b, nil
}

// Run runs the availability store subsystem
func (av *AvailabilityStoreSubsystem) Run(ctx context.Context, OverseerToSubsystem chan any,
	SubsystemToOverseer chan any) {

	av.wg.Add(1)
	go av.processMessages()
}

// Name returns the name of the availability store subsystem
func (*AvailabilityStoreSubsystem) Name() parachaintypes.SubSystemName {
	return parachaintypes.AvailabilityStore
}

func (av *AvailabilityStoreSubsystem) processMessages() {
	for {
		select {
		case msg := <-av.OverseerToSubSystem:
			logger.Infof("received message %T, %v", msg, msg)
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
				err := av.ProcessActiveLeavesUpdateSignal(msg)
				if err != nil {
					logger.Errorf("failed to process active leaves update signal: %w", err)
				}

			case parachaintypes.BlockFinalizedSignal:
				err := av.ProcessBlockFinalizedSignal(msg)
				if err != nil {
					logger.Errorf("failed to process block finalized signal: %w", err)
				}

			default:
				if msg != nil {
					logger.Infof("unknown message type %T", msg)
					logger.Error(parachaintypes.ErrUnknownOverseerMessage.Error())
					// this error shouldn't happen, so we'll panic to catch it during development
					panic(parachaintypes.ErrUnknownOverseerMessage.Error())
				}
			}
		case <-av.ctx.Done():
			if err := av.ctx.Err(); err != nil {
				logger.Errorf("ctx error: %v", err)
			}
			av.wg.Done()
			return
		case <-time.After(av.pruningConfig.PruningInterval):
			av.pruneAll()
		}
	}
}

func (av *AvailabilityStoreSubsystem) ProcessActiveLeavesUpdateSignal(signal parachaintypes.
	ActiveLeavesUpdateSignal) error {
	now := timeNow()
	logger.Infof("ProcessActiveLeavesUpdateSignal %s", signal)

	respChan := make(chan any)
	message := chainapi.ChainAPIMessage[chainapi.BlockHeader]{
		Message: chainapi.BlockHeader{
			Hash: signal.Activated.Hash,
		},
		ResponseChannel: respChan,
	}
	response, err := chainapi.Call(av.SubSystemToOverseer, message, message.ResponseChannel)
	if err != nil {
		return fmt.Errorf("sending message to get block header: %w", err)
	}

	newBlocks, err := chainapi.DetermineNewBlocks(av.SubSystemToOverseer, av.knownUnfinalizedBlocks.isKnown,
		signal.Activated.Hash,
		response.(types.Header),
		av.finalizedBlockNumber)
	if err != nil {
		return fmt.Errorf("failed to determine new blocks: %w", err)
	}

	for _, v := range newBlocks {
		// start db batch
		tx := newAvailabilityStoreBatch(&av.availabilityStore)

		err := av.processNewHead(tx, v.Hash, now, v.Header)
		if err != nil {
			return fmt.Errorf("failed to process new head: %w", err)
		}

		// add to known blocks
		av.knownUnfinalizedBlocks.insert(v.Hash, parachaintypes.BlockNumber(v.Header.Number))

		// end db batch
		err = tx.flush()
		if err != nil {
			return fmt.Errorf("failed to flush tx: %w", err)
		}
	}
	return nil
}

func (av *AvailabilityStoreSubsystem) processNewHead(tx *availabilityStoreBatch, hash common.Hash, now BETimestamp,
	header types.Header) error {
	logger.Infof("processNewHead hash %s, now %v, header %v\n", hash, now, header)
	// TODO: call requestValidators to determine number of validators, see issue #3932
	nValidators := uint(10)

	// call to get runtime
	respChan := make(chan any)
	message := parachain.RuntimeAPIMessage{Hash: hash, Resp: respChan}

	rtRes, err := chainapi.Call(av.SubSystemToOverseer, message, respChan)
	if err != nil {
		return fmt.Errorf("sending message to get block header: %w", err)
	}
	runtime := rtRes.(parachain.RuntimeInstance)

	candidateEvents, err := runtime.ParachainHostCandidateEvents()
	if err != nil {
		return fmt.Errorf("failed to get candidate events: %w", err)
	}

	for _, v := range candidateEvents.Types {
		event, err := v.Value()
		if err != nil {
			return fmt.Errorf("failed to get candidate event value: %w", err)
		}
		switch event := event.(type) {
		case parachaintypes.CandidateBacked:
			err := av.noteBlockBacked(tx, now, nValidators, event.CandidateReceipt)
			if err != nil {
				return fmt.Errorf("failed to note block backed: %w", err)
			}
		case parachaintypes.CandidateIncluded:
			err := av.noteBlockIncluded(tx, parachaintypes.BlockNumber(header.Number), header.Hash(),
				event.CandidateReceipt)
			if err != nil {
				return fmt.Errorf("failed to note block included: %w", err)
			}
		}
	}
	return nil
}

func (av *AvailabilityStoreSubsystem) noteBlockBacked(tx *availabilityStoreBatch, now BETimestamp, nValidators uint,
	candidate parachaintypes.CandidateReceipt) error {
	hash, err := candidate.Hash()
	if err != nil {
		return fmt.Errorf("failed to hash candidate: %w", err)
	}
	candidateHash := parachaintypes.CandidateHash{Value: hash}
	meta, err := av.availabilityStore.loadMeta(candidateHash)
	if err != nil {
		return fmt.Errorf("failed to load meta for candidate %v: %w", candidateHash, err)
	}
	if meta == nil {
		state := NewStateVDT()
		err := state.Set(Unavailable{now})
		if err != nil {
			return fmt.Errorf("failed to set state to unavailable: %w", err)
		}
		meta = &CandidateMeta{
			State:         state,
			DataAvailable: false,
			ChunksStored:  make([]bool, nValidators),
		}

		err = writeMeta(tx.meta, candidateHash, meta)
		if err != nil {
			return fmt.Errorf("storing metadata for candidate %v: %w", candidate, err)
		}

		pruneAt := now + BETimestamp(av.pruningConfig.KeepUnavailableFor.Seconds())
		err = av.writePruningKey(tx.pruneByTime, pruneAt, candidateHash)
		if err != nil {
			return fmt.Errorf("writing pruning key: %w", err)
		}
	}
	return nil
}

func (av *AvailabilityStoreSubsystem) noteBlockIncluded(tx *availabilityStoreBatch,
	blockNumber parachaintypes.BlockNumber, blockHash common.Hash,
	candidate parachaintypes.CandidateReceipt) error {
	hash, err := candidate.Hash()
	if err != nil {
		return fmt.Errorf("failed to hash candidate: %w", err)
	}
	candidateHash := parachaintypes.CandidateHash{Value: hash}
	meta, err := av.availabilityStore.loadMeta(candidateHash)
	if err != nil {
		return fmt.Errorf("failed to load meta for candidate %v: %w", candidateHash, err)
	}

	if meta == nil {
		return fmt.Errorf("Candidate included without being backed %v", candidateHash)
	}
	beBlock := BlockEntry{
		BlockNumber: blockNumber,
		BlockHash:   blockHash,
	}
	stateValue, err := meta.State.Value()
	if err != nil {
		return fmt.Errorf("failed to get state value: %w", err)
	}

	switch val := stateValue.(type) {
	case Unavailable:
		pruneAt := val.Timestamp + BETimestamp(av.pruningConfig.KeepUnavailableFor.Seconds())

		pruneKey := append(uint32ToBytes(uint32(pruneAt)), candidateHash.Value[:]...)
		err = tx.pruneByTime.Del(pruneKey)
		if err != nil {
			return fmt.Errorf("failed to delete pruning key: %w", err)
		}
		err = meta.State.Set(Unfinalized{
			Timestamp:  val.Timestamp,
			BlockEntry: []BlockEntry{beBlock},
		})
		if err != nil {
			return fmt.Errorf("failed to set state to unfinalized: %w", err)
		}
	case Unfinalized:
		err = meta.State.Set(Unfinalized{
			Timestamp:  val.Timestamp,
			BlockEntry: append(val.BlockEntry, beBlock),
		})
		if err != nil {
			return fmt.Errorf("failed to set state to unfinalized: %w", err)
		}
	case Finalized:
		// This should never happen as a candidate would have to be included after
		// finality.
	}

	// write unfinalized block contains
	key := append(uint32ToBytes(uint32(blockNumber)), blockHash[:]...)
	key = append(key, candidateHash.Value[:]...)
	err = tx.unfinalized.Put(key, nil)
	if err != nil {
		return fmt.Errorf("failed to put unfinalized key: %w", err)
	}

	err = writeMeta(tx.meta, candidateHash, meta)
	if err != nil {
		return fmt.Errorf("failed to put meta key: %w", err)
	}
	return nil
}

func (av *AvailabilityStoreSubsystem) ProcessBlockFinalizedSignal(signal parachaintypes.BlockFinalizedSignal) error {
	logger.Infof("ProcessBlockFinalizedSignal %T, %v", signal, signal)
	now := timeNow()

	// load all of finalized height
	batch := av.loadAllAtFinalizedHeight(signal.BlockNumber, signal.Hash)

	// delete unfinalized height
	tx := newAvailabilityStoreBatch(&av.availabilityStore)
	av.deleteUnfinalizedHeight(tx.unfinalized, signal.BlockNumber)

	// update blocks at finalized height
	av.updateBlockAtFinalizedHeight(tx, batch, signal.BlockNumber, now)

	err := tx.flush()
	if err != nil {
		return fmt.Errorf("failed to flush tx: %w", err)
	}
	return nil
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
	// TODO: add to metric on_chunks_received
	res, err := av.availabilityStore.storeAvailableData(av, msg.CandidateHash, msg.NumValidators,
		msg.AvailableData,
		msg.ExpectedErasureRoot)
	if res {
		msg.Sender <- nil
		return nil
	}
	if err != nil && errors.Is(err, errInvalidErasureRoot) {
		msg.Sender <- err
		return fmt.Errorf("store available data: %w", err)
	}
	if err != nil {
		// We do not bubble up internal errors to caller subsystems, instead the
		// tx channel is dropped and that error is caught by the caller subsystem.
		//
		// We bubble up the specific error here so `av-store` logs still tell what
		// happened.
		msg.Sender <- fmt.Errorf("store available data: %w", err)
		return fmt.Errorf("store available data: %w", err)
	}
	return nil
}

func (av *AvailabilityStoreSubsystem) pruneAll() {
	now := timeNow()
	iter, err := av.availabilityStore.pruneByTime.NewIterator()
	if err != nil {
		logger.Errorf("creating iterator: %w", err)
		return
	}
	defer iter.Release()
	for iter.First(); iter.Valid(); iter.Next() {
		key := iter.Key()

		pruneAt := binary.BigEndian.Uint32(key[len(pruneByTimePrefix) : len(pruneByTimePrefix)+4])
		if pruneAt > uint32(now) {
			continue
		}

		err := av.processPruneKey(key)
		if err != nil {
			logger.Errorf("failed to process prune key: %w", err)
		}
	}
}

func (av *AvailabilityStoreSubsystem) writePruningKey(writer database.Writer, pruneTime BETimestamp,
	hash parachaintypes.CandidateHash) error {
	key := append((uint32ToBytes(uint32(pruneTime))), hash.Value[:]...)
	return writer.Put(key, nil)
}

func (av *AvailabilityStoreSubsystem) processPruneKey(key []byte) error {
	candidateHash := key[len(pruneByTimePrefix)+4:]
	err := av.availabilityStore.pruneByTime.Del(key[len(pruneByTimePrefix):])
	if err != nil {
		logger.Errorf("failed to delete key: %w", err)
		return err
	}
	meta, err := av.availabilityStore.loadMeta(parachaintypes.CandidateHash{Value: common.Hash(candidateHash)})
	if err != nil {
		logger.Errorf("failed to load meta: %w", err)
		return err
	}

	if meta.DataAvailable {
		// delete key from available
		err = av.availabilityStore.available.Del(candidateHash)
		if err != nil {
			logger.Errorf("failed to delete key: %w", err)
		}
	}

	// delete chunks
	for i := range meta.ChunksStored {
		err = av.availabilityStore.chunk.Del(append(candidateHash, uint32ToBytes(uint32(i))...))
		if err != nil {
			logger.Errorf("failed to delete key: %w", err)
		}
	}

	// delete from meta
	err = av.availabilityStore.meta.Del(candidateHash)
	if err != nil {
		logger.Errorf("failed to delete key: %w", err)
	}

	stateValue, err := meta.State.Value()
	if err != nil {
		logger.Errorf("failed to get state value: %w", err)
	}

	switch state := stateValue.(type) {
	case Unfinalized:
		for _, v := range state.BlockEntry {
			key := append(uint32ToBytes(uint32(v.BlockNumber)), v.BlockHash[:]...)
			key = append(key, candidateHash...)
			err = av.availabilityStore.unfinalized.Del(key)
			if err != nil {
				logger.Errorf("failed to delete key: %w", err)
			}
		}
	}
	return nil
}

func (av *AvailabilityStoreSubsystem) Stop() {
	av.cancel()
	av.wg.Wait()
}

func (av *AvailabilityStoreSubsystem) deleteUnfinalizedHeight(writer database.Writer,
	blockNumber uint32) {
	iter, err := av.availabilityStore.unfinalized.NewIterator()
	if err != nil {
		logger.Errorf("creating iterator: %w", err)
	}
	defer iter.Release()

	for iter.First(); iter.Valid(); iter.Next() {
		key := iter.Key()
		number, _, _ := decodeUnfinalizedKey(key)
		if number < blockNumber {
			continue
		}
		if number == blockNumber {
			err := writer.Del(key[len(unfinalizedPrefix):])
			if err != nil {
				logger.Errorf("failed to delete unfinalized key: %w", err)
			}
		}
		if number > blockNumber {
			break
		}
	}
}

func (av *AvailabilityStoreSubsystem) loadAllAtFinalizedHeight(finalizedNumber uint32,
	finalizedHash common.Hash) map[parachaintypes.CandidateHash]bool {
	result := make(map[parachaintypes.CandidateHash]bool)
	iter, err := av.availabilityStore.unfinalized.NewIterator()
	if err != nil {
		logger.Errorf("creating iterator: %w", err)
	}
	defer iter.Release()

	for iter.First(); iter.Valid(); iter.Next() {
		key := iter.Key()
		blockNum, blockHash, candidateHash := decodeUnfinalizedKey(key)
		if blockNum < finalizedNumber {
			continue
		}
		if blockNum == finalizedNumber {
			if blockHash == finalizedHash {
				result[candidateHash] = true
			} else {
				result[candidateHash] = false
			}
		}
		if blockNum > finalizedNumber {
			break
		}
	}

	return result
}

func decodeUnfinalizedKey(key []byte) (blockNumber uint32, blockHash common.Hash,
	candidateHash parachaintypes.CandidateHash) {
	prefixLen := len(unfinalizedPrefix)
	blockNumber = binary.BigEndian.Uint32(key[prefixLen : prefixLen+4])
	blockHash = common.Hash(key[prefixLen+4 : prefixLen+36])
	candidateHash = parachaintypes.CandidateHash{Value: common.Hash(key[prefixLen+36:])}
	return
}

func writeMeta(writer database.Writer, hash parachaintypes.CandidateHash, meta *CandidateMeta) error {
	dataBytes, err := scale.Marshal(*meta)
	if err != nil {
		return err
	}
	err = writer.Put(hash.Value[:], dataBytes)
	if err != nil {
		return err
	}
	return nil
}

func (av *AvailabilityStoreSubsystem) updateBlockAtFinalizedHeight(tx *availabilityStoreBatch,
	candidates map[parachaintypes.CandidateHash]bool,
	blockNumber uint32, now BETimestamp) {
	for candidateHash, isFinalized := range candidates {
		meta, err := av.availabilityStore.loadMeta(candidateHash)
		if err != nil {
			logger.Errorf("failed to load meta for candidate %v: %w", candidateHash, err)
		}
		if isFinalized {
			stateValue, err := meta.State.Value()
			if err != nil {
				logger.Errorf("failed to get state value: %w", err)
			}
			switch val := stateValue.(type) {
			case Finalized:
				continue // sanity check
			case Unavailable:
				// This is also not going to happen; the very fact that we are
				// iterating over the candidate here indicates that `State` should
				// be `Unfinalized`.
				err = tx.pruneByTime.Del(append(uint32ToBytes(uint32(val.Timestamp)), candidateHash.Value[:]...))
				if err != nil {
					logger.Errorf("failed to delete pruning key: %w", err)
				}
			case Unfinalized:
				for _, v := range val.BlockEntry {
					if v.BlockNumber != parachaintypes.BlockNumber(blockNumber) {
						// deleteUnfinalizedInclusion
						key := append(uint32ToBytes(uint32(v.BlockNumber)), v.BlockHash[:]...)
						key = append(key, candidateHash.Value[:]...)
						err = tx.unfinalized.Del(key)
						if err != nil {
							logger.Errorf("failed to delete unfinalized key: %w", err)
						}
					}
				}
			}

			err = meta.State.Set(Finalized{Timestamp: now})
			if err != nil {
				logger.Errorf("failed to set state to finalized: %w", err)
			}

			// write meta
			err = writeMeta(tx.meta, candidateHash, meta)
			if err != nil {
				logger.Errorf("storing metadata for candidate %v: %w", candidateHash, err)
			}

			// write pruning key
			pruneAt := now + BETimestamp(av.pruningConfig.KeepFinalizedFor.Seconds())
			err = av.writePruningKey(tx.pruneByTime, pruneAt, candidateHash)
			if err != nil {
				logger.Errorf("writing pruning key: %w", err)
			}

		} else {
			stateValue, err := meta.State.Value()
			if err != nil {
				logger.Errorf("failed to get state value: %w", err)
			}
			switch val := stateValue.(type) {
			case Finalized:
				continue // sanity
			case Unavailable:
				continue // sanity
			case Unfinalized:
				retainedBlocks := []BlockEntry{}
				for _, v := range val.BlockEntry {
					if v.BlockNumber != parachaintypes.BlockNumber(blockNumber) {
						retainedBlocks = append(retainedBlocks, v)
					}
				}
				if len(retainedBlocks) == 0 {
					// write pruning key
					pruneAt := val.Timestamp + BETimestamp(av.pruningConfig.KeepUnavailableFor.Seconds())
					err = av.writePruningKey(tx.pruneByTime, pruneAt, candidateHash)
					if err != nil {
						logger.Errorf("writing pruning key: %w", err)
					}

					err = meta.State.Set(Unavailable{Timestamp: val.Timestamp})
					if err != nil {
						logger.Errorf("failed to set state to unavailable: %w", err)
					}
				} else {
					err = meta.State.Set(Unfinalized{Timestamp: val.Timestamp, BlockEntry: retainedBlocks})
					if err != nil {
						logger.Errorf("failed to set state to unfinalized: %w", err)
					}
				}
			}

			// write meta
			err = writeMeta(tx.meta, candidateHash, meta)
			if err != nil {
				logger.Errorf("failed to put meta: %w", err)
			}
		}
	}
}
