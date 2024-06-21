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
	"github.com/ChainSafe/gossamer/tests/utils/config"
	"github.com/ChainSafe/gossamer/tests/utils/node"
	"github.com/ChainSafe/gossamer/tests/utils/retry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const peerIDRegex = `^[a-zA-Z0-9]{52}$`

func TestSystemRPC(t *testing.T) { //nolint:tparallel
	const testTimeout = 8 * time.Minute
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)

	const numberOfNodes = 3

	genesisPath := libutils.GetWestendLocalRawGenesisPath(t)
	tomlConfig := config.Default()
	tomlConfig.Network.MinPeers = 1
	tomlConfig.Network.MaxPeers = 3
	tomlConfig.ChainSpec = genesisPath
	nodes := node.MakeNodes(t, numberOfNodes, tomlConfig)

	nodes.InitAndStartTest(ctx, t, cancel)

	t.Run("system_health", func(t *testing.T) {
		t.Parallel()

		const method = "system_health"
		const params = "{}"

		expected := modules.SystemHealthResponse{
			Peers:           numberOfNodes - 1,
			ShouldHavePeers: true,
		}

		var response modules.SystemHealthResponse
		err := retry.UntilOK(ctx, time.Second, func() (ok bool, err error) {
			getResponseCtx, getResponseCancel := context.WithTimeout(ctx, time.Second)
			err = getResponse(getResponseCtx, method, params, &response)
			getResponseCancel()
			if err != nil {
				return false, err
			}
			return response.Peers == expected.Peers, nil
		})
		require.NoError(t, err)

		// IsSyncing can be true or false
		response.IsSyncing = false

		assert.Equal(t, expected, response)
	})

	t.Run("system_peers", func(t *testing.T) {
		t.Parallel()

		// Wait for N-1 peers connected and no syncing
		err := retry.UntilOK(ctx, time.Second, func() (ok bool, err error) {
			getResponseCtx, getResponseCancel := context.WithTimeout(ctx, time.Second)
			const method = "system_health"
			const params = "{}"
			var healthResponse modules.SystemHealthResponse
			err = getResponse(getResponseCtx, method, params, &healthResponse)
			getResponseCancel()
			if err != nil {
				return false, err // error and stop retrying
			}

			ok = healthResponse.Peers == numberOfNodes-1 && !healthResponse.IsSyncing
			return ok, nil
		})
		require.NoError(t, err)

		var response modules.SystemPeersResponse
		// Wait for N-1 peers with peer IDs set
		err = retry.UntilOK(ctx, time.Second, func() (ok bool, err error) {
			t.Log("TestSystemRPC/system_peers 000000000000000000000")
			getResponseCtx, getResponseCancel := context.WithTimeout(ctx, time.Second)
			const method = "system_peers"
			const params = "{}"
			err = getResponse(getResponseCtx, method, params, &response)
			getResponseCancel()
			if err != nil {
				t.Logf("TestSystemRPC/system_peers error is not nil: %s", err)
				return false, err // error and stop retrying
			}

			if len(response) != numberOfNodes-1 {
				t.Logf("TestSystemRPC/system_peers len(response) is different from numberOfNodes-1: %d", numberOfNodes-1)
				return false, nil // retry
			}

			for _, peer := range response {
				// wait for all peers to have the same best block number
				if peer.PeerID == "" || peer.BestHash.IsEmpty() {
					t.Logf("TestSystemRPC/system_peers peer.PeerID is %s and peer.BestHash.IsEmpty: %v",
						peer.PeerID, peer.BestHash.IsEmpty())
					return false, nil // retry
				}
			}

			return true, nil // success, stop retrying
		})
		require.NoError(t, err)

		expectedResponse := modules.SystemPeersResponse{
			// Assert they all have the same best block number and hash
			{Role: 4, PeerID: ""},
			{Role: 4, PeerID: ""},
		}
		for i := range response {
			// Check randomly generated peer IDs and clear them
			assert.Regexp(t, peerIDRegex, response[i].PeerID)
			response[i].PeerID = ""
			// TODO assert these are all the same,
			// see https://github.com/ChainSafe/gossamer/issues/2498
			response[i].BestHash = common.Hash{}
			response[i].BestNumber = 0
		}

		assert.Equal(t, expectedResponse, response)
	})

	t.Run("system_networkState", func(t *testing.T) {
		t.Parallel()

		const method = "system_networkState"
		const params = "{}"

		var response modules.SystemNetworkStateResponse
		fetchWithTimeout(ctx, t, method, params, &response)

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
