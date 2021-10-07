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
		actionCall: reportPeer,
		setID:      0,
		reputation: newReputationChange(BannedThreshold-1, ""),
		peerID:     peer1,
		peerIds:    nil,
	}

	// We need one for the message to be processed.
	handler.actionQueue <- action
	time.Sleep(time.Millisecond * 100)

	checkMessage(t, <-ps.resultMsgCh, Drop)

	// Check that an incoming connection from that node gets refused.
	err = ps.incoming(0, peer1)
	require.NoError(t, err)
	checkMessage(t, <-ps.resultMsgCh, Reject)

	// Wait a bit for the node's reputation to go above the threshold.
	time.Sleep(time.Millisecond * 1200)

	// Try again. This time the node should be accepted.
	err = ps.incoming(0, peer1)
	require.NoError(t, err)
	checkMessage(t, <-ps.resultMsgCh, Accept)
}

func TestAddReservedPeers(t *testing.T) {
	handler := newTestPeerSet(t, 0, 2, []peer.ID{bootNode}, []peer.ID{})
	ps := handler.peerSet

	handler.actionQueue <- Action{
		actionCall: addReservedPeer,
		setID:      0,
		peerID:     reservedPeer,
	}

	handler.actionQueue <- Action{
		actionCall: addReservedPeer,
		setID:      0,
		peerID:     reservedPeer2,
	}

	time.Sleep(time.Millisecond * 200)

	expectedMsgs := []Message{
		{messageStatus: Connect, setID: 0, peerID: bootNode},
		{messageStatus: Connect, setID: 0, peerID: reservedPeer},
		{messageStatus: Connect, setID: 0, peerID: reservedPeer2},
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

	// Connect message will be added ingoing queue for bootnode.
	checkMessage(t, <-ps.resultMsgCh, Connect)

	err := ps.incoming(0, incomingPeer)
	require.NoError(t, err)
	checkMessage(t, <-ps.resultMsgCh, Accept)

	err = ps.incoming(0, incoming2)
	require.NoError(t, err)
	checkMessage(t, <-ps.resultMsgCh, Accept)

	err = ps.incoming(0, incoming3)
	require.NoError(t, err)
	checkMessage(t, <-ps.resultMsgCh, Reject)
}

// TestPeerSetDiscovered tests the addition of discovered peers to the PeerSet as connected.
func TestPeerSetDiscovered(t *testing.T) {
	handler := newTestPeerSet(t, 0, 2, []peer.ID{}, []peer.ID{reservedPeer})

	ps := handler.peerSet

	action1 := Action{
		actionCall: addToPeerSet,
		setID:      0,
		peerID:     discovered1,
	}

	action2 := Action{
		actionCall: addToPeerSet,
		setID:      0,
		peerID:     discovered2,
	}

	handler.actionQueue <- action1
	handler.actionQueue <- action1
	handler.actionQueue <- action2

	time.Sleep(200 * time.Millisecond)

	require.Equal(t, 3, len(ps.resultMsgCh))
	for i := 0; i < len(ps.resultMsgCh); i++ {
		checkMessage(t, <-ps.resultMsgCh, Connect)
	}
}

func TestReAllocAfterBanned(t *testing.T) {
	handler := newTestPeerSet(t, 25, 25, []peer.ID{}, []peer.ID{})

	ps := handler.peerSet
	// resultMsgCh length before allocate slot.
	mqLen := 1
	// Adding peer1 with incoming slot.
	if ps.peerState.peerStatus(0, peer1) == unknownPeer {
		ps.peerState.discover(0, peer1)
		err := ps.peerState.tryAcceptIncoming(0, peer1)
		require.NoError(t, err)
	}

	// We ban a node by setting its reputation under the threshold.
	action := Action{
		actionCall: reportPeer,
		setID:      0,
		reputation: newReputationChange(BannedThreshold-1, ""),
		peerID:     peer1,
		peerIds:    nil,
	}
	// We need one for the message to be processed.
	handler.actionQueue <- action
	time.Sleep(time.Millisecond * 100)

	require.Equal(t, len(ps.resultMsgCh), 1)
	checkMessage(t, <-ps.resultMsgCh, Drop)

	// Check that an incoming connection from that node gets refused.
	err := ps.incoming(0, peer1)
	require.NoError(t, err)
	checkMessage(t, <-ps.resultMsgCh, Reject)

	for {
		if len(ps.resultMsgCh) == mqLen {
			continue
		}
		break
	}

	checkMessage(t, <-ps.resultMsgCh, Connect)
}
