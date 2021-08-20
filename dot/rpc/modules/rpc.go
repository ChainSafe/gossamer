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

package modules

import (
	"net/http"
)

var (
	// UnsafeMethods is a list of all unsafe rpc methods of https://github.com/w3f/PSPs/blob/master/PSPs/drafts/psp-6.md
	UnsafeMethods = []string{
		"system_addReservedPeer",
		"system_removeReservedPeer",
		"author_submitExtrinsic",
		"author_removeExtrinsic",
		"author_insertKey",
		"author_rotateKeys",
		"state_getPairs",
		"state_getKeysPaged",
		"state_queryStorage",
	}

	// AliasesMethods is a map that links the original methods to their aliases
	AliasesMethods = map[string]string{
		"chain_getHead":          "chain_getBlockHash",
		"account_nextIndex":      "system_accountNextIndex",
		"chain_getFinalisedHead": "chain_getFinalizedHead",
	}
)

// RPCModule is a RPC module providing access to RPC methods
type RPCModule struct {
	rPCAPI RPCAPI
}

// MethodsResponse struct representing methods
type MethodsResponse struct {
	Methods []string `json:"methods"`
}

// NewRPCModule creates a new RPC api module
func NewRPCModule(rpcapi RPCAPI) *RPCModule {
	return &RPCModule{
		rPCAPI: rpcapi,
	}
}

// Methods responds with list of methods available via RPC call
func (rm *RPCModule) Methods(r *http.Request, req *EmptyRequest, res *MethodsResponse) error {
	res.Methods = rm.rPCAPI.Methods()

	return nil
}

// IsUnsafe returns true if the `name` has the  suffix
func IsUnsafe(name string) bool {
	for _, unsafe := range UnsafeMethods {
		if name == unsafe {
			return true
		}
	}

	return false
}
