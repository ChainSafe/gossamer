// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package availabilitystore

import (
	"bytes"
	"errors"
	"testing"

	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/dot/state"
	"github.com/ChainSafe/gossamer/internal/database"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/stretchr/testify/require"
)

var (
	testChunk1 = ErasureChunk{
		Chunk: []byte("chunk1"),
		Index: 0,
		Proof: []byte("proof1"),
	}
	testChunk2 = ErasureChunk{
		Chunk: []byte("chunk2"),
		Index: 1,
		Proof: []byte("proof2"),
	}
	testavailableData1 = AvailableData{
		PoV: parachaintypes.PoV{BlockData: []byte("blockdata")},
		ValidationData: parachaintypes.PersistedValidationData{
			ParentHead: parachaintypes.HeadData{Data: []byte("parentHead")},
		},
	}

	testCandidateHash = parachaintypes.CandidateHash{Value: common.Hash{0x01}}
)

func setupTestDB(t *testing.T) database.Database {
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
	err = batch.meta.Put(testCandidateHash.Value[:], dataBytes)
	require.NoError(t, err)

	err = batch.flush()
	require.NoError(t, err)

	stored, err := as.storeChunk(parachaintypes.CandidateHash{Value: common.Hash{0x01}}, testChunk1)
	require.NoError(t, err)
	require.Equal(t, true, stored)
	stored, err = as.storeChunk(parachaintypes.CandidateHash{Value: common.Hash{0x01}}, testChunk2)
	require.NoError(t, err)
	require.Equal(t, true, stored)

	batch = newAvailabilityStoreBatch(as)
	dataBytes, err = scale.Marshal(testavailableData1)
	require.NoError(t, err)
	err = batch.available.Put(testCandidateHash.Value[:], dataBytes)
	require.NoError(t, err)

	err = batch.flush()
	require.NoError(t, err)

	return inmemoryDB
}

func TestAvailabilityStore_WriteLoadDeleteAvailableData(t *testing.T) {
	inmemoryDB := state.NewInMemoryDB(t)
	as := NewAvailabilityStore(inmemoryDB)
	batch := newAvailabilityStoreBatch(as)

	dataBytes, err := scale.Marshal(testavailableData1)
	require.NoError(t, err)
	err = batch.available.Put(testCandidateHash.Value[:], dataBytes)
	require.NoError(t, err)

	err = batch.flush()
	require.NoError(t, err)

	got, err := as.loadAvailableData(parachaintypes.CandidateHash{Value: common.Hash{0x01}})
	require.NoError(t, err)
	require.Equal(t, &testavailableData1, got)

	got, err = as.loadAvailableData(parachaintypes.CandidateHash{Value: common.Hash{0x02}})
	require.EqualError(t, err, "getting candidate 0x0200000000000000000000000000000000000000000000000000000000000000"+
		" from available table: pebble: not found")
	require.Equal(t, (*AvailableData)(nil), got)

	batch = newAvailabilityStoreBatch(as)

	err = batch.available.Del(testCandidateHash.Value[:])
	require.NoError(t, err)

	err = batch.flush()
	require.NoError(t, err)

	got, err = as.loadAvailableData(parachaintypes.CandidateHash{Value: common.Hash{0x01}})
	require.EqualError(t, err, "getting candidate 0x0100000000000000000000000000000000000000000000000000000000000000"+
		" from available table: pebble: not found")
	require.Equal(t, (*AvailableData)(nil), got)
}

func TestAvailabilityStore_WriteLoadDeleteChuckData(t *testing.T) {
	inmemoryDB := state.NewInMemoryDB(t)
	as := NewAvailabilityStore(inmemoryDB)
	batch := newAvailabilityStoreBatch(as)
	metaState := NewStateVDT()
	err := metaState.Set(Unavailable{})
	require.NoError(t, err)
	meta := CandidateMeta{
		State:         metaState,
		DataAvailable: false,
		ChunksStored:  []bool{false, false},
	}
	dataBytes, err := scale.Marshal(meta)
	require.NoError(t, err)
	err = batch.meta.Put(parachaintypes.CandidateHash{Value: common.Hash{0x01}}.Value.ToBytes(), dataBytes)
	require.NoError(t, err)

	err = batch.flush()
	require.NoError(t, err)

	got, err := as.storeChunk(parachaintypes.CandidateHash{Value: common.Hash{0x01}}, testChunk1)
	require.NoError(t, err)
	require.Equal(t, true, got)
	got, err = as.storeChunk(parachaintypes.CandidateHash{Value: common.Hash{0x01}}, testChunk2)
	require.NoError(t, err)
	require.Equal(t, true, got)

	resultChunk, err := as.loadChunk(parachaintypes.CandidateHash{Value: common.Hash{0x01}}, 0)
	require.NoError(t, err)
	require.Equal(t, &testChunk1, resultChunk)

	resultChunk, err = as.loadChunk(parachaintypes.CandidateHash{Value: common.Hash{0x01}}, 1)
	require.NoError(t, err)
	require.Equal(t, &testChunk2, resultChunk)

	batch = newAvailabilityStoreBatch(as)
	err = batch.chunk.Del(append(testCandidateHash.Value[:], uint32ToBytes(0)...))
	require.NoError(t, err)

	err = batch.flush()
	require.NoError(t, err)

	resultChunk, err = as.loadChunk(parachaintypes.CandidateHash{Value: common.Hash{0x01}}, 0)
	require.EqualError(t, err, "getting candidate 0x0100000000000000000000000000000000000000000000000000000000000000,"+
		" index 0 from chunk table: pebble: not found")
	require.Equal(t, (*ErasureChunk)(nil), resultChunk)
}

func TestAvailabilityStore_WriteLoadDeleteMeta(t *testing.T) {
	inmemoryDB := state.NewInMemoryDB(t)
	as := NewAvailabilityStore(inmemoryDB)
	batch := newAvailabilityStoreBatch(as)
	metaState := NewStateVDT()
	err := metaState.Set(Unavailable{})
	require.NoError(t, err)
	meta := &CandidateMeta{
		State: metaState,
	}

	dataBytes, err := scale.Marshal(*meta)
	require.NoError(t, err)
	err = batch.meta.Put(testCandidateHash.Value[:], dataBytes)
	require.NoError(t, err)

	err = batch.flush()
	require.NoError(t, err)

	got, err := as.loadMeta(parachaintypes.CandidateHash{Value: common.Hash{0x01}})
	require.NoError(t, err)
	require.Equal(t, meta, got)

	batch = newAvailabilityStoreBatch(as)

	err = batch.meta.Del(testCandidateHash.Value[:])
	require.NoError(t, err)

	err = batch.flush()
	require.NoError(t, err)

	got, err = as.loadMeta(parachaintypes.CandidateHash{Value: common.Hash{0x01}})
	require.EqualError(t, err, "getting candidate 0x0100000000000000000000000000000000000000000000000000000000000000"+
		" from meta table: pebble: not found")
	require.Equal(t, (*CandidateMeta)(nil), got)
}

func TestAvailabilityStore_WriteLoadDeleteUnfinalizedHeight(t *testing.T) {
	inmemoryDB := state.NewInMemoryDB(t)
	as := NewAvailabilityStore(inmemoryDB)
	batch := newAvailabilityStoreBatch(as)
	blockNumber := parachaintypes.BlockNumber(1)
	hash := common.Hash{0x02}
	hash6 := common.Hash{0x06}
	candidateHash := parachaintypes.CandidateHash{Value: common.Hash{0x03}}

	key := append(uint32ToBytesBigEndian(uint32(blockNumber)), hash[:]...)
	key = append(key, candidateHash.Value[:]...)
	err := batch.unfinalized.Put(key, nil)
	require.NoError(t, err)

	key = append(uint32ToBytesBigEndian(uint32(blockNumber)), hash6[:]...)
	key = append(key, candidateHash.Value[:]...)
	err = batch.unfinalized.Put(key, nil)
	require.NoError(t, err)
	key = append(uint32ToBytesBigEndian(uint32(0)), hash[:]...)
	key = append(key, candidateHash.Value[:]...)
	err = batch.unfinalized.Put(key, nil)
	require.NoError(t, err)
	key = append(uint32ToBytesBigEndian(uint32(2)), hash[:]...)
	key = append(key, candidateHash.Value[:]...)
	err = batch.unfinalized.Put(key, nil)

	require.NoError(t, err)

	err = batch.flush()
	require.NoError(t, err)

	// check that the key is written
	key12 := append(uint32ToBytesBigEndian(uint32(blockNumber)), hash[:]...)
	key12 = append(key12, candidateHash.Value[:]...)

	got, err := as.unfinalized.Get(key12)
	require.NoError(t, err)
	require.Equal(t, []byte{}, got)

	key16 := append(uint32ToBytesBigEndian(uint32(blockNumber)), hash6[:]...)
	key16 = append(key16, candidateHash.Value[:]...)

	got, err = as.unfinalized.Get(key16)
	require.NoError(t, err)
	require.Equal(t, []byte{}, got)

	// delete height, (block 1)
	batch = newAvailabilityStoreBatch(as)
	keyPrefix := append([]byte(unfinalizedPrefix), uint32ToBytesBigEndian(uint32(blockNumber))...)
	itr, err := as.unfinalized.NewIterator()
	require.NoError(t, err)
	defer itr.Release()

	for itr.First(); itr.Valid(); itr.Next() {
		comp := bytes.Compare(itr.Key()[0:len(keyPrefix)], keyPrefix)
		if comp < 0 {
			continue
		} else if comp > 0 {
			break
		}
		err := batch.unfinalized.Del(itr.Key()[len(unfinalizedPrefix):])
		require.NoError(t, err)
	}
	err = batch.flush()
	require.NoError(t, err)

	// check that the key is deleted
	got, err = as.unfinalized.Get(key12)
	require.EqualError(t, err, "pebble: not found")
	require.Equal(t, []byte(nil), got)

	got, err = as.unfinalized.Get(key16)
	require.EqualError(t, err, "pebble: not found")
	require.Equal(t, []byte(nil), got)

	// check that the other keys are not deleted
	key = append(uint32ToBytesBigEndian(uint32(0)), hash[:]...)
	key = append(key, candidateHash.Value[:]...)
	got, err = as.unfinalized.Get(key)
	require.NoError(t, err)
	require.Equal(t, []byte{}, got)
}

func TestAvailabilityStore_WriteLoadDeleteUnfinalizedInclusion(t *testing.T) {
	inmemoryDB := state.NewInMemoryDB(t)
	as := NewAvailabilityStore(inmemoryDB)
	batch := newAvailabilityStoreBatch(as)
	blockNumber := parachaintypes.BlockNumber(1)
	hash := common.Hash{0x02}
	hash6 := common.Hash{0x06}
	candidateHash := parachaintypes.CandidateHash{Value: common.Hash{0x03}}
	key := append(uint32ToBytesBigEndian(uint32(blockNumber)), hash[:]...)
	key = append(key, candidateHash.Value[:]...)
	err := batch.unfinalized.Put(key, nil)
	require.NoError(t, err)
	key = append(uint32ToBytesBigEndian(uint32(blockNumber)), hash6[:]...)
	key = append(key, candidateHash.Value[:]...)
	err = batch.unfinalized.Put(key, nil)
	require.NoError(t, err)
	key = append(uint32ToBytesBigEndian(uint32(0)), hash[:]...)
	key = append(key, candidateHash.Value[:]...)
	err = batch.unfinalized.Put(key, nil)
	require.NoError(t, err)
	key = append(uint32ToBytesBigEndian(uint32(2)), hash[:]...)
	key = append(key, candidateHash.Value[:]...)
	err = batch.unfinalized.Put(key, nil)
	require.NoError(t, err)

	err = batch.flush()
	require.NoError(t, err)

	// check that the key is written
	key12 := append(uint32ToBytesBigEndian(uint32(blockNumber)), hash[:]...)
	key12 = append(key12, candidateHash.Value[:]...)

	got, err := as.unfinalized.Get(key12)
	require.NoError(t, err)
	require.Equal(t, []byte{}, got)

	key16 := append(uint32ToBytesBigEndian(uint32(blockNumber)), hash6[:]...)
	key16 = append(key16, candidateHash.Value[:]...)

	got, err = as.unfinalized.Get(key16)
	require.NoError(t, err)
	require.Equal(t, []byte{}, got)

	// delete inclusion, (block 1, hash 2)
	batch = newAvailabilityStoreBatch(as)
	key = append(uint32ToBytesBigEndian(uint32(blockNumber)), hash[:]...)
	key = append(key, candidateHash.Value[:]...)
	err = batch.unfinalized.Del(key)
	require.NoError(t, err)

	err = batch.flush()
	require.NoError(t, err)

	// check that the key is deleted
	got, err = as.unfinalized.Get(key12)
	require.EqualError(t, err, "pebble: not found")
	require.Equal(t, []byte(nil), got)

	// check that the other keys are not deleted
	got, err = as.unfinalized.Get(key16)
	require.NoError(t, err)
	require.Equal(t, []byte{}, got)

	key = append(uint32ToBytesBigEndian(uint32(0)), hash[:]...)
	key = append(key, candidateHash.Value[:]...)
	got, err = as.unfinalized.Get(key)
	require.NoError(t, err)
	require.Equal(t, []byte{}, got)
}

func TestAvailabilityStore_WriteDeletePruningKey(t *testing.T) {
	inmemoryDB := state.NewInMemoryDB(t)
	as := NewAvailabilityStore(inmemoryDB)
	batch := newAvailabilityStoreBatch(as)
	candidateHash := parachaintypes.CandidateHash{Value: common.Hash{0x03}}

	pruneKey := append(BETimestamp(1).ToBigEndianBytes(), candidateHash.Value[:]...)
	err := batch.pruneByTime.Put(pruneKey, nil)
	require.NoError(t, err)

	pruneKey = append(BETimestamp(2).ToBigEndianBytes(), candidateHash.Value[:]...)
	err = batch.pruneByTime.Put(pruneKey, nil)
	require.NoError(t, err)

	err = batch.flush()
	require.NoError(t, err)

	// check that the key is written
	key1 := append(BETimestamp(1).ToBigEndianBytes(), candidateHash.Value[:]...)

	got, err := as.pruneByTime.Get(key1)
	require.NoError(t, err)
	require.Equal(t, []byte{}, got)

	key2 := append(BETimestamp(2).ToBigEndianBytes(), candidateHash.Value[:]...)
	got, err = as.pruneByTime.Get(key2)
	require.NoError(t, err)
	require.Equal(t, []byte{}, got)

	// delete pruning key, timestamp 1
	batch = newAvailabilityStoreBatch(as)
	pruneKey = append(BETimestamp(1).ToBigEndianBytes(), candidateHash.Value[:]...)
	err = batch.pruneByTime.Del(pruneKey)
	require.NoError(t, err)

	err = batch.flush()
	require.NoError(t, err)

	// check that the key is deleted
	got, err = as.pruneByTime.Get(key1)
	require.EqualError(t, err, "pebble: not found")
	require.Equal(t, []byte(nil), got)

	// check that the other keys are not deleted
	got, err = as.pruneByTime.Get(key2)
	require.NoError(t, err)
	require.Equal(t, []byte{}, got)
}

func TestAvailabilityStoreSubsystem_handleQueryAvailableData(t *testing.T) {
	t.Parallel()
	inmemoryDB := setupTestDB(t)
	availabilityStore := NewAvailabilityStore(inmemoryDB)
	availabilityStoreSubsystem := AvailabilityStoreSubsystem{
		availabilityStore: *availabilityStore,
	}

	tests := map[string]struct {
		msg            QueryAvailableData
		msgSenderChan  chan AvailableData
		expectedResult AvailableData
		err            error
	}{
		"available_data_found": {
			msg: QueryAvailableData{
				CandidateHash: parachaintypes.CandidateHash{Value: common.Hash{0x01}},
			},
			msgSenderChan:  make(chan AvailableData),
			expectedResult: testavailableData1,
			err:            nil,
		},
		"available_data_not_found": {
			msg: QueryAvailableData{
				CandidateHash: parachaintypes.CandidateHash{Value: common.Hash{0x07}},
			},
			msgSenderChan:  make(chan AvailableData),
			expectedResult: AvailableData{},
			err: errors.New("load available data: getting candidate" +
				" 0x0700000000000000000000000000000000000000000000000000000000000000 from available table: pebble" +
				": not found"),
		},
	}
	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			tt.msg.Sender = tt.msgSenderChan

			go func() {
				err := availabilityStoreSubsystem.handleQueryAvailableData(tt.msg)
				if tt.err == nil {
					require.NoError(t, err)
				} else {
					require.EqualError(t, err, tt.err.Error())
				}
			}()

			msgSenderChanResult := <-tt.msgSenderChan
			require.Equal(t, tt.expectedResult, msgSenderChanResult)
		})
	}
}

func TestAvailabilityStoreSubsystem_handleQueryDataAvailability(t *testing.T) {
	t.Parallel()
	inmemoryDB := setupTestDB(t)
	availabilityStore := NewAvailabilityStore(inmemoryDB)
	availabilityStoreSubsystem := AvailabilityStoreSubsystem{
		availabilityStore: *availabilityStore,
	}

	tests := map[string]struct {
		msg            QueryDataAvailability
		msgSenderChan  chan bool
		expectedResult bool
		wantErr        bool
	}{
		"data_available_true": {
			msg: QueryDataAvailability{
				CandidateHash: parachaintypes.CandidateHash{Value: common.Hash{0x01}},
			},
			msgSenderChan:  make(chan bool),
			expectedResult: true,
			wantErr:        false,
		},
		"data_available_false": {
			msg: QueryDataAvailability{
				CandidateHash: parachaintypes.CandidateHash{Value: common.Hash{0x07}},
			},
			msgSenderChan:  make(chan bool),
			expectedResult: false,
			wantErr:        false,
		},
	}
	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			tt.msg.Sender = tt.msgSenderChan

			go func() {
				if err := availabilityStoreSubsystem.handleQueryDataAvailability(tt.msg); (err != nil) != tt.wantErr {
					t.Errorf("handleQueryDataAvailability() error = %v, wantErr %v", err, tt.wantErr)
				}
			}()

			msgSenderChanResult := <-tt.msgSenderChan
			require.Equal(t, tt.expectedResult, msgSenderChanResult)
		})
	}
}

func TestAvailabilityStoreSubsystem_handleQueryChunk(t *testing.T) {
	t.Parallel()
	inmemoryDB := setupTestDB(t)
	availabilityStore := NewAvailabilityStore(inmemoryDB)
	availabilityStoreSubsystem := AvailabilityStoreSubsystem{
		availabilityStore: *availabilityStore,
	}

	tests := map[string]struct {
		msg            QueryChunk
		msgSenderChan  chan ErasureChunk
		expectedResult ErasureChunk
		err            error
	}{
		"chunk_found": {
			msg: QueryChunk{
				CandidateHash: parachaintypes.CandidateHash{Value: common.Hash{0x01}},
			},
			msgSenderChan:  make(chan ErasureChunk),
			expectedResult: testChunk1,
			err:            nil,
		},
		"query_chunk_not_found": {
			msg: QueryChunk{
				CandidateHash: parachaintypes.CandidateHash{Value: common.Hash{0x07}},
			},
			msgSenderChan:  make(chan ErasureChunk),
			expectedResult: ErasureChunk{},
			err: errors.New("load chunk: getting candidate " +
				"0x0700000000000000000000000000000000000000000000000000000000000000, " +
				"index 0 from chunk table: pebble: not found"),
		},
	}
	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			tt.msg.Sender = tt.msgSenderChan

			go func() {
				err := availabilityStoreSubsystem.handleQueryChunk(tt.msg)
				if tt.err == nil {
					require.NoError(t, err)
				} else {
					require.EqualError(t, err, tt.err.Error())
				}
			}()

			msgSenderChanResult := <-tt.msgSenderChan
			require.Equal(t, tt.expectedResult, msgSenderChanResult)
		})
	}
}

func TestAvailabilityStoreSubsystem_handleQueryAllChunks(t *testing.T) {
	t.Parallel()
	inmemoryDB := setupTestDB(t)
	availabilityStore := NewAvailabilityStore(inmemoryDB)
	availabilityStoreSubsystem := AvailabilityStoreSubsystem{
		availabilityStore: *availabilityStore,
	}

	tests := map[string]struct {
		msg            QueryAllChunks
		msgSenderChan  chan []ErasureChunk
		expectedResult []ErasureChunk
		err            error
	}{
		"chunks_found": {
			msg: QueryAllChunks{
				CandidateHash: parachaintypes.CandidateHash{Value: common.Hash{0x01}},
			},
			msgSenderChan:  make(chan []ErasureChunk),
			expectedResult: []ErasureChunk{testChunk1, testChunk2},
			err:            nil,
		},
		"query_chunks_not_found": {
			msg: QueryAllChunks{
				CandidateHash: parachaintypes.CandidateHash{Value: common.Hash{0x07}},
			},
			msgSenderChan:  make(chan []ErasureChunk),
			expectedResult: []ErasureChunk{},
			err: errors.New(
				"load metadata: getting candidate 0x0700000000000000000000000000000000000000000000000000000000000000" +
					" from meta table: pebble: not found"),
		},
	}
	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			tt.msg.Sender = tt.msgSenderChan

			go func() {
				err := availabilityStoreSubsystem.handleQueryAllChunks(tt.msg)
				if tt.err == nil {
					require.NoError(t, err)
				} else {
					require.EqualError(t, err, tt.err.Error())
				}
			}()

			msgSenderChanResult := <-tt.msgSenderChan
			require.Equal(t, tt.expectedResult, msgSenderChanResult)
		})
	}
}

func TestAvailabilityStoreSubsystem_handleQueryChunkAvailability(t *testing.T) {
	t.Parallel()
	inmemoryDB := setupTestDB(t)
	availabilityStore := NewAvailabilityStore(inmemoryDB)
	availabilityStoreSubsystem := AvailabilityStoreSubsystem{
		availabilityStore: *availabilityStore,
	}

	tests := map[string]struct {
		msg            QueryChunkAvailability
		msgSenderChan  chan bool
		expectedResult bool
		err            error
	}{
		"query_chuck_availability_true": {
			msg: QueryChunkAvailability{
				CandidateHash:  parachaintypes.CandidateHash{Value: common.Hash{0x01}},
				ValidatorIndex: 0,
			},
			msgSenderChan:  make(chan bool),
			expectedResult: true,
		},
		"query_chuck_availability_false": {
			msg: QueryChunkAvailability{
				CandidateHash:  parachaintypes.CandidateHash{Value: common.Hash{0x01}},
				ValidatorIndex: 2,
			},
			msgSenderChan:  make(chan bool),
			expectedResult: false,
		},
		"query_chuck_availability_candidate_not_found_false": {
			msg: QueryChunkAvailability{
				CandidateHash:  parachaintypes.CandidateHash{Value: common.Hash{0x07}},
				ValidatorIndex: 0,
			},
			msgSenderChan:  make(chan bool),
			expectedResult: false,
			err: errors.New(
				"load metadata: getting candidate 0x0700000000000000000000000000000000000000000000000000000000000000" +
					" from meta table: pebble: not found"),
		},
	}
	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			tt.msg.Sender = tt.msgSenderChan

			go func() {
				err := availabilityStoreSubsystem.handleQueryChunkAvailability(tt.msg)
				if tt.err == nil {
					require.NoError(t, err)
				} else {
					require.EqualError(t, err, tt.err.Error())
				}
			}()

			msgSenderChanResult := <-tt.msgSenderChan
			require.Equal(t, tt.expectedResult, msgSenderChanResult)
		})
	}
}

func TestAvailabilityStore_handleStoreChunk(t *testing.T) {
	inmemoryDB := setupTestDB(t)
	as := NewAvailabilityStore(inmemoryDB)
	asSub := AvailabilityStoreSubsystem{
		availabilityStore: *as,
	}
	msgSenderChan := make(chan any)
	msg := StoreChunk{
		CandidateHash: parachaintypes.CandidateHash{Value: common.Hash{0x01}},
		Chunk:         testChunk1,
		Sender:        msgSenderChan,
	}

	go asSub.handleStoreChunk(msg)
	msgSenderChanResult := <-msg.Sender
	require.Equal(t, nil, msgSenderChanResult)
}

func TestAvailabilityStore_handleStoreAvailableData(t *testing.T) {
	inmemoryDB := setupTestDB(t)
	as := NewAvailabilityStore(inmemoryDB)
	asSub := AvailabilityStoreSubsystem{
		availabilityStore: *as,
	}
	msgSenderChan := make(chan error)
	msg := StoreAvailableData{
		CandidateHash:       parachaintypes.CandidateHash{Value: common.Hash{0x01}},
		NumValidators:       2,
		AvailableData:       AvailableData{},
		ExpectedErasureRoot: common.MustHexToHash("0xc3d486f444a752cbf49857ceb2fce0a235268fb8b63e9e019eab619d192650bc"),
		Sender:              msgSenderChan,
	}

	go func() {
		err := asSub.handleStoreAvailableData(msg)
		require.NoError(t, err)
	}()
	msgSenderChanResult := <-msg.Sender
	require.Equal(t, nil, msgSenderChanResult)
}

func TestAvailabilityStore_storeAvailableData(t *testing.T) {
	t.Parallel()
	type args struct {
		candidate           parachaintypes.CandidateHash
		nValidators         uint
		data                AvailableData
		expectedErasureRoot common.Hash
	}
	tests := map[string]struct {
		args args
		want bool
		err  error
	}{
		"empty_availableData": {
			args: args{
				candidate:           parachaintypes.CandidateHash{},
				nValidators:         0,
				data:                AvailableData{},
				expectedErasureRoot: common.Hash{},
			},
			want: false,
			err:  errors.New("obtaining chunks: expected at least 2 validators"),
		},
		"2_validators": {
			args: args{
				candidate:   parachaintypes.CandidateHash{},
				nValidators: 2,
				data: AvailableData{
					PoV: parachaintypes.PoV{BlockData: []byte{2}},
				},
				expectedErasureRoot: common.MustHexToHash("0x513489282098e960bfd57ed52d62838ce9395f3f59257f1f40fadd02261a7991"),
			},
			want: true,
			err:  nil,
		},
		"2_validators_error_erasure_root": {
			args: args{
				candidate:   parachaintypes.CandidateHash{},
				nValidators: 2,
				data: AvailableData{
					PoV: parachaintypes.PoV{BlockData: []byte{2}},
				},
				expectedErasureRoot: common.Hash{},
			},
			want: false,
			err:  errInvalidErasureRoot,
		},
	}
	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			inmemoryDB := setupTestDB(t)
			as := NewAvailabilityStore(inmemoryDB)
			asSub := &AvailabilityStoreSubsystem{
				availabilityStore: *as,
			}
			got, err := as.storeAvailableData(asSub, tt.args.candidate, tt.args.nValidators,
				tt.args.data, tt.args.expectedErasureRoot)
			if tt.err == nil {
				require.NoError(t, err)
			} else {
				require.EqualError(t, err, tt.err.Error())
			}
			require.Equal(t, tt.want, got)
		})
	}
}
