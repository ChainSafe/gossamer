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
)

// Gossip describes the gossip submodule
type Gossip struct {
	host        *host
	msgReceived map[string]bool
}

// newGossip creates a new gossip instance from the host
func newGossip(host *host) (gossip *Gossip, err error) {
	gossip = &Gossip{
		host:        host,
		msgReceived: make(map[string]bool),
	}

	return gossip, err
}

// gossip broadcasts the message to all connected peers if gossip is enabled,
// message is valid message type, and message has not already been gossipped
func (g *Gossip) handleMessage(msg Message) {

	// exit if gossip is disabled
	if g.host.noGossip {
		return // exit
	}

	// check if valid message type and message has not already been gossipped
	if g.shouldGossip(msg) {

		// loop through connected peers
		for _, peer := range g.host.peers() {

			// send message to each connected peer
			err := g.host.send(peer, msg)
			if err != nil {
				log.Error("Failed to send message during gossip", "err", err)
			}
		}
	}
}

// shouldGossip checks if message has already been gossipped
func (g *Gossip) shouldGossip(msg Message) bool {

	// check if message stored in received message mapping
	if g.msgReceived[msg.Id()] {
		return false
	}

	// update message in received message mapping
	g.msgReceived[msg.Id()] = true

	return true
}
