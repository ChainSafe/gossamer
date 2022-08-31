package grandpa

import (
	"sort"
	"testing"

	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/stretchr/testify/require"
)

func TestNewPeerViewTracker(t *testing.T) {
	t.Parallel()

	expected := &peerViewTracker{
		peers: map[peer.ID]view{},
	}

	actual := newPeerViewTracker()
	require.Equal(t, expected, actual)
}

func TestUpdatePeerView(t *testing.T) {
	tests := map[string]struct {
		incoming []struct {
			peerID           peer.ID
			neighbourMessage *NeighbourPacketV1
			wantErr          error
			wantErrString    string
		}
		expectedLen int
	}{
		"successful_peer_neighbor_message": {
			expectedLen: 2,
			incoming: []struct {
				peerID           peer.ID
				neighbourMessage *NeighbourPacketV1
				wantErr          error
				wantErrString    string
			}{
				{
					peerID: peer.ID("peer1"),
					neighbourMessage: &NeighbourPacketV1{
						Round:  1,
						SetID:  0,
						Number: 0,
					},
				},
				{
					peerID: peer.ID("peer2"),
					neighbourMessage: &NeighbourPacketV1{
						Round:  2,
						SetID:  0,
						Number: 10,
					},
				},
				{
					peerID: peer.ID("peer1"),
					neighbourMessage: &NeighbourPacketV1{
						Round:  2,
						SetID:  0,
						Number: 10,
					},
				},
			},
		},

		"invalid_peer_update_view_receiving_lower_round": {
			expectedLen: 1,
			incoming: []struct {
				peerID           peer.ID
				neighbourMessage *NeighbourPacketV1
				wantErr          error
				wantErrString    string
			}{
				{
					peerID: peer.ID("peer1"),
					neighbourMessage: &NeighbourPacketV1{
						Round:  1,
						SetID:  0,
						Number: 0,
					},
				},
				{
					peerID: peer.ID("peer1"),
					neighbourMessage: &NeighbourPacketV1{
						Round:  0,
						SetID:  0,
						Number: 10,
					},
					wantErr:       ErrInvalidRound,
					wantErrString: "invalid round: expecting a round greater or equal to 1 got 0",
				},
			},
		},
	}

	for tname, tt := range tests {
		tt := tt
		t.Run(tname, func(t *testing.T) {
			t.Parallel()

			tracker := newPeerViewTracker()

			for _, incoming := range tt.incoming {
				err := tracker.updatePeerView(incoming.peerID, incoming.neighbourMessage)

				if incoming.wantErr != nil {
					require.Error(t, err)
					require.ErrorIs(t, err, incoming.wantErr)
					require.EqualError(t, err, incoming.wantErrString)
				} else {
					require.NoError(t, err)
				}
			}

			require.Len(t, tracker.peers, tt.expectedLen)
		})
	}
}

func TestGetPeersAtRound(t *testing.T) {
	tests := map[string]struct {
		incoming []struct {
			peerID           peer.ID
			neighbourMessage *NeighbourPacketV1
		}
		fromRound     uint64
		expectedPeers peer.IDSlice
	}{
		"get_2_peers_from_round_2": {
			incoming: []struct {
				peerID           peer.ID
				neighbourMessage *NeighbourPacketV1
			}{
				{
					peerID: peer.ID("peer1"),
					neighbourMessage: &NeighbourPacketV1{
						Round:  1,
						SetID:  0,
						Number: 0,
					},
				},
				{
					peerID: peer.ID("peer2"),
					neighbourMessage: &NeighbourPacketV1{
						Round:  2,
						SetID:  0,
						Number: 10,
					},
				},
				{
					peerID: peer.ID("peer3"),
					neighbourMessage: &NeighbourPacketV1{
						Round:  2,
						SetID:  0,
						Number: 10,
					},
				},
			},
			fromRound:     2,
			expectedPeers: peer.IDSlice{peer.ID("peer2"), peer.ID("peer3")},
		},

		"empty_peer_views_should_return_empty_peer_IDSlice": {
			incoming:      nil,
			fromRound:     1,
			expectedPeers: peer.IDSlice{},
		},
	}

	for tname, tt := range tests {
		tt := tt
		t.Run(tname, func(t *testing.T) {
			t.Parallel()

			tracker := newPeerViewTracker()

			for _, incoming := range tt.incoming {
				err := tracker.updatePeerView(incoming.peerID, incoming.neighbourMessage)
				require.NoError(t, err)
			}

			actual := tracker.getPeersAtRound(tt.fromRound)

			// ensure the same order before compare
			sort.Sort(actual)
			sort.Sort(tt.expectedPeers)

			require.Equal(t, tt.expectedPeers, actual)
			require.Len(t, actual, len(tt.expectedPeers))
		})
	}
}
