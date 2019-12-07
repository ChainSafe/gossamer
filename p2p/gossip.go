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
// GNU Lesser General Public License for more detailg.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the gossamer library. If not, see <http://www.gnu.org/licenses/>.

package p2p

import (
	log "github.com/ChainSafe/log15"
	"github.com/libp2p/go-libp2p-core/network"
)

// gossip submodule
type gossip struct {
	host         *host
	hasGossipped map[string]bool
}

// newGossip creates a new gossip instance from the host
func newGossip(host *host) (g *gossip, err error) {
	g = &gossip{
		host:         host,
		hasGossipped: make(map[string]bool),
	}
	return g, err
}

// handleMessage gossips messages that have not already been gossipped
func (g *gossip) handleMessage(stream network.Stream, msg Message) {

	// check if message has been gossipped
	if !g.hasGossipped[msg.Id()] {

		// broadcast message to peers if message has not been gossipped
		g.sendMessage(stream, msg)

		// update message to gossipped
		g.hasGossipped[msg.Id()] = true

	}
}

// sendMessage broadcasts the message to connected peers
func (g *gossip) sendMessage(stream network.Stream, msg Message) {

	// loop through connected peers
	for _, peer := range g.host.peers() {

		// send message to each connected peer
		err := g.host.send(peer, msg)
		if err != nil {
			log.Error("Failed to send message during gossip", "err", err)
		}
	}
}
