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
)

// Dot is a container on which services can be registered.
type Dot struct {
	P2P *p2p.Service      // Currently running P2P networking layer
	Db  *polkadb.BadgerDB //BadgerDB database
	// TODO: Pending runtime PR
	//runtime *runtime.Service // WASM execution runtime
	Api *api.Service // Internal API service (utilized by RPC, etc.)
	Rpc *rpc.HTTPServer // HTTP interface for RPC server
}

func NewDot(p2p *p2p.Service, db * polkadb.BadgerDB, api *api.Service, rpc *rpc.HTTPServer) *Dot {
	return &Dot{
		p2p,
		db,
		api,
		rpc,
	}
}

func (d *Dot) Setup() {

}

// Start starts all services. API service is started last.
func (d *Dot) Start() {
	d.P2P.Start()
	d.Db.Start()
	d.Api.Start()
}
