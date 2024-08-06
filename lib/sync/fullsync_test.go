package sync

import (
	"container/list"
	"os"
	"testing"

	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/common/variadic"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestFullSyncNextActions(t *testing.T) {
	t.Run("best_block_greater_or_equal_current_target", func(t *testing.T) {
		cfg := &FullSyncConfig{
			StartHeader: types.NewEmptyHeader(),
		}

		fs := NewFullSyncStrategy(cfg)
		task, err := fs.NextActions()
		require.NoError(t, err)
		require.Empty(t, task)
	})

	t.Run("target_block_greater_than_best_block", func(t *testing.T) {
		cfg := &FullSyncConfig{
			StartHeader: types.NewEmptyHeader(),
			NumOfTasks:  2,
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

		// the current target is block 1024 (see the OnBlockAnnounceHandshake)
		// since we cap the request to the max blocks we can retrieve which is 128
		// the we should have 2 requests start from 1 and request 128 and another
		// request starting from 129 and requesting 128
		require.Len(t, task, 2)
		request := task[0].request.(*network.BlockRequestMessage)
		require.Equal(t, uint32(1), request.StartingBlock.Uint32())
		require.Equal(t, uint32(128), *request.Max)

		request = task[1].request.(*network.BlockRequestMessage)
		require.Equal(t, uint32(129), request.StartingBlock.Uint32())
		require.Equal(t, uint32(128), *request.Max)
	})

	t.Run("having_requests_in_the_queue", func(t *testing.T) {
		refTo := func(v uint32) *uint32 {
			return &v
		}

		cases := map[string]struct {
			setupRequestQueue func(*testing.T) *requestsQueue[*network.BlockRequestMessage]
			expectedTasksLen  int
			expectedTasks     []*network.BlockRequestMessage
		}{
			"should_have_one_from_request_queue_and_one_from_target_chasing": {
				setupRequestQueue: func(t *testing.T) *requestsQueue[*network.BlockRequestMessage] {
					request := network.NewAscendingBlockRequests(
						129, 129+128,
						network.BootstrapRequestData)
					require.Len(t, request, 1)

					rq := &requestsQueue[*network.BlockRequestMessage]{queue: list.New()}
					for _, req := range request {
						rq.PushBack(req)
					}
					return rq
				},
				expectedTasksLen: 2,
				expectedTasks: []*network.BlockRequestMessage{
					{
						RequestedData: network.BootstrapRequestData,
						StartingBlock: *variadic.FromUint32(129),
						Direction:     network.Ascending,
						Max:           refTo(128),
					},
					{
						RequestedData: network.BootstrapRequestData,
						StartingBlock: *variadic.FromUint32(1),
						Direction:     network.Ascending,
						Max:           refTo(128),
					},
				},
			},
			// creating a amount of 4 requests, but since we have a max num of
			// request set to 2 (see FullSyncConfig) we should only have 2 tasks
			"should_have_two_tasks": {
				setupRequestQueue: func(t *testing.T) *requestsQueue[*network.BlockRequestMessage] {
					request := network.NewAscendingBlockRequests(
						129, 129+(4*128),
						network.BootstrapRequestData)
					require.Len(t, request, 4)

					rq := &requestsQueue[*network.BlockRequestMessage]{queue: list.New()}
					for _, req := range request {
						rq.PushBack(req)
					}
					return rq
				},
				expectedTasksLen: 2,
				expectedTasks: []*network.BlockRequestMessage{
					{
						RequestedData: network.BootstrapRequestData,
						StartingBlock: *variadic.FromUint32(129),
						Direction:     network.Ascending,
						Max:           refTo(128),
					},
					{
						RequestedData: network.BootstrapRequestData,
						StartingBlock: *variadic.FromUint32(257),
						Direction:     network.Ascending,
						Max:           refTo(128),
					},
				},
			},
		}

		for tname, tt := range cases {
			tt := tt
			t.Run(tname, func(t *testing.T) {
				cfg := &FullSyncConfig{
					StartHeader: types.NewEmptyHeader(),
					NumOfTasks:  2,
				}
				fs := NewFullSyncStrategy(cfg)
				fs.requestQueue = tt.setupRequestQueue(t)

				// introduce a peer and a target
				err := fs.OnBlockAnnounceHandshake(peer.ID("peer-A"), &network.BlockAnnounceHandshake{
					Roles:           1,
					BestBlockNumber: 1024,
					BestBlockHash:   common.BytesToHash([]byte{0x01, 0x02}),
					GenesisHash:     common.BytesToHash([]byte{0x00, 0x01}),
				})
				require.NoError(t, err)

				tasks, err := fs.NextActions()
				require.NoError(t, err)
				require.Len(t, tasks, tt.expectedTasksLen)
				for idx, task := range tasks {
					require.Equal(t, task.request, tt.expectedTasks[idx])
				}
			})
		}
	})
}

func TestFullSyncIsFinished(t *testing.T) {
	fstBlocksRaw, err := os.ReadFile("./test_data/westend_1_10_blocks.out")
	require.NoError(t, err)

	fstTaskBlockResponse := &network.BlockResponseMessage{}
	err = fstTaskBlockResponse.Decode(common.MustHexToBytes(string(fstBlocksRaw)))
	require.NoError(t, err)

	sndBlocksRaw, err := os.ReadFile("./test_data/westend_129_256_blocks.out")
	require.NoError(t, err)

	sndTaskBlockResponse := &network.BlockResponseMessage{}
	err = sndTaskBlockResponse.Decode(common.MustHexToBytes(string(sndBlocksRaw)))
	require.NoError(t, err)

	t.Run("requested_max_but_received_less_blocks", func(t *testing.T) {
		syncTaskResults := []*syncTaskResult{
			// first task
			// 1 -> 10
			{
				who: peer.ID("peerA"),
				request: network.NewBlockRequest(*variadic.FromUint32(1), 128,
					network.BootstrapRequestData, network.Ascending),
				completed: true,
				response:  fstTaskBlockResponse,
			},
			// there is gap from 11 -> 128
			// second task
			// 129 -> 256
			{
				who: peer.ID("peerA"),
				request: network.NewBlockRequest(*variadic.FromUint32(1), 128,
					network.BootstrapRequestData, network.Ascending),
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
			StartHeader: types.NewEmptyHeader(),
			BlockState:  mockBlockState,
		}

		fs := NewFullSyncStrategy(cfg)
		fs.importer = mockImporter

		done, _, _, err := fs.IsFinished(syncTaskResults)
		require.NoError(t, err)
		require.False(t, done)

		require.Len(t, fs.unreadyBlocks.incompleteBlocks, 0)
		require.Len(t, fs.unreadyBlocks.disjointChains, 1)
		require.Equal(t, fs.unreadyBlocks.disjointChains[0], sndTaskBlockResponse.BlockData)

		expectedAncestorRequest := network.NewBlockRequest(
			*variadic.FromHash(sndTaskBlockResponse.BlockData[0].Header.ParentHash),
			network.MaxBlocksInResponse,
			network.BootstrapRequestData, network.Descending)

		message, ok := fs.requestQueue.PopFront()
		require.True(t, ok)
		require.Equal(t, expectedAncestorRequest, message)

		// ancestor search response
		ancestorSearchBlocksRaw, err := os.ReadFile("./test_data/westend_ancestor_blocks.out")
		require.NoError(t, err)

		ancestorSearchResponse := &network.BlockResponseMessage{}
		err = ancestorSearchResponse.Decode(common.MustHexToBytes(string(ancestorSearchBlocksRaw)))
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
		require.Len(t, fs.unreadyBlocks.disjointChains, 0)
	})
}
