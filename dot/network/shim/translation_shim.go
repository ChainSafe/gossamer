// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package shim

import (
	"github.com/ChainSafe/gossamer/dot/peerset"
	"github.com/ChainSafe/gossamer/internal/client/network"
	gossip "github.com/ChainSafe/gossamer/internal/client/network-gossip"
	"github.com/ChainSafe/gossamer/internal/client/network/config"
	"github.com/ChainSafe/gossamer/internal/client/network/event"
	"github.com/ChainSafe/gossamer/internal/client/network/role"
	"github.com/ChainSafe/gossamer/internal/client/network/sync"
	"github.com/ChainSafe/gossamer/internal/client/network/types/multiaddr"
	peerid "github.com/ChainSafe/gossamer/internal/client/network/types/peer-id"
	"github.com/ChainSafe/gossamer/lib/common"
)

// TranslationShim implements the gossip.Network and gossip.Syncing interfaces.
// This shim type provides a bridge between the new Network and Syncing interfaces
// and the existing Gossamer network implementation.
//
// The shim is placed in a separate package to avoid circular dependencies between
// internal/client/network and dot/network packages.
//
// Since both Network and Syncing interfaces have an EventStream method with different
// return types, we cannot implement both in the same type directly. Instead, we provide
// two separate types that can be used together.
type TranslationShim struct {
	handler *peerset.Handler
}

// NewTranslationShim creates a new TranslationShim instance.
func NewTranslationShim(handler *peerset.Handler) *TranslationShim {
	return &TranslationShim{handler: handler}
}

// Compile-time interface check for Network
var _ gossip.Network = (*TranslationShim)(nil)

// Compile-time interface check for Syncing
var _ gossip.Syncing[common.Hash, uint] = (*syncShim)(nil)

// ===== NetworkPeers Methods =====

// SetAuthorizedPeers sets the authorized peers.
func (s *TranslationShim) SetAuthorizedPeers(peers map[peerid.PeerID]struct{}) {
	panic("SetAuthorizedPeers: not implemented yet")
}

// SetAuthorizedOnly sets the authorized_only flag.
func (s *TranslationShim) SetAuthorizedOnly(reservedOnly bool) {
	panic("SetAuthorizedOnly: not implemented yet")
}

// AddKnownAddress adds an address known to a node.
func (s *TranslationShim) AddKnownAddress(peerID peerid.PeerID, addr multiaddr.Multiaddr) {
	panic("AddKnownAddress: not implemented yet")
}

// ReportPeer reports a given peer as either beneficial (+) or costly (-).
// It translates the network types to peerset types and calls the underlying handler.
func (s *TranslationShim) ReportPeer(peerID peerid.PeerID, costBenefit network.ReputationChange) {
	if s.handler == nil {
		panic("ReportPeer: handler is nil")
	}

	// Translate network.ReputationChange to peerset.ReputationChange
	peersetChange := peerset.ReputationChange{
		Value:  peerset.Reputation(costBenefit.Value),
		Reason: costBenefit.Reason,
	}

	// Translate peerid.PeerID to peer.ID and call the handler
	s.handler.ReportPeer(peersetChange, peerID.ID)
}

// PeerReputation gets the reputation of a peer.
func (s *TranslationShim) PeerReputation(peerID peerid.PeerID) int32 {
	panic("PeerReputation: not implemented yet")
}

// DisconnectPeer disconnects from a node as soon as possible.
func (s *TranslationShim) DisconnectPeer(who peerid.PeerID, protocol network.ProtocolName) {
	panic("DisconnectPeer: not implemented yet")
}

// AcceptUnreservedPeers connects to unreserved peers.
func (s *TranslationShim) AcceptUnreservedPeers() {
	panic("AcceptUnreservedPeers: not implemented yet")
}

// DenyUnreservedPeers disconnects from unreserved peers.
func (s *TranslationShim) DenyUnreservedPeers() {
	panic("DenyUnreservedPeers: not implemented yet")
}

// AddReservedPeer adds a PeerID and its MultiAddr as reserved.
func (s *TranslationShim) AddReservedPeer(peer config.MultiaddrPeerId) error {
	panic("AddReservedPeer: not implemented yet")
}

// RemoveReservedPeer removes a PeerID from the list of reserved peers.
func (s *TranslationShim) RemoveReservedPeer(peerID peerid.PeerID) {
	panic("RemoveReservedPeer: not implemented yet")
}

// SetReservedPeers sets the reserved set of a protocol.
func (s *TranslationShim) SetReservedPeers(
	protocol network.ProtocolName,
	peers map[multiaddr.Multiaddr]struct{},
) error {
	panic("SetReservedPeers: not implemented yet")
}

// AddPeersToReservedSet adds peers to a peer set.
func (s *TranslationShim) AddPeersToReservedSet(
	protocol network.ProtocolName,
	peers map[multiaddr.Multiaddr]struct{},
) error {
	panic("AddPeersToReservedSet: not implemented yet")
}

// RemovePeersFromReservedSet removes peers from a peer set.
func (s *TranslationShim) RemovePeersFromReservedSet(
	protocol network.ProtocolName,
	peers []peerid.PeerID,
) {
	panic("RemovePeersFromReservedSet: not implemented yet")
}

// SyncNumConnected returns the number of connected sync peers.
func (s *TranslationShim) SyncNumConnected() uint {
	panic("SyncNumConnected: not implemented yet")
}

// PeerRole attempts to get peer role from handshake.
func (s *TranslationShim) PeerRole(peerID peerid.PeerID, handshake []byte) *role.ObservedRole {
	panic("PeerRole: not implemented yet")
}

// ReservedPeers returns the list of reserved peers.
func (s *TranslationShim) ReservedPeers() <-chan struct {
	Peers []peerid.PeerID
	Error error
} {
	panic("ReservedPeers: not implemented yet")
}

// ===== NetworkEventStream Methods =====

// EventStream returns a stream containing network events.
func (s *TranslationShim) EventStream(name string) chan event.Event {
	panic("EventStream (NetworkEventStream): not implemented yet")
}

// ===== Network Methods =====

// AddSetReserved adds a peer to the reserved set.
func (s *TranslationShim) AddSetReserved(who peerid.PeerID, protocol network.ProtocolName) {
	panic("AddSetReserved: not implemented yet")
}

// RemoveSetReserved removes a peer from the reserved set.
func (s *TranslationShim) RemoveSetReserved(who peerid.PeerID, protocol network.ProtocolName) {
	panic("RemoveSetReserved: not implemented yet")
}

// syncShim implements the gossip.Syncing interface.
// This is a separate type to avoid method name conflicts with Network.
type syncShim struct {
	*TranslationShim
}

// NewSyncShim creates a new syncShim instance.
func NewSyncShim(handler *peerset.Handler) *syncShim {
	return &syncShim{
		TranslationShim: NewTranslationShim(handler),
	}
}

// ===== SyncEventStream Methods =====

// EventStream subscribes to syncing related events.
func (s *syncShim) EventStream(name string) chan sync.SyncEvent {
	panic("EventStream (SyncEventStream): not implemented yet")
}

// ===== NetworkBlock Methods =====

// AnnounceBlock announces a block to the network.
func (s *syncShim) AnnounceBlock(hash common.Hash, data []byte) {
	panic("AnnounceBlock: not implemented yet")
}

// NewBestBlockImported informs the network about a new best imported block.
func (s *syncShim) NewBestBlockImported(hash common.Hash, number uint) {
	panic("NewBestBlockImported: not implemented yet")
}
