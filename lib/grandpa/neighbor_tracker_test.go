// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"fmt"
	"go.uber.org/mock/gomock"
	"testing"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/require"
)

func TestNeighborTracker_UpdatePeer(t *testing.T) {
	initPeerview := map[peer.ID]neighborState{}
	initPeerview["testPeer"] = neighborState{
		setID:            1,
		round:            2,
		highestFinalized: 3,
	}
	type args struct {
		p                peer.ID
		setID            uint64
		round            uint64
		highestFinalized uint32
	}
	tests := []struct {
		name          string
		tracker       *neighborTracker
		args          args
		expectedState neighborState
	}{
		{
			name: "simple_update",
			tracker: &neighborTracker{
				peerview: map[peer.ID]neighborState{},
			},
			args: args{
				p:                "testPeer",
				setID:            1,
				round:            2,
				highestFinalized: 3,
			},
			expectedState: neighborState{
				setID:            1,
				round:            2,
				highestFinalized: 3,
			},
		},
		{
			name: "updating_existing_peer",
			tracker: &neighborTracker{
				peerview: map[peer.ID]neighborState{},
			},
			args: args{
				p:                "testPeer",
				setID:            4,
				round:            5,
				highestFinalized: 6,
			},
			expectedState: neighborState{
				setID:            4,
				round:            5,
				highestFinalized: 6,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nt := tt.tracker
			nt.UpdatePeer(tt.args.p, tt.args.setID, tt.args.round, tt.args.highestFinalized)
			require.Equal(t, tt.expectedState, nt.peerview[tt.args.p])
		})
	}
}

func TestNeighborTracker_UpdateState(t *testing.T) {
	type args struct {
		setID            uint64
		round            uint64
		highestFinalized uint32
	}
	tests := []struct {
		name    string
		tracker *neighborTracker
		args    args
	}{
		{
			name:    "happy_path",
			tracker: &neighborTracker{},
			args: args{
				setID:            1,
				round:            2,
				highestFinalized: 3,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nt := tt.tracker
			nt.UpdateState(tt.args.setID, tt.args.round, tt.args.highestFinalized)
			require.Equal(t, nt.currentSetID, tt.args.setID)
			require.Equal(t, nt.currentRound, tt.args.round)
			require.Equal(t, nt.highestFinalized, tt.args.highestFinalized)
		})
	}
}

func TestNeighborTracker_BroadcastNeighborMsg(t *testing.T) {
	ctrl := gomock.NewController(t)
	// Err path
	mockNetworkErr := NewMockNetwork(ctrl)
	packet := NeighbourPacketV1{
		Round: 5,
		SetID: 5,
	}
	cm, err := packet.ToConsensusMessage()
	mockNetworkErr.EXPECT().SendMessage(peer.ID("error"), cm).Return(fmt.Errorf("test error sending message"))

	grandpaServiceErr := &Service{
		network: mockNetworkErr,
	}
	peerViewErr := make(map[peer.ID]neighborState)
	peerViewErr["error"] = neighborState{
		round: 5,
		setID: 5,
	}

	neighborTrackerErr := neighborTracker{
		grandpa:      grandpaServiceErr,
		peerview:     peerViewErr,
		currentRound: 5,
		currentSetID: 5,
	}
	err = neighborTrackerErr.BroadcastNeighborMsg()
	require.Error(t, err)

	// Happy path
	mockNetworkOk := NewMockNetwork(ctrl)
	mockNetworkOk.EXPECT().SendMessage(peer.ID("equal"), cm).Return(nil)
	mockNetworkOk.EXPECT().SendMessage(peer.ID("ahead"), cm).Return(nil)

	grandpaService := &Service{
		network: mockNetworkOk,
	}

	peerViewOk := make(map[peer.ID]neighborState)
	peerViewOk["lowSet"] = neighborState{
		setID: 1,
	}
	peerViewOk["lowRound"] = neighborState{
		round: 1,
	}
	peerViewOk["equal"] = neighborState{
		round: 5,
		setID: 5,
	}
	peerViewOk["ahead"] = neighborState{
		round: 7,
		setID: 5,
	}
	neighborTrackerOk := neighborTracker{
		grandpa:      grandpaService,
		peerview:     peerViewOk,
		currentRound: 5,
		currentSetID: 5,
	}
	err = neighborTrackerOk.BroadcastNeighborMsg()
	require.NoError(t, err)
}
