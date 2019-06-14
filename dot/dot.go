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

package dot

import (
	api "github.com/ChainSafe/gossamer/internal"
	"github.com/ChainSafe/gossamer/p2p"
	"github.com/ChainSafe/gossamer/polkadb"
	"github.com/ChainSafe/gossamer/rpc"
	log "github.com/ChainSafe/log15"
)

// Dot is a container for all the components of a node.
type Dot struct {
	P2P *p2p.Service      // P2P networking layer
	Db  *polkadb.BadgerDB // BadgerDB database
	// TODO: Pending runtime PR
	//runtime *runtime.Service // WASM execution runtime
	Api *api.Service    // Internal API service (utilized by RPC, etc.)
	Rpc *rpc.HttpServer // HTTP interface for RPC server

	stop chan struct{} // Used to signal node shutdown
}

// NewDot initializes a Dot with provided components.
func NewDot(p2p *p2p.Service, db * polkadb.BadgerDB, api *api.Service, rpc *rpc.HttpServer) *Dot {
	return &Dot{
		P2P: p2p,
		Db: db,
		Api: api,
		Rpc: rpc,
		stop: make(chan struct{}),
	}
}

// Start starts all services. API service is started last.
func (d *Dot) Start() {
	log.Debug("Starting core services.")
	d.P2P.Start()
	if d.Rpc != nil {
		d.Rpc.Start()
	}
	d.Wait()
}

// Wait is used to force the node to stay alive until a signal is passed into `Dot.stop`
func (d *Dot) Wait() {
	<- d.stop
}
