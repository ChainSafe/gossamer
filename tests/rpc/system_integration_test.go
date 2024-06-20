// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package rpc

import (
	"context"
	"fmt"
	"testing"
	"time"

	cfg "github.com/ChainSafe/gossamer/config"

	"github.com/ChainSafe/gossamer/tests/utils/retry"

	"github.com/ChainSafe/gossamer/dot/rpc/modules"
	"github.com/ChainSafe/gossamer/lib/common"
	libutils "github.com/ChainSafe/gossamer/lib/utils"
	"github.com/ChainSafe/gossamer/tests/utils"
	"github.com/ChainSafe/gossamer/tests/utils/config"
	"github.com/ChainSafe/gossamer/tests/utils/node"
	"github.com/ChainSafe/gossamer/tests/utils/rpc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStableNetworkRPC(t *testing.T) { //nolint:tparallel
	if utils.MODE != "rpc" {
		//t.Skip("RPC tests are disabled, going to skip.")
	}

	genesisPath := libutils.GetWestendDevRawGenesisPath(t)
	con := config.Default()
	con.ChainSpec = genesisPath
	con.Core.Role = common.FullNodeRole
	con.RPC.Modules = []string{"system", "author", "chain"}
	con.Network.MinPeers = 1
	con.Network.MaxPeers = 20
	con.Core.BabeAuthority = true
	con.Log.Sync = "trace"

	babeAuthorityNode := node.New(t, con, node.SetIndex(0))

	peerConfig := cfg.Copy(&con)
	peerConfig.Core.BabeAuthority = false
	peer1 := node.New(t, peerConfig, node.SetIndex(1))
	peer2 := node.New(t, peerConfig, node.SetIndex(2))
	nodes := []*node.Node{&babeAuthorityNode, &peer1, &peer2}

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	for _, node := range nodes {
		fmt.Println("00000000000000000")
		node.InitAndStartTest(ctx, t, cancel)
		const timeBetweenStart = 0 * time.Second
		timer := time.NewTimer(timeBetweenStart)
		select {
		case <-timer.C:
			fmt.Println("111111111111111111111")
		case <-ctx.Done():
			fmt.Println("22222222222222")
			timer.Stop()
			return
		}
	}
	fmt.Println("33333333333333333")
	// wait until all nodes are connected
	t.Log("waiting for all nodes to be connected")
	peerTimeout, peerCancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer peerCancel()
	err := retry.UntilOK(peerTimeout, 10*time.Second, func() (bool, error) {
		fmt.Println("44444444444444444444")
		for _, node := range nodes {
			fmt.Println("555555555555555555555")
			endpoint := rpc.NewEndpoint(node.RPCPort())
			fmt.Println("66666666666666666666")
			t.Logf("requesting node %s with port %s", node.String(), endpoint)
			var response modules.SystemHealthResponse
			fetchWithTimeoutFromEndpoint(t, endpoint, "system_health", &response)
			fmt.Println("777777777777777777")
			t.Logf("Response: %+v", response)
			if response.Peers != len(nodes)-1 {
				return false, nil
			}
		}
		return true, nil
	})
	require.NoError(t, err)
	fmt.Println("888888888888888888888")
	// wait until all nodes are synced
	t.Log("waiting for all nodes to be synced")
	syncTimeout, syncCancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer syncCancel()
	err = retry.UntilOK(syncTimeout, 10*time.Second, func() (bool, error) {
		fmt.Println("9999999999999999")
		for _, node := range nodes {
			fmt.Println("10101010101010101010")
			//TODO: remove this once the issue has been addressed
			// https://github.com/ChainSafe/gossamer/issues/3030
			if node.Key() == config.AliceKey {
				continue
			}
			endpoint := rpc.NewEndpoint(node.RPCPort())
			fmt.Println(" 11-11-11-11-11-11-11-11-11-11")
			var response modules.SystemHealthResponse
			fetchWithTimeoutFromEndpoint(t, endpoint, "system_health", &response)
			fmt.Println("121212121212121212")
			if response.IsSyncing {
				return false, nil
			}
		}
		return true, nil
	})
	require.NoError(t, err)
	fmt.Println("131313131313131313131313131")
	t.Logf("All nodes have %d peers and synced", len(nodes)-1)

	// wait for a bit and then run the test suite to ensure that the nodes are still connected and synced
	t.Logf("Waiting for 60 seconds before running the test suite")
	time.Sleep(60 * time.Second)
	fmt.Println("14141414141414141414141414")
	for _, node := range nodes {
		fmt.Println("1515151515151515151515151")
		node := node
		t.Run(node.String(), func(t *testing.T) {
			t.Parallel()

			//TODO: remove this once the issue has been addressed
			// https://github.com/ChainSafe/gossamer/issues/3030
			if node.Key() == config.AliceKey {
				t.Logf("Skipping test for alice")
				t.Skip()
			}
			fmt.Println("1515151515151515151515151")
			endpoint := rpc.NewEndpoint(node.RPCPort())
			fmt.Println("1616161616161616161616161")
			t.Run("system_health", func(t *testing.T) {
				t.Parallel()
				fmt.Println("171717171717171717171")
				var response modules.SystemHealthResponse
				fetchWithTimeoutFromEndpoint(t, endpoint, "system_health", &response)
				fmt.Println("18181818181818181818181818181")
				expectedResponse := modules.SystemHealthResponse{
					Peers:           len(nodes) - 1,
					IsSyncing:       false,
					ShouldHavePeers: true,
				}
				assert.Equal(t, expectedResponse, response)
			})

			t.Run("system_networkState", func(t *testing.T) {
				t.Parallel()
				fmt.Println("19191919191919191919191919191919191")
				var response modules.SystemNetworkStateResponse

				fetchWithTimeoutFromEndpoint(t, endpoint, "system_networkState", &response)
				fmt.Println("20202020202020202020202020202020202")
				// TODO assert response
			})

			t.Run("system_peers", func(t *testing.T) {
				t.Parallel()

				var response modules.SystemPeersResponse
				fmt.Println("212121212121212121212121212121212121212121")
				fetchWithTimeoutFromEndpoint(t, endpoint, "system_peers", &response)
				fmt.Println("23-23-23-23-23-23-23-23-23-23-23-23-23-23")
				assert.GreaterOrEqual(t, len(response), len(nodes)-2)

				// TODO assert response
			})
		})
	}
}

func fetchWithTimeoutFromEndpoint(t *testing.T, endpoint, method string, target interface{}) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	body, err := rpc.Post(ctx, endpoint, method, "{}")
	require.NoError(t, err)

	err = rpc.Decode(body, target)
	require.NoError(t, err)
}
