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

package modules

import (
	"math/big"
	"net/http"

	"github.com/ChainSafe/gossamer/common"
	"github.com/ChainSafe/gossamer/internal/api"
)

// SystemModule is an RPC module providing access to core API points.
type SystemModule struct {
	api *api.Api
}

// EmptyRequest represents an RPC request with no fields
type EmptyRequest struct{}

type StringResponse string

type SystemHealthResponse struct {
	Peers           int  `json:"peers"`
	IsSyncing       bool `json:"isSyncing"`
	ShouldHavePeers bool `json:"shouldHavePeers"`
}

type SystemNetworkStateResponse struct {
	Id string `json:"Id"`
}

// TODO: This should probably live elsewhere
type Peer struct {
	PeerId          string      `json:"peerId"`
	Roles           string      `json:"roles"`
	ProtocolVersion int         `json:"protocolVersion"`
	BestHash        common.Hash `json:"bestHash"`
	BestNumber      *big.Int    `json:"bestNumber"`
}

type SystemPeersResponse struct {
	Peers []string `json:"peers"`
}

type SystemPropertiesResponse struct {
	Ss58Format    int    `json:"ss58Format"`
	TokenDecimals int    `json:"tokenDecimals"`
	TokenSymbol   string `json:"tokenSymbol"`
}

// NewSystemModule creates a new net API instance.
func NewSystemModule(api *api.Api) *SystemModule {
	return &SystemModule{
		api: api,
	}
}
func (sm *SystemModule) Chain(r *http.Request, req *EmptyRequest, res *StringResponse) {
	*res = "not yet implemented"

}

func (sm *SystemModule) Health(r *http.Request, req *EmptyRequest, res *SystemHealthResponse) {
	res.Peers = len(sm.api.P2pSystem.Peers())
	res.IsSyncing = sm.api.P2pSystem.IsSyncing()
	res.ShouldHavePeers = sm.api.P2pSystem.ShouldHavePeers()
}

func (sm *SystemModule) Name(r *http.Request, req *EmptyRequest, res *StringResponse) {
	*res = "not yet implemented"
}

func (sm *SystemModule) NetworkState(r *http.Request, req *EmptyRequest, res *SystemNetworkStateResponse) {
	res.Id = sm.api.P2pSystem.ID()
}

func (sm *SystemModule) Peers(r *http.Request, req *EmptyRequest, res *SystemPeersResponse) {
	res.Peers = sm.api.P2pSystem.Peers()
}

func (sm *SystemModule) Properties(r *http.Request, req *EmptyRequest, res *SystemPropertiesResponse) {
	return
}

func (sm *SystemModule) Version(r *http.Request, req *EmptyRequest, res *StringResponse) {
	*res = "not yet implemented"
	return
}
