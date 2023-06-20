package sync

import (
	"testing"
	"time"

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
