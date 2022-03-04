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

func Test_Ban_Reject_Accept_Peer(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	processor := NewMockMessageProcessor(ctrl)

	handler := newTestPeerSet(t, 25, 25, nil, nil, false, processor)

	ps := handler.peerSet
	require.Equal(t, unknownPeer, ps.peerState.peerStatus(0, peer1))
	ps.peerState.discover(0, peer1)
	// adding peer1 with incoming slot.
	err := ps.peerState.tryAcceptIncoming(0, peer1)
	require.NoError(t, err)

	// we ban a node by setting its reputation under the threshold.
	rpc := newReputationChange(BannedThresholdValue-1, "")

	// we need one for the message to be processed.
	processor.EXPECT().Process(Message{Status: Drop, setID: 0x0, PeerID: "testPeer1"})
	handler.ReportPeer(rpc, peer1)
	time.Sleep(time.Millisecond * 100)

	// check that an incoming connection from that node gets refused.
	processor.EXPECT().Process(Message{Status: Reject, setID: 0x0, PeerID: "testPeer1"})
	handler.Incoming(0, peer1)

	// wait a bit for the node's reputation to go above the threshold.
	time.Sleep(time.Millisecond * 1200)

	// try again. This time the node should be accepted.
	processor.EXPECT().Process(Message{Status: Accept, setID: 0x0, PeerID: "testPeer1"})
	handler.Incoming(0, peer1)
}

func TestAddReservedPeers(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	processor := NewMockMessageProcessor(ctrl)
	processor.EXPECT().Process(Message{Status: Connect, setID: 0, PeerID: bootNode})
	processor.EXPECT().Process(Message{Status: Connect, setID: 0, PeerID: reservedPeer})
	processor.EXPECT().Process(Message{Status: Connect, setID: 0, PeerID: reservedPeer2})

	handler := newTestPeerSet(t, 0, 2, []peer.ID{bootNode}, []peer.ID{}, false, processor)
	ps := handler.peerSet

	handler.AddReservedPeer(0, reservedPeer)
	handler.AddReservedPeer(0, reservedPeer2)

	time.Sleep(time.Millisecond * 200)

	require.Equal(t, uint32(1), ps.peerState.sets[0].numOut)
}

func TestPeerSetIncoming(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	processor := NewMockMessageProcessor(ctrl)

	processor.EXPECT().Process(Message{Status: Connect, setID: 0, PeerID: bootNode})
	handler := newTestPeerSet(t, 2, 1, []peer.ID{bootNode},
		[]peer.ID{}, false, processor)

	processor.EXPECT().Process(Message{Status: Accept, setID: 0, PeerID: incomingPeer})
	handler.Incoming(0, incomingPeer)

	processor.EXPECT().Process(Message{Status: Accept, setID: 0, PeerID: incoming2})
	handler.Incoming(0, incoming2)

	processor.EXPECT().Process(Message{Status: Reject, setID: 0, PeerID: incoming3})
	handler.Incoming(0, incoming3)
}

func TestPeerSetDiscovered(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	processor := NewMockMessageProcessor(ctrl)
	handler := newTestPeerSet(t, 0, 2, []peer.ID{}, []peer.ID{reservedPeer}, false, processor)

	processor.EXPECT().Process(Message{Status: Connect, setID: 0, PeerID: reservedPeer})
	handler.AddPeer(0, discovered1)

	processor.EXPECT().Process(Message{Status: Connect, setID: 0, PeerID: discovered1})
	handler.AddPeer(0, discovered1)

	processor.EXPECT().Process(Message{Status: Connect, setID: 0, PeerID: discovered2})
	handler.AddPeer(0, discovered2)
}

func TestReAllocAfterBanned(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	processor := NewMockMessageProcessor(ctrl)
	handler := newTestPeerSet(t, 25, 25, []peer.ID{}, []peer.ID{}, false, processor)

	ps := handler.peerSet
	// adding peer1 with incoming slot.
	if ps.peerState.peerStatus(0, peer1) == unknownPeer {
		ps.peerState.discover(0, peer1)
		err := ps.peerState.tryAcceptIncoming(0, peer1)
		require.NoError(t, err)
	}

	// We ban a node by setting its reputation under the threshold.
	processor.EXPECT().Process(Message{Status: Drop, setID: 0, PeerID: peer1})
	rep := newReputationChange(BannedThresholdValue-1, "")

	// we need one for the message to be processed.
	processor.EXPECT().Process(Message{Status: Reject, setID: 0, PeerID: peer1})
	handler.ReportPeer(rep, peer1)
	time.Sleep(time.Millisecond * 100)

	// Check that an incoming connection from that node gets refused.
	processor.EXPECT().Process(Message{Status: Connect, setID: 0, PeerID: peer1})
	handler.Incoming(0, peer1)
	time.Sleep(time.Second * 2)

}

func TestRemovePeer(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	processor := NewMockMessageProcessor(ctrl)

	processor.EXPECT().Process(Message{Status: Connect, setID: 0, PeerID: "testDiscovered1"})
	processor.EXPECT().Process(Message{Status: Connect, setID: 0, PeerID: "testDiscovered2"})
	handler := newTestPeerSet(t, 0, 2, []peer.ID{discovered1, discovered2},
		nil, false, processor)

	ps := handler.peerSet
	time.Sleep(time.Millisecond * 500)

	processor.EXPECT().Process(Message{Status: Drop, setID: 0, PeerID: "testDiscovered1"})
	processor.EXPECT().Process(Message{Status: Drop, setID: 0, PeerID: "testDiscovered2"})
	handler.RemovePeer(0, discovered1, discovered2)

	require.Equal(t, 0, len(ps.peerState.nodes))
}

func TestSetReservePeer(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	processor := NewMockMessageProcessor(ctrl)

	processor.EXPECT().Process(Message{Status: Connect, setID: 0, PeerID: reservedPeer})
	processor.EXPECT().Process(Message{Status: Connect, setID: 0, PeerID: reservedPeer2})
	handler := newTestPeerSet(t, 0, 2, nil, []peer.ID{reservedPeer, reservedPeer2},
		true, processor)

	ps := handler.peerSet

	newRsrPeerSet := peer.IDSlice{reservedPeer, peer.ID("newRsrPeer")}
	// add newRsrPeer but remove reservedPeer2
	processor.EXPECT().Process(Message{Status: Connect, setID: 0, PeerID: "newRsrPeer"})
	processor.EXPECT().Process(Message{Status: Drop, setID: 0, PeerID: reservedPeer2})
	handler.SetReservedPeer(0, newRsrPeerSet...)

	require.Equal(t, len(newRsrPeerSet), len(ps.reservedNode))
	for _, p := range newRsrPeerSet {
		require.Contains(t, ps.reservedNode, p)
	}
}
