// Copyright 2019 ChainSafe Systems (ON) Corp.
// This file is part of gossamer.
//
// The gossamer library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The gossamer library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the gossamer library. If not, see <http://www.gnu.org/licenses/>.

package network

import (
	"context"
	"math/rand"
	"sync"

	"github.com/libp2p/go-libp2p-core/connmgr"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"

	ma "github.com/multiformats/go-multiaddr"
)

// ConnManager implements connmgr.ConnManager
type ConnManager struct {
	sync.Mutex
	host              *host
	min, max          int
	disconnectHandler func(peer.ID)

	// closeHandlerMap contains close handler corresponding to a protocol.
	closeHandlerMap map[protocol.ID]func(peerID peer.ID)

	// protectedPeers contains a list of peers that are protected from pruning
	// when we reach the maximum numbers of peers.
	protectedPeers *sync.Map // map[peer.ID]struct{}

	// persistentPeers contains peers we should remain connected to.
	persistentPeers *sync.Map // map[peer.ID]struct{}
}

func newConnManager(min, max int) *ConnManager {
	return &ConnManager{
		min:             min,
		max:             max,
		closeHandlerMap: make(map[protocol.ID]func(peerID peer.ID)),
		protectedPeers:  new(sync.Map),
		persistentPeers: new(sync.Map),
	}
}

// Notifee is used to monitor changes to a connection
func (cm *ConnManager) Notifee() network.Notifiee {
	nb := new(network.NotifyBundle)

	nb.ListenF = cm.Listen
	nb.ListenCloseF = cm.ListenClose
	nb.ConnectedF = cm.Connected
	nb.DisconnectedF = cm.Disconnected
	nb.OpenedStreamF = cm.OpenedStream
	nb.ClosedStreamF = cm.ClosedStream

	return nb
}

// TagPeer peer
func (*ConnManager) TagPeer(peer.ID, string, int) {}

// UntagPeer peer
func (*ConnManager) UntagPeer(peer.ID, string) {}

// UpsertTag peer
func (*ConnManager) UpsertTag(peer.ID, string, func(int) int) {}

// GetTagInfo peer
func (*ConnManager) GetTagInfo(peer.ID) *connmgr.TagInfo { return &connmgr.TagInfo{} }

// TrimOpenConns peer
func (*ConnManager) TrimOpenConns(ctx context.Context) {}

// Protect peer will add the given peer to the protectedPeerMap which will
// protect the peer from pruning.
func (cm *ConnManager) Protect(id peer.ID, tag string) {
	cm.protectedPeers.Store(id, struct{}{})
}

// Unprotect peer will remove the given peer from prune protection.
// returns true if we have successfully removed the peer from the
// protectedPeerMap. False otherwise.
func (cm *ConnManager) Unprotect(id peer.ID, tag string) bool {
	_, wasDeleted := cm.protectedPeers.LoadAndDelete(id)
	return wasDeleted
}

// Close peer
func (*ConnManager) Close() error { return nil }

// IsProtected returns whether the given peer is protected from pruning or not.
func (cm *ConnManager) IsProtected(id peer.ID, tag string) (protected bool) {
	_, ok := cm.protectedPeers.Load(id)
	return ok
}

// Listen is called when network starts listening on an address
func (cm *ConnManager) Listen(n network.Network, addr ma.Multiaddr) {
	logger.Trace(
		"Started listening",
		"host", n.LocalPeer(),
		"address", addr,
	)
}

// ListenClose is called when network stops listening on an address
func (cm *ConnManager) ListenClose(n network.Network, addr ma.Multiaddr) {
	logger.Trace(
		"Stopped listening",
		"host", n.LocalPeer(),
		"address", addr,
	)
}

// returns a slice of peers that are unprotected and may be pruned.
func (cm *ConnManager) unprotectedPeers(peers []peer.ID) []peer.ID {
	unprot := []peer.ID{}
	for _, id := range peers {
		if !cm.IsProtected(id, "") && !cm.isPersistent(id) {
			unprot = append(unprot, id)
		}
	}

	return unprot
}

// Connected is called when a connection opened
func (cm *ConnManager) Connected(n network.Network, c network.Conn) {
	logger.Trace(
		"Connected to peer",
		"host", c.LocalPeer(),
		"peer", c.RemotePeer(),
	)

	cm.Lock()
	defer cm.Unlock()

	// TODO: this should be updated to disconnect from (total_peers - maximum) peers, instead of just one peer
	if len(n.Peers()) > cm.max {
		unprotPeers := cm.unprotectedPeers(n.Peers())
		if len(unprotPeers) == 0 {
			return
		}

		// TODO: change to crypto/rand
		i := rand.Intn(len(unprotPeers)) //nolint

		logger.Trace("Over max peer count, disconnecting from random unprotected peer", "peer", unprotPeers[i])
		err := n.ClosePeer(unprotPeers[i])
		if err != nil {
			logger.Trace("failed to close connection to peer", "peer", unprotPeers[i], "num peers", len(n.Peers()))
		}
	}
}

// Disconnected is called when a connection closed
func (cm *ConnManager) Disconnected(n network.Network, c network.Conn) {
	logger.Trace(
		"Disconnected from peer",
		"host", c.LocalPeer(),
		"peer", c.RemotePeer(),
	)

	cm.Unprotect(c.RemotePeer(), "")
	if cm.disconnectHandler != nil {
		cm.disconnectHandler(c.RemotePeer())
	}

	if !cm.isPersistent(c.RemotePeer()) {
		return
	}

	addrs := cm.host.h.Peerstore().Addrs(c.RemotePeer())
	info := peer.AddrInfo{
		ID:    c.RemotePeer(),
		Addrs: addrs,
	}

	err := cm.host.connect(info)
	if err != nil {
		logger.Warn("failed to reconnect to persistent peer", "peer", c.RemotePeer(), "error", err)
	}

	// TODO: if number of peers falls below the min desired peer count, we should try to connect to previously discovered peers
}

func (cm *ConnManager) registerDisconnectHandler(cb func(peer.ID)) {
	cm.disconnectHandler = cb
}

// OpenedStream is called when a stream opened
func (cm *ConnManager) OpenedStream(n network.Network, s network.Stream) {
	logger.Trace(
		"Opened stream",
		"host", s.Conn().LocalPeer(),
		"peer", s.Conn().RemotePeer(),
		"protocol", s.Protocol(),
	)
}

func (cm *ConnManager) registerCloseHandler(protocolID protocol.ID, cb func(id peer.ID)) {
	cm.closeHandlerMap[protocolID] = cb
}

// ClosedStream is called when a stream closed
func (cm *ConnManager) ClosedStream(n network.Network, s network.Stream) {
	logger.Trace(
		"Closed stream",
		"host", s.Conn().LocalPeer(),
		"peer", s.Conn().RemotePeer(),
		"protocol", s.Protocol(),
	)

	cm.Lock()
	defer cm.Unlock()
	if closeCB, ok := cm.closeHandlerMap[s.Protocol()]; ok {
		closeCB(s.Conn().RemotePeer())
	}
}

func (cm *ConnManager) isPersistent(p peer.ID) bool {
	_, ok := cm.persistentPeers.Load(p)
	return ok
}
