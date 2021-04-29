// Copyright 2020 ChainSafe Systems (ON) Corp.
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

package grandpa

import (
	"sync"
	"sync/atomic"
)

type catchUp struct {
	isStarted *atomic.Value
	peers     *sync.Map //map[peer.ID]struct{}
}

func newCatchUp() *catchUp {
	isStarted := new(atomic.Value).Store(false)

	return &catchUp{
		isStarted: isStarted,
		peers:     new(sync.Map),
	}
}

func (c *catchUp) addPeer(id peer.ID) {
	c.peers.Store(id, struct{}{})
}

func (c *catchUp) beginCatchUp(setID, round uint64) {
	resp, err := h.grandpa.network.SendCatchUpRequest(from, messageID, &ConsensusMessage{})
}
