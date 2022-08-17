package grandpa

import (
	"testing"

	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/stretchr/testify/require"
)

func TestUpdatePeerView(t *testing.T) {
	tests := map[string]struct {
		incoming map[peer.ID]struct {
			neighbourMessage *NeighbourPacketV1
			wantErr          error
			wantErrString    string
		}
		expectedLen int
	}{
		"successful_peer_neighbor_message": {
			incoming: map[peer.ID]struct {
				neighbourMessage *NeighbourPacketV1
				wantErr          error
				wantErrString    string
			}{},
		},
	}

	for tname, tt := range tests {
		tt := tt
		t.Run(tname, func(t *testing.T) {
			tracker := newPeerViewTracker()

			err := tracker.updatePeerView(tt.incomingPeer, tt.neighborMessage)
			if tt.wantErr != nil {
				require.Error(t, err)
				require.ErrorIs(t, err, tt.wantErr)
				require.EqualError(t, err, tt.wantErrString)
			}
		})
	}

}
