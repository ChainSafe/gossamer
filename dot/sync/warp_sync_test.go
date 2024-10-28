// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package sync

import (
	"testing"

	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/dot/network/messages"
	"github.com/ChainSafe/gossamer/dot/peerset"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestWarpSyncBlockAnnounce(t *testing.T) {
	peer := peer.ID("peer")

	t.Run("successful_block_announce", func(t *testing.T) {
		peersView := NewPeerViewSet()

		strategy := NewWarpSyncStrategy(&WarpSyncConfig{
			Peers: peersView,
		})

		blockAnnounce := &network.BlockAnnounceMessage{
			ParentHash:     common.BytesToHash([]byte{0, 1, 2}),
			Number:         1024,
			StateRoot:      common.BytesToHash([]byte{3, 3, 3, 3}),
			ExtrinsicsRoot: common.BytesToHash([]byte{4, 4, 4, 4}),
			Digest:         types.NewDigest(),
			BestBlock:      true,
		}

		expectedRepChange := &Change{
			who: peer,
			rep: peerset.ReputationChange{
				Value:  peerset.GossipSuccessValue,
				Reason: peerset.GossipSuccessReason,
			},
		}

		rep, err := strategy.OnBlockAnnounce(peer, blockAnnounce)
		require.NoError(t, err)
		require.NotNil(t, rep)
		require.Equal(t, expectedRepChange, rep)
		require.Equal(t, blockAnnounce.Number, uint(peersView.getTarget()))
	})

	t.Run("successful_block_announce_handshake", func(t *testing.T) {
		peersView := NewPeerViewSet()

		strategy := NewWarpSyncStrategy(&WarpSyncConfig{
			Peers: peersView,
		})

		handshake := &network.BlockAnnounceHandshake{
			Roles:           1,
			BestBlockNumber: 17,
			BestBlockHash:   common.BytesToHash([]byte{0, 1, 2}),
			GenesisHash:     common.BytesToHash([]byte{1, 1, 1, 1}),
		}
		err := strategy.OnBlockAnnounceHandshake(peer, handshake)
		require.NoError(t, err)
		require.Equal(t, handshake.BestBlockNumber, peersView.getTarget())
	})

	t.Run("successful_block_announce", func(t *testing.T) {
		peersView := NewPeerViewSet()

		blockAnnounce := &network.BlockAnnounceMessage{
			ParentHash:     common.BytesToHash([]byte{0, 1, 2}),
			Number:         1024,
			StateRoot:      common.BytesToHash([]byte{3, 3, 3, 3}),
			ExtrinsicsRoot: common.BytesToHash([]byte{4, 4, 4, 4}),
			Digest:         types.NewDigest(),
			BestBlock:      true,
		}

		blockAnnounceHash, err := blockAnnounce.Hash()
		require.NoError(t, err)

		strategy := NewWarpSyncStrategy(&WarpSyncConfig{
			Peers:     peersView,
			BadBlocks: []string{blockAnnounceHash.String()},
		})

		expectedRepChange := &Change{
			who: peer,
			rep: peerset.ReputationChange{
				Value:  peerset.BadBlockAnnouncementValue,
				Reason: peerset.BadBlockAnnouncementReason,
			},
		}

		rep, err := strategy.OnBlockAnnounce(peer, blockAnnounce)
		require.NotNil(t, err)
		require.ErrorIs(t, err, errBadBlockReceived)
		require.NotNil(t, rep)
		require.Equal(t, expectedRepChange, rep)
		require.Equal(t, 0, int(peersView.getTarget()))
	})
}

func TestWarpSyncNextActions(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockBlockState := NewMockBlockState(ctrl)

	genesisHeader := &types.Header{
		Number: 1,
	}
	mockBlockState.EXPECT().GetHighestFinalisedHeader().Return(genesisHeader, nil).AnyTimes()

	tc := map[string]struct {
		phase                WarpSyncPhase
		lastBlock            *types.Header
		expectedRequestType  interface{}
		expectedResponseType interface{}
	}{
		"warp_sync_phase": {
			phase:                WarpProof,
			expectedRequestType:  &messages.WarpProofRequest{},
			expectedResponseType: &messages.WarpSyncProof{},
		},
		"target_block_phase": {
			phase:                TargetBlock,
			expectedRequestType:  &messages.BlockRequestMessage{},
			expectedResponseType: &messages.BlockResponseMessage{},
		},
	}

	for name, c := range tc {
		t.Run(name, func(t *testing.T) {
			strategy := NewWarpSyncStrategy(&WarpSyncConfig{
				BlockState: mockBlockState,
			})

			strategy.phase = c.phase

			tasks, err := strategy.NextActions()
			require.NoError(t, err)
			require.Equal(t, 1, len(tasks), "expected 1 task")

			task := tasks[0]
			require.IsType(t, c.expectedRequestType, task.request)
			require.IsType(t, c.expectedResponseType, task.response)
		})
	}
}
