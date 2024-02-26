// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package network

import (
	"context"
	"sync"

	"github.com/libp2p/go-libp2p/core/connmgr"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	ma "github.com/multiformats/go-multiaddr"

	"github.com/ChainSafe/gossamer/dot/peerset"
)

// ConnManager implements connmgr.ConnManager
type ConnManager struct {
	sync.Mutex
	host              *host
	maxPeers          int
	connectHandler    func(peer.ID)
	disconnectHandler func(peer.ID)

	// protectedPeers contains a list of peers that are protected from pruning
	// when we reach the maximum numbers of peers.
	protectedPeers *sync.Map // map[peer.ID]struct{}

	// persistentPeers contains peers we should remain connected to.
	persistentPeers *sync.Map // map[peer.ID]struct{}

	peerSetHandler PeerSetHandler
}

func newConnManager(max int, peerSetCfg *peerset.ConfigSet) (*ConnManager, error) {
	// TODO: peerSetHandler never used from within connection manager and also referred outside through cm,
	// so this should be refactored
	psh, err := peerset.NewPeerSetHandler(peerSetCfg)
	if err != nil {
		return nil, err
	}

	return &ConnManager{
		maxPeers:        max,
		protectedPeers:  new(sync.Map),
		persistentPeers: new(sync.Map),
		peerSetHandler:  psh,
	}, nil
}

// Notifee is used to monitor changes to a connection
func (cm *ConnManager) Notifee() network.Notifiee {
	nb := new(network.NotifyBundle)

	nb.ListenF = cm.Listen
	nb.ListenCloseF = cm.ListenClose
	nb.ConnectedF = cm.Connected
	nb.DisconnectedF = cm.Disconnected

	return nb
}

// TagPeer is unimplemented
func (*ConnManager) TagPeer(peer.ID, string, int) {}

// UntagPeer is unimplemented
func (*ConnManager) UntagPeer(peer.ID, string) {}

// UpsertTag is unimplemented
func (*ConnManager) UpsertTag(peer.ID, string, func(int) int) {}

// GetTagInfo is unimplemented
func (*ConnManager) GetTagInfo(peer.ID) *connmgr.TagInfo { return &connmgr.TagInfo{} }

// TrimOpenConns is unimplemented
func (*ConnManager) TrimOpenConns(context.Context) {}

// Protect peer will add the given peer to the protectedPeerMap which will
// protect the peer from pruning.
func (cm *ConnManager) Protect(id peer.ID, _ string) {
	cm.protectedPeers.Store(id, struct{}{})
}

// Unprotect peer will remove the given peer from prune protection.
// returns true if we have successfully removed the peer from the
// protectedPeerMap. False otherwise.
func (cm *ConnManager) Unprotect(id peer.ID, _ string) bool {
	_, wasDeleted := cm.protectedPeers.LoadAndDelete(id)
	return wasDeleted
}

// Close is unimplemented
func (*ConnManager) Close() error { return nil }

// IsProtected returns whether the given peer is protected from pruning or not.
func (cm *ConnManager) IsProtected(id peer.ID, _ string) (protected bool) {
	_, ok := cm.protectedPeers.Load(id)
	return ok
}

// Listen is called when network starts listening on an address
func (cm *ConnManager) Listen(n network.Network, addr ma.Multiaddr) {
	logger.Tracef(
		"Host %s started listening on address %s", n.LocalPeer(), addr)
}

// ListenClose is called when network stops listening on an address
func (cm *ConnManager) ListenClose(n network.Network, addr ma.Multiaddr) {
	logger.Tracef(
		"Host %s stopped listening on address %s", n.LocalPeer(), addr)
}

// Connected is called when a connection opened
func (cm *ConnManager) Connected(n network.Network, c network.Conn) {
	logger.Tracef(
		"Host %s connected to peer %s", n.LocalPeer(), c.RemotePeer())

	if cm.connectHandler != nil {
		cm.connectHandler(c.RemotePeer())
	}
}

// Disconnected is called when a connection closed
func (cm *ConnManager) Disconnected(_ network.Network, c network.Conn) {
	logger.Tracef("Host %s disconnected from peer %s", c.LocalPeer(), c.RemotePeer())

	cm.Unprotect(c.RemotePeer(), "")
	if cm.disconnectHandler != nil {
		cm.disconnectHandler(c.RemotePeer())
	}
}
