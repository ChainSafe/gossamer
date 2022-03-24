// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package stress

import (
	"context"
	"testing"
	"time"

	libutils "github.com/ChainSafe/gossamer/lib/utils"
	"github.com/ChainSafe/gossamer/tests/utils"
	"github.com/ChainSafe/gossamer/tests/utils/rpc"

	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/stretchr/testify/require"
)

func TestNetwork_MaxPeers(t *testing.T) {
	numNodes := 9 // 9 block producers
	genesisPath := libutils.GetGssmrGenesisRawPathTest(t)
	utils.Logger.Patch(log.SetLevel(log.Info))
	config := utils.CreateDefaultConfig(t)
	nodes, err := utils.InitializeAndStartNodes(t, numNodes, genesisPath, config)
	require.NoError(t, err)

	defer func() {
		errList := utils.TearDown(t, nodes)
		require.Len(t, errList, 0)
	}()

	// wait for nodes to connect
	time.Sleep(time.Second * 10)

	ctx := context.Background()

	for i, node := range nodes {
		const getPeersTimeout = time.Second
		getPeersCtx, cancel := context.WithTimeout(ctx, getPeersTimeout)
		peers, err := rpc.GetPeers(getPeersCtx, node.RPCPort)
		cancel()

		require.NoError(t, err)

		t.Logf("node %d: peer count=%d", i, len(peers))
		require.LessOrEqual(t, len(peers), 5)
	}
}
