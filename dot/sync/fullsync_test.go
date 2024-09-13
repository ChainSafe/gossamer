// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package sync

import (
	"container/list"
	"testing"

	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/dot/network/messages"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/common/variadic"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"gopkg.in/yaml.v3"

	_ "embed"
)

//go:embed testdata/westend_blocks.yaml
var rawWestendBlocks []byte

type WestendBlocks struct {
	Blocks1To10    string `yaml:"blocks_1_to_10"`
	Blocks129To256 string `yaml:"blocks_129_to_256"`
	Blocks1To128   string `yaml:"blocks_1_to_128"`
}

func TestFullSyncNextActions(t *testing.T) {
	t.Run("best_block_greater_or_equal_current_target", func(t *testing.T) {
		// current target is 0 and best block is 0, then we should
		// get an empty set of tasks

		mockBlockState := NewMockBlockState(gomock.NewController(t))
		mockBlockState.EXPECT().BestBlockHeader().Return(
			types.NewEmptyHeader(), nil)

		cfg := &FullSyncConfig{
			BlockState: mockBlockState,
		}

		fs := NewFullSyncStrategy(cfg)
		task, err := fs.NextActions()
		require.NoError(t, err)
		require.Empty(t, task)
	})

	t.Run("target_block_greater_than_best_block", func(t *testing.T) {
		mockBlockState := NewMockBlockState(gomock.NewController(t))
		mockBlockState.EXPECT().BestBlockHeader().Return(
			types.NewEmptyHeader(), nil)

		cfg := &FullSyncConfig{
			BlockState: mockBlockState,
		}

		fs := NewFullSyncStrategy(cfg)
		err := fs.OnBlockAnnounceHandshake(peer.ID("peer-A"), &network.BlockAnnounceHandshake{
			Roles:           1,
			BestBlockNumber: 1024,
			BestBlockHash:   common.BytesToHash([]byte{0x01, 0x02}),
			GenesisHash:     common.BytesToHash([]byte{0x00, 0x01}),
		})
		require.NoError(t, err)

		task, err := fs.NextActions()
		require.NoError(t, err)

		require.Len(t, task, int(maxRequestsAllowed))
		request := task[0].request.(*messages.BlockRequestMessage)
		require.Equal(t, uint32(1), request.StartingBlock.Uint32())
		require.Equal(t, uint32(128), *request.Max)
	})

	t.Run("having_requests_in_the_queue", func(t *testing.T) {
		refTo := func(v uint32) *uint32 {
			return &v
		}

		cases := map[string]struct {
			setupRequestQueue func(*testing.T) *requestsQueue[*messages.BlockRequestMessage]
			expectedQueueLen  int
			expectedTasks     []*messages.BlockRequestMessage
		}{
			"should_get_all_from_request_queue": {
				setupRequestQueue: func(t *testing.T) *requestsQueue[*messages.BlockRequestMessage] {
					// insert a task to retrieve the block body of a single block
					request := messages.NewAscendingBlockRequests(129, 129, messages.RequestedDataBody)
					require.Len(t, request, 1)

					rq := &requestsQueue[*messages.BlockRequestMessage]{queue: list.New()}
					rq.PushBack(request[0])
					return rq
				},
				expectedQueueLen: 0,
				expectedTasks: []*messages.BlockRequestMessage{
					{
						RequestedData: messages.RequestedDataBody,
						StartingBlock: *variadic.Uint32OrHashFrom(uint32(129)),
						Direction:     messages.Ascending,
						Max:           refTo(1),
					},
					{
						RequestedData: messages.BootstrapRequestData,
						StartingBlock: *variadic.Uint32OrHashFrom(uint32(1)),
						Direction:     messages.Ascending,
						Max:           refTo(127),
					},
				},
			},
			"should_remain_1_in_request_queue": {
				setupRequestQueue: func(t *testing.T) *requestsQueue[*messages.BlockRequestMessage] {
					rq := &requestsQueue[*messages.BlockRequestMessage]{queue: list.New()}

					fstReqByHash := messages.NewBlockRequest(
						*variadic.Uint32OrHashFrom(common.BytesToHash([]byte{0, 1, 1, 2})),
						1, messages.RequestedDataBody, messages.Ascending)
					rq.PushBack(fstReqByHash)

					sndReqByHash := messages.NewBlockRequest(
						*variadic.Uint32OrHashFrom(common.BytesToHash([]byte{1, 2, 2, 4})),
						1, messages.RequestedDataBody, messages.Ascending)
					rq.PushBack(sndReqByHash)

					return rq
				},
				expectedQueueLen: 1,
				expectedTasks: []*messages.BlockRequestMessage{
					{
						RequestedData: messages.RequestedDataBody,
						StartingBlock: *variadic.Uint32OrHashFrom(common.BytesToHash([]byte{0, 1, 1, 2})),
						Direction:     messages.Ascending,
						Max:           refTo(1),
					},
					{
						RequestedData: messages.BootstrapRequestData,
						StartingBlock: *variadic.Uint32OrHashFrom(uint32(1)),
						Direction:     messages.Ascending,
						Max:           refTo(127),
					},
				},
			},
		}

		for tname, tt := range cases {
			tt := tt
			t.Run(tname, func(t *testing.T) {
				fs := NewFullSyncStrategy(&FullSyncConfig{})
				fs.requestQueue = tt.setupRequestQueue(t)
				fs.numOfTasks = 1

				ctrl := gomock.NewController(t)
				mockBlockState := NewMockBlockState(ctrl)
				mockBlockState.EXPECT().
					BestBlockHeader().
					Return(&types.Header{Number: 0}, nil)
				fs.blockState = mockBlockState

				// introduce a peer and a target
				err := fs.OnBlockAnnounceHandshake(peer.ID("peer-A"), &network.BlockAnnounceHandshake{
					Roles:           1,
					BestBlockNumber: 1024,
					BestBlockHash:   common.BytesToHash([]byte{0x01, 0x02}),
					GenesisHash:     common.BytesToHash([]byte{0x00, 0x01}),
				})
				require.NoError(t, err)

				task, err := fs.NextActions()
				require.NoError(t, err)

				require.Equal(t, task[0].request, tt.expectedTasks[0])
				require.Equal(t, fs.requestQueue.Len(), tt.expectedQueueLen)
			})
		}
	})
}

func TestFullSyncIsFinished(t *testing.T) {
	westendBlocks := &WestendBlocks{}
	err := yaml.Unmarshal(rawWestendBlocks, westendBlocks)
	require.NoError(t, err)

	fstTaskBlockResponse := &messages.BlockResponseMessage{}
	err = fstTaskBlockResponse.Decode(common.MustHexToBytes(westendBlocks.Blocks1To10))
	require.NoError(t, err)

	sndTaskBlockResponse := &messages.BlockResponseMessage{}
	err = sndTaskBlockResponse.Decode(common.MustHexToBytes(westendBlocks.Blocks129To256))
	require.NoError(t, err)

	t.Run("requested_max_but_received_less_blocks", func(t *testing.T) {
		syncTaskResults := []*syncTaskResult{
			// first task
			// 1 -> 10
			{
				who: peer.ID("peerA"),
				request: messages.NewBlockRequest(*variadic.Uint32OrHashFrom(1), 128,
					messages.BootstrapRequestData, messages.Ascending),
				completed: true,
				response:  fstTaskBlockResponse,
			},
			// there is gap from 11 -> 128
			// second task
			// 129 -> 256
			{
				who: peer.ID("peerA"),
				request: messages.NewBlockRequest(*variadic.Uint32OrHashFrom(1), 128,
					messages.BootstrapRequestData, messages.Ascending),
				completed: true,
				response:  sndTaskBlockResponse,
			},
		}

		genesisHeader := types.NewHeader(fstTaskBlockResponse.BlockData[0].Header.ParentHash,
			common.Hash{}, common.Hash{}, 0, types.NewDigest())

		ctrl := gomock.NewController(t)
		mockBlockState := NewMockBlockState(ctrl)

		mockBlockState.EXPECT().GetHighestFinalisedHeader().
			Return(genesisHeader, nil).
			Times(3)

		mockBlockState.EXPECT().
			HasHeader(fstTaskBlockResponse.BlockData[0].Header.ParentHash).
			Return(true, nil).
			Times(2)

		mockBlockState.EXPECT().
			HasHeader(sndTaskBlockResponse.BlockData[0].Header.ParentHash).
			Return(false, nil).
			Times(2)

		mockImporter := NewMockImporter(ctrl)
		mockImporter.EXPECT().
			handle(gomock.AssignableToTypeOf(&types.BlockData{}), networkInitialSync).
			Return(true, nil).
			Times(10 + 128 + 128)

		cfg := &FullSyncConfig{
			BlockState: mockBlockState,
		}

		fs := NewFullSyncStrategy(cfg)
		fs.importer = mockImporter

		done, _, _, err := fs.IsFinished(syncTaskResults)
		require.NoError(t, err)
		require.False(t, done)

		require.Len(t, fs.unreadyBlocks.incompleteBlocks, 0)
		require.Len(t, fs.unreadyBlocks.disjointFragments, 1)
		require.Equal(t, fs.unreadyBlocks.disjointFragments[0], sndTaskBlockResponse.BlockData)

		expectedAncestorRequest := messages.NewBlockRequest(
			*variadic.Uint32OrHashFrom(sndTaskBlockResponse.BlockData[0].Header.ParentHash),
			messages.MaxBlocksInResponse,
			messages.BootstrapRequestData, messages.Descending)

		message, ok := fs.requestQueue.PopFront()
		require.True(t, ok)
		require.Equal(t, expectedAncestorRequest, message)

		// ancestor search response
		ancestorSearchResponse := &messages.BlockResponseMessage{}
		err = ancestorSearchResponse.Decode(common.MustHexToBytes(westendBlocks.Blocks1To128))
		require.NoError(t, err)

		syncTaskResults = []*syncTaskResult{
			// ancestor search task
			// 128 -> 1
			{
				who:       peer.ID("peerA"),
				request:   expectedAncestorRequest,
				completed: true,
				response:  ancestorSearchResponse,
			},
		}

		done, _, _, err = fs.IsFinished(syncTaskResults)
		require.NoError(t, err)
		require.False(t, done)

		require.Len(t, fs.unreadyBlocks.incompleteBlocks, 0)
		require.Len(t, fs.unreadyBlocks.disjointFragments, 0)
	})
}

func TestFullSyncBlockAnnounce(t *testing.T) {
	t.Run("announce_a_block_without_any_commom_ancestor", func(t *testing.T) {
		highestFinalizedHeader := &types.Header{
			ParentHash:     common.BytesToHash([]byte{0}),
			StateRoot:      common.BytesToHash([]byte{3, 3, 3, 3}),
			ExtrinsicsRoot: common.BytesToHash([]byte{4, 4, 4, 4}),
			Number:         0,
			Digest:         types.NewDigest(),
		}

		ctrl := gomock.NewController(t)
		mockBlockState := NewMockBlockState(ctrl)
		mockBlockState.EXPECT().IsPaused().Return(false)
		mockBlockState.EXPECT().
			GetHighestFinalisedHeader().
			Return(highestFinalizedHeader, nil)

		mockBlockState.EXPECT().
			HasHeader(gomock.AnyOf(common.Hash{})).
			Return(false, nil)

		fsCfg := &FullSyncConfig{
			BlockState: mockBlockState,
		}

		fs := NewFullSyncStrategy(fsCfg)

		firstPeer := peer.ID("fst-peer")
		firstHandshake := &network.BlockAnnounceHandshake{
			Roles:           1,
			BestBlockNumber: 1024,
			BestBlockHash:   common.BytesToHash([]byte{0, 1, 2}),
			GenesisHash:     common.BytesToHash([]byte{1, 1, 1, 1}),
		}

		err := fs.OnBlockAnnounceHandshake(firstPeer, firstHandshake)
		require.NoError(t, err)

		firstBlockAnnounce := &network.BlockAnnounceMessage{
			ParentHash:     common.BytesToHash([]byte{0, 1, 2}),
			Number:         1024,
			StateRoot:      common.BytesToHash([]byte{3, 3, 3, 3}),
			ExtrinsicsRoot: common.BytesToHash([]byte{4, 4, 4, 4}),
			Digest:         types.NewDigest(),
			BestBlock:      true,
		}

		_, rep, err := fs.OnBlockAnnounce(firstPeer, firstBlockAnnounce)
		require.NoError(t, err)
		require.Nil(t, rep)
	})
}
