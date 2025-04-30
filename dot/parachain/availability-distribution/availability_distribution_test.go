// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package availabilitydistribution

import (
	"errors"
	"math"
	"sync"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/dot/network"
	availabilitystore "github.com/ChainSafe/gossamer/dot/parachain/availability-store"
	"github.com/ChainSafe/gossamer/dot/parachain/network-bridge/messages"
	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/libp2p/go-libp2p/core/protocol"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestHandleChunkFetchingRequest(t *testing.T) {
	var (
		ctrl           *gomock.Controller
		netMock        *MockNetwork
		blockStateMock *MockBlockState
		overseerCh     chan any
		ad             *AvailabilityDistribution
	)

	setup := func(t *testing.T) {
		ctrl = gomock.NewController(t)
		netMock = NewMockNetwork(ctrl)
		blockStateMock = NewMockBlockState(ctrl)

		netMock.EXPECT().RegisterRequestHandler(protocol.ID("req_chunk/2"), gomock.Any())
		netMock.EXPECT().RegisterRequestHandler(protocol.ID("req_pov/1"), gomock.Any())

		overseerCh = make(chan any)
		ad = NewAvailabilityDistribution(overseerCh, netMock, blockStateMock, nil)
	}

	t.Run("invalid_request", func(t *testing.T) {
		setup(t)

		response, err := ad.handleChunkFetchingRequest("bob", []byte("0xDECAFBAD"))

		assert.Error(t, err)
		assert.Nil(t, response)
	})

	t.Run("chunk_not_found", func(t *testing.T) {
		setup(t)

		request := &messages.ChunkFetchingRequest{
			CandidateHash: parachaintypes.CandidateHash{Value: common.Hash{0x01}},
			Index:         parachaintypes.ValidatorIndex(0),
		}

		encodedRequest, err := request.Encode()
		assert.NoError(t, err)

		var response network.ResponseMessage

		wg := sync.WaitGroup{}
		wg.Add(1)
		go func() {
			var err error
			response, err = ad.handleChunkFetchingRequest("bob", encodedRequest)
			assert.NoError(t, err)
			assert.NotNil(t, response)
			cfResponse, ok := response.(*messages.ChunkFetchingResponse)
			assert.True(t, ok)

			value, err := cfResponse.Value()
			assert.NoError(t, err)
			assert.Equal(t, messages.NoSuchChunk{}, value)
			wg.Done()
		}()

		query, ok := (<-overseerCh).(availabilitystore.QueryChunk)
		assert.True(t, ok)
		assert.Equal(t, request.CandidateHash, query.CandidateHash)
		assert.Equal(t, request.Index, query.ValidatorIndex)

		query.Sender <- availabilitystore.ErasureChunk{}
		wg.Wait()
	})

	t.Run("chunk_found", func(t *testing.T) {
		setup(t)

		request := messages.ChunkFetchingRequest{
			CandidateHash: parachaintypes.CandidateHash{Value: common.Hash{0x01}},
			Index:         parachaintypes.ValidatorIndex(0),
		}

		encodedRequest, err := request.Encode()
		assert.NoError(t, err)

		testChunk := availabilitystore.ErasureChunk{
			Chunk: []byte{0x01, 0x02},
			Index: 23,
			Proof: [][]byte{{0x03, 0x04}},
		}

		var response network.ResponseMessage

		wg := sync.WaitGroup{}
		wg.Add(1)
		go func() {
			var err error
			response, err = ad.handleChunkFetchingRequest("bob", encodedRequest)
			assert.NoError(t, err)
			assert.NotNil(t, response)
			cfResponse, ok := response.(*messages.ChunkFetchingResponse)
			assert.True(t, ok)

			value, err := cfResponse.Value()
			assert.NoError(t, err)

			chunkRes, ok := value.(messages.ChunkResponse)
			assert.True(t, ok)
			assert.Equal(t, testChunk.Chunk, chunkRes.Chunk)
			assert.Equal(t, testChunk.Index, chunkRes.Index)
			assert.Equal(t, testChunk.Proof, chunkRes.Proof)
			wg.Done()
		}()

		query, ok := (<-overseerCh).(availabilitystore.QueryChunk)
		assert.True(t, ok)
		assert.Equal(t, request.CandidateHash, query.CandidateHash)
		assert.Equal(t, request.Index, query.ValidatorIndex)

		query.Sender <- testChunk
		wg.Wait()
	})
}

func TestHandlePoVFetchingRequest(t *testing.T) {
	var (
		ctrl           *gomock.Controller
		netMock        *MockNetwork
		blockStateMock *MockBlockState
		overseerCh     chan any
		ad             *AvailabilityDistribution
	)

	setup := func(t *testing.T) {
		ctrl = gomock.NewController(t)
		netMock = NewMockNetwork(ctrl)
		blockStateMock = NewMockBlockState(ctrl)

		netMock.EXPECT().RegisterRequestHandler(protocol.ID("req_chunk/2"), gomock.Any())
		netMock.EXPECT().RegisterRequestHandler(protocol.ID("req_pov/1"), gomock.Any())

		overseerCh = make(chan any)
		ad = NewAvailabilityDistribution(overseerCh, netMock, blockStateMock, nil)
	}

	t.Run("invalid_request", func(t *testing.T) {
		setup(t)

		response, err := ad.handlePoVFetchingRequest("bob", []byte("0xDECAFBAD"))

		assert.Error(t, err)
		assert.Nil(t, response)
	})

	t.Run("pov_not_found", func(t *testing.T) {
		setup(t)

		request := messages.PoVFetchingRequest{
			CandidateHash: parachaintypes.CandidateHash{Value: common.Hash{0x01}},
		}

		encodedRequest, err := request.Encode()
		assert.NoError(t, err)

		var response network.ResponseMessage

		wg := sync.WaitGroup{}
		wg.Add(1)
		go func() {
			var err error
			response, err = ad.handlePoVFetchingRequest("bob", encodedRequest)
			assert.NoError(t, err)
			assert.NotNil(t, response)
			cfResponse, ok := response.(*messages.PoVFetchingResponse)
			assert.True(t, ok)

			value, err := cfResponse.Value()
			assert.NoError(t, err)
			assert.Equal(t, parachaintypes.NoSuchPoV{}, value)
			wg.Done()
		}()

		query, ok := (<-overseerCh).(availabilitystore.QueryAvailableData)
		assert.True(t, ok)
		assert.Equal(t, request.CandidateHash, query.CandidateHash)

		query.Sender <- availabilitystore.AvailableData{}
		wg.Wait()
	})

	t.Run("pov_found", func(t *testing.T) {
		setup(t)

		request := messages.PoVFetchingRequest{
			CandidateHash: parachaintypes.CandidateHash{Value: common.Hash{0x01}},
		}

		encodedRequest, err := request.Encode()
		assert.NoError(t, err)

		testPoV := availabilitystore.AvailableData{
			PoV: parachaintypes.PoV{BlockData: []byte{0x01, 0x02}},
			ValidationData: parachaintypes.PersistedValidationData{
				ParentHead: parachaintypes.HeadData{Data: []byte{0x42}},
			},
		}

		var response network.ResponseMessage

		wg := sync.WaitGroup{}
		wg.Add(1)
		go func() {
			var err error
			response, err = ad.handlePoVFetchingRequest("bob", encodedRequest)
			assert.NoError(t, err)
			assert.NotNil(t, response)
			cfResponse, ok := response.(*messages.PoVFetchingResponse)
			assert.True(t, ok)

			value, err := cfResponse.Value()
			assert.NoError(t, err)

			povRes, ok := value.(parachaintypes.PoV)
			assert.True(t, ok)
			assert.Equal(t, testPoV.PoV, povRes)
			wg.Done()
		}()

		query, ok := (<-overseerCh).(availabilitystore.QueryAvailableData)
		assert.True(t, ok)
		assert.Equal(t, request.CandidateHash, query.CandidateHash)

		query.Sender <- testPoV
		wg.Wait()
	})
}

func TestGetBlockAncestorsInSameSession(t *testing.T) {
	var (
		ctrl           *gomock.Controller
		netMock        *MockNetwork
		blockStateMock *MockBlockState
		runtimeMock    *MockInstance
		overseerCh     chan any
		ad             *AvailabilityDistribution
	)

	setup := func(t *testing.T) {
		ctrl = gomock.NewController(t)
		defer ctrl.Finish()

		netMock = NewMockNetwork(ctrl)
		blockStateMock = NewMockBlockState(ctrl)
		runtimeMock = NewMockInstance(ctrl)

		netMock.EXPECT().RegisterRequestHandler(protocol.ID("req_chunk/2"), gomock.Any())
		netMock.EXPECT().RegisterRequestHandler(protocol.ID("req_pov/1"), gomock.Any())

		blockStateMock.EXPECT().
			GetRuntime(gomock.AssignableToTypeOf(common.Hash{})).
			MaxTimes(leafAncestryLenWithinSession+1).
			Return(runtimeMock, nil)

		runtimeMock.EXPECT().Stop().MaxTimes(leafAncestryLenWithinSession + 1)

		overseerCh = make(chan any)
		ad = NewAvailabilityDistribution(overseerCh, netMock, blockStateMock, NewLRUSessionCache(nil))
	}

	t.Run("no_header_for_leaf_hash", func(t *testing.T) {
		setup(t)

		blockStateMock.EXPECT().
			GetHeader(gomock.AssignableToTypeOf(common.Hash{})).
			Return(nil, errors.New("block not found"))

		sessionIndex, ancestors, err := ad.getBlockAncestorsInSameSession(common.Hash{0x01}, 0)

		assert.Error(t, err)
		assert.Equal(t, parachaintypes.SessionIndex(0), sessionIndex)
		assert.Empty(t, ancestors)
	})

	t.Run("session_index_for_leaf_fails", func(t *testing.T) {
		setup(t)

		blockStateMock.EXPECT().
			GetHeader(gomock.AssignableToTypeOf(common.Hash{})).
			Return(&types.Header{}, nil)

		runtimeMock.EXPECT().
			ParachainHostSessionIndexForChild().
			Return(parachaintypes.SessionIndex(0), errors.New("fail"))

		sessionIndex, ancestors, err := ad.getBlockAncestorsInSameSession(
			common.Hash{0x01},
			leafAncestryLenWithinSession,
		)

		assert.Error(t, err)
		assert.Equal(t, parachaintypes.SessionIndex(0), sessionIndex)
		assert.Empty(t, ancestors)
	})

	t.Run("ancestors_include_genesis", func(t *testing.T) {
		setup(t)

		blockNum := uint(3)

		blockStateMock.EXPECT().
			GetHeader(gomock.AssignableToTypeOf(common.Hash{})).
			MaxTimes(leafAncestryLenWithinSession + 1).
			DoAndReturn(func(hash common.Hash) (*types.Header, error) {
				h := &types.Header{
					Number:     blockNum,
					ParentHash: common.Hash{byte(blockNum)},
				}

				blockNum -= 1
				return h, nil
			})

		runtimeMock.EXPECT().
			ParachainHostSessionIndexForChild().
			MaxTimes(leafAncestryLenWithinSession+1).
			Return(parachaintypes.SessionIndex(5), nil)

		sessionIndex, ancestors, err := ad.getBlockAncestorsInSameSession(
			common.Hash{0x01},
			leafAncestryLenWithinSession,
		)

		assert.NoError(t, err)
		assert.Equal(t, parachaintypes.SessionIndex(5), sessionIndex)
		assert.Len(t, ancestors, 2)
	})

	t.Run("max_number_of_ancestors", func(t *testing.T) {
		setup(t)

		blockNum := uint(10)

		blockStateMock.EXPECT().
			GetHeader(gomock.AssignableToTypeOf(common.Hash{})).
			MaxTimes(leafAncestryLenWithinSession + 1).
			DoAndReturn(func(hash common.Hash) (*types.Header, error) {
				h := &types.Header{
					Number:     blockNum,
					ParentHash: common.Hash{byte(blockNum)},
				}

				blockNum -= 1
				return h, nil
			})

		runtimeMock.EXPECT().
			ParachainHostSessionIndexForChild().
			MaxTimes(leafAncestryLenWithinSession+1).
			// All blocks are in the same session. Should stop at leafAncestryLenWithinSession ancestors.
			Return(parachaintypes.SessionIndex(5), nil)

		sessionIndex, ancestors, err := ad.getBlockAncestorsInSameSession(
			common.Hash{0x01},
			leafAncestryLenWithinSession,
		)

		assert.NoError(t, err)
		assert.Equal(t, parachaintypes.SessionIndex(5), sessionIndex)
		assert.Len(t, ancestors, leafAncestryLenWithinSession)
	})

	t.Run("some_ancestors_in_previous_session", func(t *testing.T) {
		setup(t)

		blockNum := uint(11)

		blockStateMock.EXPECT().
			GetHeader(gomock.AssignableToTypeOf(common.Hash{})).
			MaxTimes(leafAncestryLenWithinSession + 1).
			// Block 10 is the new leaf, block 9 its first parent, 8 its grandparent, etc.
			DoAndReturn(func(hash common.Hash) (*types.Header, error) {
				// Decrement this first so we can check the block number in the mocked call to
				// ParachainHostSessionIndexForChild().
				blockNum -= 1

				h := &types.Header{
					Number:     blockNum,
					ParentHash: common.Hash{byte(blockNum)},
				}

				return h, nil
			})

		runtimeMock.EXPECT().
			ParachainHostSessionIndexForChild().
			MaxTimes(leafAncestryLenWithinSession + 1).
			// Only blocks 10 (the new leaf), 9 and 8 are in the same session. Block 7 should not be included.
			DoAndReturn(func() (parachaintypes.SessionIndex, error) {
				if blockNum <= 7 {
					return parachaintypes.SessionIndex(4), nil
				}
				return parachaintypes.SessionIndex(5), nil
			})

		sessionIndex, ancestors, err := ad.getBlockAncestorsInSameSession(common.Hash{0x01}, 3)

		assert.NoError(t, err)
		assert.Equal(t, parachaintypes.SessionIndex(5), sessionIndex)
		assert.Len(t, ancestors, 2)
	})
}

// This is only one test case that provides decent coverage of ProcessActiveLeavesUpdateSignal().
// Other scenarios are tested in the tests for the functions called inside ProcessActiveLeavesUpdateSignal()
// (e.g. getOccupiedCoresFor(), addCores(), etc.)
func TestProcessActiveLeavesUpdateSignal(t *testing.T) {
	ctrl := gomock.NewController(t)
	netMock := NewMockNetwork(ctrl)
	blockStateMock := NewMockBlockState(ctrl)
	runtimeMock := NewMockInstance(ctrl)
	sessionCacheMock := NewMockSessionCache(ctrl)

	activatedLeaf := common.Hash{0x42}
	deactivatedLeaf := common.Hash{0x43}
	sessionIndex := parachaintypes.SessionIndex(1)
	candidate1 := parachaintypes.CandidateHash{Value: common.Hash{0x01}}
	candidate2 := parachaintypes.CandidateHash{Value: common.Hash{0x02}}

	coreCandidate1 := parachaintypes.OccupiedCore{
		CandidateHash:    candidate1.Value,
		GroupResponsible: parachaintypes.GroupIndex(1),
		CandidateDescriptor: parachaintypes.CandidateDescriptor{
			RelayParent: activatedLeaf,
		},
	}

	coreCandidate2 := parachaintypes.OccupiedCore{
		CandidateHash:    candidate2.Value,
		GroupResponsible: parachaintypes.GroupIndex(2),
		CandidateDescriptor: parachaintypes.CandidateDescriptor{
			RelayParent: activatedLeaf,
		},
	}

	coreState := parachaintypes.CoreState{}
	require.NoError(t, coreState.SetValue(coreCandidate1))

	netMock.EXPECT().RegisterRequestHandler(protocol.ID("req_chunk/2"), gomock.Any())
	netMock.EXPECT().RegisterRequestHandler(protocol.ID("req_pov/1"), gomock.Any())

	runtimeMock.EXPECT().Stop().AnyTimes()
	runtimeMock.EXPECT().ParachainHostSessionIndexForChild().Return(sessionIndex, nil).MaxTimes(1)

	runtimeMock.EXPECT().
		ParachainHostAvailabilityCores().
		Return([]parachaintypes.CoreState{coreState}, nil)

	blockStateMock.EXPECT().
		GetRuntime(gomock.AssignableToTypeOf(common.Hash{})).
		AnyTimes().
		Return(runtimeMock, nil)

	blockStateMock.EXPECT().
		GetHeader(gomock.AssignableToTypeOf(common.Hash{})).
		AnyTimes().
		// Returning block number 0 here means no ancestors of the activated leaf will be considered.
		Return(&types.Header{Number: 0, ParentHash: common.Hash{0xAF}}, nil)

	setUpSessionCacheMock(
		t,
		sessionCacheMock,
		runtimeMock,
		sessionIndex,
		parachaintypes.GroupIndex(0),
	)

	overseerCh := make(chan any)
	ad := NewAvailabilityDistribution(overseerCh, netMock, blockStateMock, sessionCacheMock)
	t.Cleanup(ad.Stop)

	removedTask := newFetchChunkTask(deactivatedLeaf, 42, nil, &coreCandidate2, overseerCh, nil)
	ad.fetchTasks[candidate2] = removedTask

	activeLeavesUpdateSignal := parachaintypes.ActiveLeavesUpdateSignal{
		Activated: &parachaintypes.ActivatedLeaf{
			Hash:   activatedLeaf,
			Number: 42,
		},
		Deactivated: []common.Hash{deactivatedLeaf},
	}

	err := ad.ProcessActiveLeavesUpdateSignal(activeLeavesUpdateSignal)
	require.NoError(t, err)

	// Check that a new task for candidate1 has been created.
	newTask, ok := ad.fetchTasks[candidate1]
	require.True(t, ok)

	_, liveInActivatedLeaf := newTask.liveIn[activatedLeaf]
	require.True(t, liveInActivatedLeaf)

	// Check that the fetch task for candidate2 was removed.
	require.Empty(t, removedTask.liveIn)
	_, ok = ad.fetchTasks[candidate2]
	require.False(t, ok)
}

func TestAddCores(t *testing.T) {
	var (
		ctrl             *gomock.Controller
		netMock          *MockNetwork
		runtimeMock      *MockInstance
		sessionCacheMock *MockSessionCache
		overseerCh       chan any
		ad               *AvailabilityDistribution

		candidateHash = parachaintypes.CandidateHash{Value: common.Hash{0x01}}
		leaf          = common.Hash{0x02}
		sessionIndex  = parachaintypes.SessionIndex(1)
	)

	setup := func(t *testing.T) {
		ctrl = gomock.NewController(t)
		netMock = NewMockNetwork(ctrl)
		runtimeMock = NewMockInstance(ctrl)
		sessionCacheMock = NewMockSessionCache(ctrl)

		netMock.EXPECT().RegisterRequestHandler(protocol.ID("req_chunk/2"), gomock.Any())
		netMock.EXPECT().RegisterRequestHandler(protocol.ID("req_pov/1"), gomock.Any())

		runtimeMock.EXPECT().Stop().MaxTimes(leafAncestryLenWithinSession + 1)

		overseerCh = make(chan any)
		ad = NewAvailabilityDistribution(overseerCh, netMock, nil, sessionCacheMock)

		t.Cleanup(ad.Stop)
	}

	t.Run("skips_own_backing_group", func(t *testing.T) {
		setup(t)

		groupIndex := parachaintypes.GroupIndex(1)
		setUpSessionCacheMock(t, sessionCacheMock, runtimeMock, sessionIndex, groupIndex)

		core := parachaintypes.OccupiedCore{
			GroupResponsible: groupIndex,
			CandidateHash:    candidateHash.Value,
		}

		otherLeaf := common.Hash{0x05}
		// There is a fetch task for this candidate hash but it should not be marked live in the new leaf.
		ad.fetchTasks[candidateHash] = newFetchChunkTask(otherLeaf, 42, nil, &core, overseerCh, nil)

		err := ad.addCores(
			runtimeMock,
			leaf,
			parachaintypes.SessionIndex(1),
			[]*parachaintypes.OccupiedCore{&core},
		)

		require.NoError(t, err)
		require.Contains(t, ad.fetchTasks[candidateHash].liveIn, otherLeaf)
		require.NotContains(t, ad.fetchTasks[candidateHash].liveIn, leaf)
	})

	t.Run("creates_new_task", func(t *testing.T) {
		setup(t)

		ourGroup := parachaintypes.GroupIndex(1)
		setUpSessionCacheMock(t, sessionCacheMock, runtimeMock, sessionIndex, ourGroup)

		core := parachaintypes.OccupiedCore{
			GroupResponsible: parachaintypes.GroupIndex(2),
			CandidateHash:    candidateHash.Value,
		}

		require.Empty(t, ad.fetchTasks, "subsystem should not have any fetch tasks yet")

		err := ad.addCores(
			runtimeMock,
			leaf,
			sessionIndex,
			[]*parachaintypes.OccupiedCore{&core},
		)

		require.NoError(t, err)
		require.Len(t, ad.fetchTasks, 1, "subsystem should have one fetch task")

		task, ok := ad.fetchTasks[candidateHash]
		require.True(t, ok)

		require.Len(t, task.liveIn, 1)
		_, ok = task.liveIn[leaf]
		require.True(t, ok)
	})

	t.Run("updates_existing_task", func(t *testing.T) {
		setup(t)

		ourGroup := parachaintypes.GroupIndex(1)
		setUpSessionCacheMock(t, sessionCacheMock, runtimeMock, sessionIndex, ourGroup)

		core := parachaintypes.OccupiedCore{
			GroupResponsible: parachaintypes.GroupIndex(2),
			CandidateHash:    candidateHash.Value,
		}

		otherLeaf := common.Hash{0x05}
		// There is a fetch task for this candidate hash already, marked live in a different leaf.
		ad.fetchTasks[candidateHash] = newFetchChunkTask(otherLeaf, 42, nil, &core, overseerCh, nil)

		err := ad.addCores(
			runtimeMock,
			leaf,
			sessionIndex,
			[]*parachaintypes.OccupiedCore{&core},
		)

		require.NoError(t, err)
		require.Len(t, ad.fetchTasks, 1)

		task, ok := ad.fetchTasks[candidateHash]
		require.True(t, ok)

		require.Len(t, task.liveIn, 2)
		_, ok = task.liveIn[leaf]
		require.True(t, ok)
	})
}

func setUpSessionCacheMock(
	t *testing.T,
	mock *MockSessionCache,
	runtimeMock *MockInstance,
	sessionIndex parachaintypes.SessionIndex,
	ourGroup parachaintypes.GroupIndex,
) {
	t.Helper()

	validatorGroups := [][]parachaintypes.AuthorityDiscoveryID{
		{
			parachaintypes.AuthorityDiscoveryID{0x01},
			parachaintypes.AuthorityDiscoveryID{0x02},
			parachaintypes.AuthorityDiscoveryID{0x03},
		},
		{
			parachaintypes.AuthorityDiscoveryID{0x04},
			parachaintypes.AuthorityDiscoveryID{0x05},
			parachaintypes.AuthorityDiscoveryID{0x06},
		},
		{
			parachaintypes.AuthorityDiscoveryID{0x07},
			parachaintypes.AuthorityDiscoveryID{0x08},
			parachaintypes.AuthorityDiscoveryID{0x09},
		},
	}

	nodeFeatures, err := parachaintypes.NewBitVec([]bool{false, false, false, false})
	require.NoError(t, err)

	mock.EXPECT().
		GetSessionIndexForChild(gomock.AssignableToTypeOf(common.Hash{}), gomock.AssignableToTypeOf(runtimeMock)).
		Return(sessionIndex, nil).
		AnyTimes()

	mock.EXPECT().
		GetSessionInfo(
			gomock.AssignableToTypeOf(sessionIndex),
			gomock.AssignableToTypeOf(runtimeMock),
		).
		Return(&SessionInfo{
			SessionIndex:    sessionIndex,
			ValidatorGroups: validatorGroups,
			OurIndex:        parachaintypes.ValidatorIndex(2),
			OurGroup:        &ourGroup,
			NodeFeatures:    nodeFeatures,
		}, nil)

	mock.EXPECT().
		ReportBadValidators(gomock.Any(), gomock.Any(), gomock.Any()).
		Times(0)
}

func TestGetOccupiedCoresFor(t *testing.T) {
	relayParent := common.Hash{0x01}

	occupiedCoreSameParent := parachaintypes.OccupiedCore{
		CandidateDescriptor: parachaintypes.CandidateDescriptor{
			RelayParent: relayParent,
		},
	}

	occupiedCoreStateSameParent := parachaintypes.CoreState{}
	require.NoError(t, occupiedCoreStateSameParent.SetValue(occupiedCoreSameParent))

	occupiedCoreDifferentParent := parachaintypes.OccupiedCore{
		CandidateDescriptor: parachaintypes.CandidateDescriptor{
			RelayParent: common.Hash{0x02},
		},
	}

	occupiedCoreStateDifferentParent := parachaintypes.CoreState{}
	require.NoError(t, occupiedCoreStateDifferentParent.SetValue(occupiedCoreDifferentParent))

	scheduledCoreState := parachaintypes.CoreState{}
	require.NoError(t, scheduledCoreState.SetValue(parachaintypes.ScheduledCore{}))

	freeCoreState := parachaintypes.CoreState{}
	require.NoError(t, freeCoreState.SetValue(parachaintypes.Free{}))

	testCases := []struct {
		description   string
		relayParent   common.Hash
		coreStates    []parachaintypes.CoreState
		expectedCores []*parachaintypes.OccupiedCore
	}{
		{
			description: "filters_out_not_occupied_cores",
			relayParent: relayParent,
			coreStates: []parachaintypes.CoreState{
				occupiedCoreStateSameParent,
				freeCoreState,
				scheduledCoreState,
			},
			expectedCores: []*parachaintypes.OccupiedCore{
				&occupiedCoreSameParent,
			},
		},
		{
			description: "filters_out_different_relay_parent",
			relayParent: relayParent,
			coreStates: []parachaintypes.CoreState{
				occupiedCoreStateDifferentParent,
				occupiedCoreStateSameParent,
			},
			expectedCores: []*parachaintypes.OccupiedCore{
				&occupiedCoreSameParent,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			ad := AvailabilityDistribution{}

			cores, err := ad.getOccupiedCoresFor(tc.relayParent, tc.coreStates)

			require.NoError(t, err)
			require.Equal(t, tc.expectedCores, cores)
		})
	}
}

func TestHandleFetchTaskTermination(t *testing.T) {
	testCases := []struct {
		description           string
		fetchTasks            map[parachaintypes.CandidateHash]*fetchChunkTask
		candidateHash         parachaintypes.CandidateHash
		reason                taskTerminationReason
		badValidators         []parachaintypes.AuthorityDiscoveryID
		expectedNumberOfTasks int
	}{
		{
			description:           "no_tasks",
			fetchTasks:            map[parachaintypes.CandidateHash]*fetchChunkTask{},
			candidateHash:         parachaintypes.CandidateHash{Value: common.Hash{0x01}},
			reason:                taskSucceeded{},
			badValidators:         []parachaintypes.AuthorityDiscoveryID{},
			expectedNumberOfTasks: 0,
		},
		{
			description: "matching_task_is_removed",
			fetchTasks: map[parachaintypes.CandidateHash]*fetchChunkTask{
				{Value: common.Hash{0x01}}: new(fetchChunkTask),
			},
			candidateHash:         parachaintypes.CandidateHash{Value: common.Hash{0x01}},
			reason:                taskSucceeded{},
			badValidators:         []parachaintypes.AuthorityDiscoveryID{},
			expectedNumberOfTasks: 0,
		},
		{
			description: "not_matching_task_is_retained",
			fetchTasks: map[parachaintypes.CandidateHash]*fetchChunkTask{
				{Value: common.Hash{0x01}}: new(fetchChunkTask),
			},
			candidateHash:         parachaintypes.CandidateHash{Value: common.Hash{0x02}},
			reason:                taskSucceeded{},
			badValidators:         []parachaintypes.AuthorityDiscoveryID{},
			expectedNumberOfTasks: 1,
		},
		{
			description: "cancelled_matching_task_is_removed",
			fetchTasks: map[parachaintypes.CandidateHash]*fetchChunkTask{
				{Value: common.Hash{0x01}}: new(fetchChunkTask),
			},
			candidateHash:         parachaintypes.CandidateHash{Value: common.Hash{0x01}},
			reason:                taskCancelled{},
			badValidators:         []parachaintypes.AuthorityDiscoveryID{},
			expectedNumberOfTasks: 0,
		},
		{
			description: "failed_matching_task_is_removed",
			fetchTasks: map[parachaintypes.CandidateHash]*fetchChunkTask{
				{Value: common.Hash{0x01}}: new(fetchChunkTask),
			},
			candidateHash:         parachaintypes.CandidateHash{Value: common.Hash{0x01}},
			reason:                taskFailed{},
			badValidators:         []parachaintypes.AuthorityDiscoveryID{},
			expectedNumberOfTasks: 0,
		},
		{
			description: "bad_validators_are_reported",
			fetchTasks: map[parachaintypes.CandidateHash]*fetchChunkTask{
				{Value: common.Hash{0x01}}: new(fetchChunkTask),
			},
			candidateHash: parachaintypes.CandidateHash{Value: common.Hash{0x01}},
			reason: taskSucceeded{
				sessionIndex:  parachaintypes.SessionIndex(1),
				groupIndex:    parachaintypes.GroupIndex(2),
				badValidators: []parachaintypes.AuthorityDiscoveryID{{1}, {2}, {3}},
			},
			expectedNumberOfTasks: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			ad := AvailabilityDistribution{
				fetchTasks: tc.fetchTasks,
			}

			success, ok := tc.reason.(taskSucceeded)
			if ok && len(success.badValidators) > 0 {
				ctrl := gomock.NewController(t)
				sessionCacheMock := NewMockSessionCache(ctrl)
				sessionCacheMock.EXPECT().
					ReportBadValidators(
						success.sessionIndex,
						success.groupIndex,
						success.badValidators,
					)

				ad.sessionCache = sessionCacheMock
			}

			ad.handleFetchTaskTermination(tc.candidateHash, tc.reason)

			require.Len(t, ad.fetchTasks, tc.expectedNumberOfTasks)
		})
	}
}

func TestAvailabilityChunkIndex(t *testing.T) {
	chunkMappingEnabled, err := parachaintypes.NewBitVec([]bool{false, false, true})
	require.NoError(t, err)

	chunkMappingDisabled, err := parachaintypes.NewBitVec([]bool{false, true, false})
	require.NoError(t, err)

	nValidators := uint(20)
	coreIndex := 13
	validatorIndex := parachaintypes.ValidatorIndex(2)

	testCases := []struct {
		description    string
		nodeFeatures   parachaintypes.BitVec
		nValidators    uint
		coreIndex      int
		validatorIndex parachaintypes.ValidatorIndex
		expected       uint32
		expectedErr    string
	}{
		{
			description:    "chunk_mapping_disabled",
			nodeFeatures:   chunkMappingDisabled,
			nValidators:    nValidators,
			coreIndex:      coreIndex,
			validatorIndex: validatorIndex,
			expected:       2, // without availability chunk mapping it should be the validator index
			expectedErr:    "",
		},
		{
			description:    "chunk_mapping_enabled",
			nodeFeatures:   chunkMappingEnabled,
			nValidators:    nValidators,
			coreIndex:      coreIndex,
			validatorIndex: validatorIndex,
			expected:       14,
			expectedErr:    "",
		},
		{
			description:    "too_many_validators",
			nodeFeatures:   chunkMappingEnabled,
			nValidators:    math.MaxUint,
			coreIndex:      coreIndex,
			validatorIndex: validatorIndex,
			expected:       0,
			expectedErr:    "there are too many validators",
		},
		{
			description:    "too_few_validators",
			nodeFeatures:   chunkMappingEnabled,
			nValidators:    1,
			coreIndex:      coreIndex,
			validatorIndex: validatorIndex,
			expected:       0,
			expectedErr:    "expected at least 2 validators",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			actual, err := availabilityChunkIndex(
				tc.nodeFeatures,
				tc.nValidators,
				tc.coreIndex,
				tc.validatorIndex,
			)

			if tc.expectedErr != "" {
				assert.EqualError(t, err, tc.expectedErr)
			} else {
				assert.Equal(t, tc.expected, actual)
			}
		})
	}
}

func TestProcessAvailabilityDistributionMessageFetchPoV(t *testing.T) { //nolint:tparallel
	t.Parallel()

	var (
		ctrl        *gomock.Controller
		netMock     *MockNetwork
		stateMock   *MockBlockState
		runtimeMock *MockInstance
		cacheMock   *MockSessionCache
		overseerCh  chan any
		ad          *AvailabilityDistribution

		relayParent    = common.Hash{0x01}
		validatorIndex = parachaintypes.ValidatorIndex(1)
		authorityID    = parachaintypes.AuthorityDiscoveryID{0x05}

		msg parachaintypes.AvailabilityDistributionMessageFetchPoV
	)

	setup := func(t *testing.T) {
		ctrl = gomock.NewController(t)
		defer ctrl.Finish()

		netMock = NewMockNetwork(ctrl)
		stateMock = NewMockBlockState(ctrl)
		runtimeMock = NewMockInstance(ctrl)
		cacheMock = NewMockSessionCache(ctrl)

		netMock.EXPECT().RegisterRequestHandler(protocol.ID("req_chunk/2"), gomock.Any())
		netMock.EXPECT().RegisterRequestHandler(protocol.ID("req_pov/1"), gomock.Any())

		stateMock.EXPECT().GetRuntime(gomock.Any()).Return(runtimeMock, nil).AnyTimes()

		cacheMock.EXPECT().
			GetAuthorityID(
				gomock.AssignableToTypeOf(validatorIndex),
				gomock.AssignableToTypeOf(relayParent),
				gomock.Any()).
			Return(authorityID, nil).
			AnyTimes()

		overseerCh = make(chan any)
		ad = NewAvailabilityDistribution(overseerCh, netMock, stateMock, cacheMock)
		require.NotNil(t, ad)

		msg = parachaintypes.AvailabilityDistributionMessageFetchPoV{
			RelayParent:   relayParent,
			FromValidator: validatorIndex,
			PovCh:         make(chan parachaintypes.OverseerFuncRes[parachaintypes.PoV]),
		}
	}

	testCases := []struct {
		description         string
		makeRequestResponse func() messages.ReqRespResult
		errorExpected       bool
	}{
		{
			description: "request_fails",
			makeRequestResponse: func() messages.ReqRespResult {
				return messages.ReqRespResult{
					Response: nil,
					Error:    errors.New("network failure"),
				}
			},
			errorExpected: true,
		},
		{
			description: "no_such_pov",
			makeRequestResponse: func() messages.ReqRespResult {
				response := messages.PoVFetchingResponse{}
				require.NoError(t, response.SetValue(parachaintypes.NoSuchPoV{}))

				return messages.ReqRespResult{
					Response: &response,
					Error:    nil,
				}
			},
			errorExpected: true,
		},
		{
			description: "happy_path",
			makeRequestResponse: func() messages.ReqRespResult {
				response := messages.PoVFetchingResponse{}
				require.NoError(t, response.SetValue(parachaintypes.PoV{BlockData: []byte{0x01, 0x02}}))

				return messages.ReqRespResult{
					Response: &response,
					Error:    nil,
				}
			},
			errorExpected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			setup(t)

			wg := sync.WaitGroup{}
			wg.Add(1)
			go func() {
				err := ad.processMessage(msg)
				require.NoError(t, err)
				wg.Done()
			}()

			select {
			case msg := <-overseerCh:
				sendRequests, ok := msg.(messages.SendRequests)
				require.True(t, ok)
				require.Equal(t, 1, len(sendRequests.Requests))

				request := sendRequests.Requests[0]

				_, ok = request.Payload.(*messages.PoVFetchingRequest)
				require.True(t, ok)

				request.Result <- tc.makeRequestResponse()
			case <-time.After(100 * time.Millisecond):
				t.Fatal("timeout")
			}

			select {
			case msg := <-msg.PovCh:
				if tc.errorExpected {
					require.Error(t, msg.Err)
					require.Empty(t, msg.Data)
				} else {
					require.NoError(t, msg.Err)
					require.NotEmpty(t, msg.Data)
				}
			case <-time.After(100 * time.Millisecond):
				t.Fatal("timeout")
			}

			wg.Wait()
		})
	}
}
