// Copyright 2019 ChainSafe Systems (ON) Corp.
// This file is part of gossamer.

// The gossamer library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

// The gossamer library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.

// You should have received a copy of the GNU Lesser General Public License
// along with the gossamer library. If not, see <http://www.gnu.org/licenses/>.

package api

import (
	"net/http"

	"github.com/ChainSafe/gossamer/p2p"
)

// PublicRPC offers network related RPC methods
type PublicRPC struct {
	Net *p2p.Service
}

type Uint uint

type PublicRPCResponse struct {
	Count Uint
}

type PublicRPCRequest struct{}

// NewPublicNetAPI creates a new net API instance.
func NewPublicRPC(net *p2p.Service) *PublicRPC {
	return &PublicRPC{
		Net: net,
	}
}

// PeerCount returns the number of connected peers
func (s *PublicRPC) PeerCount(r *http.Request, args *PublicRPCRequest, res *PublicRPCResponse) error {
	res.Count = Uint(s.Net.PeerCount())
	return nil
}
