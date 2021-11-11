// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: Apache-2.0

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
