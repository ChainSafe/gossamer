// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package shim

import (
	"context"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/dot/peerset"
	"github.com/ChainSafe/gossamer/internal/client/network"
	gossip "github.com/ChainSafe/gossamer/internal/client/network-gossip"
	"github.com/ChainSafe/gossamer/internal/client/network/config"
	"github.com/ChainSafe/gossamer/internal/client/network/types/multiaddr"
	peerid "github.com/ChainSafe/gossamer/internal/client/network/types/peer-id"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/stretchr/testify/require"
)

// TestTranslationShImplementsInterfaces verifies that TranslationShim
// implements the required interfaces at compile time.
func TestTranslationShImplementsInterfaces(t *testing.T) {
	var _ gossip.Network = (*TranslationShim)(nil)
	var _ gossip.Syncing[common.Hash, uint] = (*syncShim)(nil)

	require.NotNil(t, NewTranslationShim(nil))
	require.NotNil(t, NewSyncShim(nil))
}

// TestNewTranslationShim tests the constructor.
func TestNewTranslationShim(t *testing.T) {
	shim := NewTranslationShim(nil)
	require.NotNil(t, shim)
	require.IsType(t, &TranslationShim{}, shim)
}

// TestSetAuthorizedPeers tests that SetAuthorizedPeers panics.
func TestSetAuthorizedPeers(t *testing.T) {
	shim := NewTranslationShim(nil)
	require.Panics(t, func() {
		shim.SetAuthorizedPeers(map[peerid.PeerID]struct{}{})
	})
}

// TestSetAuthorizedOnly tests that SetAuthorizedOnly panics.
func TestSetAuthorizedOnly(t *testing.T) {
	shim := NewTranslationShim(nil)
	require.Panics(t, func() {
		shim.SetAuthorizedOnly(true)
	})
	require.Panics(t, func() {
		shim.SetAuthorizedOnly(false)
	})
}

// TestAddKnownAddress tests that AddKnownAddress panics.
func TestAddKnownAddress(t *testing.T) {
	shim := NewTranslationShim(nil)
	peerID := peerid.NewRandomPeerID()
	var addr multiaddr.Multiaddr

	require.Panics(t, func() {
		shim.AddKnownAddress(peerID, addr)
	})
}

// TestReportPeerNilHandler tests that ReportPeer panics when handler is nil.
func TestReportPeerNilHandler(t *testing.T) {
	shim := NewTranslationShim(nil)
	peerID := peerid.NewRandomPeerID()

	require.Panics(t, func() {
		shim.ReportPeer(peerID, network.ReputationChange{})
	})
}

// TestReportPeer tests that ReportPeer correctly translates types and calls the handler.
func TestReportPeer(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := peerset.NewConfigSet(10, 10, false, time.Second)
	handler, err := peerset.NewPeerSetHandler(cfg)
	require.NoError(t, err)
	require.NotNil(t, handler)

	// Start the handler
	handler.Start(ctx)

	shim := NewTranslationShim(handler)
	peerID := peerid.NewRandomPeerID()

	// This should not panic and should successfully call the handler
	require.NotPanics(t, func() {
		shim.ReportPeer(peerID, network.ReputationChange{
			Value:  100,
			Reason: "test reputation change",
		})
	})
}

// TestPeerReputation tests that PeerReputation panics.
func TestPeerReputation(t *testing.T) {
	shim := NewTranslationShim(nil)
	peerID := peerid.NewRandomPeerID()

	require.Panics(t, func() {
		shim.PeerReputation(peerID)
	})
}

// TestDisconnectPeer tests that DisconnectPeer panics.
func TestDisconnectPeer(t *testing.T) {
	shim := NewTranslationShim(nil)
	peerID := peerid.NewRandomPeerID()

	require.Panics(t, func() {
		shim.DisconnectPeer(peerID, network.ProtocolName(""))
	})
}

// TestAcceptUnreservedPeers tests that AcceptUnreservedPeers panics.
func TestAcceptUnreservedPeers(t *testing.T) {
	shim := NewTranslationShim(nil)
	require.Panics(t, func() {
		shim.AcceptUnreservedPeers()
	})
}

// TestDenyUnreservedPeers tests that DenyUnreservedPeers panics.
func TestDenyUnreservedPeers(t *testing.T) {
	shim := NewTranslationShim(nil)
	require.Panics(t, func() {
		shim.DenyUnreservedPeers()
	})
}

// TestAddReservedPeer tests that AddReservedPeer panics.
func TestAddReservedPeer(t *testing.T) {
	shim := NewTranslationShim(nil)
	require.Panics(t, func() {
		shim.AddReservedPeer(config.MultiaddrPeerId{})
	})
}

// TestRemoveReservedPeer tests that RemoveReservedPeer panics.
func TestRemoveReservedPeer(t *testing.T) {
	shim := NewTranslationShim(nil)
	peerID := peerid.NewRandomPeerID()

	require.Panics(t, func() {
		shim.RemoveReservedPeer(peerID)
	})
}

// TestSetReservedPeers tests that SetReservedPeers panics.
func TestSetReservedPeers(t *testing.T) {
	shim := NewTranslationShim(nil)
	require.Panics(t, func() {
		shim.SetReservedPeers(network.ProtocolName(""), map[multiaddr.Multiaddr]struct{}{})
	})
}

// TestAddPeersToReservedSet tests that AddPeersToReservedSet panics.
func TestAddPeersToReservedSet(t *testing.T) {
	shim := NewTranslationShim(nil)
	require.Panics(t, func() {
		shim.AddPeersToReservedSet(network.ProtocolName(""), map[multiaddr.Multiaddr]struct{}{})
	})
}

// TestRemovePeersFromReservedSet tests that RemovePeersFromReservedSet panics.
func TestRemovePeersFromReservedSet(t *testing.T) {
	shim := NewTranslationShim(nil)
	peerID := peerid.NewRandomPeerID()

	require.Panics(t, func() {
		shim.RemovePeersFromReservedSet(network.ProtocolName(""), []peerid.PeerID{peerID})
	})
}

// TestSyncNumConnected tests that SyncNumConnected panics.
func TestSyncNumConnected(t *testing.T) {
	shim := NewTranslationShim(nil)
	require.Panics(t, func() {
		shim.SyncNumConnected()
	})
}

// TestPeerRole tests that PeerRole panics.
func TestPeerRole(t *testing.T) {
	shim := NewTranslationShim(nil)
	peerID := peerid.NewRandomPeerID()

	require.Panics(t, func() {
		shim.PeerRole(peerID, []byte{})
	})
}

// TestReservedPeers tests that ReservedPeers panics.
func TestReservedPeers(t *testing.T) {
	shim := NewTranslationShim(nil)
	require.Panics(t, func() {
		shim.ReservedPeers()
	})
}

// TestEventStreamNetwork tests that EventStream (NetworkEventStream) panics.
func TestEventStreamNetwork(t *testing.T) {
	shim := NewTranslationShim(nil)
	require.Panics(t, func() {
		shim.EventStream("test")
	})
}

// TestAddSetReserved tests that AddSetReserved panics.
func TestAddSetReserved(t *testing.T) {
	shim := NewTranslationShim(nil)
	peerID := peerid.NewRandomPeerID()

	require.Panics(t, func() {
		shim.AddSetReserved(peerID, network.ProtocolName(""))
	})
}

// TestRemoveSetReserved tests that RemoveSetReserved panics.
func TestRemoveSetReserved(t *testing.T) {
	shim := NewTranslationShim(nil)
	peerID := peerid.NewRandomPeerID()

	require.Panics(t, func() {
		shim.RemoveSetReserved(peerID, network.ProtocolName(""))
	})
}

// TestNewSyncShim tests the syncShim constructor.
func TestNewSyncShim(t *testing.T) {
	syncShim := NewSyncShim(nil)
	require.NotNil(t, syncShim)
	require.NotNil(t, syncShim.TranslationShim)
}

// TestEventStreamSyncShim tests that EventStream (SyncEventStream) panics on syncShim.
func TestEventStreamSyncShim(t *testing.T) {
	syncShim := NewSyncShim(nil)
	require.Panics(t, func() {
		syncShim.EventStream("test")
	})
}

// TestAnnounceBlockSyncShim tests that AnnounceBlock panics on syncShim.
func TestAnnounceBlockSyncShim(t *testing.T) {
	syncShim := NewSyncShim(nil)
	hash := common.Hash{}

	require.Panics(t, func() {
		syncShim.AnnounceBlock(hash, []byte{})
	})
}

// TestNewBestBlockImportedSyncShim tests that NewBestBlockImported panics on syncShim.
func TestNewBestBlockImportedSyncShim(t *testing.T) {
	syncShim := NewSyncShim(nil)
	hash := common.Hash{}

	require.Panics(t, func() {
		syncShim.NewBestBlockImported(hash, 0)
	})
}
