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
	log "github.com/ChainSafe/log15"
)

type P2pModule struct {
	P2p P2pApi
}

// P2pApi is the interface expected to implemented by `p2p` package
type P2pApi interface {
	PeerCount() int
	Peers() []string
	NoBootstrapping() bool
	ID() string
}

func NewP2PModule(p2papi P2pApi) *P2pModule {
	return &P2pModule{p2papi}
}

func (p *P2pModule) PeerCount() int {
	log.Debug("[rpc] Executing System.PeerCount", "params", nil)
	return len(p.Peers())
}

// Peers of the node
func (p *P2pModule) Peers() []string {
	log.Debug("[rpc] Executing System.Peers", "params", nil)
	return p.P2p.Peers()
}

func (p *P2pModule) NoBootstrapping() bool {
	log.Debug("[rpc] Executing System.NoBootstrapping", "params", nil)
	return p.P2p.NoBootstrapping()
}

func (p *P2pModule) ID() string {
	log.Debug("[rpc] Executing System.NetworkState", "params", nil)
	return p.P2p.ID()
}

func (p *P2pModule) IsSyncing() bool {
	return false
}
