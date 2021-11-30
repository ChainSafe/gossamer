// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package peerset

import (
	"testing"
	"time"

	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/stretchr/testify/require"
)

const (
	bootNode      = peer.ID("testBootNode")
	reservedPeer  = peer.ID("testReservedPeer")
	reservedPeer2 = peer.ID("testReservedPeer2")
	discovered1   = peer.ID("testDiscovered1")
	discovered2   = peer.ID("testDiscovered2")
	incomingPeer  = peer.ID("testIncoming")
	incoming2     = peer.ID("testIncoming2")
	incoming3     = peer.ID("testIncoming3")
	peer1         = peer.ID("testPeer1")
	peer2         = peer.ID("testPeer2")
)

func newTestPeerSet(t *testing.T, in, out uint32, bootNodes, reservedPeers []peer.ID, reservedOnly bool) *Handler {
	t.Helper()
	con := &ConfigSet{
		Set: []*config{
			{
				inPeers:           in,
				outPeers:          out,
				reservedOnly:      reservedOnly,
				periodicAllocTime: time.Second * 2,
			},
		},
	}

	handler, err := NewPeerSetHandler(con)
	require.NoError(t, err)

	handler.Start()

	handler.AddPeer(0, bootNodes...)
	handler.AddReservedPeer(0, reservedPeers...)
	time.Sleep(time.Millisecond * 100)

	return handler
}

func newTestPeerState(t *testing.T, maxIn, maxOut uint32) *PeersState {
	t.Helper()
	state, err := NewPeerState([]*config{
		{
			inPeers:  maxIn,
			outPeers: maxOut,
		},
	})
	require.NoError(t, err)

	return state
}

func checkMessageStatus(t *testing.T, m interface{}, expectedStatus Status) {
	t.Helper()
	msg, ok := m.(Message)
	require.True(t, ok)
	require.Equal(t, expectedStatus, msg.Status)
}
