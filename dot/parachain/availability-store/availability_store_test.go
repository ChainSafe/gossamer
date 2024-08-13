// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package availabilitystore

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/dot/parachain/chainapi"
	parachain "github.com/ChainSafe/gossamer/dot/parachain/runtime"
	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/dot/parachain/util"
	"github.com/ChainSafe/gossamer/dot/state"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/internal/database"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/erasure"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/ChainSafe/gossamer/pkg/trie"
	"github.com/ChainSafe/gossamer/pkg/trie/inmemory"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
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
	testAvailableData1 = AvailableData{
		PoV: parachaintypes.PoV{BlockData: []byte("blockdata")},
		ValidationData: parachaintypes.PersistedValidationData{
			ParentHead: parachaintypes.HeadData{Data: []byte("parentHead")},
		},
	}

	testCandidateReceipt = parachaintypes.CandidateReceipt{
		Descriptor: parachaintypes.CandidateDescriptor{
			ParaID:      0xd05,
			RelayParent: common.MustHexToHash("0x2245952bd39fbc912d3bcf6ab6d9c191b5c9df70a4e1da6f670f4761bf96cc9f"),
			Collator: parachaintypes.CollatorID{0x54, 0xde, 0x49, 0x5b, 0x57, 0xc7, 0xc3, 0x50,
				0x5c, 0x62, 0x63, 0x3f, 0x1a, 0xc3, 0xa2, 0xf2, 0x2b, 0xe7, 0x4e, 0xd4, 0x97, 0xa5, 0x88, 0x43, 0x79,
				0xe0, 0x82, 0x16, 0x7, 0xd9, 0x17, 0x3b},
			PersistedValidationDataHash: common.MustHexToHash(
				"0xeb5dd269e10d71dc754ce3fac591364afce61b007925730544bf530068087c7d"),
			PovHash: common.MustHexToHash(
				"0x38e3e2cd8bdf7fef72ad23b076e0620ab5d97d3c0a98207dd8b6af8c3becff69"),
			ErasureRoot: common.MustHexToHash(
				"0x6cd2c59aefd1a3bd654df00febe502e9f6ba788b8584c7278c47a2983fd8e87b"),
			ParaHead: common.MustHexToHash(
				"0xfedc8cf6b2555199ecc95e7092742c5959f96bce04becaebfd9266f7642c23d7"),
			ValidationCodeHash: parachaintypes.ValidationCodeHash{110, 37, 119, 28, 37, 5, 245, 73, 181,
				175, 119, 52, 200, 66, 19, 189, 31, 211, 146, 120, 250, 143, 7, 41, 139, 166, 157, 165, 90, 92, 112, 137},
		},
		CommitmentsHash: common.MustHexToHash("0x9ece96d300d33d733840cfd4035249b50618e4e81f1cd425bd304b0cffc13b8e"),
	}

	testCandidateReceiptHash, _ = testCandidateReceipt.Hash()
)

func setupTestDB(t *testing.T) database.Database {
	inmemoryDB, err := database.NewPebble("", true)
	require.NoError(t, err)
	as := NewAvailabilityStore(inmemoryDB)
	batch := newAvailabilityStoreBatch(as)
	metaState := newStateVDT()
	err = metaState.SetValue(Unavailable{})
	require.NoError(t, err)
	meta := CandidateMeta{
		State:         metaState,
		DataAvailable: false,
		ChunksStored:  []bool{false, false, false},
	}

	dataBytes, err := scale.Marshal(meta)
	require.NoError(t, err)

	err = batch.meta.Put(testCandidateReceiptHash[:], dataBytes)
	require.NoError(t, err)
	err = batch.meta.Put(common.Hash{0x02}.ToBytes(), dataBytes)
	require.NoError(t, err)

	err = batch.flush()
	require.NoError(t, err)

	stored, err := as.storeChunk(parachaintypes.CandidateHash{Value: testCandidateReceiptHash}, testChunk1)
	require.NoError(t, err)
	require.Equal(t, true, stored)
	stored, err = as.storeChunk(parachaintypes.CandidateHash{Value: testCandidateReceiptHash}, testChunk2)
	require.NoError(t, err)
	require.Equal(t, true, stored)

	batch = newAvailabilityStoreBatch(as)
	dataBytes, err = scale.Marshal(testAvailableData1)
	require.NoError(t, err)
	err = batch.available.Put(testCandidateReceiptHash[:], dataBytes)
	require.NoError(t, err)

	err = batch.flush()
	require.NoError(t, err)

	return inmemoryDB
}

func TestAvailabilityStore_WriteLoadDeleteAvailableData(t *testing.T) {
	inmemoryDB := state.NewInMemoryDB(t)
	as := NewAvailabilityStore(inmemoryDB)
	batch := newAvailabilityStoreBatch(as)

	dataBytes, err := scale.Marshal(testAvailableData1)
	require.NoError(t, err)
	err = batch.available.Put(testCandidateReceiptHash.ToBytes(), dataBytes)
	require.NoError(t, err)

	err = batch.flush()
	require.NoError(t, err)

	got, err := as.loadAvailableData(parachaintypes.CandidateHash{Value: testCandidateReceiptHash})
	require.NoError(t, err)
	require.Equal(t, &testAvailableData1, got)

	got, err = as.loadAvailableData(parachaintypes.CandidateHash{Value: common.Hash{0x02}})
	require.EqualError(t, err, "getting candidate 0x0200000000000000000000000000000000000000000000000000000000000000"+
		" from available table: pebble: not found")
	require.Equal(t, (*AvailableData)(nil), got)

	batch = newAvailabilityStoreBatch(as)

	err = batch.available.Del(testCandidateReceiptHash.ToBytes())
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
	metaState := newStateVDT()
	err := metaState.SetValue(Unavailable{})
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
	err = batch.chunk.Del(append(testCandidateReceiptHash.ToBytes(), uint32ToBytes(0)...))
	require.NoError(t, err)

	err = batch.flush()
	require.NoError(t, err)

	resultChunk, err = as.loadChunk(parachaintypes.CandidateHash{Value: testCandidateReceiptHash}, 0)
	require.EqualError(t, err, "getting candidate 0xbe7d49d790273a96e6c0c3c16ed1ed6895ff57a57b573c7eb081e9aeda7835f5,"+
		" index 0 from chunk table: pebble: not found")
	require.Equal(t, (*ErasureChunk)(nil), resultChunk)
}

func TestAvailabilityStore_WriteLoadDeleteMeta(t *testing.T) {
	inmemoryDB := state.NewInMemoryDB(t)
	as := NewAvailabilityStore(inmemoryDB)
	batch := newAvailabilityStoreBatch(as)
	metaState := newStateVDT()
	err := metaState.SetValue(Unavailable{BETimestamp(1711026139)})
	require.NoError(t, err)
	meta := &CandidateMeta{
		State:         metaState,
		DataAvailable: false,
		ChunksStored:  make([]bool, 10),
	}

	dataBytes, err := scale.Marshal(*meta)
	require.NoError(t, err)
	err = batch.meta.Put(testCandidateReceiptHash.ToBytes(), dataBytes)
	require.NoError(t, err)

	err = batch.flush()
	require.NoError(t, err)

	got, err := as.loadMeta(parachaintypes.CandidateHash{Value: testCandidateReceiptHash})
	require.NoError(t, err)
	require.Equal(t, meta, got)

	batch = newAvailabilityStoreBatch(as)

	err = batch.meta.Del(testCandidateReceiptHash.ToBytes())
	require.NoError(t, err)

	err = batch.flush()
	require.NoError(t, err)

	got, err = as.loadMeta(parachaintypes.CandidateHash{Value: testCandidateReceiptHash})
	require.EqualError(t, err, "getting candidate 0xbe7d49d790273a96e6c0c3c16ed1ed6895ff57a57b573c7eb081e9aeda7835f5"+
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

	key := append(uint32ToBytes(uint32(blockNumber)), hash[:]...)
	key = append(key, candidateHash.Value[:]...)
	err := batch.unfinalized.Put(key, nil)
	require.NoError(t, err)

	key = append(uint32ToBytes(uint32(blockNumber)), hash6[:]...)
	key = append(key, candidateHash.Value[:]...)
	err = batch.unfinalized.Put(key, nil)
	require.NoError(t, err)
	key = append(uint32ToBytes(uint32(0)), hash[:]...)
	key = append(key, candidateHash.Value[:]...)
	err = batch.unfinalized.Put(key, nil)
	require.NoError(t, err)
	key = append(uint32ToBytes(uint32(2)), hash[:]...)
	key = append(key, candidateHash.Value[:]...)
	err = batch.unfinalized.Put(key, nil)

	require.NoError(t, err)

	err = batch.flush()
	require.NoError(t, err)

	// check that the key is written
	key12 := append(uint32ToBytes(uint32(blockNumber)), hash[:]...)
	key12 = append(key12, candidateHash.Value[:]...)

	got, err := as.unfinalized.Get(key12)
	require.NoError(t, err)
	require.Equal(t, []byte{}, got)

	key16 := append(uint32ToBytes(uint32(blockNumber)), hash6[:]...)
	key16 = append(key16, candidateHash.Value[:]...)

	got, err = as.unfinalized.Get(key16)
	require.NoError(t, err)
	require.Equal(t, []byte{}, got)

	// delete height, (block 1)
	batch = newAvailabilityStoreBatch(as)
	keyPrefix := append([]byte(unfinalizedPrefix), uint32ToBytes(uint32(blockNumber))...)
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
	key = append(uint32ToBytes(uint32(0)), hash[:]...)
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
	key := append(uint32ToBytes(uint32(blockNumber)), hash[:]...)
	key = append(key, candidateHash.Value[:]...)
	err := batch.unfinalized.Put(key, nil)
	require.NoError(t, err)
	key = append(uint32ToBytes(uint32(blockNumber)), hash6[:]...)
	key = append(key, candidateHash.Value[:]...)
	err = batch.unfinalized.Put(key, nil)
	require.NoError(t, err)
	key = append(uint32ToBytes(uint32(0)), hash[:]...)
	key = append(key, candidateHash.Value[:]...)
	err = batch.unfinalized.Put(key, nil)
	require.NoError(t, err)
	key = append(uint32ToBytes(uint32(2)), hash[:]...)
	key = append(key, candidateHash.Value[:]...)
	err = batch.unfinalized.Put(key, nil)
	require.NoError(t, err)

	err = batch.flush()
	require.NoError(t, err)

	// check that the key is written
	key12 := append(uint32ToBytes(uint32(blockNumber)), hash[:]...)
	key12 = append(key12, candidateHash.Value[:]...)

	got, err := as.unfinalized.Get(key12)
	require.NoError(t, err)
	require.Equal(t, []byte{}, got)

	key16 := append(uint32ToBytes(uint32(blockNumber)), hash6[:]...)
	key16 = append(key16, candidateHash.Value[:]...)

	got, err = as.unfinalized.Get(key16)
	require.NoError(t, err)
	require.Equal(t, []byte{}, got)

	// delete inclusion, (block 1, hash 2)
	batch = newAvailabilityStoreBatch(as)
	key = append(uint32ToBytes(uint32(blockNumber)), hash[:]...)
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

	key = append(uint32ToBytes(uint32(0)), hash[:]...)
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
	availabilityStoreSubsystem := NewAvailabilityStoreSubsystem(inmemoryDB)

	tests := map[string]struct {
		msg            QueryAvailableData
		msgSenderChan  chan AvailableData
		expectedResult AvailableData
		err            error
	}{
		"available_data_found": {
			msg: QueryAvailableData{
				CandidateHash: parachaintypes.CandidateHash{Value: testCandidateReceiptHash},
			},
			msgSenderChan:  make(chan AvailableData),
			expectedResult: testAvailableData1,
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
	availabilityStoreSubsystem := NewAvailabilityStoreSubsystem(inmemoryDB)

	tests := map[string]struct {
		msg            QueryDataAvailability
		msgSenderChan  chan bool
		expectedResult bool
		wantErr        bool
	}{
		"data_available_true": {
			msg: QueryDataAvailability{
				CandidateHash: parachaintypes.CandidateHash{Value: testCandidateReceiptHash},
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
	availabilityStoreSubsystem := NewAvailabilityStoreSubsystem(inmemoryDB)

	tests := map[string]struct {
		msg            QueryChunk
		msgSenderChan  chan ErasureChunk
		expectedResult ErasureChunk
		err            error
	}{
		"chunk_found": {
			msg: QueryChunk{
				CandidateHash: parachaintypes.CandidateHash{Value: testCandidateReceiptHash},
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
	availabilityStoreSubsystem := NewAvailabilityStoreSubsystem(inmemoryDB)

	tests := map[string]struct {
		msg            QueryAllChunks
		msgSenderChan  chan []ErasureChunk
		expectedResult []ErasureChunk
		err            error
	}{
		"chunks_found": {
			msg: QueryAllChunks{
				CandidateHash: parachaintypes.CandidateHash{Value: testCandidateReceiptHash},
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
	availabilityStoreSubsystem := NewAvailabilityStoreSubsystem(inmemoryDB)

	tests := map[string]struct {
		msg            QueryChunkAvailability
		msgSenderChan  chan bool
		expectedResult bool
		err            error
	}{
		"query_chuck_availability_true": {
			msg: QueryChunkAvailability{
				CandidateHash:  parachaintypes.CandidateHash{Value: testCandidateReceiptHash},
				ValidatorIndex: 0,
			},
			msgSenderChan:  make(chan bool),
			expectedResult: true,
		},
		"query_chuck_availability_false": {
			msg: QueryChunkAvailability{
				CandidateHash:  parachaintypes.CandidateHash{Value: testCandidateReceiptHash},
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
	asSub := NewAvailabilityStoreSubsystem(inmemoryDB)

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
	asSub := NewAvailabilityStoreSubsystem(inmemoryDB)

	msgSenderChan := make(chan error)
	msg := StoreAvailableData{
		CandidateHash:       parachaintypes.CandidateHash{Value: common.Hash{0x01}},
		NumValidators:       2,
		AvailableData:       AvailableData{},
		ExpectedErasureRoot: common.MustHexToHash("0xdf3484f071ceeda34e5464806bbca678dd1f1d6155a3af700044deee134ce400"),
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
		nValidators         uint32
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
				expectedErasureRoot: common.MustHexToHash("0xaa4f3f9672310c0cb2e302118b8897cb3e3388ff818e8c0451c2edd403484dce"),
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
			asSub := NewAvailabilityStoreSubsystem(inmemoryDB)
			got, err := asSub.availabilityStore.storeAvailableData(asSub, tt.args.candidate, tt.args.nValidators,
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

func TestAvailabilityStoreSubsystem_noteBlockBacked(t *testing.T) {
	inmemoryDB := setupTestDB(t)
	as := NewAvailabilityStore(inmemoryDB)
	tx := newAvailabilityStoreBatch(as)

	type fields struct {
		availabilityStore availabilityStore
	}
	type args struct {
		tx          *availabilityStoreBatch
		now         BETimestamp
		nValidators uint
		candidate   parachaintypes.CandidateReceipt
	}
	tests := map[string]struct {
		fields   fields
		args     args
		expected map[string][]byte
	}{
		"base_case": {
			fields: fields{
				availabilityStore: *as,
			},
			args: args{
				tx: tx,
			},
			expected: map[string][]byte{
				string([]byte{99, 104, 117, 110, 107, 190, 125, 73, 215, 144, 39, 58, 150, 230, 192, 195, 193, 110,
					209, 237, 104, 149, 255, 87, 165, 123, 87, 60, 126, 176, 129, 233, 174, 218, 120, 53, 245, 0, 0,
					0, 0}): {24, 99, 104, 117, 110, 107, 49, 0, 0, 0, 0, 24, 112, 114, 111, 111, 102, 49},
				string([]byte{99, 104, 117, 110, 107, 190, 125, 73, 215, 144, 39, 58, 150, 230, 192, 195, 193, 110,
					209, 237, 104, 149, 255, 87, 165, 123, 87, 60, 126, 176, 129, 233, 174, 218, 120, 53, 245, 0, 0,
					0, 1}): {24, 99, 104, 117, 110, 107, 50, 1, 0, 0, 0, 24,
					112, 114, 111, 111, 102, 50},
				string([]byte{109, 101, 116, 97, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
					0, 0, 0, 0, 0, 0, 0, 0, 0}): {0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 12, 0, 0, 0},
				string([]byte{109, 101, 116, 97, 115, 137, 2, 67, 55, 164, 51, 156, 149, 98, 11, 193, 131, 84, 203,
					139, 23, 220, 30, 2, 96, 246, 142, 145, 249, 127, 57, 9, 41, 1, 193, 202}): {0, 0, 0, 0, 0,
					0, 0, 0, 0, 0, 0},
				string([]byte{109, 101, 116, 97, 190, 125, 73, 215, 144, 39, 58, 150, 230, 192, 195, 193, 110, 209,
					237, 104, 149, 255, 87, 165, 123, 87, 60, 126, 176, 129, 233, 174, 218, 120, 53, 245}): {0, 0, 0,
					0, 0, 0, 0, 0, 0, 0, 12, 1, 1, 0},
				string([]byte{112, 114, 117, 110, 101, 95, 98, 121, 95, 116, 105, 109, 101, 0, 0, 0, 0, 115, 137, 2,
					67, 55, 164, 51, 156, 149, 98, 11, 193, 131, 84, 203, 139, 23, 220, 30, 2, 96, 246, 142, 145,
					249, 127, 57, 9, 41, 1, 193, 202}): {},
			},
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			av := &AvailabilityStoreSubsystem{
				availabilityStore: tt.fields.availabilityStore,
			}
			err := av.noteBlockBacked(tt.args.tx, tt.args.now, tt.args.nValidators, tt.args.candidate)
			require.NoError(t, err)
			err = tt.args.tx.flushAll()
			require.NoError(t, err)
			itr, err := inmemoryDB.NewIterator()
			require.NoError(t, err)
			defer itr.Release()
			itr.First()
			for itr.Next() {
				key := itr.Key()
				value := itr.Value()
				require.Equal(t, tt.expected[string(key)], value)
			}
		})
	}
}

func TestAvailabilityStoreSubsystem_noteBlockIncluded(t *testing.T) {
	inmemoryDB := setupTestDB(t)
	as := NewAvailabilityStore(inmemoryDB)
	tx := newAvailabilityStoreBatch(as)

	type fields struct {
		availabilityStore availabilityStore
	}
	type args struct {
		tx          *availabilityStoreBatch
		blockNumber parachaintypes.BlockNumber
		blockHash   common.Hash
		candidate   parachaintypes.CandidateReceipt
	}
	tests := map[string]struct {
		fields   fields
		args     args
		expected map[string][]byte
	}{
		"baseCase": {
			fields: fields{
				availabilityStore: *as,
			},
			args: args{
				tx:        tx,
				candidate: testCandidateReceipt,
			},
			expected: map[string][]byte{
				string([]byte{99, 104, 117, 110, 107, 190, 125, 73, 215, 144, 39, 58, 150, 230, 192, 195, 193, 110,
					209, 237, 104, 149, 255, 87, 165, 123, 87, 60, 126, 176, 129, 233, 174, 218, 120, 53, 245, 0, 0,
					0, 0}): {24, 99, 104, 117, 110, 107, 49, 0, 0, 0, 0, 24, 112, 114, 111, 111, 102, 49},
				string([]byte{99, 104, 117, 110, 107, 190, 125, 73, 215, 144, 39, 58, 150, 230, 192, 195, 193, 110,
					209, 237, 104, 149, 255, 87, 165, 123, 87, 60, 126, 176, 129, 233, 174, 218, 120, 53, 245, 0, 0,
					0, 1}): {24, 99, 104, 117, 110, 107, 50, 1, 0, 0, 0, 24,
					112, 114, 111, 111, 102, 50},
				string([]byte{109, 101, 116, 97, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
					0, 0, 0, 0, 0, 0, 0, 0, 0}): {0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 12, 0, 0, 0},
				string([]byte{109, 101, 116, 97, 190, 125, 73, 215, 144, 39, 58, 150, 230, 192, 195, 193, 110, 209,
					237, 104, 149, 255, 87, 165, 123, 87, 60, 126, 176, 129, 233, 174, 218, 120, 53, 245}): {1, 0, 0,
					0, 0, 0, 0, 0, 0, 4, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
					0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 12, 1, 1, 0},
				string([]byte{117, 110, 102, 105, 110, 97, 108, 105, 122, 101, 100, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
					0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 190, 125, 73, 215,
					144, 39, 58, 150, 230, 192, 195, 193, 110, 209, 237, 104, 149, 255, 87, 165, 123, 87, 60, 126,
					176, 129, 233, 174, 218, 120, 53, 245}): {},
			},
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			av := &AvailabilityStoreSubsystem{
				availabilityStore: tt.fields.availabilityStore,
			}
			err := av.noteBlockIncluded(tt.args.tx, tt.args.blockNumber, tt.args.blockHash, tt.args.candidate)
			require.NoError(t, err)
			err = tt.args.tx.flushAll()
			require.NoError(t, err)
			itr, err := inmemoryDB.NewIterator()
			require.NoError(t, err)
			defer itr.Release()
			itr.First()
			for itr.Next() {
				key := itr.Key()
				value := itr.Value()
				require.Equal(t, tt.expected[string(key)], value)
			}
		})
	}
}

func newTestHarness(t *testing.T) *testHarness {
	overseer := newTestOverseer()
	harness := &testHarness{
		overseer:       overseer,
		broadcastIndex: 0,
		t:              t,
	}

	harness.db = setupTestDB(t)

	testPruningConfig := &pruningConfig{
		keepUnavailableFor: time.Second * 2,
		keepFinalizedFor:   time.Second * 5,
		pruningInterval:    time.Second * 1,
	}

	availabilityStore, err := Register(harness.overseer.GetSubsystemToOverseerChannel(),
		harness.db, testPruningConfig)

	require.NoError(t, err)

	availabilityStore.OverseerToSubSystem = harness.overseer.RegisterSubsystem(availabilityStore)

	return harness
}

type testHarness struct {
	overseer          *testOverseer
	t                 *testing.T
	broadcastMessages []any
	broadcastIndex    int
	processes         []func(msg any)
	db                database.Database
}

func (h *testHarness) processMessages() {
	processIndex := 0
	for {
		select {
		case msg := <-h.overseer.SubsystemsToOverseer:
			if h.processes != nil && processIndex < len(h.processes) {
				h.processes[processIndex](msg)
				processIndex++
			}
		case <-h.overseer.ctx.Done():
			if err := h.overseer.ctx.Err(); err != nil {
				logger.Errorf("ctx error: %v\n", err)
			}
			h.overseer.wg.Done()
			return
		}
	}
}

func (h *testHarness) triggerBroadcast() {
	h.overseer.broadcast(h.broadcastMessages[h.broadcastIndex])
	h.broadcastIndex++
}

func (h *testHarness) printDB(caption string) {
	fmt.Printf("db contents %v:\n", caption)
	iterator, err := h.db.NewIterator()
	require.NoError(h.t, err)
	defer iterator.Release()

	for iterator.First(); iterator.Valid(); iterator.Next() {
		fmt.Printf("key: %x, value: %x\n", iterator.Key(), iterator.Value())
	}
}

func (h *testHarness) importLeaf(t *testing.T, parentHash common.Hash,
	blockNumber uint32, candidateEvents []parachaintypes.CandidateEvent) common.Hash {
	header := types.Header{
		ParentHash: parentHash,
		Number:     uint(blockNumber),
	}
	activatedLeaf := header.Hash()

	h.processes = append(h.processes, func(msg any) {
		msg2, _ := msg.(chainapi.ChainAPIMessage[chainapi.BlockHeader])
		msg2.ResponseChannel <- header
		require.Equal(t, chainapi.BlockHeader(activatedLeaf), msg2.Message)
	})

	h.processes = append(h.processes, func(msg any) {
		msg2, _ := msg.(chainapi.ChainAPIMessage[util.Ancestors])
		msg2.ResponseChannel <- util.AncestorsResponse{
			Ancestors: []common.Hash{{0x01}, {0x02}},
		}
		require.Equal(t, activatedLeaf, msg2.Message.Hash)
	})

	h.processes = append(h.processes, func(msg any) {
		msg2, _ := msg.(parachain.RuntimeAPIMessage)
		require.Equal(t, activatedLeaf, msg2.Hash)
		ctrl := gomock.NewController(h.t)
		inst := NewMockRuntimeInstance(ctrl)

		inst.EXPECT().ParachainHostCandidateEvents().Return(candidateEvents, nil)

		msg2.Resp <- inst
	})

	h.overseer.broadcast(parachaintypes.ActiveLeavesUpdateSignal{
		Activated: &parachaintypes.ActivatedLeaf{
			Hash:   activatedLeaf,
			Number: uint32(1),
		},
	})

	return activatedLeaf
}

func (h *testHarness) hasAllChunks(candidateHash parachaintypes.CandidateHash, numValidators uint32, //nolint
	expectPresent bool) bool {

	for i := uint32(0); i < numValidators; i++ {
		msgQueryChan := make(chan ErasureChunk)
		queryChunk := QueryChunk{
			CandidateHash:  candidateHash,
			ValidatorIndex: i,
			Sender:         msgQueryChan,
		}
		h.broadcastMessages = append(h.broadcastMessages, queryChunk)
		h.triggerBroadcast()

		msgQueryChanResult := <-queryChunk.Sender
		if msgQueryChanResult.Chunk == nil && expectPresent {
			return false
		}
	}
	return true
}

func (h *testHarness) queryAvailableData(candidateHash parachaintypes.CandidateHash) AvailableData {
	msgSenderQueryChan := make(chan AvailableData)
	queryData := QueryAvailableData{
		CandidateHash: candidateHash,
		Sender:        msgSenderQueryChan,
	}
	h.broadcastMessages = append(h.broadcastMessages, queryData)

	h.triggerBroadcast()
	msgQueryChan := <-msgSenderQueryChan
	return msgQueryChan
}

func buildAvailableDataBranchesRoot(t *testing.T, numValidators uint32, availableData AvailableData) common.Hash {
	availableDataEnc, err := scale.Marshal(availableData)
	require.NoError(t, err)

	chunksExpected, err := erasure.ObtainChunks(uint(numValidators), availableDataEnc)
	require.NoError(t, err)

	tr := inmemory.NewEmptyTrie()

	for i, chunk := range chunksExpected {
		result := make([]byte, 4)
		binary.BigEndian.PutUint32(result, uint32(i))
		err := tr.Put(result, common.MustBlake2bHash(chunk).ToBytes())
		require.NoError(t, err)
	}
	branchHash, err := trie.V1.Hash(tr)
	require.NoError(t, err)
	return branchHash
}

func newTestOverseer() *testOverseer {
	ctx, cancel := context.WithCancel(context.Background())

	return &testOverseer{
		ctx:                  ctx,
		cancel:               cancel,
		subsystems:           make(map[parachaintypes.Subsystem]chan any),
		SubsystemsToOverseer: make(chan any),
	}
}

type testOverseer struct {
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	subsystems           map[parachaintypes.Subsystem]chan any
	SubsystemsToOverseer chan any
}

func (to *testOverseer) GetSubsystemToOverseerChannel() chan any {
	return to.SubsystemsToOverseer
}

func (to *testOverseer) RegisterSubsystem(subsystem parachaintypes.Subsystem) chan any {
	overseerToSubSystem := make(chan any)
	to.subsystems[subsystem] = overseerToSubSystem

	return overseerToSubSystem
}

func (to *testOverseer) Start() error {
	// start subsystems
	for subsystem, overseerToSubSystem := range to.subsystems {
		to.wg.Add(1)
		go func(sub parachaintypes.Subsystem, overseerToSubSystem chan any) {
			sub.Run(to.ctx, overseerToSubSystem, to.SubsystemsToOverseer)
			logger.Infof("subsystem %v stopped", sub)
			to.wg.Done()
		}(subsystem, overseerToSubSystem)
	}
	return nil
}

func (to *testOverseer) Stop() error {
	return nil
}

func (to *testOverseer) broadcast(msg any) {
	for _, overseerToSubSystem := range to.subsystems {
		overseerToSubSystem <- msg
	}
}

func TestRuntimeApiErrorDoesNotStopTheSubsystemTestHarness(t *testing.T) {
	ctrl := gomock.NewController(t)
	harness := newTestHarness(t)
	defer harness.overseer.Stop()

	activeLeavesUpdate := parachaintypes.ActiveLeavesUpdateSignal{
		Activated: &parachaintypes.ActivatedLeaf{
			Hash:   common.Hash{},
			Number: uint32(1),
		},
		Deactivated: []common.Hash{{}},
	}

	harness.broadcastMessages = append(harness.broadcastMessages, activeLeavesUpdate)
	harness.processes = append(harness.processes, func(msg any) {
		msg2, _ := msg.(chainapi.ChainAPIMessage[chainapi.BlockHeader])
		msg2.ResponseChannel <- types.Header{
			Number: 3,
		}
	})
	harness.processes = append(harness.processes, func(msg any) {
		msg2, _ := msg.(chainapi.ChainAPIMessage[util.Ancestors])
		msg2.ResponseChannel <- util.AncestorsResponse{
			Ancestors: []common.Hash{{0x01}, {0x02}},
		}
	})
	harness.processes = append(harness.processes, func(msg any) {
		msg2, _ := msg.(parachain.RuntimeAPIMessage)

		// return error from runtime call, and check that the subsystem continues to run
		inst := NewMockRuntimeInstance(ctrl)
		inst.EXPECT().ParachainHostCandidateEvents().Return(nil, errors.New("error"))

		msg2.Resp <- inst
	})

	err := harness.overseer.Start()
	require.NoError(t, err)

	go harness.processMessages()

	harness.triggerBroadcast()
	// time to process messages
	time.Sleep(500 * time.Millisecond)

	err = harness.overseer.Stop()
	require.NoError(t, err)
}

func TestStoreChunkWorks(t *testing.T) {
	harness := newTestHarness(t)
	defer harness.overseer.Stop()

	msgSenderChan := make(chan any)

	chunkMsg := StoreChunk{
		CandidateHash: parachaintypes.CandidateHash{Value: testCandidateReceiptHash},
		Chunk:         testChunk1,
		Sender:        msgSenderChan,
	}

	harness.broadcastMessages = append(harness.broadcastMessages, chunkMsg)
	msgSenderQueryChan := make(chan ErasureChunk)

	harness.broadcastMessages = append(harness.broadcastMessages, QueryChunk{
		CandidateHash:  parachaintypes.CandidateHash{Value: testCandidateReceiptHash},
		ValidatorIndex: 0,
		Sender:         msgSenderQueryChan,
	})

	err := harness.overseer.Start()
	require.NoError(t, err)

	go harness.processMessages()

	harness.triggerBroadcast()
	time.Sleep(100 * time.Millisecond)

	msgSenderChanResult := <-chunkMsg.Sender
	require.Nil(t, msgSenderChanResult)

	harness.triggerBroadcast()

	msgQueryChan := <-msgSenderQueryChan
	require.Equal(t, testChunk1, msgQueryChan)

	err = harness.overseer.Stop()
	require.NoError(t, err)
}

func TestStoreChunkDoesNothingIfNoEntryAlready(t *testing.T) {
	harness := newTestHarness(t)
	defer harness.overseer.Stop()

	msgSenderChan := make(chan any)

	chunkMsg := StoreChunk{
		CandidateHash: parachaintypes.CandidateHash{Value: common.Hash{0x01}},
		Chunk:         testChunk1,
		Sender:        msgSenderChan,
	}

	harness.broadcastMessages = append(harness.broadcastMessages, chunkMsg)
	msgSenderQueryChan := make(chan ErasureChunk)

	harness.broadcastMessages = append(harness.broadcastMessages, QueryChunk{
		CandidateHash:  parachaintypes.CandidateHash{Value: common.Hash{0x01}},
		ValidatorIndex: 0,
		Sender:         msgSenderQueryChan,
	})

	err := harness.overseer.Start()
	require.NoError(t, err)

	go harness.processMessages()

	harness.triggerBroadcast()

	msgSenderChanResult := <-chunkMsg.Sender
	require.Equal(t, nil, msgSenderChanResult)

	harness.triggerBroadcast()

	msgQueryChan := <-msgSenderQueryChan
	require.Equal(t, ErasureChunk{}, msgQueryChan)

	err = harness.overseer.Stop()
	require.NoError(t, err)
}

func TestQueryChunkChecksMetadata(t *testing.T) {
	harness := newTestHarness(t)
	defer harness.overseer.Stop()

	msgSenderChan := make(chan bool)

	queryChunkMsg := QueryChunkAvailability{
		CandidateHash:  parachaintypes.CandidateHash{Value: testCandidateReceiptHash},
		ValidatorIndex: 0,
		Sender:         msgSenderChan,
	}

	harness.broadcastMessages = append(harness.broadcastMessages, queryChunkMsg)
	msgSender2Chan := make(chan bool)

	queryChunk2Msg := QueryChunkAvailability{
		CandidateHash:  parachaintypes.CandidateHash{Value: common.Hash{0x01}},
		ValidatorIndex: 2,
		Sender:         msgSender2Chan,
	}
	harness.broadcastMessages = append(harness.broadcastMessages, queryChunk2Msg)

	err := harness.overseer.Start()
	require.NoError(t, err)

	go harness.processMessages()

	harness.triggerBroadcast()

	msgSenderChanResult := <-queryChunkMsg.Sender
	require.Equal(t, true, msgSenderChanResult)

	harness.triggerBroadcast()

	msgQueryChan := <-queryChunk2Msg.Sender
	require.Equal(t, false, msgQueryChan)

	err = harness.overseer.Stop()
	require.NoError(t, err)
}

func TestStorePOVandQueryChunkWorks(t *testing.T) {
	harness := newTestHarness(t)
	defer harness.overseer.Stop()
	candidateHash := parachaintypes.CandidateHash{Value: common.Hash{0x01}}
	nValidators := uint32(10)

	pov := parachaintypes.PoV{BlockData: parachaintypes.BlockData{4, 5, 6}}

	availableData := AvailableData{
		PoV: pov,
	}
	availableDataEnc, err := scale.Marshal(availableData)
	require.NoError(t, err)

	chunksExpected, err := erasure.ObtainChunks(uint(nValidators), availableDataEnc)
	require.NoError(t, err)

	tr := inmemory.NewEmptyTrie()

	for i, chunk := range chunksExpected {
		result := make([]byte, 4)
		binary.BigEndian.PutUint32(result, uint32(i))
		err := tr.Put(result, common.MustBlake2bHash(chunk).ToBytes())
		require.NoError(t, err)
	}
	branchHash, err := trie.V1.Hash(tr)
	require.NoError(t, err)

	msgSenderChan := make(chan error)

	blockMsg := StoreAvailableData{
		CandidateHash:       candidateHash,
		NumValidators:       nValidators,
		AvailableData:       availableData,
		ExpectedErasureRoot: branchHash,
		Sender:              msgSenderChan,
	}

	harness.broadcastMessages = append(harness.broadcastMessages, blockMsg)

	err = harness.overseer.Start()
	require.NoError(t, err)

	go harness.processMessages()

	harness.triggerBroadcast()

	msgSenderChanResult := <-blockMsg.Sender
	require.Equal(t, nil, msgSenderChanResult)

	for i := uint32(0); i < nValidators; i++ {
		msgSenderQueryChan := make(chan ErasureChunk)
		harness.broadcastMessages = append(harness.broadcastMessages, QueryChunk{
			CandidateHash:  candidateHash,
			ValidatorIndex: i,
			Sender:         msgSenderQueryChan,
		})
		harness.triggerBroadcast()
		msgQueryChan := <-msgSenderQueryChan
		require.Equal(t, chunksExpected[i], msgQueryChan.Chunk)
	}
}

func TestQueryAllChunksWorks(t *testing.T) {
	harness := newTestHarness(t)
	defer harness.overseer.Stop()
	candidateHash := parachaintypes.CandidateHash{Value: testCandidateReceiptHash}
	candidateHash2 := parachaintypes.CandidateHash{Value: common.Hash{0x02}}
	candidateHash3 := parachaintypes.CandidateHash{Value: common.Hash{0x03}}

	msgChunkSenderChan := make(chan any)

	chunkMsg := StoreChunk{
		CandidateHash: parachaintypes.CandidateHash{Value: common.Hash{0x02}},
		Chunk:         testChunk1,
		Sender:        msgChunkSenderChan,
	}

	harness.broadcastMessages = append(harness.broadcastMessages, chunkMsg)

	msgSenderQueryChan := make(chan []ErasureChunk)
	harness.broadcastMessages = append(harness.broadcastMessages, QueryAllChunks{
		CandidateHash: candidateHash,
		Sender:        msgSenderQueryChan,
	})

	harness.broadcastMessages = append(harness.broadcastMessages, QueryAllChunks{
		CandidateHash: candidateHash2,
		Sender:        msgSenderQueryChan,
	})

	harness.broadcastMessages = append(harness.broadcastMessages, QueryAllChunks{
		CandidateHash: candidateHash3,
		Sender:        msgSenderQueryChan,
	})

	err := harness.overseer.Start()
	require.NoError(t, err)

	go harness.processMessages()

	//result from store chunk
	harness.triggerBroadcast()
	msgQueryChan := <-msgChunkSenderChan
	require.Equal(t, nil, msgQueryChan)

	// result from query all chunks for candidatehash
	harness.triggerBroadcast()
	msgQueryChan = <-msgSenderQueryChan
	require.Equal(t, []ErasureChunk{testChunk1, testChunk2},
		msgQueryChan)

	// result from query all chunks for candidatehash2
	harness.triggerBroadcast()
	msgQueryChan = <-msgSenderQueryChan
	require.Equal(t, []ErasureChunk{testChunk1}, msgQueryChan)

	// result from query all chunks for candidatehash3
	harness.triggerBroadcast()
	msgQueryChan = <-msgSenderQueryChan
	require.Equal(t, []ErasureChunk{}, msgQueryChan)

	err = harness.overseer.Stop()
	require.NoError(t, err)
}

func TestQueryChunkSizeWorks(t *testing.T) {
	harness := newTestHarness(t)
	defer harness.overseer.Stop()

	msgSenderChan := make(chan uint32)

	queryChunkMsg := QueryChunkSize{
		CandidateHash: parachaintypes.CandidateHash{Value: testCandidateReceiptHash},
		Sender:        msgSenderChan,
	}

	harness.broadcastMessages = append(harness.broadcastMessages, queryChunkMsg)

	err := harness.overseer.Start()
	require.NoError(t, err)

	go harness.processMessages()

	harness.triggerBroadcast()

	msgSenderChanResult := <-queryChunkMsg.Sender
	require.Equal(t, uint32(6), msgSenderChanResult)

	err = harness.overseer.Stop()
	require.NoError(t, err)
}

func TestStoreBlockWorks(t *testing.T) {
	harness := newTestHarness(t)
	defer harness.overseer.Stop()
	candidateHash := parachaintypes.CandidateHash{Value: common.Hash{0x01}}
	nValidators := uint32(10)

	pov := parachaintypes.PoV{BlockData: parachaintypes.BlockData{4, 5, 6}}

	availableData := AvailableData{
		PoV: pov,
		ValidationData: parachaintypes.PersistedValidationData{
			ParentHead: parachaintypes.HeadData{Data: []byte{}},
		},
	}
	availableDataEnc, err := scale.Marshal(availableData)
	require.NoError(t, err)

	chunksExpected, err := erasure.ObtainChunks(uint(nValidators), availableDataEnc)
	require.NoError(t, err)

	tr := inmemory.NewEmptyTrie()

	for i, chunk := range chunksExpected {
		result := make([]byte, 4)
		binary.BigEndian.PutUint32(result, uint32(i))
		err := tr.Put(result, common.MustBlake2bHash(chunk).ToBytes())
		require.NoError(t, err)
	}
	branchHash, err := trie.V1.Hash(tr)
	require.NoError(t, err)

	msgSenderChan := make(chan error)

	blockMsg := StoreAvailableData{
		CandidateHash:       candidateHash,
		NumValidators:       nValidators,
		AvailableData:       availableData,
		ExpectedErasureRoot: branchHash,
		Sender:              msgSenderChan,
	}

	harness.broadcastMessages = append(harness.broadcastMessages, blockMsg)

	msgSenderQueryChan := make(chan AvailableData)
	queryData := QueryAvailableData{
		CandidateHash: candidateHash,
		Sender:        msgSenderQueryChan,
	}
	harness.broadcastMessages = append(harness.broadcastMessages, queryData)

	msgSenderErasureChan := make(chan ErasureChunk)
	queryChunk := QueryChunk{
		CandidateHash:  candidateHash,
		ValidatorIndex: 5,
		Sender:         msgSenderErasureChan,
	}
	harness.broadcastMessages = append(harness.broadcastMessages, queryChunk)

	err = harness.overseer.Start()
	require.NoError(t, err)

	go harness.processMessages()

	harness.triggerBroadcast()

	msgSenderChanResult := <-blockMsg.Sender
	require.Equal(t, nil, msgSenderChanResult)

	harness.triggerBroadcast()
	msgQueryChan := <-msgSenderQueryChan
	require.Equal(t, availableData, msgQueryChan)

	harness.triggerBroadcast()
	msgSenderErasureChanResult := <-queryChunk.Sender
	expectedChunk := ErasureChunk{
		Chunk: chunksExpected[5],
		Index: 5,
		Proof: []byte{},
	}
	require.Equal(t, expectedChunk, msgSenderErasureChanResult)

	err = harness.overseer.Stop()
	require.NoError(t, err)
}

func TestStoreAvailableDataErasureMismatch(t *testing.T) {
	harness := newTestHarness(t)
	defer harness.overseer.Stop()
	candidateHash := parachaintypes.CandidateHash{Value: common.Hash{0x01}}
	nValidators := uint32(10)

	pov := parachaintypes.PoV{BlockData: parachaintypes.BlockData{4, 5, 6}}

	availableData := AvailableData{
		PoV: pov,
	}

	msgSenderChan := make(chan error)

	blockMsg := StoreAvailableData{
		CandidateHash:       candidateHash,
		NumValidators:       nValidators,
		AvailableData:       availableData,
		ExpectedErasureRoot: common.Hash{},
		Sender:              msgSenderChan,
	}

	harness.broadcastMessages = append(harness.broadcastMessages, blockMsg)

	err := harness.overseer.Start()
	require.NoError(t, err)

	go harness.processMessages()

	harness.triggerBroadcast()

	msgSenderChanResult := <-blockMsg.Sender
	require.Equal(t, errInvalidErasureRoot, msgSenderChanResult)

	err = harness.overseer.Stop()
	require.NoError(t, err)
}

func TestStoredButNotIncludedDataIsPruned(t *testing.T) {
	harness := newTestHarness(t)
	defer harness.overseer.Stop()
	candidateHash := parachaintypes.CandidateHash{Value: common.Hash{0x01}}
	nValidators := uint32(10)

	pov := parachaintypes.PoV{BlockData: parachaintypes.BlockData{4, 5, 6}}

	availableData := AvailableData{
		PoV: pov,
		ValidationData: parachaintypes.PersistedValidationData{
			ParentHead: parachaintypes.HeadData{Data: []byte{}},
		},
	}
	availableDataEnc, err := scale.Marshal(availableData)
	require.NoError(t, err)

	chunksExpected, err := erasure.ObtainChunks(uint(nValidators), availableDataEnc)
	require.NoError(t, err)

	tr := inmemory.NewEmptyTrie()

	for i, chunk := range chunksExpected {
		result := make([]byte, 4)
		binary.BigEndian.PutUint32(result, uint32(i))
		err := tr.Put(result, common.MustBlake2bHash(chunk).ToBytes())
		require.NoError(t, err)
	}
	branchHash, err := trie.V1.Hash(tr)
	require.NoError(t, err)

	msgSenderChan := make(chan error)

	blockMsg := StoreAvailableData{
		CandidateHash:       candidateHash,
		NumValidators:       nValidators,
		AvailableData:       availableData,
		ExpectedErasureRoot: branchHash,
		Sender:              msgSenderChan,
	}

	harness.broadcastMessages = append(harness.broadcastMessages, blockMsg)

	err = harness.overseer.Start()
	require.NoError(t, err)

	go harness.processMessages()

	harness.triggerBroadcast()

	msgSenderChanResult := <-blockMsg.Sender
	require.Equal(t, nil, msgSenderChanResult)

	// check that the data is still there
	msgSenderQueryChan := make(chan AvailableData)
	queryData := QueryAvailableData{
		CandidateHash: candidateHash,
		Sender:        msgSenderQueryChan,
	}
	harness.broadcastMessages = append(harness.broadcastMessages, queryData)

	harness.triggerBroadcast()
	msgQueryChan := <-msgSenderQueryChan
	require.Equal(t, availableData, msgQueryChan)

	// wait for pruning to occur and check that the data is gone
	time.Sleep(7000 * time.Millisecond)

	harness.broadcastMessages = append(harness.broadcastMessages, queryData)
	harness.triggerBroadcast()
	msgQueryChan = <-msgSenderQueryChan
	require.Equal(t, AvailableData{}, msgQueryChan)
	harness.printDB("after pruning")

	err = harness.overseer.Stop()
	require.NoError(t, err)
}

func TestStoredDataKeptUntilFinalized(t *testing.T) {
	harness := newTestHarness(t)
	defer harness.overseer.Stop()
	candidateHash := parachaintypes.CandidateHash{Value: testCandidateReceiptHash}
	nValidators := uint32(10)

	pov := parachaintypes.PoV{BlockData: parachaintypes.BlockData{4, 5, 6}}

	availableData := AvailableData{
		PoV: pov,
		ValidationData: parachaintypes.PersistedValidationData{
			ParentHead: parachaintypes.HeadData{Data: []byte{}},
		},
	}
	parent := common.Hash{0x02, 0x02, 0x02, 0x02}
	blockNumber := uint32(3)

	availableDataEnc, err := scale.Marshal(availableData)
	require.NoError(t, err)

	chunksExpected, err := erasure.ObtainChunks(uint(nValidators), availableDataEnc)
	require.NoError(t, err)

	tr := inmemory.NewEmptyTrie()

	for i, chunk := range chunksExpected {
		result := make([]byte, 4)
		binary.BigEndian.PutUint32(result, uint32(i))
		err := tr.Put(result, common.MustBlake2bHash(chunk).ToBytes())
		require.NoError(t, err)
	}
	branchHash, err := trie.V1.Hash(tr)
	require.NoError(t, err)

	msgSenderChan := make(chan error)

	blockMsg := StoreAvailableData{
		CandidateHash:       candidateHash,
		NumValidators:       nValidators,
		AvailableData:       availableData,
		ExpectedErasureRoot: branchHash,
		Sender:              msgSenderChan,
	}

	harness.broadcastMessages = append(harness.broadcastMessages, blockMsg)

	err = harness.overseer.Start()
	require.NoError(t, err)

	go harness.processMessages()

	harness.triggerBroadcast()

	// result from seeding data
	msgSenderChanResult := <-blockMsg.Sender
	require.Equal(t, nil, msgSenderChanResult)

	// check that the data is there
	msgSenderQueryChan := make(chan AvailableData)
	queryData := QueryAvailableData{
		CandidateHash: candidateHash,
		Sender:        msgSenderQueryChan,
	}
	harness.broadcastMessages = append(harness.broadcastMessages, queryData)

	harness.triggerBroadcast()
	msgQueryChan := <-msgSenderQueryChan
	require.Equal(t, availableData, msgQueryChan)
	harness.printDB("before import leaf")

	// trigger import leaf
	candidateEvents := parachaintypes.NewCandidateEvents()
	event := parachaintypes.NewCandidateEvent()
	err = event.SetValue(parachaintypes.CandidateIncluded{CandidateReceipt: testCandidateReceipt})
	require.NoError(t, err)
	candidateEvents = append(candidateEvents, event)
	require.NoError(harness.t, err)

	aLeaf := harness.importLeaf(t, parent, blockNumber, candidateEvents)

	time.Sleep(500 * time.Millisecond)
	harness.printDB("after import leaf")

	// check that the data is still there
	// queryAvailabeData, hasAllChunks
	harness.broadcastMessages = append(harness.broadcastMessages, queryData)

	harness.triggerBroadcast()
	msgQueryChan = <-msgSenderQueryChan
	require.Equal(t, availableData, msgQueryChan)
	harness.printDB("after queryData")

	// check that the chunks are there
	hasChunks := harness.hasAllChunks(candidateHash, nValidators, true)
	require.True(t, hasChunks)

	// trigger block finalized
	blockFinalizedSignal := parachaintypes.BlockFinalizedSignal{
		Hash:        aLeaf,
		BlockNumber: blockNumber,
	}
	harness.broadcastMessages = append(harness.broadcastMessages, blockFinalizedSignal)
	harness.triggerBroadcast()

	// wait for pruning to occur and check that the data is gone
	time.Sleep(7000 * time.Millisecond)
	harness.printDB("after block finalized")

	harness.broadcastMessages = append(harness.broadcastMessages, queryData)

	harness.triggerBroadcast()
	msgQueryChan = <-msgSenderQueryChan
	expectedResult := AvailableData{}
	require.Equal(t, expectedResult, msgQueryChan)

	// check that the chunks are gone
	hasChunks = harness.hasAllChunks(candidateHash, nValidators, false)
	require.True(t, hasChunks)

	err = harness.overseer.Stop()
	require.NoError(t, err)
}

func TestForkfullnessWorks(t *testing.T) {
	harness := newTestHarness(t)
	defer harness.overseer.Stop()
	nValidators := uint32(10)
	msgSenderChan := make(chan error)

	blockNumber1 := uint32(5)
	parent1 := common.Hash{0x03, 0x03, 0x03, 0x03}
	pov1 := parachaintypes.PoV{BlockData: parachaintypes.BlockData{1, 2, 3}}
	pov1Hash := common.MustBlake2bHash(scale.MustMarshal(pov1))
	candidate1 := parachaintypes.CandidateReceipt{
		Descriptor: parachaintypes.CandidateDescriptor{
			PovHash: pov1Hash,
		},
	}

	candidate1Hash, err := candidate1.Hash()
	require.NoError(t, err)
	availableData1 := AvailableData{
		PoV: pov1,
		ValidationData: parachaintypes.PersistedValidationData{
			ParentHead: parachaintypes.HeadData{Data: []byte{}},
		},
	}
	availableData1ErasureRoot := buildAvailableDataBranchesRoot(t, nValidators, availableData1)
	msg1StoreAvailableData := StoreAvailableData{
		CandidateHash:       parachaintypes.CandidateHash{Value: candidate1Hash},
		NumValidators:       nValidators,
		AvailableData:       availableData1,
		ExpectedErasureRoot: availableData1ErasureRoot,
		Sender:              msgSenderChan,
	}
	candidate1Events := parachaintypes.NewCandidateEvents()
	event := parachaintypes.NewCandidateEvent()
	err = event.SetValue(parachaintypes.CandidateIncluded{CandidateReceipt: candidate1})
	require.NoError(t, err)
	candidate1Events = append(candidate1Events, event)

	blockNumber2 := uint32(5)
	parent2 := common.Hash{0x04, 0x04, 0x04, 0x04}
	pov2 := parachaintypes.PoV{BlockData: parachaintypes.BlockData{4, 5, 6}}
	pov2Hash := common.MustBlake2bHash(scale.MustMarshal(pov2))
	candidate2 := parachaintypes.CandidateReceipt{
		Descriptor: parachaintypes.CandidateDescriptor{
			PovHash: pov2Hash,
		},
	}
	candidate2Hash, err := candidate2.Hash()
	require.NoError(t, err)
	availableData2 := AvailableData{
		PoV: pov2,
		ValidationData: parachaintypes.PersistedValidationData{
			ParentHead: parachaintypes.HeadData{Data: []byte{}},
		},
	}
	availableData2ErasureRoot := buildAvailableDataBranchesRoot(t, nValidators, availableData2)
	msg2StoreAvailabeData := StoreAvailableData{
		CandidateHash:       parachaintypes.CandidateHash{Value: candidate2Hash},
		NumValidators:       nValidators,
		AvailableData:       availableData2,
		ExpectedErasureRoot: availableData2ErasureRoot,
		Sender:              msgSenderChan,
	}
	candidate2Events := parachaintypes.NewCandidateEvents()
	event = parachaintypes.NewCandidateEvent()
	err = event.SetValue(parachaintypes.CandidateIncluded{CandidateReceipt: candidate2})
	require.NoError(t, err)
	candidate2Events = append(candidate2Events, event)

	err = harness.overseer.Start()
	require.NoError(t, err)

	go harness.processMessages()

	harness.broadcastMessages = append(harness.broadcastMessages, msg1StoreAvailableData)
	harness.triggerBroadcast()

	// result from seeding data
	msg1SenderChanResult := <-msg1StoreAvailableData.Sender
	require.Equal(t, nil, msg1SenderChanResult)

	harness.broadcastMessages = append(harness.broadcastMessages, msg2StoreAvailabeData)
	harness.triggerBroadcast()

	// result from seeding data 2
	msgSender2ChanResult := <-msg2StoreAvailabeData.Sender
	require.Equal(t, nil, msgSender2ChanResult)

	// confirm available data 1 and 2, and has all chunks 1 and 2
	availableDataResult := harness.queryAvailableData(parachaintypes.CandidateHash{Value: candidate1Hash})
	require.Equal(t, availableData1, availableDataResult)
	availableDataResult = harness.queryAvailableData(parachaintypes.CandidateHash{Value: candidate2Hash})
	require.Equal(t, availableData2, availableDataResult)
	hasChunks := harness.hasAllChunks(parachaintypes.CandidateHash{Value: candidate1Hash}, nValidators, true)
	require.True(t, hasChunks)
	hasChunks = harness.hasAllChunks(parachaintypes.CandidateHash{Value: candidate2Hash}, nValidators, true)
	require.True(t, hasChunks)
	harness.printDB("before import leaf")

	// import leaf for candidate 1
	activatedLeaf := harness.importLeaf(t, parent1, blockNumber1, candidate1Events)
	time.Sleep(50 * time.Millisecond)

	// import leaf for candidate 2
	harness.importLeaf(t, parent2, blockNumber2, candidate2Events)
	time.Sleep(1500 * time.Millisecond)
	harness.printDB("after import leaf")

	// signal block 1 finalized for candidate 1
	blockFinalizedSignal := parachaintypes.BlockFinalizedSignal{
		Hash:        activatedLeaf,
		BlockNumber: blockNumber1,
	}
	harness.broadcastMessages = append(harness.broadcastMessages, blockFinalizedSignal)
	harness.triggerBroadcast()
	time.Sleep(50 * time.Millisecond)

	// confirm available data 1, and 2, and has all chunks 1 and 2
	availableDataResult = harness.queryAvailableData(parachaintypes.CandidateHash{Value: candidate1Hash})
	require.Equal(t, availableData1, availableDataResult)
	availableDataResult = harness.queryAvailableData(parachaintypes.CandidateHash{Value: candidate2Hash})
	require.Equal(t, availableData2, availableDataResult)
	hasChunks = harness.hasAllChunks(parachaintypes.CandidateHash{Value: candidate1Hash}, nValidators, true)
	require.True(t, hasChunks)
	hasChunks = harness.hasAllChunks(parachaintypes.CandidateHash{Value: candidate2Hash}, nValidators, true)
	require.True(t, hasChunks)
	harness.printDB("after block finalized")

	// wait for pruning
	time.Sleep(3000 * time.Millisecond)
	// query available data 1 matches and 2 is empty, and has all chunks 1 true 2 false
	availableDataResult = harness.queryAvailableData(parachaintypes.CandidateHash{Value: candidate1Hash})
	require.Equal(t, availableData1, availableDataResult)
	availableDataResult = harness.queryAvailableData(parachaintypes.CandidateHash{Value: candidate2Hash})
	require.Equal(t, AvailableData{}, availableDataResult)
	hasChunks = harness.hasAllChunks(parachaintypes.CandidateHash{Value: candidate1Hash}, nValidators, true)
	require.True(t, hasChunks)
	hasChunks = harness.hasAllChunks(parachaintypes.CandidateHash{Value: candidate2Hash}, nValidators, false)
	require.True(t, hasChunks)
	harness.printDB("after unfinalized pruning time delay")

	// wait for finalized pruning
	time.Sleep(3000 * time.Millisecond)
	// query available data 1 and 2 are empty, and has all chunks 1 and 2 are empty
	availableDataResult = harness.queryAvailableData(parachaintypes.CandidateHash{Value: candidate1Hash})
	require.Equal(t, AvailableData{}, availableDataResult)
	availableDataResult = harness.queryAvailableData(parachaintypes.CandidateHash{Value: candidate2Hash})
	require.Equal(t, AvailableData{}, availableDataResult)
	hasChunks = harness.hasAllChunks(parachaintypes.CandidateHash{Value: candidate1Hash}, nValidators, false)
	require.True(t, hasChunks)
	hasChunks = harness.hasAllChunks(parachaintypes.CandidateHash{Value: candidate2Hash}, nValidators, false)
	require.True(t, hasChunks)
	harness.printDB("after final time pruning delay")
}
