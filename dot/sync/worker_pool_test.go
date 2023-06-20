// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package sync

import (
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/golang/mock/gomock"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/require"
)

func TestSyncWorkerPool_useConnectedPeers(t *testing.T) {
	t.Parallel()
	stablePunishmentTime := time.Now().Add(time.Minute * 2)

	cases := map[string]struct {
		setupWorkerPool func(t *testing.T) *syncWorkerPool
		expectedPool    map[peer.ID]*peerSyncWorker
	}{
		"no_connected_peers": {
			setupWorkerPool: func(t *testing.T) *syncWorkerPool {
				ctrl := gomock.NewController(t)
				networkMock := NewMockNetwork(ctrl)
				networkMock.EXPECT().
					AllConnectedPeersID().
					Return([]peer.ID{})

				return newSyncWorkerPool(networkMock)
			},
			expectedPool: make(map[peer.ID]*peerSyncWorker),
		},
		"3_available_peers": {
			setupWorkerPool: func(t *testing.T) *syncWorkerPool {
				ctrl := gomock.NewController(t)
				networkMock := NewMockNetwork(ctrl)
				networkMock.EXPECT().
					AllConnectedPeersID().
					Return([]peer.ID{
						peer.ID("available-1"),
						peer.ID("available-2"),
						peer.ID("available-3"),
					})
				return newSyncWorkerPool(networkMock)
			},
			expectedPool: map[peer.ID]*peerSyncWorker{
				peer.ID("available-1"): {status: available},
				peer.ID("available-2"): {status: available},
				peer.ID("available-3"): {status: available},
			},
		},
		"2_available_peers_1_to_ignore": {
			setupWorkerPool: func(t *testing.T) *syncWorkerPool {
				ctrl := gomock.NewController(t)
				networkMock := NewMockNetwork(ctrl)
				networkMock.EXPECT().
					AllConnectedPeersID().
					Return([]peer.ID{
						peer.ID("available-1"),
						peer.ID("available-2"),
						peer.ID("available-3"),
					})
				workerPool := newSyncWorkerPool(networkMock)
				workerPool.ignorePeers[peer.ID("available-3")] = struct{}{}
				return workerPool
			},
			expectedPool: map[peer.ID]*peerSyncWorker{
				peer.ID("available-1"): {status: available},
				peer.ID("available-2"): {status: available},
			},
		},
		"peer_punishment_not_valid_anymore": {
			setupWorkerPool: func(t *testing.T) *syncWorkerPool {
				ctrl := gomock.NewController(t)
				networkMock := NewMockNetwork(ctrl)
				networkMock.EXPECT().
					AllConnectedPeersID().
					Return([]peer.ID{
						peer.ID("available-1"),
						peer.ID("available-2"),
						peer.ID("available-3"),
					})
				workerPool := newSyncWorkerPool(networkMock)
				workerPool.workers[peer.ID("available-3")] = &peerSyncWorker{
					status:         punished,
					punishmentTime: time.Unix(1000, 0), //arbitrary unix value
				}
				return workerPool
			},
			expectedPool: map[peer.ID]*peerSyncWorker{
				peer.ID("available-1"): {status: available},
				peer.ID("available-2"): {status: available},
				peer.ID("available-3"): {
					status:         available,
					punishmentTime: time.Unix(1000, 0),
				},
			},
		},
		"peer_punishment_still_valid": {
			setupWorkerPool: func(t *testing.T) *syncWorkerPool {
				ctrl := gomock.NewController(t)
				networkMock := NewMockNetwork(ctrl)
				networkMock.EXPECT().
					AllConnectedPeersID().
					Return([]peer.ID{
						peer.ID("available-1"),
						peer.ID("available-2"),
						peer.ID("available-3"),
					})
				workerPool := newSyncWorkerPool(networkMock)
				workerPool.workers[peer.ID("available-3")] = &peerSyncWorker{
					status:         punished,
					punishmentTime: stablePunishmentTime,
				}
				return workerPool
			},
			expectedPool: map[peer.ID]*peerSyncWorker{
				peer.ID("available-1"): {status: available},
				peer.ID("available-2"): {status: available},
				peer.ID("available-3"): {
					status:         punished,
					punishmentTime: stablePunishmentTime,
				},
			},
		},
	}

	for tname, tt := range cases {
		tt := tt
		t.Run(tname, func(t *testing.T) {
			t.Parallel()

			workerPool := tt.setupWorkerPool(t)
			workerPool.useConnectedPeers()

			require.Equal(t, workerPool.workers, tt.expectedPool)
		})
	}
}

func TestSyncWorkerPool_newPeer(t *testing.T) {
	t.Parallel()
	stablePunishmentTime := time.Now().Add(time.Minute * 2)

	cases := map[string]struct {
		peerID          peer.ID
		setupWorkerPool func(t *testing.T) *syncWorkerPool
		expectedPool    map[peer.ID]*peerSyncWorker
	}{
		"very_fist_entry": {
			peerID: peer.ID("peer-1"),
			setupWorkerPool: func(*testing.T) *syncWorkerPool {
				return newSyncWorkerPool(nil)
			},
			expectedPool: map[peer.ID]*peerSyncWorker{
				peer.ID("peer-1"): {status: available},
			},
		},
		"peer_to_ignore": {
			peerID: peer.ID("to-ignore"),
			setupWorkerPool: func(*testing.T) *syncWorkerPool {
				workerPool := newSyncWorkerPool(nil)
				workerPool.ignorePeers[peer.ID("to-ignore")] = struct{}{}
				return workerPool
			},
			expectedPool: map[peer.ID]*peerSyncWorker{},
		},
		"peer_punishment_not_valid_anymore": {
			peerID: peer.ID("free-again"),
			setupWorkerPool: func(*testing.T) *syncWorkerPool {
				workerPool := newSyncWorkerPool(nil)
				workerPool.workers[peer.ID("free-again")] = &peerSyncWorker{
					status:         punished,
					punishmentTime: time.Unix(1000, 0), //arbitrary unix value
				}
				return workerPool
			},
			expectedPool: map[peer.ID]*peerSyncWorker{
				peer.ID("free-again"): {
					status:         available,
					punishmentTime: time.Unix(1000, 0),
				},
			},
		},
		"peer_punishment_still_valid": {
			peerID: peer.ID("peer_punished"),
			setupWorkerPool: func(*testing.T) *syncWorkerPool {

				workerPool := newSyncWorkerPool(nil)
				workerPool.workers[peer.ID("peer_punished")] = &peerSyncWorker{
					status:         punished,
					punishmentTime: stablePunishmentTime,
				}
				return workerPool
			},
			expectedPool: map[peer.ID]*peerSyncWorker{
				peer.ID("peer_punished"): {
					status:         punished,
					punishmentTime: stablePunishmentTime,
				},
			},
		},
	}

	for tname, tt := range cases {
		tt := tt
		t.Run(tname, func(t *testing.T) {
			t.Parallel()

			workerPool := tt.setupWorkerPool(t)
			workerPool.newPeer(tt.peerID)

			require.Equal(t, workerPool.workers, tt.expectedPool)
		})
	}
}

func TestSyncWorkerPool_listenForRequests_submitRequest(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	networkMock := NewMockNetwork(ctrl)
	workerPool := newSyncWorkerPool(networkMock)

	stopCh := make(chan struct{})
	defer close(stopCh)
	go workerPool.listenForRequests(stopCh)

	availablePeer := peer.ID("available-peer")
	workerPool.newPeer(availablePeer)

	blockHash := common.MustHexToHash("0x750646b852a29e5f3668959916a03d6243a3137e91d0cd36870364931030f707")
	blockRequest := network.NewSingleBlockRequestMessage(blockHash, network.BootstrapRequestData)
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
	networkMock.EXPECT().
		DoBlockRequest(availablePeer, blockRequest).
		DoAndReturn(func(any, any) (any, any) {
			time.Sleep(5 * time.Second)
			return mockedBlockResponse, nil
		})

	resultCh := make(chan *syncTaskResult)
	workerPool.submitRequest(blockRequest, resultCh)

	// ensure the task is in the pool and was already
	// assigned to the peer
	time.Sleep(time.Second)

	totalWorkers := workerPool.totalWorkers()
	require.Zero(t, totalWorkers)

	peerSync := workerPool.getPeerByID(availablePeer)
	require.Equal(t, peerSync.status, busy)

	syncTaskResult := <-resultCh
	require.NoError(t, syncTaskResult.err)
	require.Equal(t, syncTaskResult.who, availablePeer)
	require.Equal(t, syncTaskResult.request, blockRequest)
	require.Equal(t, syncTaskResult.response, mockedBlockResponse)
}

func TestSyncWorkerPool_listenForRequests_busyWorkers(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	networkMock := NewMockNetwork(ctrl)
	workerPool := newSyncWorkerPool(networkMock)

	stopCh := make(chan struct{})
	defer close(stopCh)
	go workerPool.listenForRequests(stopCh)

	availablePeer := peer.ID("available-peer")
	workerPool.newPeer(availablePeer)

	firstRequestBlockHash := common.MustHexToHash("0x750646b852a29e5f3668959916a03d6243a3137e91d0cd36870364931030f707")
	firstBlockRequest := network.NewSingleBlockRequestMessage(firstRequestBlockHash, network.BootstrapRequestData)

	secondRequestBlockHash := common.MustHexToHash("0x897646b852a29e5f3668959916a03d6243a3137e91d0cd36870364931030f707")
	secondBlockRequest := network.NewSingleBlockRequestMessage(firstRequestBlockHash, network.BootstrapRequestData)

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
	networkMock.EXPECT().
		DoBlockRequest(availablePeer, firstBlockRequest).
		DoAndReturn(func(any, any) (any, any) {
			time.Sleep(5 * time.Second)
			return firstMockedBlockResponse, nil
		})

	networkMock.EXPECT().
		DoBlockRequest(availablePeer, secondBlockRequest).
		DoAndReturn(func(any, any) (any, any) {
			return secondMockedBlockResponse, nil
		})

	resultCh := make(chan *syncTaskResult)

	workerPool.submitRequests(
		[]*network.BlockRequestMessage{firstBlockRequest, secondBlockRequest}, resultCh)

	// ensure the task is in the pool and was already
	// assigned to the peer
	time.Sleep(time.Second)
	require.Zero(t, workerPool.totalWorkers())

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
