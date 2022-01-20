// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package peerset

import (
	"testing"
	"time"

	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/stretchr/testify/require"
)

func TestPeerSetBanned(t *testing.T) {
	t.Parallel()

	handler := newTestPeerSet(t, 25, 25, nil, nil, false)

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

	checkMessageStatus(t, <-ps.resultMsgCh, Drop)

	// check that an incoming connection from that node gets refused.
	handler.Incoming(0, peer1)
	checkMessageStatus(t, <-ps.resultMsgCh, Reject)

	// wait a bit for the node's reputation to go above the threshold.
	time.Sleep(time.Millisecond * 1200)

	// try again. This time the node should be accepted.
	handler.Incoming(0, peer1)
	require.NoError(t, err)
	checkMessageStatus(t, <-ps.resultMsgCh, Accept)
}

func TestAddReservedPeers(t *testing.T) {
	t.Parallel()

	handler := newTestPeerSet(t, 0, 2, []peer.ID{bootNode}, []peer.ID{}, false)
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
	t.Parallel()

	handler := newTestPeerSet(t, 2, 1, []peer.ID{bootNode}, []peer.ID{}, false)
	ps := handler.peerSet

	// connect message will be added ingoing queue for bootnode.
	checkMessageStatus(t, <-ps.resultMsgCh, Connect)

	handler.Incoming(0, incomingPeer)
	checkMessageStatus(t, <-ps.resultMsgCh, Accept)

	handler.Incoming(0, incoming2)
	checkMessageStatus(t, <-ps.resultMsgCh, Accept)

	handler.Incoming(0, incoming3)
	checkMessageStatus(t, <-ps.resultMsgCh, Reject)
}

func TestPeerSetDiscovered(t *testing.T) {
	t.Parallel()

	handler := newTestPeerSet(t, 0, 2, []peer.ID{}, []peer.ID{reservedPeer}, false)

	ps := handler.peerSet

	handler.AddPeer(0, discovered1)
	handler.AddPeer(0, discovered1)
	handler.AddPeer(0, discovered2)

	time.Sleep(200 * time.Millisecond)

	require.Equal(t, 3, len(ps.resultMsgCh))
	for len(ps.resultMsgCh) == 0 {
		checkMessageStatus(t, <-ps.resultMsgCh, Connect)
	}
}

func TestReAllocAfterBanned(t *testing.T) {
	t.Parallel()

	handler := newTestPeerSet(t, 25, 25, []peer.ID{}, []peer.ID{}, false)

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

	checkMessageStatus(t, <-ps.resultMsgCh, Drop)

	// Check that an incoming connection from that node gets refused.

	handler.Incoming(0, peer1)
	checkMessageStatus(t, <-ps.resultMsgCh, Reject)

	time.Sleep(time.Millisecond * 100)
	checkMessageStatus(t, <-ps.resultMsgCh, Connect)
}

func TestRemovePeer(t *testing.T) {
	t.Parallel()

	handler := newTestPeerSet(t, 0, 2, []peer.ID{discovered1, discovered2}, nil, false)
	ps := handler.peerSet

	require.Equal(t, 2, len(ps.resultMsgCh))
	for len(ps.resultMsgCh) != 0 {
		checkMessageStatus(t, <-ps.resultMsgCh, Connect)
	}

	handler.RemovePeer(0, discovered1, discovered2)
	time.Sleep(200 * time.Millisecond)

	require.Equal(t, 2, len(ps.resultMsgCh))
	for len(ps.resultMsgCh) != 0 {
		checkMessageStatus(t, <-ps.resultMsgCh, Drop)
	}

	require.Equal(t, 0, len(ps.peerState.nodes))
}

func TestSetReservePeer(t *testing.T) {
	t.Parallel()

	handler := newTestPeerSet(t, 0, 2, nil, []peer.ID{reservedPeer, reservedPeer2}, true)
	ps := handler.peerSet

	require.Equal(t, 2, len(ps.resultMsgCh))
	for len(ps.resultMsgCh) != 0 {
		checkMessageStatus(t, <-ps.resultMsgCh, Connect)
	}

	newRsrPeerSet := peer.IDSlice{reservedPeer, peer.ID("newRsrPeer")}
	handler.SetReservedPeer(0, newRsrPeerSet...)
	time.Sleep(200 * time.Millisecond)

	require.Equal(t, len(newRsrPeerSet), len(ps.reservedNode))
	for _, p := range newRsrPeerSet {
		require.Contains(t, ps.reservedNode, p)
	}
}
