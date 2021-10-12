package peerset

import (
	"testing"
	"time"

	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/stretchr/testify/require"
)

func TestPeerSetBanned(t *testing.T) {
	handler := newTestPeerSet(t, 25, 25, nil, nil)

	ps := handler.peerSet
	require.Equal(t, unknownPeer, ps.peerState.peerStatus(0, peer1))
	ps.peerState.discover(0, peer1)
	// adding peer1 with incoming slot.
	err := ps.peerState.tryAcceptIncoming(0, peer1)
	require.NoError(t, err)

	// we ban a node by setting its reputation under the threshold.
	rpc := newReputationChange(BannedThresholdValue-1, "")
	// we need one for the message to be processed.
	handler.ReportPeer(rpc, peer1)
	time.Sleep(time.Millisecond * 100)

	checkMessage(t, <-ps.resultMsgCh, Drop)

	// check that an incoming connection from that node gets refused.
	handler.Incoming(0, peer1)
	checkMessage(t, <-ps.resultMsgCh, Reject)

	// wait a bit for the node's reputation to go above the threshold.
	time.Sleep(time.Millisecond * 1200)

	// try again. This time the node should be accepted.
	handler.Incoming(0, peer1)
	require.NoError(t, err)
	checkMessage(t, <-ps.resultMsgCh, Accept)
}

func TestAddReservedPeers(t *testing.T) {
	handler := newTestPeerSet(t, 0, 2, []peer.ID{bootNode}, []peer.ID{})
	ps := handler.peerSet

	handler.AddReservedPeer(0, reservedPeer)
	handler.AddReservedPeer(0, reservedPeer2)

	time.Sleep(time.Millisecond * 200)

	expectedMsgs := []Message{
		{Status: Connect, setID: 0, PeerID: bootNode},
		{Status: Connect, setID: 0, PeerID: reservedPeer},
		{Status: Connect, setID: 0, PeerID: reservedPeer2},
	}

	require.Equal(t, uint32(1), ps.peerState.sets[0].numOut)

	for i := 0; i < len(ps.resultMsgCh); i++ {
		m := <-ps.resultMsgCh
		msg, ok := m.(Message)
		require.True(t, ok)
		require.Equal(t, msg, expectedMsgs[i])
	}
}

func TestPeerSetIncoming(t *testing.T) {
	handler := newTestPeerSet(t, 2, 1, []peer.ID{bootNode}, []peer.ID{})
	ps := handler.peerSet

	// connect message will be added ingoing queue for bootnode.
	checkMessage(t, <-ps.resultMsgCh, Connect)

	handler.Incoming(0, incomingPeer)
	checkMessage(t, <-ps.resultMsgCh, Accept)

	handler.Incoming(0, incoming2)
	checkMessage(t, <-ps.resultMsgCh, Accept)

	handler.Incoming(0, incoming3)
	checkMessage(t, <-ps.resultMsgCh, Reject)
}

func TestPeerSetDiscovered(t *testing.T) {
	handler := newTestPeerSet(t, 0, 2, []peer.ID{}, []peer.ID{reservedPeer})

	ps := handler.peerSet

	handler.AddPeer(0, discovered1)
	handler.AddPeer(0, discovered1)
	handler.AddPeer(0, discovered2)

	time.Sleep(200 * time.Millisecond)

	require.Equal(t, 3, len(ps.resultMsgCh))
	for i := 0; i < len(ps.resultMsgCh); i++ {
		checkMessage(t, <-ps.resultMsgCh, Connect)
	}
}

func TestReAllocAfterBanned(t *testing.T) {
	handler := newTestPeerSet(t, 25, 25, []peer.ID{}, []peer.ID{})

	ps := handler.peerSet
	// adding peer1 with incoming slot.
	if ps.peerState.peerStatus(0, peer1) == unknownPeer {
		ps.peerState.discover(0, peer1)
		err := ps.peerState.tryAcceptIncoming(0, peer1)
		require.NoError(t, err)
	}

	// We ban a node by setting its reputation under the threshold.
	rep := newReputationChange(BannedThresholdValue-1, "")
	// we need one for the message to be processed.
	handler.ReportPeer(rep, peer1)
	time.Sleep(time.Millisecond * 100)

	checkMessage(t, <-ps.resultMsgCh, Drop)

	// Check that an incoming connection from that node gets refused.

	handler.Incoming(0, peer1)
	checkMessage(t, <-ps.resultMsgCh, Reject)

	time.Sleep(time.Millisecond * 100)
	checkMessage(t, <-ps.resultMsgCh, Connect)
}
