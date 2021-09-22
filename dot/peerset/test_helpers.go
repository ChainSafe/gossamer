package peerset

import (
	"testing"

	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/stretchr/testify/require"
)

var (
	bootNode      = peer.ID("testBootNode")
	reservedPeer  = peer.ID("testReservedPeer")
	reservedPeer2 = peer.ID("testReservedPeer2")
	discovered1   = peer.ID("testDiscovered1")
	discovered2   = peer.ID("testDiscovered2")
	incoming      = peer.ID("testIncoming")
	incoming2     = peer.ID("testIncoming2")
	incoming3     = peer.ID("testIncoming3")

	peer1 = peer.ID("testPeer1")
	peer2 = peer.ID("testPeer2")
)

func newTestPeerSet(t *testing.T, in, out uint32, bootNode, reservedPeer []peer.ID, reservedOnly bool) *PeerSet {
	con := &config{
		sets: []*SetConfig{
			{
				inPeers:       in,
				outPeers:      out,
				bootNodes:     bootNode,
				reservedNodes: reservedPeer,
				reservedOnly:  reservedOnly,
			},
		},
	}

	ps, err := fromConfig(con)
	require.NoError(t, err)

	return ps
}

func newTestPeerState(t *testing.T, maxIn, maxOut uint32) *PeersState {
	state, err := NewPeerState([]*SetConfig{
		{
			inPeers:  maxIn,
			outPeers: maxOut,
		},
	})
	require.NoError(t, err)

	return state
}
