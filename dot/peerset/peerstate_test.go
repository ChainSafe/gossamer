package peerset

import (
	"testing"

	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/stretchr/testify/require"
)

func TestFullSlotIn(t *testing.T) {
	state := newTestPeerState(t, 1, 1)

	// initially peer1 state will be UnknownPeer.
	require.Equal(t, UnknownPeer, state.peer(0, peer1))
	// discover peer1
	state.discover(0, peer1)
	// peer1 state will change from UnknownPeer to NotConnectedPeer, once we tried to discover it.
	require.Equal(t, NotConnectedPeer, state.peer(0, peer1))
	// try to make peer1 as an incoming connection.
	err := state.tryAcceptIncoming(0, peer1)
	require.NoError(t, err)

	// peer1 is connected
	require.Equal(t, ConnectedPeer, state.peer(0, peer1))

	// initially peer2 state will be UnknownPeer.
	require.Equal(t, UnknownPeer, state.peer(0, peer2))
	// discover peer2
	state.discover(0, peer2)
	// try to make peer2 as an incoming connection.
	err = state.tryAcceptIncoming(0, peer2)
	// peer2 will not be accepted as incoming connection, as we only have one incoming connection slot in peerState.
	require.Error(t, err)
}

func TestNoSlotNodeDoesntOccupySlot(t *testing.T) {
	state := newTestPeerState(t, 1, 1)

	// peer1 will not occupy any slot.
	state.addNoSlotNode(0, peer1)
	// initially peer1 state will be UnknownPeer.
	require.Equal(t, UnknownPeer, state.peer(0, peer1))
	// discover peer1
	state.discover(0, peer1)
	// peer1 will become an incoming connection.
	err := state.tryAcceptIncoming(0, peer1)
	require.NoError(t, err)
	// peer1 is connected
	require.Equal(t, ConnectedPeer, state.peer(0, peer1))

	// peer1 is connected, but the slot is still not occupied.
	require.Equal(t, uint32(0), state.sets[0].numIn)

	// initially peer2 state will be UnknownPeer.
	require.Equal(t, UnknownPeer, state.peer(0, peer2))
	// discover peer2
	state.discover(0, peer2)
	// peer2 state will change from UnknownPeer to NotConnectedPeer, once we tried to discover it.
	require.Equal(t, NotConnectedPeer, state.peer(0, peer2))

	// try to accept peer2 as an incoming connection.
	err = state.tryAcceptIncoming(0, peer2)
	require.NoError(t, err)

	// peer2 is connected
	require.Equal(t, ConnectedPeer, state.peer(0, peer2))

	// peer2 is connected, but the slot is still not occupied.
	require.Equal(t, uint32(1), state.sets[0].numIn)

}

func TestDisconnectingFreeSlot(t *testing.T) {
	state := newTestPeerState(t, 1, 1)

	// initially peer1 state will be UnknownPeer.
	require.Equal(t, UnknownPeer, state.peer(0, peer1))
	// discover peer1
	state.discover(0, peer1)
	err := state.tryAcceptIncoming(0, peer1) // try to make peer1 as an incoming connection.
	require.NoError(t, err)
	// peer1 is connected
	require.Equal(t, ConnectedPeer, state.peer(0, peer1))

	// initially peer2 state will be UnknownPeer.
	require.Equal(t, UnknownPeer, state.peer(0, peer2))
	// discover peer2
	state.discover(0, peer2)
	// peer2 state will change from UnknownPeer to NotConnectedPeer, once we tried to discover it.
	require.Equal(t, NotConnectedPeer, state.peer(0, peer2))
	// try to make peer2 as an incoming connection.
	err = state.tryAcceptIncoming(0, peer2)
	require.Error(t, err) // peer2 will not be accepted as incoming connection, as we only have one incoming connection slot in peerState.

	// disconnect peer1
	err = state.disconnect(0, peer1)
	require.NoError(t, err)

	// peer2 will be accepted as incoming connection, as peer1 is disconnected.
	err = state.tryAcceptIncoming(0, peer2)
	require.NoError(t, err)

}

func TestDisconnectNoSlotDoesntPanic(t *testing.T) {
	state := newTestPeerState(t, 1, 1)

	state.addNoSlotNode(0, peer1)

	require.Equal(t, UnknownPeer, state.peer(0, peer1))

	state.discover(0, peer1)
	err := state.tryOutgoing(0, peer1)
	require.NoError(t, err)

	require.Equal(t, ConnectedPeer, state.peer(0, peer1))

	err = state.disconnect(0, peer1)
	require.NoError(t, err)

	require.Equal(t, NotConnectedPeer, state.peer(0, peer1))
}

func TestHighestNotConnectedPeer(t *testing.T) {
	state, err := NewPeerState([]*SetConfig{
		{
			inPeers:  25,
			outPeers: 25,
		},
	})
	require.NoError(t, err)
	emptyPeerId := peer.ID("")

	require.Equal(t, emptyPeerId, state.highestNotConnectedPeer(0))

	require.Equal(t, UnknownPeer, state.peer(0, peer1))

	state.discover(0, peer1)
	node := state.getNode(peer1)
	require.NotNil(t, node)
	node.setReputation(50)
	state.setNode(peer1, node)

	require.Equal(t, int32(50), state.nodes[peer1].getReputation())

	require.Equal(t, UnknownPeer, state.peer(0, peer2))

	state.discover(0, peer2)
	node = state.getNode(peer2)
	require.NotNil(t, node)
	node.setReputation(25)
	state.setNode(peer2, node)

	// peer1 still has the highest reputation
	require.Equal(t, peer1, state.highestNotConnectedPeer(0))
	require.Equal(t, int32(25), state.nodes[peer2].getReputation())

	require.Equal(t, NotConnectedPeer, state.peer(0, peer2))

	node = state.getNode(peer2)
	require.NotNil(t, node)

	node.setReputation(75)
	state.setNode(peer2, node)

	require.Equal(t, peer2, state.highestNotConnectedPeer(0))
	require.Equal(t, int32(75), state.nodes[peer2].getReputation())

	require.Equal(t, NotConnectedPeer, state.peer(0, peer2))
	err = state.tryAcceptIncoming(0, peer2)
	require.NoError(t, err)

	require.Equal(t, peer1, state.highestNotConnectedPeer(0))

	require.Equal(t, ConnectedPeer, state.peer(0, peer2))
	err = state.disconnect(0, peer2)
	require.NoError(t, err)

	require.Equal(t, NotConnectedPeer, state.peer(0, peer1))
	node = state.getNode(peer1)
	node.setReputation(100)
	state.setNode(peer1, node)

	require.Equal(t, peer1, state.highestNotConnectedPeer(0))
}
