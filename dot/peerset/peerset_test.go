// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package peerset

import (
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/require"
)

func TestBanRejectAcceptPeer(t *testing.T) {
	const testSetID = 0

	t.Parallel()

	handler := newTestPeerSet(t, 25, 25, nil, nil, false)

	ps := handler.peerSet

	checkPeerStateSetNumIn(t, ps.peerState, testSetID, 0)
	peer1Status := ps.peerState.peerStatus(testSetID, peer1)
	require.Equal(t, unknownPeer, peer1Status)

	ps.peerState.insertPeer(testSetID, peer1)
	// adding peer1 with incoming slot.
	err := ps.peerState.tryAcceptIncoming(testSetID, peer1)
	require.NoError(t, err)

	checkPeerStateSetNumIn(t, ps.peerState, testSetID, 1)
	peer1Status = ps.peerState.peerStatus(testSetID, peer1)
	require.Equal(t, connectedPeer, peer1Status)

	// we ban a node by setting its reputation under the threshold.
	rpc := newReputationChange(BannedThresholdValue-1, "")

	// we need one for the message to be processed.
	// report peer will disconnect the peer and set the `lastConnected` to time.Now
	handler.ReportPeer(rpc, peer1)

	checkMessageStatus(t, <-ps.resultMsgCh, Drop)

	checkPeerStateSetNumIn(t, ps.peerState, testSetID, 0)
	peer1Status = ps.peerState.peerStatus(testSetID, peer1)
	require.Equal(t, notConnectedPeer, peer1Status)
	lastDisconectedAt := ps.peerState.nodes[peer1].lastConnected[testSetID]

	// check that an incoming connection from that node gets refused.
	// incoming should update the lastConnected time
	handler.Incoming(0, peer1)
	checkMessageStatus(t, <-ps.resultMsgCh, Reject)

	triedToConnectAt := ps.peerState.nodes[peer1].lastConnected[testSetID]
	require.True(t, lastDisconectedAt.Before(triedToConnectAt))

	// wait a bit for the node's reputation to go above the threshold.
	time.Sleep(time.Millisecond * 1200)

	// try again. This time the node should be accepted.
	handler.Incoming(0, peer1)
	checkMessageStatus(t, <-ps.resultMsgCh, Accept)
}

func TestAddReservedPeers(t *testing.T) {
	const testSetID = 0

	t.Parallel()
	handler := newTestPeerSet(t, 0, 2, []peer.ID{bootNode}, []peer.ID{}, false)
	ps := handler.peerSet

	checkNodePeerExists(t, ps.peerState, bootNode)

	require.Equal(t, connectedPeer, ps.peerState.peerStatus(testSetID, bootNode))
	checkPeerStateSetNumIn(t, ps.peerState, testSetID, 0)
	checkPeerStateSetNumOut(t, ps.peerState, testSetID, 1)

	reservedPeers := peer.IDSlice{reservedPeer, reservedPeer2}

	for _, peerID := range reservedPeers {
		handler.AddReservedPeer(testSetID, peerID)
		time.Sleep(time.Millisecond * 100)

		checkReservedNodePeerExists(t, ps, peerID)
		checkPeerIsInNoSlotsNode(t, ps.peerState, peerID, testSetID)

		require.Equal(t, connectedPeer, ps.peerState.peerStatus(testSetID, peerID))
		checkNodePeerMembershipState(t, ps.peerState, peerID, testSetID, outgoing)

		// peers in noSlotNodes maps should not increase the
		// numIn and numOut count
		checkPeerStateSetNumIn(t, ps.peerState, testSetID, 0)
		checkPeerStateSetNumOut(t, ps.peerState, testSetID, 1)
	}

	expectedMsgs := []Message{
		{Status: Connect, setID: 0, PeerID: bootNode},
		{Status: Connect, setID: 0, PeerID: reservedPeer},
		{Status: Connect, setID: 0, PeerID: reservedPeer2},
	}

	require.Equal(t, 3, len(ps.resultMsgCh))

	for i := 0; ; i++ {
		if len(ps.resultMsgCh) == 0 {
			break
		}
		msg := <-ps.resultMsgCh
		require.Equal(t, expectedMsgs[i], msg)
	}
}

func TestPeerSetIncoming(t *testing.T) {
	const testSetID = 0

	t.Parallel()
	handler := newTestPeerSet(t, 2, 1, []peer.ID{bootNode}, []peer.ID{}, false)

	ps := handler.peerSet
	checkMessageStatus(t, <-ps.resultMsgCh, Connect)

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

		// all the incoming peers are unknow before calling the Incoming method
		status := ps.peerState.peerStatus(testSetID, tt.pid)
		require.Equal(t, unknownPeer, status)

		handler.Incoming(testSetID, tt.pid)
		time.Sleep(time.Millisecond * 100)

		checkNodePeerExists(t, ps.peerState, tt.pid)

		freeSlots := ps.peerState.hasFreeIncomingSlot(testSetID)
		require.Equal(t, tt.hasFreeIncomingSlot, freeSlots)

		checkPeerStateSetNumIn(t, ps.peerState, testSetID, tt.expectedNumIn)
		// incoming peers should not chang the numOut count
		checkPeerStateSetNumOut(t, ps.peerState, testSetID, 1)

		checkMessageStatus(t, <-ps.resultMsgCh, tt.expectedStatus)
	}
}

func TestPeerSetDiscovered(t *testing.T) {
	const testSetID = 0

	t.Parallel()
	handler := newTestPeerSet(t, 0, 2, []peer.ID{}, []peer.ID{reservedPeer}, false)

	ps := handler.peerSet

	checkReservedNodePeerExists(t, ps, reservedPeer)

	_, isNoSlotNode := ps.peerState.sets[testSetID].noSlotNodes[reservedPeer]
	require.True(t, isNoSlotNode)

	// reserved nodes should not increase the numOut count
	checkPeerStateSetNumOut(t, ps.peerState, testSetID, 0)

	handler.AddPeer(0, discovered1)
	handler.AddPeer(0, discovered1)
	handler.AddPeer(0, discovered2)

	time.Sleep(200 * time.Millisecond)

	checkNodePeerExists(t, ps.peerState, discovered1)
	checkNodePeerExists(t, ps.peerState, discovered2)
	// AddPeer called twice with the same peer ID should not increase the numOut count
	checkPeerStateSetNumOut(t, ps.peerState, testSetID, 2)

	require.Equal(t, 3, len(ps.resultMsgCh))
	for len(ps.resultMsgCh) == 0 {
		checkMessageStatus(t, <-ps.resultMsgCh, Connect)
	}
}

func TestReAllocAfterBanned(t *testing.T) {
	const testSetID = 0

	t.Parallel()
	handler := newTestPeerSet(t, 25, 25, []peer.ID{}, []peer.ID{}, false)

	ps := handler.peerSet
	peer1Status := ps.peerState.peerStatus(testSetID, peer1)
	require.Equal(t, unknownPeer, peer1Status)

	ps.peerState.insertPeer(testSetID, peer1)
	err := ps.peerState.tryAcceptIncoming(testSetID, peer1)
	require.NoError(t, err)

	// accepting the income peer which is not in the reserved peers
	// should increase the numIn count by 1
	checkPeerStateSetNumIn(t, ps.peerState, testSetID, 1)

	// We ban a node by setting its reputation under the threshold.
	rep := newReputationChange(BannedThresholdValue-1, "")
	handler.ReportPeer(rep, peer1)

	time.Sleep(time.Millisecond * 100)
	checkMessageStatus(t, <-ps.resultMsgCh, Drop)

	// banning a incoming peer should decrease the numIn count by 1
	checkPeerStateSetNumIn(t, ps.peerState, testSetID, 0)
	checkNodePeerMembershipState(t, ps.peerState, peer1, testSetID, notConnected)

	n, exists := getNodePeer(ps.peerState, peer1)
	require.True(t, exists)

	// when the peer1 was banned we updated its lastConnected field to time.Now()
	lastTimeConnected := n.lastConnected[testSetID]

	// Check that an incoming connection from that node gets refused.
	handler.Incoming(testSetID, peer1)
	checkMessageStatus(t, <-ps.resultMsgCh, Reject)

	// when calling Incoming method the peer1 is with status notConnectedPeer
	// so we update its lastConnected field to time.Now() again
	n, exists = getNodePeer(ps.peerState, peer1)
	require.True(t, exists)

	currentLastTimeConnected := n.lastConnected[testSetID]
	require.True(t, lastTimeConnected.Before(currentLastTimeConnected))

	// wait a bit for the node's reputation to go above the threshold.
	time.Sleep(allocTimeDuration + time.Second)

	checkPeerStateSetNumOut(t, ps.peerState, testSetID, 1)
	checkMessageStatus(t, <-ps.resultMsgCh, Connect)
}

func TestRemovePeer(t *testing.T) {
	const testSetID = 0

	t.Parallel()
	handler := newTestPeerSet(t, 0, 2, []peer.ID{discovered1, discovered2}, nil, false)

	ps := handler.peerSet
	require.Len(t, ps.resultMsgCh, 2)
	for len(ps.resultMsgCh) != 0 {
		checkMessageStatus(t, <-ps.resultMsgCh, Connect)
	}

	require.Len(t, ps.peerState.nodes, 2)
	checkPeerStateSetNumOut(t, ps.peerState, testSetID, 2)

	handler.RemovePeer(testSetID, discovered1, discovered2)
	time.Sleep(200 * time.Millisecond)

	require.Len(t, ps.resultMsgCh, 2)
	for len(ps.resultMsgCh) != 0 {
		checkMessageStatus(t, <-ps.resultMsgCh, Drop)
	}

	checkPeerStateNodeCount(t, ps.peerState, 0)
	checkPeerStateSetNumOut(t, ps.peerState, testSetID, 0)
}

func TestSetReservePeer(t *testing.T) {
	const testSetID = 0

	t.Parallel()
	handler := newTestPeerSet(t, 0, 2, nil, []peer.ID{reservedPeer, reservedPeer2}, true)

	ps := handler.peerSet
	require.Len(t, ps.resultMsgCh, 2)

	for len(ps.resultMsgCh) != 0 {
		checkMessageStatus(t, <-ps.resultMsgCh, Connect)
	}

	require.Len(t, ps.reservedNode, 2)

	newRsrPeerSet := peer.IDSlice{reservedPeer, peer.ID("newRsrPeer")}
	// add newRsrPeer but remove reservedPeer2
	handler.SetReservedPeer(testSetID, newRsrPeerSet...)
	time.Sleep(200 * time.Millisecond)

	checkPeerSetReservedNodeCount(t, ps, 2)
	for _, p := range newRsrPeerSet {
		checkReservedNodePeerExists(t, ps, p)
	}
}

func getNodePeer(ps *PeersState, pid peer.ID) (node, bool) {
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
	t.Helper()

	ps.RLock()
	defer ps.RUnlock()

	_, exists := ps.sets[setID].noSlotNodes[pid]
	require.True(t, exists)
}

func checkPeerStateSetNumOut(t *testing.T, ps *PeersState, setID int, expectedNumOut uint32) { //nolint:unparam
	t.Helper()

	ps.RLock()
	defer ps.RUnlock()

	gotNumOut := ps.sets[setID].numOut
	require.Equal(t, expectedNumOut, gotNumOut)
}

func checkPeerStateSetNumIn(t *testing.T, ps *PeersState, setID int, expectedNumIn uint32) { //nolint:unparam
	t.Helper()

	ps.RLock()
	defer ps.RUnlock()

	gotNumIn := ps.sets[setID].numIn
	require.Equal(t, expectedNumIn, gotNumIn)
}

func checkPeerStateNodeCount(t *testing.T, ps *PeersState, expectedCount int) {
	t.Helper()

	ps.RLock()
	defer ps.RUnlock()

	require.Equal(t, expectedCount, len(ps.nodes))
}

func checkPeerSetReservedNodeCount(t *testing.T, ps *PeerSet, expectedCount int) {
	t.Helper()

	ps.reservedLock.RLock()
	defer ps.reservedLock.RUnlock()

	require.Equal(t, expectedCount, len(ps.reservedNode))
}
