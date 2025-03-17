// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package availabilitydistribution

import (
	"sync"
	"testing"

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

	testCases := []struct {
		description   string
		createRequest func(t *testing.T) (*messages.ChunkFetchingRequest, []byte)
		createChunk   func(t *testing.T) *availabilitystore.ErasureChunk
		handleQuery   func(
			t *testing.T,
			overseerCh chan any,
			request *messages.ChunkFetchingRequest,
			chunk *availabilitystore.ErasureChunk,
		)
		validateResponse func(
			t *testing.T,
			response *messages.ChunkFetchingResponse,
			err error,
			chunk *availabilitystore.ErasureChunk,
		)
	}{
		{
			description: "invalid_request",
			createRequest: func(t *testing.T) (*messages.ChunkFetchingRequest, []byte) {
				return nil, []byte("0xDECAFBAD")
			},
			createChunk: func(t *testing.T) *availabilitystore.ErasureChunk {
				return nil
			},
			handleQuery: func(
				t *testing.T,
				overseerCh chan any,
				request *messages.ChunkFetchingRequest,
				chunk *availabilitystore.ErasureChunk,
			) {
			},
			validateResponse: func(
				t *testing.T,
				response *messages.ChunkFetchingResponse,
				err error,
				chunk *availabilitystore.ErasureChunk,
			) {
				assert.Error(t, err)
				assert.Nil(t, response)
			},
		},
		{
			description: "chunk_not_found",
			createRequest: func(t *testing.T) (*messages.ChunkFetchingRequest, []byte) {
				request := messages.ChunkFetchingRequest{
					CandidateHash: parachaintypes.CandidateHash{Value: common.Hash{0x01}},
					Index:         parachaintypes.ValidatorIndex(0),
				}

				encodedRequest, err := request.Encode()
				assert.NoError(t, err)
				return &request, encodedRequest
			},
			createChunk: func(t *testing.T) *availabilitystore.ErasureChunk {
				return &availabilitystore.ErasureChunk{}
			},
			handleQuery: func(
				t *testing.T,
				overseerCh chan any,
				request *messages.ChunkFetchingRequest,
				chunk *availabilitystore.ErasureChunk,
			) {
				query, ok := (<-overseerCh).(availabilitystore.QueryChunk)
				assert.True(t, ok)
				assert.Equal(t, request.CandidateHash, query.CandidateHash)
				assert.Equal(t, request.Index, query.ValidatorIndex)

				query.Sender <- *chunk
			},
			validateResponse: func(
				t *testing.T,
				response *messages.ChunkFetchingResponse,
				err error,
				chunk *availabilitystore.ErasureChunk,
			) {
				assert.NoError(t, err)
				assert.NotNil(t, response)

				value, err := response.Value()
				assert.NoError(t, err)
				assert.Equal(t, messages.NoSuchChunk{}, value)
			},
		},
		{
			description: "chunk_found",
			createRequest: func(t *testing.T) (*messages.ChunkFetchingRequest, []byte) {
				request := messages.ChunkFetchingRequest{
					CandidateHash: parachaintypes.CandidateHash{Value: common.Hash{0x01}},
					Index:         parachaintypes.ValidatorIndex(0),
				}

				encodedRequest, err := request.Encode()
				assert.NoError(t, err)
				return &request, encodedRequest
			},
			createChunk: func(t *testing.T) *availabilitystore.ErasureChunk {
				return &availabilitystore.ErasureChunk{
					Chunk: []byte{0x01, 0x02},
					Index: 23,
					Proof: []byte{0x03, 0x04},
				}
			},
			handleQuery: func(
				t *testing.T,
				overseerCh chan any,
				request *messages.ChunkFetchingRequest,
				chunk *availabilitystore.ErasureChunk,
			) {
				query, ok := (<-overseerCh).(availabilitystore.QueryChunk)
				assert.True(t, ok)
				assert.Equal(t, request.CandidateHash, query.CandidateHash)
				assert.Equal(t, request.Index, query.ValidatorIndex)

				query.Sender <- *chunk
			},
			validateResponse: func(
				t *testing.T,
				response *messages.ChunkFetchingResponse,
				err error,
				chunk *availabilitystore.ErasureChunk,
			) {
				assert.NoError(t, err)
				assert.NotNil(t, response)

				value, err := response.Value()
				assert.NoError(t, err)

				chunkRes, ok := value.(messages.ChunkResponse)
				assert.True(t, ok)
				assert.NotNil(t, chunkRes)
				assert.Equal(t, chunk.Chunk, chunkRes.Chunk)
				assert.Equal(t, chunk.Index, chunkRes.Index)
				// assert.Equal(t, chunk.Proof, chunkRes.Proof) // FIXME see #4597
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			setup(t)

			request, encodedRequest := tc.createRequest(t)
			chunk := tc.createChunk(t)

			wg := sync.WaitGroup{}
			wg.Add(1)
			go func() {
				response, err := ad.handleChunkFetchingRequest("bob", encodedRequest)

				if response != nil {
					cfResponse, ok := response.(*messages.ChunkFetchingResponse)
					assert.True(t, ok)
					tc.validateResponse(t, cfResponse, err, chunk)
				} else {
					tc.validateResponse(t, nil, err, chunk)
				}

				wg.Done()
			}()

			tc.handleQuery(t, overseerCh, request, chunk)
			wg.Wait()
		})
	}
}
