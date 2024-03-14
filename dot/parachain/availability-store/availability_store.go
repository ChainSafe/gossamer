// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package availabilitystore

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	parachain "github.com/ChainSafe/gossamer/dot/parachain/runtime"
	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/dot/parachain/util"
	"github.com/ChainSafe/gossamer/dot/state"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/internal/database"
	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/erasure"
	"github.com/ChainSafe/gossamer/lib/trie"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/stretchr/testify/require"
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

// ToBigEndianBytes returns the big-endian encoding of the timestamp
func (b BETimestamp) ToBigEndianBytes() []byte {
	res := make([]byte, 8)
	binary.BigEndian.PutUint64(res, uint64(b))
	return res
}

type subsystemClock struct{}

func (sc *subsystemClock) Now() BETimestamp {
	return BETimestamp(time.Now().Unix())
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

	SubSystemToOverseer chan<- any
	OverseerToSubSystem <-chan any
	availabilityStore   availabilityStore
	currentMessage      any
	finalizedNumber     parachaintypes.BlockNumber
	knownBlocks         KnownUnfinalizedBlock
	pruningConfig       pruningConfig
	clock               subsystemClock
	//TODO: metrics       Metrics
}
type BlockEntry struct {
	blockNumber parachaintypes.BlockNumber
	hash        common.Hash
}

type KnownUnfinalizedBlock struct {
	byHash   map[common.Hash]struct{}
	byNumber map[BlockEntry]struct{}
}

func (kud *KnownUnfinalizedBlock) isKnown(hash common.Hash) bool {
	_, ok := kud.byHash[hash]
	return ok
}

func (kud *KnownUnfinalizedBlock) insert(hash common.Hash, number parachaintypes.BlockNumber) {
	if kud.byHash == nil {
		kud.byHash = make(map[common.Hash]struct{})
	}
	if kud.byNumber == nil {
		kud.byNumber = make(map[BlockEntry]struct{})
	}
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

var (
	TestChunk1 = ErasureChunk{
		Chunk: []byte("chunk1"),
		Index: 0,
		Proof: []byte("proof1"),
	}
	TestChunk2 = ErasureChunk{
		Chunk: []byte("chunk2"),
		Index: 1,
		Proof: []byte("proof2"),
	}
	TestavailableData1 = AvailableData{
		PoV: parachaintypes.PoV{BlockData: []byte("blockdata")},
		ValidationData: parachaintypes.PersistedValidationData{
			ParentHead: parachaintypes.HeadData{Data: []byte("parentHead")},
		},
	}

	TestCandidateHash = parachaintypes.CandidateHash{Value: common.Hash{0x01}}
)

func SetupTestDB(t *testing.T) database.Database {
	inmemoryDB := state.NewInMemoryDB(t)
	as := NewAvailabilityStore(inmemoryDB)
	batch := newAvailabilityStoreBatch(as)
	metaState := NewStateVDT()
	err := metaState.Set(Unavailable{})
	require.NoError(t, err)
	meta := CandidateMeta{
		State:         metaState,
		DataAvailable: false,
		ChunksStored:  []bool{false, false, false},
	}

	dataBytes, err := scale.Marshal(meta)
	require.NoError(t, err)
	err = batch.meta.Put(TestCandidateHash.Value[:], dataBytes)
	require.NoError(t, err)
	err = batch.meta.Put(common.Hash{0x02}.ToBytes(), dataBytes)
	require.NoError(t, err)

	err = batch.flush()
	require.NoError(t, err)

	stored, err := as.storeChunk(parachaintypes.CandidateHash{Value: common.Hash{0x01}}, TestChunk1)
	require.NoError(t, err)
	require.Equal(t, true, stored)
	stored, err = as.storeChunk(parachaintypes.CandidateHash{Value: common.Hash{0x01}}, TestChunk2)
	require.NoError(t, err)
	require.Equal(t, true, stored)

	batch = newAvailabilityStoreBatch(as)
	dataBytes, err = scale.Marshal(TestavailableData1)
	require.NoError(t, err)
	err = batch.available.Put(TestCandidateHash.Value[:], dataBytes)
	require.NoError(t, err)

	err = batch.flush()
	require.NoError(t, err)

	return inmemoryDB
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

	dataBytes, err = scale.Marshal(*meta)
	if err != nil {
		return false, fmt.Errorf("marshalling meta for candidate: %w", err)
	}
	err = batch.meta.Put(candidate.Value[:], dataBytes)
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
	candidate parachaintypes.CandidateHash, nValidators uint, data AvailableData,
	expectedErasureRoot common.Hash) (bool, error) {
	batch := newAvailabilityStoreBatch(as)
	meta, err := as.loadMeta(candidate)
	if err != nil && !errors.Is(err, database.ErrNotFound) {
		return false, fmt.Errorf("load metadata: %w", err)
	}
	if meta != nil && meta.DataAvailable {
		return true, nil // already stored
	}

	meta = &CandidateMeta{}

	now := subsystem.clock.Now()
	pruneAt := now + BETimestamp(subsystem.pruningConfig.keepUnavailableFor.Seconds())

	pruneKey := append(pruneAt.ToBigEndianBytes(), candidate.Value[:]...)
	err = batch.pruneByTime.Put(pruneKey, nil)
	if err != nil {
		return false, fmt.Errorf("writing pruning key: %w", err)
	}

	meta.State = NewStateVDT()
	err = meta.State.Set(Unavailable{Timestamp: now})
	if err != nil {
		return false, fmt.Errorf("setting state to unavailable: %w", err)
	}
	meta.DataAvailable = false
	meta.ChunksStored = make([]bool, nValidators)

	dataEncoded, err := scale.Marshal(data)
	if err != nil {
		return false, fmt.Errorf("encoding data: %w", err)
	}

	chunks, err := erasure.ObtainChunks(nValidators, dataEncoded)
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

		meta.ChunksStored[i] = true
	}

	meta.DataAvailable = true
	meta.ChunksStored = make([]bool, nValidators)
	for i := range meta.ChunksStored {
		meta.ChunksStored[i] = true
	}

	dataBytes, err := scale.Marshal(meta)
	if err != nil {
		return false, fmt.Errorf("marshalling meta for candidate: %w", err)
	}
	err = batch.meta.Put(candidate.Value[:], dataBytes)
	if err != nil {
		return false, fmt.Errorf("storing metadata for candidate %v: %w", candidate, err)
	}

	dataBytes, err = scale.Marshal(data)
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
			logger.Infof("received message %v", msg)
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
				av.ProcessBlockFinalizedSignal(msg)

			default:
				// TODO(ed): handle unknown message type (if necessary)
				if msg != nil {
					logger.Infof("unknown message type %T", msg)
					logger.Error(parachaintypes.ErrUnknownOverseerMessage.Error())
				}
			}
		case <-av.ctx.Done():
			if err := av.ctx.Err(); err != nil {
				logger.Errorf("ctx error: %v", err)
			}
			av.wg.Done()
			return
		}
	}
}

func (av *AvailabilityStoreSubsystem) ProcessActiveLeavesUpdateSignal() {
	now := av.clock.Now()
	logger.Infof("ProcessActiveLeavesUpdateSignal %s", av.currentMessage)
	activeLeave := av.currentMessage.(parachaintypes.ActiveLeavesUpdateSignal)
	// TODO: get block header from activated number

	respChan := make(chan any)
	message := util.ChainAPIMessage[util.BlockHeader]{
		Message: util.BlockHeader{
			Hash: activeLeave.Activated.Hash,
		},
		ResponseChannel: respChan,
	}
	res, err := util.Call(av.SubSystemToOverseer, message, message.ResponseChannel)
	if err != nil {
		logger.Errorf("sending message to get block header: %w", err)
	}
	logger.Infof("block header res %s", res)

	blockNumber := activeLeave.Activated.Number

	logger.Infof("ProcessActiveLeavesUpdateSignal blocknumber %s", blockNumber)

	// todo: confirm these params are correct (last param should be finalized block number or block_number -1,
	// not sure which)
	newBlocks, err := util.DetermineNewBlocks(av.SubSystemToOverseer, av.knownBlocks.isKnown, activeLeave.Activated.Hash,
		res.(types.Header),
		av.finalizedNumber)
	if err != nil {
		logger.Errorf("failed to determine new blocks: %w", err)
	}

	logger.Infof("newBlocks %v", newBlocks)

	for i, v := range newBlocks {
		// start db batch
		tx := newAvailabilityStoreBatch(&av.availabilityStore)
		logger.Infof("newBlock %v %v", i, v)
		// process new head

		av.processNewHead(tx, v.Hash, now, v.Header)
		// add to known blocks
		av.knownBlocks.insert(v.Hash, parachaintypes.BlockNumber(v.Header.Number))

		// end db batch
		err := tx.flush()
		if err != nil {
			logger.Errorf("failed to flush tx: %w", err)
		}
	}
}

func (av *AvailabilityStoreSubsystem) processNewHead(tx *availabilityStoreBatch, hash common.Hash, now BETimestamp,
	header types.Header) {
	logger.Infof("processNewHead hash %s, now %v, header %v\n", hash, now, header)
	// TODO: call requestValidators to determine number of validators
	nValidators := uint(0)

	// call to get runtime
	respChan := make(chan any)
	message := parachain.RuntimeAPIMessage{Hash: hash, Resp: respChan}

	rtRes, err := util.Call(av.SubSystemToOverseer, message, respChan)
	if err != nil {
		logger.Errorf("sending message to get block header: %w", err)
	}
	runtime := rtRes.(parachain.RuntimeInstance)

	candidateEvents, err := runtime.ParachainHostCandidateEvents()
	if err != nil {
		logger.Errorf("failed to get candidate events: %w", err)
		return
	}
	logger.Infof("candidateEvents %v", candidateEvents)

	for i, v := range candidateEvents.Types {
		logger.Infof("candidateEvent %v %v", i, v)
		event, err := v.Value()
		if err != nil {
			logger.Errorf("failed to get candidate event value: %w", err)
		}
		switch event := event.(type) {
		case parachaintypes.CandidateBacked:
			av.noteBlockBacked(tx, now, nValidators, event.CandidateReceipt)

		case parachaintypes.CandidateIncluded:
			av.noteBlockIncluded(tx, parachaintypes.BlockNumber(header.Number), header.Hash(),
				event.CandidateReceipt)
		}
	}
}

func (av *AvailabilityStoreSubsystem) noteBlockBacked(tx *availabilityStoreBatch, now BETimestamp, nValidators uint,
	candidate parachaintypes.CandidateReceipt) {
	candidateHash, err := candidate.Hash()
	if err != nil {
		logger.Errorf("failed to hash candidate: %w", err)
		return
	}
	logger.Infof("Candidate backed %x", candidateHash)
	meta, err := av.availabilityStore.loadMeta(parachaintypes.CandidateHash{Value: candidateHash})
	if err != nil {
		logger.Errorf("failed to load meta for candidate %x: %w", candidateHash, err)
	}

	if meta == nil {
		state := State{}.New()
		err := state.Set(Unavailable{now})
		if err != nil {
			logger.Errorf("failed to set state to unavailable: %w", err)
		}
		meta = &CandidateMeta{
			State:         state,
			DataAvailable: false,
			ChunksStored:  make([]bool, nValidators),
		}
		// write meta
		dataBytes, err := scale.Marshal(meta)
		if err != nil {
			logger.Errorf("marshalling meta for candidate: %w", err)
		}
		err = tx.meta.Put(candidateHash[:], dataBytes)
		if err != nil {
			logger.Errorf("storing metadata for candidate %v: %w", candidate, err)
		}
		logger.Infof("stored meta for candidate %x", candidateHash[:])
		logger.Infof("data %v", dataBytes)

		// write pruning key
		pruneAt := now + BETimestamp(av.pruningConfig.keepUnavailableFor.Seconds())
		pruneKey := append(candidateHash[:], uint32ToBytes(uint32(pruneAt))...)
		err = tx.pruneByTime.Put(pruneKey, nil)
		if err != nil {
			logger.Errorf("writing pruning key: %w", err)
		}
	}
}

func (av *AvailabilityStoreSubsystem) noteBlockIncluded(tx *availabilityStoreBatch,
	blockNumber parachaintypes.BlockNumber, blockHash common.Hash,
	candidate parachaintypes.CandidateReceipt) {
	candidateHash, err := candidate.Hash()
	if err != nil {
		logger.Errorf("failed to hash candidate: %w", err)
		return
	}
	logger.Infof("noteBlockIncluded")
	meta, err := av.availabilityStore.loadMeta(parachaintypes.CandidateHash{Value: candidateHash})
	if err != nil {
		logger.Errorf("failed to load meta for candidate %x: %w", candidateHash, err)
	}

	if meta == nil {
		logger.Warnf("Candidate included without being backed %x", candidateHash)
	}
	//switch meta.State.(type) {
	//// TODO: determine how to switch on State VDT type
	//case Unavailable:
	//// deletePruningKey
	//case Unfinalized:
	//
	//case Finalized:
	//	// This should never happen as a candidate would have to be included after
	//	// finality.
	//	return
	//}
	//write unfialized block contains

	key := append([]byte(unfinalizedPrefix), uint32ToBytes(uint32(blockNumber))...)
	key = append(key, blockHash[:]...)
	key = append(key, candidateHash[:]...)

	err = tx.unfinalized.Put(key, nil)
	if err != nil {
		logger.Errorf("failed to put unfinalized key: %w", err)
	}

	metaKey := append([]byte(metaPrefix), candidateHash[:]...)
	err = tx.meta.Put(metaKey, nil)
	if err != nil {
		logger.Errorf("failed to put meta key: %w", err)
	}
}

func (av *AvailabilityStoreSubsystem) ProcessBlockFinalizedSignal() {
	logger.Infof("ProcessBlockFinalizedSignal %s", av.currentMessage)
	// TODO: determine batch number and finalized hash
	startPrefix, endPrefix := finalizedBlockRange(av.currentMessage.(parachaintypes.BlockFinalizedSignal).BlockNumber)
	logger.Infof("startPrefix %v", startPrefix)
	logger.Infof("endPrefix %v", endPrefix)
	batchNum := 0
	finalizedHash := common.Hash{}
	// load all of finalized height
	batch, err := av.loadAllAtFinalizedHeight(batchNum, finalizedHash)
	if err != nil {
		logger.Errorf("failed to load all at finalized height: %w", err)

	}
	logger.Infof("batch %s", batch)
	// delete unfinalized height

	// update blocks at finalized height
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

	res, err := av.availabilityStore.storeAvailableData(av, msg.CandidateHash, uint(msg.NumValidators),
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
		return fmt.Errorf("store available data: %w", err)
	}
	return nil
}

func (av *AvailabilityStoreSubsystem) Stop() {
	av.cancel()
	av.wg.Wait()
}

func finalizedBlockRange(finalized parachaintypes.BlockNumber) (start, end []byte) {
	start = []byte(unfinalizedPrefix)
	end = append([]byte(unfinalizedPrefix), uint32ToBytes(uint32(finalized+1))...)
	return
}

func (av *AvailabilityStoreSubsystem) loadAllAtFinalizedHeight(blockNumber int,
	finalizedHash common.Hash) (map[parachaintypes.CandidateHash]bool, error) {
	result := make(map[parachaintypes.CandidateHash]bool)
	iter, err := av.availabilityStore.meta.NewIterator()
	if err != nil {
		return nil, fmt.Errorf("creating iterator: %w", err)
	}
	defer iter.Release()

	// TODO: use finalizedHash
	logger.Infof("finalizedHash %s", finalizedHash)

	for iter.Next() {
		key := iter.Key()
		logger.Infof("key %s", key)
		if len(key) != 36 {
			return nil, fmt.Errorf("invalid key length %d", len(key))
		}
		height := binary.BigEndian.Uint64(key[:8])
		if height != uint64(blockNumber) {
			break
		}
		// TODO:
		// decode unfianlized key (to get blockHash and CandidateHash
		// if blockHash == finalizedHash then candidatesInsert true, else insert false
	}
	return result, nil
}
