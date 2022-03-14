// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package utils

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
	respBody, err := PostRPC(ctx, endpoint, method, params)
	if err != nil {
		return nil, fmt.Errorf("cannot post RPC: %w", err)
	}

	var peersResponse modules.SystemPeersResponse
	err = DecodeRPC(respBody, &peersResponse)
	if err != nil {
		return nil, fmt.Errorf("cannot decode RPC: %w", err)
	}

	return peersResponse, nil
}
