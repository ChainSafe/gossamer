package availabilitystore

import (
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

	err := as.storeChunk(common.Hash{0x01}, testChunk1)
	require.NoError(t, err)
	err = as.storeChunk(common.Hash{0x01}, testChunk2)
	require.NoError(t, err)

	err = as.storeAvailableData(common.Hash{0x01}, testavailableData1)
	require.NoError(t, err)

	return inmemoryDB
}
func TestAvailabilityStore_StoreLoadAvailableData(t *testing.T) {
	inmemoryDB := state.NewInMemoryDB(t)
	as := NewAvailabilityStore(inmemoryDB)

	err := as.storeAvailableData(common.Hash{0x01}, testavailableData1)
	require.NoError(t, err)

	got, err := as.loadAvailableData(common.Hash{0x01})
	require.NoError(t, err)
	require.Equal(t, &testavailableData1, got)

	got, err = as.loadAvailableData(common.Hash{0x02})
	require.EqualError(t, err, "getting candidate 0x0200000000000000000000000000000000000000000000000000000000000000"+
		" from available table: pebble: not found")
	var ExpectedAvailableData *AvailableData = nil
	require.Equal(t, ExpectedAvailableData, got)
}

func TestAvailabilityStore_StoreLoadChuckData(t *testing.T) {
	inmemoryDB := state.NewInMemoryDB(t)
	as := NewAvailabilityStore(inmemoryDB)

	err := as.storeChunk(common.Hash{0x01}, testChunk1)
	require.NoError(t, err)
	err = as.storeChunk(common.Hash{0x01}, testChunk2)
	require.NoError(t, err)

	resultChunk, err := as.loadChunk(common.Hash{0x01}, 0)
	require.NoError(t, err)
	require.Equal(t, &testChunk1, resultChunk)
}

func TestAvailabilityStore_handleQueryAvailableData(t *testing.T) {
	inmemoryDB := setupTestDB(t)
	as := NewAvailabilityStore(inmemoryDB)
	asSub := AvailabilityStoreSubsystem{
		availabilityStore: *as,
	}
	msgSenderChan := make(chan AvailableData)
	msg := QueryAvailableData{
		CandidateHash: common.Hash{0x01},
		Sender:        msgSenderChan,
	}

	go asSub.handleQueryAvailableData(msg)
	msgSenderChanResult := <-msg.Sender
	require.Equal(t, testavailableData1, msgSenderChanResult)
}

func TestAvailabilityStore_handleQueryDataAvailability(t *testing.T) {
	inmemoryDB := setupTestDB(t)
	as := NewAvailabilityStore(inmemoryDB)
	asSub := AvailabilityStoreSubsystem{
		availabilityStore: *as,
	}
	msgSenderChan := make(chan bool)
	msg := QueryDataAvailability{
		CandidateHash: common.Hash{0x01},
		Sender:        msgSenderChan,
	}

	go asSub.handleQueryDataAvailability(msg)
	msgSenderChanResult := <-msg.Sender
	require.Equal(t, true, msgSenderChanResult)

	msg2 := QueryDataAvailability{
		CandidateHash: common.Hash{0x02},
		Sender:        msgSenderChan,
	}
	go asSub.handleQueryDataAvailability(msg2)
	msgSenderChanResult2 := <-msg2.Sender
	require.Equal(t, false, msgSenderChanResult2)
}

func TestAvailabilityStore_handleQueryChunk(t *testing.T) {
	inmemoryDB := setupTestDB(t)
	as := NewAvailabilityStore(inmemoryDB)
	asSub := AvailabilityStoreSubsystem{
		availabilityStore: *as,
	}
	msgSenderChan := make(chan ErasureChunk)
	msg := QueryChunk{
		CandidateHash:  common.Hash{0x01},
		ValidatorIndex: 0,
		Sender:         msgSenderChan,
	}

	go asSub.handleQueryChunk(msg)
	msgSenderChanResult := <-msg.Sender
	require.Equal(t, testChunk1, msgSenderChanResult)
}

func TestAvailabilityStore_handleQueryAllChunks(t *testing.T) {
	inmemoryDB := setupTestDB(t)
	as := NewAvailabilityStore(inmemoryDB)
	asSub := AvailabilityStoreSubsystem{
		availabilityStore: *as,
	}
	msgSenderChan := make(chan []ErasureChunk)
	msg := QueryAllChunks{
		CandidateHash: common.Hash{0x01},
		Sender:        msgSenderChan,
	}

	go asSub.handleQueryAllChunks(msg)
	msgSenderChanResult := <-msg.Sender
	require.Equal(t, []ErasureChunk{testChunk1, testChunk2}, msgSenderChanResult)
}

func TestAvailabilityStore_handleQueryChunkAvailability(t *testing.T) {
	inmemoryDB := setupTestDB(t)
	as := NewAvailabilityStore(inmemoryDB)
	asSub := AvailabilityStoreSubsystem{
		availabilityStore: *as,
	}
	msgSenderChan := make(chan bool)
	msg := QueryChunkAvailability{
		CandidateHash:  common.Hash{0x01},
		ValidatorIndex: 0,
		Sender:         msgSenderChan,
	}

	go asSub.handleQueryChunkAvailability(msg)
	msgSenderChanResult := <-msg.Sender
	require.Equal(t, true, msgSenderChanResult)

	msg2 := QueryChunkAvailability{
		CandidateHash:  common.Hash{0x01},
		ValidatorIndex: 2,
		Sender:         msgSenderChan,
	}
	go asSub.handleQueryChunkAvailability(msg2)
	msgSenderChanResult2 := <-msg2.Sender
	require.Equal(t, false, msgSenderChanResult2)
}

func TestAvailabilityStore_handleStoreChunk(t *testing.T) {
	inmemoryDB := setupTestDB(t)
	as := NewAvailabilityStore(inmemoryDB)
	asSub := AvailabilityStoreSubsystem{
		availabilityStore: *as,
	}
	msgSenderChan := make(chan any)
	msg := StoreChunk{
		CandidateHash: common.Hash{0x01},
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
