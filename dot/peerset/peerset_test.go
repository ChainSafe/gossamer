package peerset

import (
	"testing"
	"time"

	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/stretchr/testify/require"
)

func TestPeerSetBanned(t *testing.T) {
	ps := newTestPeerSet(t, 25, 25, nil, nil, false)

	require.Equal(t, UnknownPeer, ps.peerState.peer(0, peer1))
	ps.peerState.discover(0, peer1)
	// Adding peer1 with incoming slot.
	err := ps.peerState.tryAcceptIncoming(0, peer1)
	require.NoError(t, err)

	go ps.Psm()
	// We ban a node by setting its reputation under the threshold.
	action := Action{
		actionCall: ReportPeer,
		setId:      0,
		reputation: newReputationChange(bannedThreshold-1, ""),
		peerId:     peer1,
		peerIds:    nil,
	}

	// We need one for the message to be processed.
	ps.actionQueue <- action
	time.Sleep(time.Millisecond * 100)

	require.Equal(t, len(ps.messageQueue), 1)
	require.Equal(t, Drop, ps.messageQueue[0].messageStatus)

	// Check that an incoming connection from that node gets refused.
	ps.incoming(0, peer1, 0)
	require.Equal(t, len(ps.messageQueue), 2)
	require.Equal(t, Reject, ps.messageQueue[1].messageStatus)

	// Wait a bit for the node's reputation to go above the threshold.
	time.Sleep(time.Millisecond * 1200)

	// Try again. This time the node should be accepted.
	ps.incoming(0, peer1, 0)
	require.Equal(t, len(ps.messageQueue), 3)
	require.Equal(t, Accept, ps.messageQueue[2].messageStatus)
}

func TestAddReservedPeers(t *testing.T) {
	ps := newTestPeerSet(t, 0, 2, []peer.ID{bootNode}, []peer.ID{}, false)

	go ps.Psm()

	ps.actionQueue <- Action{
		actionCall: AddReservedPeer,
		setId:      0,
		peerId:     reservedPeer,
	}

	ps.actionQueue <- Action{
		actionCall: AddReservedPeer,
		setId:      0,
		peerId:     reservedPeer2,
	}

	time.Sleep(time.Millisecond * 200)

	expectedMsgs := []Message{
		{messageStatus: Connect, setId: 0, peerId: reservedPeer},
		{messageStatus: Connect, setId: 0, peerId: reservedPeer2},
	}

	require.Equal(t, uint32(1), ps.peerState.sets[0].numOut)

	checkMessage(t, ps.messageQueue, expectedMsgs)
}

func TestPeerSetIncoming(t *testing.T) {
	ii := uint64(1)
	ii2 := uint64(2)
	ii3 := uint64(3)
	ii4 := uint64(3)

	ps := newTestPeerSet(t, 2, 1, []peer.ID{bootNode}, []peer.ID{}, false)

	ps.incoming(0, incoming, ii)
	ps.incoming(0, incoming, ii4)
	ps.incoming(0, incoming2, ii2)
	ps.incoming(0, incoming3, ii3)

	expectedMsgs := []Message{
		{messageStatus: Connect, setId: 0, peerId: bootNode},
		{messageStatus: Accept, IncomingIndex: ii},
		{messageStatus: Accept, IncomingIndex: ii2},
		{messageStatus: Reject, IncomingIndex: ii3},
	}

	checkMessage(t, ps.messageQueue, expectedMsgs)
}

func checkMessage(t *testing.T, actualMsgs []Message, expectedMsgs []Message) {
	for _, msg := range expectedMsgs {
		require.True(t, contains(actualMsgs, msg))
	}
}

func contains(expectedMsgs []Message, actualMsg Message) bool {
	for _, msg := range expectedMsgs {
		if msg == actualMsg {
			return true
		}
	}

	return false
}

// TestPeerSetDiscovered tests the addition of discovered peers to the PeerSet as connected.
func TestPeerSetDiscovered(t *testing.T) {
	ps := newTestPeerSet(t, 0, 2, []peer.ID{}, []peer.ID{reservedPeer}, false)

	go ps.Psm()
	action1 := Action{
		actionCall: AddToPeersSet,
		setId:      0,
		peerId:     discovered1,
	}

	action2 := Action{
		actionCall: AddToPeersSet,
		setId:      0,
		peerId:     discovered2,
	}

	ps.actionQueue <- action1
	ps.actionQueue <- action1
	ps.actionQueue <- action2

	time.Sleep(200 * time.Millisecond)

	require.Equal(t, Connect, ps.messageQueue[0].messageStatus)
	require.Equal(t, Connect, ps.messageQueue[1].messageStatus)
	require.Equal(t, Connect, ps.messageQueue[2].messageStatus)
}
