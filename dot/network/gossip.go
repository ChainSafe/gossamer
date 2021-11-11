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

package network

import (
	"sync"

	"github.com/ChainSafe/gossamer/internal/log"
)

// gossip submodule
type gossip struct {
	logger log.LeveledLogger
	seen   *sync.Map
}

// newGossip creates a new gossip message tracker
func newGossip() *gossip {
	return &gossip{
		logger: log.NewFromGlobal(log.AddContext("module", "gossip")),
		seen:   &sync.Map{},
	}
}

// hasSeen broadcasts messages that have not been seen
func (g *gossip) hasSeen(msg NotificationsMessage) bool { //nolint
	// check if message has not been seen
	if seen, ok := g.seen.Load(msg.Hash()); !ok || !seen.(bool) {
		// set message to has been seen
		g.seen.Store(msg.Hash(), true)
		return false
	}

	return true
}
