// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package rpc

import (
	"context"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/dot/rpc/modules"
	"github.com/ChainSafe/gossamer/lib/common"
	libutils "github.com/ChainSafe/gossamer/lib/utils"
	"github.com/ChainSafe/gossamer/tests/utils"
	"github.com/ChainSafe/gossamer/tests/utils/config"
	"github.com/ChainSafe/gossamer/tests/utils/node"
	"github.com/ChainSafe/gossamer/tests/utils/retry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const peerIDRegex = `^[a-zA-Z0-9]{52}$`

func TestSystemRPC(t *testing.T) {
	if utils.MODE != rpcSuite {
		t.Log("Going to skip RPC suite tests")
		return
	}

	const numberOfNodes = 3

	genesisPath := libutils.GetGssmrGenesisRawPathTest(t)
	config := config.CreateDefault(t)
	nodes := node.MakeNodes(t, numberOfNodes, node.SetGenesis(genesisPath), node.SetConfig(config))

	ctx, cancel := context.WithCancel(context.Background())
	nodes.InitAndStartTest(ctx, t, cancel)

	t.Run("system_health", func(t *testing.T) {
		t.Parallel()

		const method = "system_health"
		const params = "{}"

		expected := modules.SystemHealthResponse{
			Peers:           numberOfNodes - 1,
			IsSyncing:       true,
			ShouldHavePeers: true,
		}

		var response modules.SystemHealthResponse
		err := retry.UntilOK(ctx, time.Second, func() (ok bool) {
			getResponseCtx, getResponseCancel := context.WithTimeout(ctx, time.Second)
			err := getResponse(getResponseCtx, method, params, &response)
			getResponseCancel()

			require.NoError(t, err)
			return response.Peers == expected.Peers
		})
		require.NoError(t, err)

		assert.Equal(t, expected, response)
	})

	t.Run("system_peers", func(t *testing.T) {
		t.Parallel()

		const method = "system_peers"
		const params = "{}"

		var response modules.SystemPeersResponse
		retry.UntilOK(ctx, time.Second, func() (ok bool) {
			getResponseCtx, getResponseCancel := context.WithTimeout(ctx, time.Second)
			err := getResponse(getResponseCtx, method, params, &response)
			getResponseCancel()

			require.NoError(t, err)

			if len(response) != numberOfNodes-1 {
				return false
			}

			for _, peer := range response {
				if peer.PeerID == "" {
					return false
				}
			}
			return true
		})

		// Check randomly generated peer IDs and clear this field
		for i := range response {
			assert.Regexp(t, peerIDRegex, response[i].PeerID)
			response[i].PeerID = ""
		}

		expectedBestHash := common.Hash{
			0xb5, 0xd8, 0xb5, 0xdd, 0xc7, 0xbb, 0x57, 0x64,
			0x00, 0x14, 0x28, 0x38, 0x5f, 0x23, 0xf2, 0x93,
			0x52, 0x11, 0xc7, 0x0c, 0xd8, 0xce, 0x6d, 0x0e,
			0x10, 0x96, 0xc7, 0xa7, 0xc3, 0x2b, 0x92, 0x6c}
		expectedResponse := modules.SystemPeersResponse{
			{Roles: 4, BestHash: expectedBestHash, BestNumber: 0},
			{Roles: 4, BestHash: expectedBestHash, BestNumber: 0},
		}
		assert.Equal(t, expectedResponse, response)
	})

	t.Run("system_networkState", func(t *testing.T) {
		t.Parallel()

		const method = "system_networkState"
		const params = "{}"

		getResponseCtx, getResponseCancel := context.WithTimeout(ctx, time.Second)
		defer getResponseCancel()
		var response modules.SystemNetworkStateResponse
		err := getResponse(getResponseCtx, method, params, &response)

		require.NoError(t, err)

		assert.Regexp(t, peerIDRegex, response.NetworkState.PeerID)
		response.NetworkState.PeerID = ""

		assert.NotEmpty(t, response.NetworkState.Multiaddrs)
		for _, addr := range response.NetworkState.Multiaddrs {
			assert.Regexp(t, "^/ip[4|6]/.+/tcp/[0-9]{1,5}/p2p/[a-zA-Z0-9]{52}$", addr)
		}
		response.NetworkState.Multiaddrs = nil

		// Ensure we don't need to assert other fields
		expectedResponse := modules.SystemNetworkStateResponse{}
		assert.Equal(t, expectedResponse, response)
	})

	t.Run("system_name", func(t *testing.T) {
		t.Parallel()
		t.Skip("test not implemented")
	})

	t.Run("system_version", func(t *testing.T) {
		t.Parallel()
		t.Skip("test not implemented")
	})

	t.Run("system_chain", func(t *testing.T) {
		t.Parallel()
		t.Skip("test not implemented")
	})

	t.Run("system_properties", func(t *testing.T) {
		t.Parallel()
		t.Skip("test not implemented")
	})

	t.Run("system_addReservedPeer", func(t *testing.T) {
		t.Parallel()
		t.Skip("test not implemented")
	})

	t.Run("system_removeReservedPeer", func(t *testing.T) {
		t.Parallel()
		t.Skip("test not implemented")
	})

	t.Run("system_nodeRoles", func(t *testing.T) {
		t.Parallel()
		t.Skip("test not implemented")
	})

	t.Run("system_accountNextIndex", func(t *testing.T) {
		t.Parallel()
		t.Skip("test not implemented")
	})
}
