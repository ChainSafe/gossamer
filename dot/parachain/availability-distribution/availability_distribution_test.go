// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package availabilitydistribution

import (
	"sync"
	"testing"

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
