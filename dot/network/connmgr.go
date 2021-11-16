// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package network

import (
	"context"
	"crypto/rand"
	"math/big"
	"sync"

	"github.com/libp2p/go-libp2p-core/connmgr"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	ma "github.com/multiformats/go-multiaddr"

	"github.com/ChainSafe/gossamer/dot/peerset"
)

// ConnManager implements connmgr.ConnManager
type ConnManager struct {
	sync.Mutex
	host              *host
	min, max          int
	connectHandler    func(peer.ID)
	disconnectHandler func(peer.ID)

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
	cm.connectHandler(c.RemotePeer())

	cm.Lock()
	defer cm.Unlock()

	over := len(n.Peers()) - cm.max
	if over <= 0 {
		return
	}

	// TODO: peer scoring doesn't seem to prevent us from going over the max.
	// if over the max peer count, disconnect from (total_peers - maximum) peers
	// (#2039)
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
	cm.disconnectHandler(c.RemotePeer())
}

// OpenedStream is called when a stream is opened
func (cm *ConnManager) OpenedStream(_ network.Network, s network.Stream) {
	logger.Tracef("Stream opened with peer %s using protocol %s",
		s.Conn().RemotePeer(), s.Protocol())
}

// ClosedStream is called when a stream is closed
func (cm *ConnManager) ClosedStream(_ network.Network, s network.Stream) {
	logger.Tracef("Stream closed with peer %s using protocol %s",
		s.Conn().RemotePeer(), s.Protocol())
}

func (cm *ConnManager) isPersistent(p peer.ID) bool {
	_, ok := cm.persistentPeers.Load(p)
	return ok
}
