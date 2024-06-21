// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package rpc

import (
	"context"
	"fmt"

	"github.com/ChainSafe/gossamer/dot/rpc/modules"
	"github.com/ChainSafe/gossamer/lib/common"
)

// GetPeers calls the endpoint system_peers
func GetPeers(ctx context.Context, rpcPort string) (peers []common.PeerInfo, err error) {
	endpoint := NewEndpoint(rpcPort)
	const method = "system_peers"
	const params = "[]"
	respBody, err := Post(ctx, endpoint, method, params)
	if err != nil {
		return nil, fmt.Errorf("cannot post RPC: %w", err)
	}

	var peersResponse modules.SystemPeersResponse
	err = Decode(respBody, &peersResponse)
	if err != nil {
		return nil, fmt.Errorf("cannot decode RPC: %w", err)
	}

	return peersResponse, nil
}

// GetHealth sends an RPC request to `system_health`.
func GetHealth(ctx context.Context, address string) (
	health modules.SystemHealthResponse, err error) {
	const method = "system_health"
	const params = "{}"
	respBody, err := Post(ctx, address, method, params)
	if err != nil {
		return health, fmt.Errorf("cannot post RPC: %w", err)
	}

	err = Decode(respBody, &health)
	if err != nil {
		return health, fmt.Errorf("cannot decode RPC: %w", err)
	}
	fmt.Println("Peers number:", health.Peers)
	return health, nil
}
