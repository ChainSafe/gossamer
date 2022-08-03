// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package rpc

import (
	"context"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/dot/config/toml"
	"github.com/ChainSafe/gossamer/dot/rpc/modules"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/tests/utils"
	"github.com/ChainSafe/gossamer/tests/utils/node"
	"github.com/ChainSafe/gossamer/tests/utils/rpc"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStableNetworkRPC(t *testing.T) {
	if utils.MODE != "rpc" {
		t.Skip("RPC tests are disabled, going to skip.")
	}

	const numberOfNodes = 3
	config := toml.Config{
		RPC: toml.RPCConfig{
			Enabled: true,
			Modules: []string{"system", "author", "chain"},
		},
		Core: toml.CoreConfig{
			Roles: types.FullNodeRole,
		},
	}

	nodes := make(node.Nodes, numberOfNodes)
	for i := range nodes {
		nodes[i] = node.New(t, config, node.SetIndex(i))
	}

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	for _, node := range nodes {
		node.InitAndStartTest(ctx, t, cancel)
		const timeBetweenStart = 0 * time.Second
		timer := time.NewTimer(timeBetweenStart)
		select {
		case <-timer.C:
		case <-ctx.Done():
			if !timer.Stop() {
				<-timer.C
			}
			return
		}
	}

	for _, node := range nodes {
		node := node
		t.Run(node.String(), func(t *testing.T) {
			t.Parallel()
			endpoint := rpc.NewEndpoint(node.RPCPort())

			t.Run("system_health", func(t *testing.T) {
				t.Parallel()

				var response modules.SystemHealthResponse

				fetchWithTimeoutFromEndpoint(t, endpoint, "system_health", "{}", &response)

				expectedResponse := modules.SystemHealthResponse{
					Peers:           numberOfNodes - 1,
					IsSyncing:       true,
					ShouldHavePeers: true,
				}
				assert.Equal(t, expectedResponse, response)
			})

			t.Run("system_networkState", func(t *testing.T) {
				t.Parallel()

				var response modules.SystemNetworkStateResponse

				fetchWithTimeoutFromEndpoint(t, endpoint, "system_networkState", "{}", &response)

				// TODO assert response
			})

			t.Run("system_peers", func(t *testing.T) {
				t.Parallel()

				var response modules.SystemPeersResponse

				fetchWithTimeoutFromEndpoint(t, endpoint, "system_peers", "{}", &response)

				assert.GreaterOrEqual(t, len(response), numberOfNodes-2)

				// TODO assert response
			})
		})
	}
}

func fetchWithTimeoutFromEndpoint(t *testing.T, endpoint, method,
	params string, target interface{}) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	body, err := rpc.Post(ctx, endpoint, method, params)
	cancel()
	require.NoError(t, err)

	err = rpc.Decode(body, target)
	require.NoError(t, err)
}
