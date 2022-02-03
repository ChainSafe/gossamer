// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package peerset

import (
	"sync"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/stretchr/testify/require"
)

type assertMessageParam struct {
	expect interface{}
	got    interface{}
}

type mockProcessMessage struct {
	callCounter      int
	executionCounter int
	expects          map[int]assertMessageParam
	mockMutex        sync.Mutex
}

func newMockProcessMessage() *mockProcessMessage {
	return &mockProcessMessage{
		expects: make(map[int]assertMessageParam),
	}
}

func (mock *mockProcessMessage) ExpectByCall(expected interface{}) {
	mock.mockMutex.Lock()
	defer mock.mockMutex.Unlock()

	mock.callCounter++
	mock.expects[mock.callCounter] = assertMessageParam{
		expect: expected,
	}
}

func (mock *mockProcessMessage) ProcessMessage() func(Message) {
	return func(m Message) {
		mock.mockMutex.Lock()
		defer mock.mockMutex.Unlock()

		mock.executionCounter++
		assert, has := mock.expects[mock.executionCounter]
		if has {
			assert.got = m
			mock.expects[mock.executionCounter] = assert
		}
	}
}

func (mock *mockProcessMessage) Assert(t *testing.T) {
	mock.mockMutex.Lock()
	defer mock.mockMutex.Unlock()

	// the number of calls to processMessage should be the same we expect
	require.Equal(t, mock.callCounter, mock.executionCounter)

	// follow the order we defined
	for i := 1; i <= mock.callCounter; i++ {
		assert, has := mock.expects[i]
		require.True(t, has)
		require.Equal(t, assert.expect, assert.got)
	}
}

func TestPeerSetBanned(t *testing.T) {
	t.Parallel()

	mock := newMockProcessMessage()
	processMessageFn := mock.ProcessMessage()

	handler := newTestPeerSet(t, 25, 25, nil, nil, false, processMessageFn)

	ps := handler.peerSet
	require.Equal(t, unknownPeer, ps.peerState.peerStatus(0, peer1))
	ps.peerState.discover(0, peer1)
	// adding peer1 with incoming slot.
	err := ps.peerState.tryAcceptIncoming(0, peer1)
	require.NoError(t, err)

	// we ban a node by setting its reputation under the threshold.
	rpc := newReputationChange(BannedThresholdValue-1, "")

	mock.ExpectByCall(Message{Status: Drop, setID: 0x0, PeerID: "testPeer1"})
	// we need one for the message to be processed.
	handler.ReportPeer(rpc, peer1)
	time.Sleep(time.Millisecond * 100)

	mock.ExpectByCall(Message{Status: Reject, setID: 0x0, PeerID: "testPeer1"})
	// check that an incoming connection from that node gets refused.
	handler.Incoming(0, peer1)

	// wait a bit for the node's reputation to go above the threshold.
	time.Sleep(time.Millisecond * 1200)

	mock.ExpectByCall(Message{Status: Accept, setID: 0x0, PeerID: "testPeer1"})
	// try again. This time the node should be accepted.
	handler.Incoming(0, peer1)

	mock.Assert(t)
}

func TestAddReservedPeers(t *testing.T) {
	t.Parallel()

	mock := newMockProcessMessage()
	mock.ExpectByCall(Message{Status: Connect, setID: 0, PeerID: bootNode})
	mock.ExpectByCall(Message{Status: Connect, setID: 0, PeerID: reservedPeer})
	mock.ExpectByCall(Message{Status: Connect, setID: 0, PeerID: reservedPeer2})
	processMessageFn := mock.ProcessMessage()

	handler := newTestPeerSet(t, 0, 2, []peer.ID{bootNode}, []peer.ID{}, false, processMessageFn)
	ps := handler.peerSet

	handler.AddReservedPeer(0, reservedPeer)
	handler.AddReservedPeer(0, reservedPeer2)

	time.Sleep(time.Millisecond * 200)

	require.Equal(t, uint32(1), ps.peerState.sets[0].numOut)
	require.Equal(t, 3, mock.executionCounter)

	mock.Assert(t)
}

func TestPeerSetIncoming(t *testing.T) {
	t.Parallel()

	mock := newMockProcessMessage()
	processMessageFn := mock.ProcessMessage()

	mock.ExpectByCall(Message{Status: Connect, setID: 0, PeerID: bootNode})
	mock.ExpectByCall(Message{Status: Accept, setID: 0, PeerID: incomingPeer})
	mock.ExpectByCall(Message{Status: Accept, setID: 0, PeerID: incoming2})
	mock.ExpectByCall(Message{Status: Reject, setID: 0, PeerID: incoming3})

	handler := newTestPeerSet(t, 2, 1, []peer.ID{bootNode},
		[]peer.ID{}, false, processMessageFn)

	handler.Incoming(0, incomingPeer)
	handler.Incoming(0, incoming2)
	handler.Incoming(0, incoming3)

	mock.Assert(t)
}

func TestPeerSetDiscovered(t *testing.T) {
	t.Parallel()

	mock := newMockProcessMessage()
	processMessageFn := mock.ProcessMessage()

	mock.ExpectByCall(Message{Status: Connect, setID: 0, PeerID: reservedPeer})
	mock.ExpectByCall(Message{Status: Connect, setID: 0, PeerID: discovered1})
	mock.ExpectByCall(Message{Status: Connect, setID: 0, PeerID: discovered2})

	handler := newTestPeerSet(t, 0, 2, []peer.ID{}, []peer.ID{reservedPeer}, false, processMessageFn)

	handler.AddPeer(0, discovered1)

	handler.AddPeer(0, discovered1)

	handler.AddPeer(0, discovered2)

	mock.Assert(t)
}

func TestReAllocAfterBanned(t *testing.T) {
	t.Parallel()

	mock := newMockProcessMessage()
	processMessageFn := mock.ProcessMessage()

	handler := newTestPeerSet(t, 25, 25, []peer.ID{}, []peer.ID{}, false, processMessageFn)

	ps := handler.peerSet
	// adding peer1 with incoming slot.
	if ps.peerState.peerStatus(0, peer1) == unknownPeer {
		ps.peerState.discover(0, peer1)
		err := ps.peerState.tryAcceptIncoming(0, peer1)
		require.NoError(t, err)
	}

	// We ban a node by setting its reputation under the threshold.
	rep := newReputationChange(BannedThresholdValue-1, "")

	mock.ExpectByCall(Message{Status: Drop, setID: 0, PeerID: peer1})
	// we need one for the message to be processed.
	handler.ReportPeer(rep, peer1)
	time.Sleep(time.Millisecond * 100)

	// Check that an incoming connection from that node gets refused.
	mock.ExpectByCall(Message{Status: Reject, setID: 0, PeerID: peer1})
	mock.ExpectByCall(Message{Status: Connect, setID: 0, PeerID: peer1})
	handler.Incoming(0, peer1)
	time.Sleep(time.Second * 2)

	mock.Assert(t)
}

func TestRemovePeer(t *testing.T) {
	t.Parallel()

	mock := newMockProcessMessage()
	processMessageFn := mock.ProcessMessage()

	mock.ExpectByCall(Message{Status: Connect, setID: 0, PeerID: "testDiscovered1"})
	mock.ExpectByCall(Message{Status: Connect, setID: 0, PeerID: "testDiscovered2"})
	handler := newTestPeerSet(t, 0, 2, []peer.ID{discovered1, discovered2},
		nil, false, processMessageFn)

	ps := handler.peerSet
	require.Equal(t, 2, mock.executionCounter)

	time.Sleep(time.Millisecond * 500)

	mock.ExpectByCall(Message{Status: Drop, setID: 0, PeerID: "testDiscovered1"})
	mock.ExpectByCall(Message{Status: Drop, setID: 0, PeerID: "testDiscovered2"})
	handler.RemovePeer(0, discovered1, discovered2)

	require.Equal(t, 0, len(ps.peerState.nodes))

	mock.Assert(t)
}

func TestSetReservePeer(t *testing.T) {
	t.Parallel()

	mock := newMockProcessMessage()
	processMessageFn := mock.ProcessMessage()

	mock.ExpectByCall(Message{Status: Connect, setID: 0, PeerID: reservedPeer})
	mock.ExpectByCall(Message{Status: Connect, setID: 0, PeerID: reservedPeer2})
	handler := newTestPeerSet(t, 0, 2, nil, []peer.ID{reservedPeer, reservedPeer2},
		true, processMessageFn)

	ps := handler.peerSet

	require.Equal(t, 2, mock.executionCounter)

	mock.ExpectByCall(Message{Status: Connect, setID: 0, PeerID: "newRsrPeer"})
	mock.ExpectByCall(Message{Status: Drop, setID: 0, PeerID: reservedPeer2})

	newRsrPeerSet := peer.IDSlice{reservedPeer, peer.ID("newRsrPeer")}
	// add newRsrPeer but remove reservedPeer2
	handler.SetReservedPeer(0, newRsrPeerSet...)

	require.Equal(t, len(newRsrPeerSet), len(ps.reservedNode))
	for _, p := range newRsrPeerSet {
		require.Contains(t, ps.reservedNode, p)
	}

	mock.Assert(t)
}
