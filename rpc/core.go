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

package rpc

import (
	api "github.com/ChainSafe/gossamer/internal"
	"net/http"
)

// CoreModule is an RPC module providing access to core API points.
type CoreModule struct {
	api *api.Service
}

// PublicP2PRequest represents RPC request type
type EmptyRequest struct{}

// CoreVersionResponse represents response from RPC call
type CoreVersionResponse struct {
	Version string
}

// NewPublicRPC creates a new net API instance.
func NewCoreModule(api *api.Service) *CoreModule {
	return &CoreModule{
		api: api,
	}
}

// PeerCount returns the number of connected peers
func (s *CoreModule) Version(r *http.Request, args *EmptyRequest, res *CoreVersionResponse) error {
	res.Version = s.api.Core.Version()
	return nil
}
