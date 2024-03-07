// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package network

import (
	cfg "github.com/ChainSafe/gossamer/config"
	"github.com/ChainSafe/gossamer/dot/rpc/modules"
	"github.com/ChainSafe/gossamer/lib/common"
	libutils "github.com/ChainSafe/gossamer/lib/utils"
	"github.com/ChainSafe/gossamer/tests/utils"
	"github.com/ChainSafe/gossamer/tests/utils/config"
	"github.com/ChainSafe/gossamer/tests/utils/node"
	"github.com/ChainSafe/gossamer/tests/utils/retry"
	"github.com/ChainSafe/gossamer/tests/utils/rpc"
	"github.com/stretchr/testify/require"

	"context"
	"os"
	"testing"
	"time"
)

func TestKadDHTNetworkDiscovery(t *testing.T) {
	if utils.MODE != "network" {
		t.Skip("RPC tests are disabled, going to skip.")
	}

	genesisPath := libutils.GetWestendDevRawGenesisPath(t)
	con := config.Default()
	con.ChainSpec = genesisPath
	con.Core.Role = common.FullNodeRole
	con.RPC.Modules = []string{"system", "author", "chain"}
	con.Network.MinPeers = 1
	con.Network.MaxPeers = 20
	con.Network.NoMDNS = true // Turning off mDNS, purpose of this test is purly use only kadDHT discovery
	con.Core.BabeAuthority = true
	con.Log.Sync = "trace"
	con.Network.Port = 7001

	peerConfigBoB := cfg.Copy(&con)
	peerConfigBoB.Network.Bootnodes = []string{
		"/ip4/127.0.0.1/tcp/7001/p2p/12D3KooWARREmJv5sDF3TFsnUsRrwPUQaXC5BaGGACHLGutNSGeV",
	}
	peerConfigBoB.Core.BabeAuthority = false
	peerConfigBoB.Network.Port = 7002

	peerConfigCharlie := cfg.Copy(&peerConfigBoB)
	peerConfigCharlie.Network.Port = 7003

	alice := node.New(t, con, node.SetIndex(0), node.SetWriter(os.Stdout))
	charlie := node.New(t, peerConfigCharlie, node.SetIndex(1), node.SetWriter(os.Stdout))
	bob := node.New(t, peerConfigBoB, node.SetIndex(2), node.SetWriter(os.Stdout))
	nodes := []*node.Node{&alice, &charlie, &bob}
	nodeKeyMap := map[string]*node.Node{
		"5260036532755806706566077008373229892172734656718221275791488860": &alice,
		"5260036532755806706566077008373229892172734656718221275791488861": &bob,
		"5260036532755806706566077008373229892172734656718221275791488862": &charlie,
	}

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	for key, node := range nodeKeyMap {
		node.InitAndStartTest(ctx, t, cancel, "--node-key", key)
		const timeBetweenStart = 0 * time.Second
		timer := time.NewTimer(timeBetweenStart)
		select {
		case <-timer.C:
		case <-ctx.Done():
			timer.Stop()
			return
		}
	}

	t.Log("waiting for all nodes to be connected")
	peerTimeout, peerCancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer peerCancel()
	err := retry.UntilOK(peerTimeout, 10*time.Second, func() (bool, error) {
		for _, node := range nodes {
			endpoint := rpc.NewEndpoint(node.RPCPort())
			t.Logf("requesting node %s with port %s", node.String(), endpoint)
			var response modules.SystemHealthResponse
			fetchWithTimeoutFromEndpoint(t, endpoint, "system_health", &response)
			t.Logf("Response: %+v", response)
			if response.Peers != len(nodes)-1 {
				return false, nil
			}
		}
		return true, nil
	})
	require.NoError(t, err)
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
