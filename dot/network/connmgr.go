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
	"crypto/rand"
	"math/big"
	"sync"

	"github.com/libp2p/go-libp2p-core/connmgr"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
	ma "github.com/multiformats/go-multiaddr"

	"github.com/ChainSafe/gossamer/dot/peerset"
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

	peerSetHandler PeerSetHandler
}

func newConnManager(min, max int, peerSetCfg *peerset.ConfigSet) (*ConnManager, error) {
	psh, err := peerset.NewPeerSetHandler(peerSetCfg)
	if err != nil {
		return nil, err
	}

	return &ConnManager{
		min:             min,
		max:             max,
		closeHandlerMap: make(map[protocol.ID]func(peerID peer.ID)),
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

// Close peer
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
	logger.Tracef(
		"Host %s connected to peer %s", n.LocalPeer(), c.RemotePeer())

	cm.Lock()
	defer cm.Unlock()

	over := len(n.Peers()) - cm.max
	if over <= 0 {
		return
	}

	// if over the max peer count, disconnect from (total_peers - maximum) peers
	for i := 0; i < over; i++ {
		unprotPeers := cm.unprotectedPeers(n.Peers())
		if len(unprotPeers) == 0 {
			return
		}

		i, err := rand.Int(rand.Reader, big.NewInt(int64(len(unprotPeers))))
		if err != nil {
			logger.Errorf("error generating random number: %s", err)
			return
		}

		up := unprotPeers[i.Int64()]
		logger.Tracef("Over max peer count, disconnecting from random unprotected peer %s", up)
		err = n.ClosePeer(up)
		if err != nil {
			logger.Tracef("failed to close connection to peer %s", up)
		}
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

// OpenedStream is called when a stream opened
func (cm *ConnManager) OpenedStream(_ network.Network, s network.Stream) {
	logger.Tracef("Stream opened with peer %s using protocol %s",
		s.Conn().RemotePeer(), s.Protocol())
}

func (cm *ConnManager) registerCloseHandler(protocolID protocol.ID, cb func(id peer.ID)) {
	cm.closeHandlerMap[protocolID] = cb
}

// ClosedStream is called when a stream closed
func (cm *ConnManager) ClosedStream(_ network.Network, s network.Stream) {
	logger.Tracef("Stream closed with peer %s using protocol %s",
		s.Conn().RemotePeer(), s.Protocol())

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
