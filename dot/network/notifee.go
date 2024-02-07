// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package network

import (
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/peerstore"
	"github.com/multiformats/go-multiaddr"
)

// AddressAdder is an interface that adds addresses.
type AddressAdder interface {
	AddAddrs(p peer.ID, addrs []multiaddr.Multiaddr, ttl time.Duration)
}

// PeerAdder adds peers.
type PeerAdder interface {
	AddPeer(setID int, peerIDs ...peer.ID)
}

// NewNotifeeTracker returns a new notifee tracker.
func NewNotifeeTracker(addressAdder AddressAdder, peerAdder PeerAdder) *NotifeeTracker {
	return &NotifeeTracker{
		addressAdder: addressAdder,
		peerAdder:    peerAdder,
	}
}

// NotifeeTracker tracks new peers found.
type NotifeeTracker struct {
	addressAdder AddressAdder
	peerAdder    PeerAdder
}

// HandlePeerFound tracks the address info from the peer found.
func (n *NotifeeTracker) HandlePeerFound(p peer.AddrInfo) {
	logger.Infof("HANDLE PEER FOUND %v", p)
	n.addressAdder.AddAddrs(p.ID, p.Addrs, peerstore.PermanentAddrTTL)
	n.peerAdder.AddPeer(0, p.ID)
}
