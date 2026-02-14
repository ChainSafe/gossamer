// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package shim

import (
	"testing"

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

	require.NotNil(t, NewTranslationShim())
	require.NotNil(t, NewSyncShim())
}

// TestNewTranslationShim tests the constructor.
func TestNewTranslationShim(t *testing.T) {
	shim := NewTranslationShim()
	require.NotNil(t, shim)
	require.IsType(t, &TranslationShim{}, shim)
}

// TestSetAuthorizedPeers tests that SetAuthorizedPeers panics.
func TestSetAuthorizedPeers(t *testing.T) {
	shim := NewTranslationShim()
	require.Panics(t, func() {
		shim.SetAuthorizedPeers(map[peerid.PeerID]struct{}{})
	})
}

// TestSetAuthorizedOnly tests that SetAuthorizedOnly panics.
func TestSetAuthorizedOnly(t *testing.T) {
	shim := NewTranslationShim()
	require.Panics(t, func() {
		shim.SetAuthorizedOnly(true)
	})
	require.Panics(t, func() {
		shim.SetAuthorizedOnly(false)
	})
}

// TestAddKnownAddress tests that AddKnownAddress panics.
func TestAddKnownAddress(t *testing.T) {
	shim := NewTranslationShim()
	peerID := peerid.NewRandomPeerID()
	var addr multiaddr.Multiaddr

	require.Panics(t, func() {
		shim.AddKnownAddress(peerID, addr)
	})
}

// TestReportPeer tests that ReportPeer panics.
func TestReportPeer(t *testing.T) {
	shim := NewTranslationShim()
	peerID := peerid.NewRandomPeerID()

	require.Panics(t, func() {
		shim.ReportPeer(peerID, network.ReputationChange{})
	})
}

// TestPeerReputation tests that PeerReputation panics.
func TestPeerReputation(t *testing.T) {
	shim := NewTranslationShim()
	peerID := peerid.NewRandomPeerID()

	require.Panics(t, func() {
		shim.PeerReputation(peerID)
	})
}

// TestDisconnectPeer tests that DisconnectPeer panics.
func TestDisconnectPeer(t *testing.T) {
	shim := NewTranslationShim()
	peerID := peerid.NewRandomPeerID()

	require.Panics(t, func() {
		shim.DisconnectPeer(peerID, network.ProtocolName(""))
	})
}

// TestAcceptUnreservedPeers tests that AcceptUnreservedPeers panics.
func TestAcceptUnreservedPeers(t *testing.T) {
	shim := NewTranslationShim()
	require.Panics(t, func() {
		shim.AcceptUnreservedPeers()
	})
}

// TestDenyUnreservedPeers tests that DenyUnreservedPeers panics.
func TestDenyUnreservedPeers(t *testing.T) {
	shim := NewTranslationShim()
	require.Panics(t, func() {
		shim.DenyUnreservedPeers()
	})
}

// TestAddReservedPeer tests that AddReservedPeer panics.
func TestAddReservedPeer(t *testing.T) {
	shim := NewTranslationShim()
	require.Panics(t, func() {
		shim.AddReservedPeer(config.MultiaddrPeerId{})
	})
}

// TestRemoveReservedPeer tests that RemoveReservedPeer panics.
func TestRemoveReservedPeer(t *testing.T) {
	shim := NewTranslationShim()
	peerID := peerid.NewRandomPeerID()

	require.Panics(t, func() {
		shim.RemoveReservedPeer(peerID)
	})
}

// TestSetReservedPeers tests that SetReservedPeers panics.
func TestSetReservedPeers(t *testing.T) {
	shim := NewTranslationShim()
	require.Panics(t, func() {
		shim.SetReservedPeers(network.ProtocolName(""), map[multiaddr.Multiaddr]struct{}{})
	})
}

// TestAddPeersToReservedSet tests that AddPeersToReservedSet panics.
func TestAddPeersToReservedSet(t *testing.T) {
	shim := NewTranslationShim()
	require.Panics(t, func() {
		shim.AddPeersToReservedSet(network.ProtocolName(""), map[multiaddr.Multiaddr]struct{}{})
	})
}

// TestRemovePeersFromReservedSet tests that RemovePeersFromReservedSet panics.
func TestRemovePeersFromReservedSet(t *testing.T) {
	shim := NewTranslationShim()
	peerID := peerid.NewRandomPeerID()

	require.Panics(t, func() {
		shim.RemovePeersFromReservedSet(network.ProtocolName(""), []peerid.PeerID{peerID})
	})
}

// TestSyncNumConnected tests that SyncNumConnected panics.
func TestSyncNumConnected(t *testing.T) {
	shim := NewTranslationShim()
	require.Panics(t, func() {
		shim.SyncNumConnected()
	})
}

// TestPeerRole tests that PeerRole panics.
func TestPeerRole(t *testing.T) {
	shim := NewTranslationShim()
	peerID := peerid.NewRandomPeerID()

	require.Panics(t, func() {
		shim.PeerRole(peerID, []byte{})
	})
}

// TestReservedPeers tests that ReservedPeers panics.
func TestReservedPeers(t *testing.T) {
	shim := NewTranslationShim()
	require.Panics(t, func() {
		shim.ReservedPeers()
	})
}

// TestEventStreamNetwork tests that EventStream (NetworkEventStream) panics.
func TestEventStreamNetwork(t *testing.T) {
	shim := NewTranslationShim()
	require.Panics(t, func() {
		shim.EventStream("test")
	})
}

// TestAddSetReserved tests that AddSetReserved panics.
func TestAddSetReserved(t *testing.T) {
	shim := NewTranslationShim()
	peerID := peerid.NewRandomPeerID()

	require.Panics(t, func() {
		shim.AddSetReserved(peerID, network.ProtocolName(""))
	})
}

// TestRemoveSetReserved tests that RemoveSetReserved panics.
func TestRemoveSetReserved(t *testing.T) {
	shim := NewTranslationShim()
	peerID := peerid.NewRandomPeerID()

	require.Panics(t, func() {
		shim.RemoveSetReserved(peerID, network.ProtocolName(""))
	})
}

// TestEventStreamSync tests that EventStream (SyncEventStream) panics.
func TestEventStreamSync(t *testing.T) {
	shim := NewTranslationShim()
	require.Panics(t, func() {
		shim.EventStream("test")
	})
}

// TestNewSyncShim tests the syncShim constructor.
func TestNewSyncShim(t *testing.T) {
	syncShim := NewSyncShim()
	require.NotNil(t, syncShim)
	require.NotNil(t, syncShim.TranslationShim)
}

// TestEventStreamSyncShim tests that EventStream (SyncEventStream) panics on syncShim.
func TestEventStreamSyncShim(t *testing.T) {
	syncShim := NewSyncShim()
	require.Panics(t, func() {
		syncShim.EventStream("test")
	})
}

// TestAnnounceBlockSyncShim tests that AnnounceBlock panics on syncShim.
func TestAnnounceBlockSyncShim(t *testing.T) {
	syncShim := NewSyncShim()
	hash := common.Hash{}

	require.Panics(t, func() {
		syncShim.AnnounceBlock(hash, []byte{})
	})
}

// TestNewBestBlockImportedSyncShim tests that NewBestBlockImported panics on syncShim.
func TestNewBestBlockImportedSyncShim(t *testing.T) {
	syncShim := NewSyncShim()
	hash := common.Hash{}

	require.Panics(t, func() {
		syncShim.NewBestBlockImported(hash, 0)
	})
}
