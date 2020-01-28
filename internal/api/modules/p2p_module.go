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

package module

import (
	"github.com/ChainSafe/gossamer/p2p"
	log "github.com/ChainSafe/log15"
)

// P2pModule holds the fields for manipulating the API
type P2pModule struct {
	P2pAPI P2pAPI
}

// P2pAPI is the interface for the p2p package
type P2pAPI interface {
	Health() p2p.Health
	NetworkState() p2p.NetworkState
	Peers() []p2p.PeerInfo
}

// NewP2pModule implements P2pAPI
func NewP2pModule(p2pAPI P2pAPI) *P2pModule {
	return &P2pModule{p2pAPI}
}

// Health returns p2p service Health()
func (m *P2pModule) Health() p2p.Health {
	log.Debug("[rpc] Executing System.Health", "params", nil)
	return m.P2pAPI.Health()
}

// NetworkState returns p2p service NetworkState()
func (m *P2pModule) NetworkState() p2p.NetworkState {
	log.Debug("[rpc] Executing System.NetworkState", "params", nil)
	return m.P2pAPI.NetworkState()
}

// Peers returns p2p service Peers()
func (m *P2pModule) Peers() []p2p.PeerInfo {
	log.Debug("[rpc] Executing System.Peers", "params", nil)
	return m.P2pAPI.Peers()
}
