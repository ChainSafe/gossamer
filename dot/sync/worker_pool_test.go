// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package sync

import (
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/common/variadic"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"golang.org/x/exp/maps"
)

func TestSyncWorkerPool_useConnectedPeers(t *testing.T) {
	t.Parallel()
	cases := map[string]struct {
		setupWorkerPool  func(t *testing.T) *syncWorkerPool
		exepectedWorkers []peer.ID
	}{
		"no_connected_peers": {
			setupWorkerPool: func(t *testing.T) *syncWorkerPool {
				ctrl := gomock.NewController(t)
				networkMock := NewMockNetwork(ctrl)
				networkMock.EXPECT().
					AllConnectedPeersIDs().
					Return([]peer.ID{})

				return newSyncWorkerPool(networkMock, nil)
			},
			exepectedWorkers: []peer.ID{},
		},
		"3_available_peers": {
			setupWorkerPool: func(t *testing.T) *syncWorkerPool {
				ctrl := gomock.NewController(t)
				networkMock := NewMockNetwork(ctrl)
				networkMock.EXPECT().
					AllConnectedPeersIDs().
					Return([]peer.ID{
						peer.ID("available-1"),
						peer.ID("available-2"),
						peer.ID("available-3"),
					})
				return newSyncWorkerPool(networkMock, nil)
			},
			exepectedWorkers: []peer.ID{
				peer.ID("available-1"),
				peer.ID("available-2"),
				peer.ID("available-3"),
			},
		},
		"2_available_peers_1_to_ignore": {
			setupWorkerPool: func(t *testing.T) *syncWorkerPool {
				ctrl := gomock.NewController(t)
				networkMock := NewMockNetwork(ctrl)
				networkMock.EXPECT().
					AllConnectedPeersIDs().
					Return([]peer.ID{
						peer.ID("available-1"),
						peer.ID("available-2"),
						peer.ID("available-3"),
					})
				workerPool := newSyncWorkerPool(networkMock, nil)
				workerPool.ignorePeers[peer.ID("available-3")] = struct{}{}
				return workerPool
			},
			exepectedWorkers: []peer.ID{
				peer.ID("available-1"),
				peer.ID("available-2"),
			},
		},
		"peer_already_in_workers_set": {
			setupWorkerPool: func(t *testing.T) *syncWorkerPool {
				ctrl := gomock.NewController(t)
				networkMock := NewMockNetwork(ctrl)
				networkMock.EXPECT().
					AllConnectedPeersIDs().
					Return([]peer.ID{
						peer.ID("available-1"),
						peer.ID("available-2"),
						peer.ID("available-3"),
					})
				workerPool := newSyncWorkerPool(networkMock, nil)
				syncWorker := &syncWorker{
					worker: &worker{},
					queue:  make(chan *syncTask),
				}
				workerPool.workers[peer.ID("available-3")] = syncWorker
				return workerPool
			},
			exepectedWorkers: []peer.ID{
				peer.ID("available-1"),
				peer.ID("available-2"),
				peer.ID("available-3"),
			},
		},
	}

	for tname, tt := range cases {
		tt := tt
		t.Run(tname, func(t *testing.T) {
			t.Parallel()

			workerPool := tt.setupWorkerPool(t)
			workerPool.useConnectedPeers()
			defer workerPool.stop()

			require.ElementsMatch(t,
				maps.Keys(workerPool.workers),
				tt.exepectedWorkers)
		})
	}
}

func TestSyncWorkerPool_listenForRequests_submitRequest(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	networkMock := NewMockNetwork(ctrl)
	requestMakerMock := NewMockRequestMaker(ctrl)
	workerPool := newSyncWorkerPool(networkMock, requestMakerMock)

	availablePeer := peer.ID("available-peer")
	workerPool.newPeer(availablePeer)
	defer workerPool.stop()

	blockHash := common.MustHexToHash("0x750646b852a29e5f3668959916a03d6243a3137e91d0cd36870364931030f707")
	blockRequest := network.NewBlockRequest(*variadic.MustNewUint32OrHash(blockHash),
		1, network.BootstrapRequestData, network.Descending)
	mockedBlockResponse := &network.BlockResponseMessage{
		BlockData: []*types.BlockData{
			{
				Hash: blockHash,
				Header: &types.Header{
					ParentHash: common.
						MustHexToHash("0x5895897f12e1a670609929433ac7a69dcae90e0cc2d9c32c0dce0e2a5e5e614e"),
				},
			},
		},
	}

	// introduce a timeout of 5s then we can test the
	// peer status change to busy
	requestMakerMock.EXPECT().
		Do(availablePeer, blockRequest, &network.BlockResponseMessage{}).
		DoAndReturn(func(_, _, response any) any {
			responsePtr := response.(*network.BlockResponseMessage)
			*responsePtr = *mockedBlockResponse
			return nil
		})

	resultCh := make(chan *syncTaskResult)
	workerPool.submitRequest(blockRequest, nil, resultCh)

	syncTaskResult := <-resultCh
	require.NoError(t, syncTaskResult.err)
	require.Equal(t, syncTaskResult.who, availablePeer)
	require.Equal(t, syncTaskResult.request, blockRequest)
	require.Equal(t, syncTaskResult.response, mockedBlockResponse)

}

func TestSyncWorkerPool_singleWorker_multipleRequests(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	networkMock := NewMockNetwork(ctrl)
	requestMakerMock := NewMockRequestMaker(ctrl)
	workerPool := newSyncWorkerPool(networkMock, requestMakerMock)
	defer workerPool.stop()

	availablePeer := peer.ID("available-peer")
	workerPool.newPeer(availablePeer)

	firstRequestBlockHash := common.MustHexToHash("0x750646b852a29e5f3668959916a03d6243a3137e91d0cd36870364931030f707")
	firstBlockRequest := network.NewBlockRequest(*variadic.MustNewUint32OrHash(firstRequestBlockHash),
		1, network.BootstrapRequestData, network.Descending)

	secondRequestBlockHash := common.MustHexToHash("0x897646b852a29e5f3668959916a03d6243a3137e91d0cd36870364931030f707")
	secondBlockRequest := network.NewBlockRequest(*variadic.MustNewUint32OrHash(firstRequestBlockHash),
		1, network.BootstrapRequestData, network.Descending)

	firstMockedBlockResponse := &network.BlockResponseMessage{
		BlockData: []*types.BlockData{
			{
				Hash: firstRequestBlockHash,
				Header: &types.Header{
					ParentHash: common.
						MustHexToHash("0x5895897f12e1a670609929433ac7a69dcae90e0cc2d9c32c0dce0e2a5e5e614e"),
				},
			},
		},
	}

	secondMockedBlockResponse := &network.BlockResponseMessage{
		BlockData: []*types.BlockData{
			{
				Hash: secondRequestBlockHash,
				Header: &types.Header{
					ParentHash: common.
						MustHexToHash("0x8965897f12e1a670609929433ac7a69dcae90e0cc2d9c32c0dce0e2a5e5e614e"),
				},
			},
		},
	}

	// introduce a timeout of 5s then we can test the
	// then we can simulate a busy peer
	requestMakerMock.EXPECT().
		Do(availablePeer, firstBlockRequest, &network.BlockResponseMessage{}).
		DoAndReturn(func(_, _, response any) any {
			time.Sleep(5 * time.Second)
			responsePtr := response.(*network.BlockResponseMessage)
			*responsePtr = *firstMockedBlockResponse
			return nil
		})

	requestMakerMock.EXPECT().
		Do(availablePeer, firstBlockRequest, &network.BlockResponseMessage{}).
		DoAndReturn(func(_, _, response any) any {
			responsePtr := response.(*network.BlockResponseMessage)
			*responsePtr = *secondMockedBlockResponse
			return nil
		})

	resultCh := workerPool.submitRequests(
		[]*network.BlockRequestMessage{firstBlockRequest, secondBlockRequest})

	syncTaskResult := <-resultCh
	require.NoError(t, syncTaskResult.err)
	require.Equal(t, syncTaskResult.who, availablePeer)
	require.Equal(t, syncTaskResult.request, firstBlockRequest)
	require.Equal(t, syncTaskResult.response, firstMockedBlockResponse)

	syncTaskResult = <-resultCh
	require.NoError(t, syncTaskResult.err)
	require.Equal(t, syncTaskResult.who, availablePeer)
	require.Equal(t, syncTaskResult.request, secondBlockRequest)
	require.Equal(t, syncTaskResult.response, secondMockedBlockResponse)

	require.Equal(t, uint(1), workerPool.totalWorkers())
}
