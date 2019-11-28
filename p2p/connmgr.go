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

package p2p

import (
	"context"

	log "github.com/ChainSafe/log15"

	"github.com/libp2p/go-libp2p-core/connmgr"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"

	ma "github.com/multiformats/go-multiaddr"
)

// ConnManager implement connmgr.ConnManager
// https://godoc.org/github.com/libp2p/go-libp2p-core/connmgr#ConnManager
type ConnManager struct{}

// Notifee is used to monitor changes to a connection
func (cm ConnManager) Notifee() network.Notifiee {
	nb := new(network.NotifyBundle)

	nb.ListenF = Listen
	nb.ListenCloseF = ListenClose
	nb.ConnectedF = Connected
	nb.DisconnectedF = Disconnected
	nb.OpenedStreamF = OpenedStream
	nb.ClosedStreamF = ClosedStream

	return nb
}

func (_ ConnManager) TagPeer(peer.ID, string, int)             {}
func (_ ConnManager) UntagPeer(peer.ID, string)                {}
func (_ ConnManager) UpsertTag(peer.ID, string, func(int) int) {}
func (_ ConnManager) GetTagInfo(peer.ID) *connmgr.TagInfo      { return &connmgr.TagInfo{} }
func (_ ConnManager) TrimOpenConns(ctx context.Context)        {}
func (_ ConnManager) Protect(peer.ID, string)                  {}
func (_ ConnManager) Unprotect(peer.ID, string) bool           { return false }
func (_ ConnManager) Close() error                             { return nil }

// Listen is called when network starts listening on an address
func Listen(n network.Network, address ma.Multiaddr) {
	log.Debug(
		"started listening",
		"host", n.LocalPeer(),
		"address", address,
	)
}

// ListenClose is called when network stops listening on an address
func ListenClose(n network.Network, address ma.Multiaddr) {
	log.Debug(
		"stopped listening",
		"host", n.LocalPeer(),
		"address", address,
	)
}

// Connected is called when a connection opened
func Connected(n network.Network, c network.Conn) {
	log.Debug(
		"connected",
		"host", c.LocalPeer(),
		"peer", c.RemotePeer(),
	)
}

// Disconnected is called when a connection closed
func Disconnected(n network.Network, c network.Conn) {
	log.Debug(
		"disconnected",
		"host", c.LocalPeer(),
		"peer", c.RemotePeer(),
	)
}

// OpenedStream is called when a stream opened
func OpenedStream(n network.Network, s network.Stream) {
	protocol := s.Protocol()
	if protocol != "" {
		log.Trace(
			"opened stream",
			"host", s.Conn().LocalPeer(),
			"peer", s.Conn().RemotePeer(),
			"protocol", protocol,
		)
	}
}

// ClosedStream is called when a stream closed
func ClosedStream(n network.Network, s network.Stream) {
	protocol := s.Protocol()
	if protocol != "" {
		log.Trace(
			"closed stream",
			"host", s.Conn().LocalPeer(),
			"peer", s.Conn().RemotePeer(),
			"protocol", protocol,
		)
	}
}
