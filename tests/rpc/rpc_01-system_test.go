// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package rpc

import (
	"context"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/dot/rpc/modules"
	libutils "github.com/ChainSafe/gossamer/lib/utils"
	"github.com/ChainSafe/gossamer/tests/utils"
	"github.com/ChainSafe/gossamer/tests/utils/config"
	"github.com/ChainSafe/gossamer/tests/utils/node"
	"github.com/stretchr/testify/require"
)

func TestSystemRPC(t *testing.T) {
	if utils.MODE != rpcSuite {
		t.Log("Going to skip RPC suite tests")
		return
	}

	testCases := []*testCase{
		{ //TODO
			description: "test system_name",
			method:      "system_name",
			skip:        true,
		},
		{ //TODO
			description: "test system_version",
			method:      "system_version",
			skip:        true,
		},
		{ //TODO
			description: "test system_chain",
			method:      "system_chain",
			skip:        true,
		},
		{ //TODO
			description: "test system_properties",
			method:      "system_properties",
			skip:        true,
		},
		{
			description: "test system_health",
			method:      "system_health",
			expected: modules.SystemHealthResponse{
				Peers:           2,
				IsSyncing:       true,
				ShouldHavePeers: true,
			},
			params: "{}",
		},
		{
			description: "test system_peers",
			method:      "system_peers",
			expected:    modules.SystemPeersResponse{},
			params:      "{}",
		},
		{
			description: "test system_network_state",
			method:      "system_networkState",
			expected: modules.SystemNetworkStateResponse{
				NetworkState: modules.NetworkStateString{
					PeerID: "",
				},
			},
			params: "{}",
		},
		{ //TODO
			description: "test system_addReservedPeer",
			method:      "system_addReservedPeer",
			skip:        true,
		},
		{ //TODO
			description: "test system_removeReservedPeer",
			method:      "system_removeReservedPeer",
			skip:        true,
		},
		{ //TODO
			description: "test system_nodeRoles",
			method:      "system_nodeRoles",
			skip:        true,
		},
		{ //TODO
			description: "test system_accountNextIndex",
			method:      "system_accountNextIndex",
			skip:        true,
		},
	}

	genesisPath := libutils.GetGssmrGenesisRawPathTest(t)
	config := config.CreateDefault(t)
	nodes := node.MakeNodes(t, 3, node.SetGenesis(genesisPath), node.SetConfig(config))

	ctx, cancel := context.WithCancel(context.Background())
	nodes.InitAndStartTest(ctx, t, cancel)

	time.Sleep(time.Second) // give server a second to start

	for _, test := range testCases {
		t.Run(test.description, func(t *testing.T) {
			getResponseCtx, getResponseCancel := context.WithTimeout(ctx, time.Second)
			defer getResponseCancel()
			target := getResponse(getResponseCtx, t, test)

			switch v := target.(type) {
			case *modules.SystemHealthResponse:
				t.Log("Will assert SystemHealthResponse", "target", target)

				require.Equal(t, test.expected.(modules.SystemHealthResponse).IsSyncing, v.IsSyncing)
				require.Equal(t, test.expected.(modules.SystemHealthResponse).ShouldHavePeers, v.ShouldHavePeers)
				require.GreaterOrEqual(t, v.Peers, test.expected.(modules.SystemHealthResponse).Peers)

			case *modules.SystemNetworkStateResponse:
				t.Log("Will assert SystemNetworkStateResponse", "target", target)

				require.NotNil(t, v.NetworkState)
				require.NotNil(t, v.NetworkState.PeerID)

			case *modules.SystemPeersResponse:
				t.Log("Will assert SystemPeersResponse", "target", target)

				require.NotNil(t, v)

				//TODO: #807
				//this assertion requires more time on init to be enabled
				//require.GreaterOrEqual(t, len(v.Peers), 2)

				for _, vv := range *v {
					require.NotNil(t, vv.PeerID)
					require.NotNil(t, vv.Roles)
					require.NotNil(t, vv.BestHash)
					require.NotNil(t, vv.BestNumber)
				}

			}

		})
	}
}
