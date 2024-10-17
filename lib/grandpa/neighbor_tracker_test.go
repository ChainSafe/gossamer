// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"fmt"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestNeighbourTracker_UpdatePeer(t *testing.T) {
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
			nt.updatePeer(tt.args.p, tt.args.setID, tt.args.round, tt.args.highestFinalized)
			require.Equal(t, tt.expectedState, nt.peerview[tt.args.p])
		})
	}
}

func TestNeighbourTracker_UpdateState(t *testing.T) {
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
			nt.updateState(tt.args.setID, tt.args.round, tt.args.highestFinalized)
			require.Equal(t, nt.currentSetID, tt.args.setID)
			require.Equal(t, nt.currentRound, tt.args.round)
			require.Equal(t, nt.highestFinalized, tt.args.highestFinalized)
		})
	}
}

func TestNeighbourTracker_BroadcastNeighborMsg(t *testing.T) {
	ctrl := gomock.NewController(t)
	// Err path
	mockNetworkErr := NewMockNetwork(ctrl)
	packet := NeighbourPacketV1{
		Round: 5,
		SetID: 5,
	}
	cm, err := packet.ToConsensusMessage()
	require.NoError(t, err)
	mockNetworkErr.EXPECT().SendMessage(peer.ID("error"), cm).Return(fmt.Errorf("test error sending message"))

	grandpaServiceErr := &Service{
		network: mockNetworkErr,
	}
	peerViewErr := make(map[peer.ID]neighborState)
	peerViewErr["error"] = neighborState{
		round: 5,
		setID: 5,
	}

	neighbourTrackerErr := neighborTracker{
		grandpa:      grandpaServiceErr,
		peerview:     peerViewErr,
		currentRound: 5,
		currentSetID: 5,
	}
	err = neighbourTrackerErr.BroadcastNeighborMsg()
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
	neighbourTrackerOk := neighborTracker{
		grandpa:      grandpaService,
		peerview:     peerViewOk,
		currentRound: 5,
		currentSetID: 5,
	}
	err = neighbourTrackerOk.BroadcastNeighborMsg()
	require.NoError(t, err)
}

func TestNeighbourTracker_StartStop_viaFunctionCall(t *testing.T) {
	ctrl := gomock.NewController(t)
	finalizationChan := make(chan *types.FinalisationInfo)
	blockStateMock := NewMockBlockState(ctrl)
	blockStateMock.EXPECT().
		GetFinalisedNotifierChannel().
		Return(finalizationChan)
	blockStateMock.EXPECT().
		FreeFinalisedNotifierChannel(finalizationChan)

	grandpaService := &Service{
		blockState: blockStateMock,
	}
	nt := newNeighborTracker(grandpaService, make(chan neighborData))
	nt.Start()
	nt.Stop()
}

func TestNeighbourTracker_StartStop_viaChannel(t *testing.T) {
	ctrl := gomock.NewController(t)
	finalizationChan := make(chan *types.FinalisationInfo)
	blockStateMock := NewMockBlockState(ctrl)
	blockStateMock.EXPECT().
		GetFinalisedNotifierChannel().
		Return(finalizationChan)

	grandpaService := &Service{
		blockState: blockStateMock,
	}
	nt := newNeighborTracker(grandpaService, make(chan neighborData))
	nt.Start()
	nt.stoppedNeighbor <- struct{}{}
}

func TestNeighbourTracker_UpdatePeer_viaChannel(t *testing.T) {
	ctrl := gomock.NewController(t)
	finalizationChan := make(chan *types.FinalisationInfo)
	blockStateMock := NewMockBlockState(ctrl)
	blockStateMock.EXPECT().
		GetFinalisedNotifierChannel().
		Return(finalizationChan)
	blockStateMock.EXPECT().
		FreeFinalisedNotifierChannel(finalizationChan)

	grandpaService := &Service{
		blockState: blockStateMock,
	}
	neighbourChan := make(chan neighborData)
	nt := newNeighborTracker(grandpaService, neighbourChan)
	nt.Start()

	neighbourChan <- neighborData{
		peer: "testPeer",
		neighborMsg: &NeighbourPacketV1{
			Round:  5,
			SetID:  6,
			Number: 7,
		},
	}

	time.Sleep(100 * time.Millisecond)

	testPeer := nt.getPeer("testPeer")
	require.Equal(t, uint64(5), testPeer.round)
	require.Equal(t, uint64(6), testPeer.setID)
	require.Equal(t, uint32(7), testPeer.highestFinalized)

	nt.Stop()
}
