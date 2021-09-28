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
	// Adding peer1 with incoming slot.
	err := ps.peerState.tryAcceptIncoming(0, peer1)
	require.NoError(t, err)

	// We ban a node by setting its reputation under the threshold.
	action := Action{
		actionCall: ReportPeer,
		setID:      0,
		reputation: newReputationChange(BannedThreshold-1, ""),
		peerID:     peer1,
		peerIds:    nil,
	}

	// We need one for the message to be processed.
	handler.actionQueue <- action
	time.Sleep(time.Millisecond * 100)

	require.Equal(t, len(ps.messageQueue), 1)
	require.Equal(t, Drop, ps.messageQueue[0].messageStatus)

	// Check that an incoming connection from that node gets refused.
	m, err := ps.incoming(0, peer1, 0)
	require.NoError(t, err)
	require.Equal(t, Reject, m.messageStatus)

	// Wait a bit for the node's reputation to go above the threshold.
	time.Sleep(time.Millisecond * 1200)

	// Try again. This time the node should be accepted.
	m, err = ps.incoming(0, peer1, 0)
	require.NoError(t, err)
	require.Equal(t, Accept, m.messageStatus)
}

func TestAddReservedPeers(t *testing.T) {
	handler := newTestPeerSet(t, 0, 2, []peer.ID{bootNode}, []peer.ID{})
	ps := handler.peerSet

	handler.actionQueue <- Action{
		actionCall: AddReservedPeer,
		setID:      0,
		peerID:     reservedPeer,
	}

	handler.actionQueue <- Action{
		actionCall: AddReservedPeer,
		setID:      0,
		peerID:     reservedPeer2,
	}

	time.Sleep(time.Millisecond * 200)

	expectedMsgs := []Message{
		{messageStatus: Connect, setID: 0, peerID: reservedPeer},
		{messageStatus: Connect, setID: 0, peerID: reservedPeer2},
	}

	require.Equal(t, uint32(1), ps.peerState.sets[0].numOut)

	checkMessage(t, ps.messageQueue, expectedMsgs)
}

func TestPeerSetIncoming(t *testing.T) {
	ii2 := uint64(2)
	ii3 := uint64(3)
	ii4 := uint64(3)

	handler := newTestPeerSet(t, 2, 1, []peer.ID{bootNode}, []peer.ID{})
	ps := handler.peerSet

	// Connect message will be added ingoing queue for bootnode.
	require.Equal(t, Connect, ps.messageQueue[0].messageStatus)

	m, err := ps.incoming(0, incoming, ii4)
	require.NoError(t, err)
	require.Equal(t, Accept, m.messageStatus)

	m, err = ps.incoming(0, incoming2, ii2)
	require.NoError(t, err)
	require.Equal(t, Accept, m.messageStatus)

	m, err = ps.incoming(0, incoming3, ii3)
	require.NoError(t, err)
	require.Equal(t, Reject, m.messageStatus)
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
	handler := newTestPeerSet(t, 0, 2, []peer.ID{}, []peer.ID{reservedPeer})

	ps := handler.peerSet

	action1 := Action{
		actionCall: AddToPeerSet,
		setID:      0,
		peerID:     discovered1,
	}

	action2 := Action{
		actionCall: AddToPeerSet,
		setID:      0,
		peerID:     discovered2,
	}

	handler.actionQueue <- action1
	handler.actionQueue <- action1
	handler.actionQueue <- action2

	time.Sleep(200 * time.Millisecond)

	require.Equal(t, Connect, ps.messageQueue[0].messageStatus)
	require.Equal(t, Connect, ps.messageQueue[1].messageStatus)
	require.Equal(t, Connect, ps.messageQueue[2].messageStatus)
}

func TestReAllocAfterBanned(t *testing.T) {
	handler := newTestPeerSet(t, 25, 25, []peer.ID{}, []peer.ID{})

	ps := handler.peerSet
	// messageQueue length before allocate slot.
	mqLen := 1
	// Adding peer1 with incoming slot.
	if ps.peerState.peerStatus(0, peer1) == unknownPeer {
		ps.peerState.discover(0, peer1)
		err := ps.peerState.tryAcceptIncoming(0, peer1)
		require.NoError(t, err)
	}

	// We ban a node by setting its reputation under the threshold.
	action := Action{
		actionCall: ReportPeer,
		setID:      0,
		reputation: newReputationChange(BannedThreshold-1, ""),
		peerID:     peer1,
		peerIds:    nil,
	}
	// We need one for the message to be processed.
	handler.actionQueue <- action
	time.Sleep(time.Millisecond * 100)

	require.Equal(t, len(ps.messageQueue), 1)
	require.Equal(t, Drop, ps.messageQueue[0].messageStatus)

	// Check that an incoming connection from that node gets refused.
	m, err := ps.incoming(0, peer1, 0)
	require.NoError(t, err)
	require.Equal(t, Reject, m.messageStatus)
	require.Equal(t, len(ps.messageQueue), 1)

	for {
		if len(ps.messageQueue) == mqLen {
			continue
		}
		break
	}
	require.Equal(t, Connect, ps.messageQueue[1].messageStatus)
}
