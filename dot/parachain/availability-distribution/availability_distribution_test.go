// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package availabilitydistribution

import (
	"errors"
	"sync"
	"testing"

	"github.com/ChainSafe/gossamer/dot/types"

	"github.com/ChainSafe/gossamer/dot/network"
	availabilitystore "github.com/ChainSafe/gossamer/dot/parachain/availability-store"
	"github.com/ChainSafe/gossamer/dot/parachain/network-bridge/messages"
	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/libp2p/go-libp2p/core/protocol"

	"github.com/stretchr/testify/assert"
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
		ad = NewAvailabilityDistribution(overseerCh, netMock, blockStateMock)
	}

	t.Run("invalid_request", func(t *testing.T) {
		setup(t)

		response, err := ad.handleChunkFetchingRequest("bob", []byte("0xDECAFBAD"))

		assert.Error(t, err)
		assert.Nil(t, response)
	})

	t.Run("chunk_not_found", func(t *testing.T) {
		setup(t)

		request := messages.ChunkFetchingRequest{
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
			Proof: []byte{0x03, 0x04},
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
			// assert.Equal(t, testChunk.Proof, chunkRes.Proof) // FIXME see #4597
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
		ad = NewAvailabilityDistribution(overseerCh, netMock, blockStateMock)
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
		ad = NewAvailabilityDistribution(overseerCh, netMock, blockStateMock)
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
