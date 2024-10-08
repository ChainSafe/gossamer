package grandpa

import (
	"fmt"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/require"
	"testing"
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
			name: "simple update",
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
			name:    "nil peerview",
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
			name: "updating existing peer",
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
		{
			name: "nil tracker",
			args: args{
				p:                "testPeer",
				setID:            1,
				round:            2,
				highestFinalized: 3,
			},
			expectedErr: fmt.Errorf("neighbor tracker is nil"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nt := tt.tracker
			err := nt.UpdatePeer(tt.args.p, tt.args.setID, tt.args.round, tt.args.highestFinalized)
			require.Equal(t, err, tt.expectedErr)
			if nt != nil {
				require.Equal(t, tt.expectedState, nt.peerview[tt.args.p])
			}
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
		name        string
		tracker     *NeighborTracker
		args        args
		expectedErr error
	}{
		{
			name: "nil tracker",
			args: args{
				setID:            1,
				round:            2,
				highestFinalized: 3,
			},
			expectedErr: fmt.Errorf("neighbor tracker is nil"),
		},
		{
			name:    "happy path",
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
			err := nt.UpdateState(tt.args.setID, tt.args.round, tt.args.highestFinalized)
			require.Equal(t, err, tt.expectedErr)
			if nt != nil {
				require.Equal(t, nt.currentSetID, tt.args.setID)
				require.Equal(t, nt.currentRound, tt.args.round)
				require.Equal(t, nt.highestFinalized, tt.args.highestFinalized)
			}
		})
	}
}
