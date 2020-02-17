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
	"net/http"

	"github.com/ChainSafe/gossamer/common"
)

// NOT_IMPLEMENTED used as placeholder for not implemented yet funcs
const NOT_IMPLEMENTED = "not yet implemented"

// SystemModule is an RPC module providing access to core API points
type SystemModule struct {
	networkApi NetworkApi
}

// EmptyRequest represents an RPC request with no fields
type EmptyRequest struct{}

// StringResponse holds the string response
type StringResponse string

// SystemHealthResponse struct to marshal json
type SystemHealthResponse struct {
	Health common.Health `json:"health"`
}

// SystemNetworkStateResponse struct to marshal json
type SystemNetworkStateResponse struct {
	NetworkState common.NetworkState `json:"networkState"`
}

// SystemPeersResponse struct to marshal json
type SystemPeersResponse struct {
	Peers []common.PeerInfo `json:"peers"`
}

// SystemPropertiesResponse struct to marshal json
type SystemPropertiesResponse struct {
	Ss58Format    int    `json:"ss58Format"`
	TokenDecimals int    `json:"tokenDecimals"`
	TokenSymbol   string `json:"tokenSymbol"`
}

// NewSystemModule creates a new API instance
func NewSystemModule(net NetworkApi) *SystemModule {
	return &SystemModule{
		networkApi: net, // TODO: migrate to network state
	}
}

// Chain returns the runtime chain
func (sm *SystemModule) Chain(r *http.Request, req *EmptyRequest, res *StringResponse) error {
	*res = NOT_IMPLEMENTED
	return nil
}

// Name returns the runtime name
func (sm *SystemModule) Name(r *http.Request, req *EmptyRequest, res *StringResponse) error {
	*res = "gossamer v0.0"
	return nil
}

// Properties returns the runtime properties
func (sm *SystemModule) Properties(r *http.Request, req *EmptyRequest, res *StringResponse) error {
	*res = NOT_IMPLEMENTED
	return nil
}

// Version returns the runtime version
func (sm *SystemModule) Version(r *http.Request, req *EmptyRequest, res *StringResponse) error {
	*res = NOT_IMPLEMENTED
	return nil
}

// Health returns the information about the health of the network
func (sm *SystemModule) Health(r *http.Request, req *EmptyRequest, res *SystemHealthResponse) error {
	// TODO: rename to not have `Get` to match API
	health := sm.networkApi.Health()
	res.Health = *health
	return nil
}

// NetworkState returns the network state (basic information about the host)
func (sm *SystemModule) NetworkState(r *http.Request, req *EmptyRequest, res *SystemNetworkStateResponse) error {
	// TODO: rename to not have `Get` to match API
	networkState := sm.networkApi.NetworkState()
	res.NetworkState = *networkState
	return nil
}

// Peers returns peer information for each connected and confirmed peer
func (sm *SystemModule) Peers(r *http.Request, req *EmptyRequest, res *SystemPeersResponse) error {
	// TODO: rename to not have `Get` to match API
	peers := sm.networkApi.Peers()
	res.Peers = peers
	return nil
}
