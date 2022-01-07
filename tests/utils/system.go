// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package utils

import (
	"testing"

	"github.com/ChainSafe/gossamer/dot/rpc/modules"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/stretchr/testify/require"
)

// GetPeers calls the endpoint system_peers
func GetPeers(t *testing.T, node *Node) []common.PeerInfo {
	respBody, err := PostRPC("system_peers", NewEndpoint(node.RPCPort), "[]")
	require.NoError(t, err)

	resp := new(modules.SystemPeersResponse)
	err = DecodeRPC(t, respBody, resp)
	require.NoError(t, err)
	require.NotNil(t, resp)

	return *resp
}
