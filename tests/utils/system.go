// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package utils

import (
	"context"
	"testing"

	"github.com/ChainSafe/gossamer/dot/rpc/modules"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/stretchr/testify/require"
)

// GetPeers calls the endpoint system_peers
func GetPeers(ctx context.Context, t *testing.T, node *Node) []common.PeerInfo {
	endpoint := NewEndpoint(node.RPCPort)
	const method = "system_peers"
	const params = "[]"
	respBody, err := PostRPC(ctx, endpoint, method, params)
	require.NoError(t, err)

	resp := new(modules.SystemPeersResponse)
	err = DecodeRPC(t, respBody, resp)
	require.NoError(t, err)
	require.NotNil(t, resp)

	return *resp
}
