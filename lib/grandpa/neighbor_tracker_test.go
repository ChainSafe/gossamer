// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"fmt"
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
		tracker       *NeighborTracker
		args          args
		expectedState neighborState
		expectedErr   error
	}{
		{
			name: "simple_update",
			tracker: &NeighborTracker{
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
			name:    "nil_peerview",
			tracker: &NeighborTracker{},
			args: args{
				p:                "testPeer",
				setID:            1,
				round:            2,
				highestFinalized: 3,
			},
			expectedErr: fmt.Errorf("neighbour tracker has nil peer tracker"),
		},
		{
			name: "updating_existing_peer",
			tracker: &NeighborTracker{
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
			err := nt.UpdatePeer(tt.args.p, tt.args.setID, tt.args.round, tt.args.highestFinalized)
			require.Equal(t, err, tt.expectedErr)
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
		tracker *NeighborTracker
		args    args
	}{
		{
			name:    "happy_path",
			tracker: &NeighborTracker{},
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
