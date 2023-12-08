// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package availabilitystore

import (
	"errors"
	"testing"

	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/dot/state"
	"github.com/ChainSafe/gossamer/internal/database"
	"github.com/ChainSafe/gossamer/lib/common"
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
)

func setupTestDB(t *testing.T) database.Database {
	inmemoryDB := state.NewInMemoryDB(t)
	as := NewAvailabilityStore(inmemoryDB)
	batch := NewAvailabilityStoreBatch(as)

	_, err := as.storeChunk(parachaintypes.CandidateHash{Value: common.Hash{0x01}}, testChunk1)
	require.NoError(t, err)
	_, err = as.storeChunk(parachaintypes.CandidateHash{Value: common.Hash{0x01}}, testChunk2)
	require.NoError(t, err)

	err = as.writeAvailableData(batch, parachaintypes.CandidateHash{Value: common.Hash{0x01}}, testavailableData1)
	require.NoError(t, err)
	err = batch.Write()
	require.NoError(t, err)

	return inmemoryDB
}

func TestAvailabilityStore_WriteLoadDeleteAvailableData(t *testing.T) {
	inmemoryDB := state.NewInMemoryDB(t)
	as := NewAvailabilityStore(inmemoryDB)
	batch := NewAvailabilityStoreBatch(as)

	err := as.writeAvailableData(batch, parachaintypes.CandidateHash{Value: common.Hash{0x01}}, testavailableData1)
	require.NoError(t, err)
	err = batch.Write()
	require.NoError(t, err)

	got, err := as.loadAvailableData(parachaintypes.CandidateHash{Value: common.Hash{0x01}})
	require.NoError(t, err)
	require.Equal(t, &testavailableData1, got)

	got, err = as.loadAvailableData(parachaintypes.CandidateHash{Value: common.Hash{0x02}})
	require.EqualError(t, err, "getting candidate 0x0200000000000000000000000000000000000000000000000000000000000000"+
		" from available table: pebble: not found")
	require.Equal(t, (*AvailableData)(nil), got)

	batch = NewAvailabilityStoreBatch(as)
	err = as.deleteAvailableData(batch, parachaintypes.CandidateHash{Value: common.Hash{0x01}})
	require.NoError(t, err)
	err = batch.Write()
	require.NoError(t, err)

	got, err = as.loadAvailableData(parachaintypes.CandidateHash{Value: common.Hash{0x01}})
	require.EqualError(t, err, "getting candidate 0x0100000000000000000000000000000000000000000000000000000000000000"+
		" from available table: pebble: not found")
	require.Equal(t, (*AvailableData)(nil), got)
}

func TestAvailabilityStore_WriteLoadDeleteChuckData(t *testing.T) {
	inmemoryDB := state.NewInMemoryDB(t)
	as := NewAvailabilityStore(inmemoryDB)
	batch := NewAvailabilityStoreBatch(as)
	meta := CandidateMeta{
		State:         State{},
		DataAvailable: false,
		ChunksStored:  []bool{false, false},
	}
	err := as.writeMeta(batch, parachaintypes.CandidateHash{Value: common.Hash{0x01}}, meta)
	require.NoError(t, err)
	err = batch.Write()
	require.NoError(t, err)

	_, err = as.storeChunk(parachaintypes.CandidateHash{Value: common.Hash{0x01}}, testChunk1)
	require.NoError(t, err)
	_, err = as.storeChunk(parachaintypes.CandidateHash{Value: common.Hash{0x01}}, testChunk2)
	require.NoError(t, err)

	resultChunk, err := as.loadChunk(parachaintypes.CandidateHash{Value: common.Hash{0x01}}, 0)
	require.NoError(t, err)
	require.Equal(t, &testChunk1, resultChunk)

	resultChunk, err = as.loadChunk(parachaintypes.CandidateHash{Value: common.Hash{0x01}}, 1)
	require.NoError(t, err)
	require.Equal(t, &testChunk2, resultChunk)

	batch = NewAvailabilityStoreBatch(as)
	err = as.deleteChunk(batch, parachaintypes.CandidateHash{Value: common.Hash{0x01}}, 0)
	require.NoError(t, err)
	err = batch.Write()
	require.NoError(t, err)

	resultChunk, err = as.loadChunk(parachaintypes.CandidateHash{Value: common.Hash{0x01}}, 0)
	require.EqualError(t, err, "getting candidate 0x0100000000000000000000000000000000000000000000000000000000000000,"+
		" index 0 from chunk table: pebble: not found")
	require.Equal(t, (*ErasureChunk)(nil), resultChunk)
}

func TestAvailabilityStore_WriteLoadDeleteMeta(t *testing.T) {
	inmemoryDB := state.NewInMemoryDB(t)
	as := NewAvailabilityStore(inmemoryDB)
	batch := NewAvailabilityStoreBatch(as)
	meta := &CandidateMeta{}
	err := as.writeMeta(batch, parachaintypes.CandidateHash{Value: common.Hash{0x01}}, *meta)
	require.NoError(t, err)
	err = batch.Write()
	require.NoError(t, err)

	got, err := as.loadMeta(parachaintypes.CandidateHash{Value: common.Hash{0x01}})
	require.NoError(t, err)
	require.Equal(t, meta, got)

	batch = NewAvailabilityStoreBatch(as)
	err = as.deleteMeta(batch, parachaintypes.CandidateHash{Value: common.Hash{0x01}})
	require.NoError(t, err)
	err = batch.Write()
	require.NoError(t, err)

	got, err = as.loadMeta(parachaintypes.CandidateHash{Value: common.Hash{0x01}})
	require.EqualError(t, err, "getting candidate 0x0100000000000000000000000000000000000000000000000000000000000000"+
		" from meta table: pebble: not found")
	require.Equal(t, (*CandidateMeta)(nil), got)
}

func TestAvailabilityStore_WriteLoadDeleteUnfinalizedHeight(t *testing.T) {
	inmemoryDB := state.NewInMemoryDB(t)
	as := NewAvailabilityStore(inmemoryDB)
	batch := NewAvailabilityStoreBatch(as)
	blockNumber := parachaintypes.BlockNumber(1)
	hash := common.Hash{0x02}
	hash6 := common.Hash{0x06}
	candidateHash := parachaintypes.CandidateHash{Value: common.Hash{0x03}}
	err := as.writeUnfinalizedBlockContains(batch, blockNumber, hash, candidateHash)
	require.NoError(t, err)
	err = as.writeUnfinalizedBlockContains(batch, blockNumber, hash6, candidateHash)
	require.NoError(t, err)
	err = as.writeUnfinalizedBlockContains(batch, parachaintypes.BlockNumber(0), hash, candidateHash)
	require.NoError(t, err)
	err = as.writeUnfinalizedBlockContains(batch, parachaintypes.BlockNumber(2), hash, candidateHash)
	require.NoError(t, err)

	err = batch.Write()
	require.NoError(t, err)

	// check that the key is written
	key12 := append(uint32ToBytesBigEndian(uint32(blockNumber)), hash[:]...)
	key12 = append(key12, candidateHash.Value[:]...)

	got, err := as.unfinalizedTable.Get(key12)
	require.NoError(t, err)
	require.Equal(t, []byte(tombstoneValue), got)

	key16 := append(uint32ToBytesBigEndian(uint32(blockNumber)), hash6[:]...)
	key16 = append(key16, candidateHash.Value[:]...)

	got, err = as.unfinalizedTable.Get(key16)
	require.NoError(t, err)
	require.Equal(t, []byte(tombstoneValue), got)

	// delete height, (block 1)
	batch = NewAvailabilityStoreBatch(as)
	err = as.deleteUnfinalizedHeight(batch, blockNumber)
	require.NoError(t, err)
	err = batch.Write()
	require.NoError(t, err)

	// check that the key is deleted
	got, err = as.unfinalizedTable.Get(key12)
	require.EqualError(t, err, "pebble: not found")
	require.Equal(t, []byte(nil), got)

	got, err = as.unfinalizedTable.Get(key16)
	require.EqualError(t, err, "pebble: not found")
	require.Equal(t, []byte(nil), got)

	// check that the other keys are not deleted
	key := append(uint32ToBytesBigEndian(uint32(0)), hash[:]...)
	key = append(key, candidateHash.Value[:]...)
	got, err = as.unfinalizedTable.Get(key)
	require.NoError(t, err)
	require.Equal(t, []byte(tombstoneValue), got)
}

func TestAvailabilityStore_WriteLoadDeleteUnfinalizedInclusion(t *testing.T) {
	inmemoryDB := state.NewInMemoryDB(t)
	as := NewAvailabilityStore(inmemoryDB)
	batch := NewAvailabilityStoreBatch(as)
	blockNumber := parachaintypes.BlockNumber(1)
	hash := common.Hash{0x02}
	hash6 := common.Hash{0x06}
	candidateHash := parachaintypes.CandidateHash{Value: common.Hash{0x03}}
	err := as.writeUnfinalizedBlockContains(batch, blockNumber, hash, candidateHash)
	require.NoError(t, err)
	err = as.writeUnfinalizedBlockContains(batch, blockNumber, hash6, candidateHash)
	require.NoError(t, err)
	err = as.writeUnfinalizedBlockContains(batch, parachaintypes.BlockNumber(0), hash, candidateHash)
	require.NoError(t, err)
	err = as.writeUnfinalizedBlockContains(batch, parachaintypes.BlockNumber(2), hash, candidateHash)
	require.NoError(t, err)

	err = batch.Write()
	require.NoError(t, err)

	// check that the key is written
	key12 := append(uint32ToBytesBigEndian(uint32(blockNumber)), hash[:]...)
	key12 = append(key12, candidateHash.Value[:]...)

	got, err := as.unfinalizedTable.Get(key12)
	require.NoError(t, err)
	require.Equal(t, []byte(tombstoneValue), got)

	key16 := append(uint32ToBytesBigEndian(uint32(blockNumber)), hash6[:]...)
	key16 = append(key16, candidateHash.Value[:]...)

	got, err = as.unfinalizedTable.Get(key16)
	require.NoError(t, err)
	require.Equal(t, []byte(tombstoneValue), got)

	// delete inclusion, (block 1, hash 2)
	batch = NewAvailabilityStoreBatch(as)
	err = as.deleteUnfinalizedInclusion(batch, blockNumber, hash, candidateHash)
	require.NoError(t, err)
	err = batch.Write()
	require.NoError(t, err)

	// check that the key is deleted
	got, err = as.unfinalizedTable.Get(key12)
	require.EqualError(t, err, "pebble: not found")
	require.Equal(t, []byte(nil), got)

	// check that the other keys are not deleted
	got, err = as.unfinalizedTable.Get(key16)
	require.NoError(t, err)
	require.Equal(t, []byte(tombstoneValue), got)

	key := append(uint32ToBytesBigEndian(uint32(0)), hash[:]...)
	key = append(key, candidateHash.Value[:]...)
	got, err = as.unfinalizedTable.Get(key)
	require.NoError(t, err)
	require.Equal(t, []byte(tombstoneValue), got)
}

func TestAvailabilityStore_WriteDeletePruningKey(t *testing.T) {
	inmemoryDB := state.NewInMemoryDB(t)
	as := NewAvailabilityStore(inmemoryDB)
	batch := NewAvailabilityStoreBatch(as)
	candidateHash := parachaintypes.CandidateHash{Value: common.Hash{0x03}}

	err := as.writePruningKey(batch, BETimestamp(1), candidateHash)
	require.NoError(t, err)
	err = as.writePruningKey(batch, BETimestamp(2), candidateHash)
	require.NoError(t, err)

	err = batch.Write()
	require.NoError(t, err)

	// check that the key is written
	key1 := append(BETimestamp(1).ToBytes(), candidateHash.Value[:]...)

	got, err := as.pruneByTimeTable.Get(key1)
	require.NoError(t, err)
	require.Equal(t, []byte(tombstoneValue), got)

	key2 := append(BETimestamp(2).ToBytes(), candidateHash.Value[:]...)
	got, err = as.pruneByTimeTable.Get(key2)
	require.NoError(t, err)
	require.Equal(t, []byte(tombstoneValue), got)

	// delete pruning key, timestamp 1
	batch = NewAvailabilityStoreBatch(as)
	err = as.deletePruningKey(batch, BETimestamp(1), candidateHash)
	require.NoError(t, err)
	err = batch.Write()
	require.NoError(t, err)

	// check that the key is deleted
	got, err = as.pruneByTimeTable.Get(key1)
	require.EqualError(t, err, "pebble: not found")
	require.Equal(t, []byte(nil), got)

	// check that the other keys are not deleted
	got, err = as.pruneByTimeTable.Get(key2)
	require.NoError(t, err)
	require.Equal(t, []byte(tombstoneValue), got)
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
					" from available table: pebble: not found"),
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
					" from available table: pebble: not found"),
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
	msgSenderChan := make(chan any)
	msg := StoreAvailableData{
		CandidateHash:       common.Hash{0x01},
		NValidators:         0,
		AvailableData:       AvailableData{},
		ExpectedErasureRoot: common.Hash{},
		Sender:              msgSenderChan,
	}

	go asSub.handleStoreAvailableData(msg)
	msgSenderChanResult := <-msg.Sender
	require.Equal(t, nil, msgSenderChanResult)
}
