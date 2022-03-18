// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package peerset

import (
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/stretchr/testify/require"
)

const testSetID = 0

func Test_Ban_Reject_Accept_Peer(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	processor := NewMockMessageProcessor(ctrl)

	handler := newTestPeerSet(t, 25, 25, nil, nil, false, processor)

	ps := handler.peerSet

	checkPeerStateSetNumIn(t, ps.peerState, testSetID, 0)
	require.Equal(t, unknownPeer, ps.peerState.peerStatus(testSetID, peer1))

	ps.peerState.discover(testSetID, peer1)
	// adding peer1 with incoming slot.
	err := ps.peerState.tryAcceptIncoming(testSetID, peer1)
	require.NoError(t, err)

	checkPeerStateSetNumIn(t, ps.peerState, testSetID, 1)
	require.Equal(t, connectedPeer, ps.peerState.peerStatus(testSetID, peer1))

	// we ban a node by setting its reputation under the threshold.
	rpc := newReputationChange(BannedThresholdValue-1, "")

	// we need one for the message to be processed.
	processor.EXPECT().Process(Message{Status: Drop, setID: testSetID, PeerID: peer1})
	// report peer will disconnect the peer and set the `lastConnected` to time.Now
	handler.ReportPeer(rpc, peer1)
	checkPeerStateSetNumIn(t, ps.peerState, testSetID, 0)
	require.Equal(t, notConnectedPeer, ps.peerState.peerStatus(testSetID, peer1))

	lastDisconectedAt := ps.peerState.nodes[peer1].lastConnected[testSetID]

	// simple wait to ensure the triedToConnectAt will be greater than the lastDisconectedAt
	time.Sleep(100 * time.Millisecond)

	// check that an incoming connection from that node gets refused.
	processor.EXPECT().Process(Message{Status: Reject, setID: testSetID, PeerID: peer1})
	// incoming should update the lastConnected time
	handler.Incoming(0, peer1)

	triedToConnectAt := ps.peerState.nodes[peer1].lastConnected[testSetID]
	require.True(t, lastDisconectedAt.Before(triedToConnectAt))

	// wait a bit for the node's reputation to go above the threshold.
	time.Sleep(time.Millisecond * 1200)

	// try again. This time the node should be accepted.
	processor.EXPECT().Process(Message{Status: Accept, setID: testSetID, PeerID: peer1})
	handler.Incoming(0, peer1)
}

func TestAddReservedPeers(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	processor := NewMockMessageProcessor(ctrl)

	processor.EXPECT().Process(Message{Status: Connect, setID: testSetID, PeerID: bootNode})
	handler := newTestPeerSet(t, 0, 2, []peer.ID{bootNode}, []peer.ID{}, false, processor)
	ps := handler.peerSet

	checkNodePeerExists(t, ps.peerState, bootNode)

	require.Equal(t, connectedPeer, ps.peerState.peerStatus(testSetID, bootNode))
	checkPeerStateSetNumIn(t, ps.peerState, testSetID, 0)
	checkPeerStateSetNumOut(t, ps.peerState, testSetID, 1)

	reservedPeers := []struct {
		peerID peer.ID
	}{
		{
			peerID: reservedPeer,
		},
		{
			peerID: reservedPeer2,
		},
	}

	for _, tt := range reservedPeers {
		processor.EXPECT().Process(Message{Status: Connect, setID: testSetID, PeerID: tt.peerID})
		handler.AddReservedPeer(testSetID, tt.peerID)

		checkReservedNodePeerExists(t, ps, tt.peerID)
		checkPeerIsInNoSlotsNode(t, ps.peerState, tt.peerID, testSetID)

		require.Equal(t, connectedPeer, ps.peerState.peerStatus(testSetID, tt.peerID))
		checkNodePeerMembershipState(t, ps.peerState, tt.peerID, testSetID, outgoing)

		// peers in noSlotNodes maps should not increase the
		// numIn and numOut count
		checkPeerStateSetNumIn(t, ps.peerState, testSetID, 0)
		checkPeerStateSetNumOut(t, ps.peerState, testSetID, 1)
	}
}

func TestPeerSetIncoming(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	processor := NewMockMessageProcessor(ctrl)

	processor.EXPECT().Process(Message{Status: Connect, setID: testSetID, PeerID: bootNode})
	handler := newTestPeerSet(t, 2, 1, []peer.ID{bootNode},
		[]peer.ID{}, false, processor)

	ps := handler.peerSet

	require.Equal(t, connectedPeer, ps.peerState.peerStatus(testSetID, bootNode))
	checkPeerStateSetNumIn(t, ps.peerState, testSetID, 0)
	checkPeerStateSetNumOut(t, ps.peerState, testSetID, 1)

	incomingPeers := []struct {
		pid            peer.ID
		expectedStatus Status
		expectedNumIn  uint32
		// hasFreeIncomingSlot indicates the next slots
		// are only available to noSlotNodes nodes
		hasFreeIncomingSlot bool
	}{
		{
			pid:                 incomingPeer,
			expectedStatus:      Accept,
			expectedNumIn:       1,
			hasFreeIncomingSlot: false,
		},
		{
			pid:                 incoming2,
			expectedStatus:      Accept,
			expectedNumIn:       2,
			hasFreeIncomingSlot: true,
		},
		{
			pid:                 incoming3,
			expectedStatus:      Reject,
			expectedNumIn:       2,
			hasFreeIncomingSlot: true,
		},
	}

	for _, tt := range incomingPeers {
		processor.EXPECT().Process(Message{Status: tt.expectedStatus, setID: testSetID, PeerID: tt.pid})

		// all the incoming peers are unknow before calling the Incoming method
		status := ps.peerState.peerStatus(testSetID, tt.pid)
		require.Equal(t, unknownPeer, status)

		handler.Incoming(testSetID, tt.pid)

		checkNodePeerExists(t, ps.peerState, tt.pid)

		freeSlots := ps.peerState.hasFreeIncomingSlot(testSetID)
		require.Equal(t, tt.hasFreeIncomingSlot, freeSlots)

		checkPeerStateSetNumIn(t, ps.peerState, testSetID, tt.expectedNumIn)
		// incoming peers should not chang the numOut count
		checkPeerStateSetNumOut(t, ps.peerState, testSetID, 1)
	}
}

func TestPeerSetDiscovered(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	processor := NewMockMessageProcessor(ctrl)

	processor.EXPECT().Process(Message{Status: Connect, setID: 0, PeerID: reservedPeer})
	handler := newTestPeerSet(t, 0, 2, []peer.ID{}, []peer.ID{reservedPeer}, false, processor)

	ps := handler.peerSet

	_, isReservedNode := ps.reservedNode[reservedPeer]
	require.True(t, isReservedNode)

	_, isNoSlotNode := ps.peerState.sets[testSetID].noSlotNodes[reservedPeer]
	require.True(t, isNoSlotNode)

	// reserved nodes should not increase the numOut count
	checkPeerStateSetNumOut(t, ps.peerState, testSetID, 0)

	processor.EXPECT().Process(Message{Status: Connect, setID: 0, PeerID: discovered1})
	handler.AddPeer(0, discovered1)
	handler.AddPeer(0, discovered1)

	checkNodePeerExists(t, ps.peerState, discovered1)

	// AddPeer called twice with the same peer ID should not increase the numOut count
	checkPeerStateSetNumOut(t, ps.peerState, testSetID, 1)

	processor.EXPECT().Process(Message{Status: Connect, setID: 0, PeerID: discovered2})
	handler.AddPeer(0, discovered2)

	checkNodePeerExists(t, ps.peerState, discovered2)
	checkPeerStateSetNumOut(t, ps.peerState, testSetID, 2)
}

func TestReAllocAfterBanned(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	processor := NewMockMessageProcessor(ctrl)
	handler := newTestPeerSet(t, 25, 25, []peer.ID{}, []peer.ID{}, false, processor)

	ps := handler.peerSet
	require.Equal(t, unknownPeer, ps.peerState.peerStatus(testSetID, peer1))

	ps.peerState.discover(testSetID, peer1)
	err := ps.peerState.tryAcceptIncoming(testSetID, peer1)
	require.NoError(t, err)

	// accepting the income peer which is not in the reserved peers
	// should increase the numIn count by 1
	checkPeerStateSetNumIn(t, ps.peerState, testSetID, 1)

	// We ban a node by setting its reputation under the threshold.
	processor.EXPECT().Process(Message{Status: Drop, setID: testSetID, PeerID: peer1})
	rep := newReputationChange(BannedThresholdValue-1, "")
	handler.ReportPeer(rep, peer1)

	// banning a incoming peer should decrease the numIn count by 1
	checkPeerStateSetNumIn(t, ps.peerState, testSetID, 0)

	checkNodePeerMembershipState(t, ps.peerState, peer1, testSetID, notConnected)

	n, exists := getNodePeer(t, ps.peerState, peer1)
	require.True(t, exists)

	// when the peer1 was banned we updated its lastConnected field to time.Now()
	lastTimeConnected := n.lastConnected[testSetID]

	// Check that an incoming connection from that node gets refused.
	processor.EXPECT().Process(Message{Status: Reject, setID: testSetID, PeerID: peer1})
	handler.Incoming(testSetID, peer1)

	// when calling Incoming method the peer1 is with status notConnectedPeer
	// so we update its lastConnected field to time.Now() again
	n, exists = getNodePeer(t, ps.peerState, peer1)
	require.True(t, exists)

	currentLastTimeConnected := n.lastConnected[testSetID]
	require.True(t, lastTimeConnected.Before(currentLastTimeConnected))

	// wait a bit for the node's reputation to go above the threshold.
	processor.EXPECT().Process(Message{Status: Connect, setID: testSetID, PeerID: peer1})
	<-time.After(allocTimeDuration + time.Second)

	checkPeerStateSetNumOut(t, ps.peerState, testSetID, 1)
}

func TestRemovePeer(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	processor := NewMockMessageProcessor(ctrl)

	processor.EXPECT().Process(Message{Status: Connect, setID: testSetID, PeerID: "testDiscovered1"})
	processor.EXPECT().Process(Message{Status: Connect, setID: testSetID, PeerID: "testDiscovered2"})
	handler := newTestPeerSet(t, 0, 2, []peer.ID{discovered1, discovered2},
		nil, false, processor)

	ps := handler.peerSet
	require.Equal(t, 2, len(ps.peerState.nodes))
	checkPeerStateSetNumOut(t, ps.peerState, testSetID, 2)

	processor.EXPECT().Process(Message{Status: Drop, setID: 0, PeerID: "testDiscovered1"})
	processor.EXPECT().Process(Message{Status: Drop, setID: 0, PeerID: "testDiscovered2"})
	handler.RemovePeer(testSetID, discovered1, discovered2)

	require.Equal(t, 0, len(ps.peerState.nodes))
	checkPeerStateSetNumOut(t, ps.peerState, testSetID, 0)
}

func TestSetReservePeer(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	processor := NewMockMessageProcessor(ctrl)

	processor.EXPECT().Process(Message{Status: Connect, setID: testSetID, PeerID: reservedPeer})
	processor.EXPECT().Process(Message{Status: Connect, setID: testSetID, PeerID: reservedPeer2})
	handler := newTestPeerSet(t, 0, 2, nil, []peer.ID{reservedPeer, reservedPeer2},
		true, processor)

	ps := handler.peerSet
	require.Equal(t, 2, len(ps.reservedNode))

	newRsrPeerSet := peer.IDSlice{reservedPeer, peer.ID("newRsrPeer")}
	// add newRsrPeer but remove reservedPeer2
	processor.EXPECT().Process(Message{Status: Connect, setID: testSetID, PeerID: "newRsrPeer"})
	processor.EXPECT().Process(Message{Status: Drop, setID: testSetID, PeerID: reservedPeer2})
	handler.SetReservedPeer(testSetID, newRsrPeerSet...)

	require.Equal(t, 2, len(ps.reservedNode))
	for _, p := range newRsrPeerSet {
		require.Contains(t, ps.reservedNode, p)
	}
}

func getNodePeer(t *testing.T, ps *PeersState, pid peer.ID) (node, bool) {
	t.Helper()

	ps.RLock()
	defer ps.RUnlock()

	n, exists := ps.nodes[pid]
	if !exists {
		return node{}, false
	}

	return *n, exists
}

func checkNodePeerMembershipState(t *testing.T, ps *PeersState, pid peer.ID,
	setID int, ms MembershipState) {
	t.Helper()

	ps.RLock()
	defer ps.RUnlock()

	node, exists := ps.nodes[pid]

	require.True(t, exists)
	require.Equal(t, ms, node.state[setID])
}

func checkNodePeerExists(t *testing.T, ps *PeersState, pid peer.ID) {
	t.Helper()

	ps.RLock()
	defer ps.RUnlock()

	_, exists := ps.nodes[pid]
	require.True(t, exists)
}

func checkReservedNodePeerExists(t *testing.T, ps *PeerSet, pid peer.ID) {
	t.Helper()

	ps.Lock()
	defer ps.Unlock()

	_, exists := ps.reservedNode[pid]
	require.True(t, exists)
}

func checkPeerIsInNoSlotsNode(t *testing.T, ps *PeersState, pid peer.ID, setID int) {
	ps.RLock()
	defer ps.RUnlock()

	_, exists := ps.sets[setID].noSlotNodes[pid]
	require.True(t, exists)
}

//nolint:unparam
func checkPeerStateSetNumOut(t *testing.T, ps *PeersState, setID int, expectedNumOut uint32) {
	ps.RLock()
	defer ps.RUnlock()

	gotNumOut := ps.sets[testSetID].numOut
	require.Equal(t, expectedNumOut, gotNumOut)
}

//nolint:unparam
func checkPeerStateSetNumIn(t *testing.T, ps *PeersState, setID int, expectedNumIn uint32) {
	ps.RLock()
	defer ps.RUnlock()

	gotNumIn := ps.sets[testSetID].numIn
	require.Equal(t, expectedNumIn, gotNumIn)
}
