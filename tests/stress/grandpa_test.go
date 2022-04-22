// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package stress

import (
	"context"
	"testing"
	"time"

	libutils "github.com/ChainSafe/gossamer/lib/utils"
	"github.com/ChainSafe/gossamer/tests/utils"
	"github.com/ChainSafe/gossamer/tests/utils/config"
	"github.com/ChainSafe/gossamer/tests/utils/node"
	"github.com/stretchr/testify/require"
)

func TestStress_Grandpa_OneAuthority(t *testing.T) {
	genesisPath := libutils.GetDevGenesisSpecPathTest(t)
	tomlConfig := config.Default()
	tomlConfig.Core.BABELead = true
	tomlConfig.Init.Genesis = genesisPath
	n := node.New(t, tomlConfig)

	ctx, cancel := context.WithCancel(context.Background())

	n.InitAndStartTest(ctx, t, cancel)
	nodes := node.Nodes{n}

	time.Sleep(time.Second * 10)

	const getChainHeadTimeout = time.Second
	compareChainHeadsWithRetry(ctx, nodes, getChainHeadTimeout)

	const getFinalizedHeadTimeout = time.Second
	prev, _ := compareFinalizedHeads(ctx, t, nodes, getFinalizedHeadTimeout)

	time.Sleep(time.Second * 10)
	curr, _ := compareFinalizedHeads(ctx, t, nodes, getFinalizedHeadTimeout)
	require.NotEqual(t, prev, curr)
}

func TestStress_Grandpa_ThreeAuthorities(t *testing.T) {
	t.Skip()

	const numNodes = 3

	genesisPath := utils.GenerateGenesisAuths(t, numNodes)

	tomlConfig := config.Default()
	tomlConfig.Init.Genesis = genesisPath
	nodes := node.MakeNodes(t, numNodes, tomlConfig)

	ctx, cancel := context.WithCancel(context.Background())

	nodes.InitAndStartTest(ctx, t, cancel)

	numRounds := 5
	for i := 1; i < numRounds+1; i++ {
		const getFinalizedHeadByRoundTimeout = time.Second
		const retryWait = time.Second
		fin, err := compareFinalizedHeadsWithRetry(ctx,
			nodes, uint64(i), getFinalizedHeadByRoundTimeout, retryWait)
		require.NoError(t, err)
		t.Logf("finalised hash in round %d: %s", i, fin)
	}
}

func TestStress_Grandpa_SixAuthorities(t *testing.T) {
	t.Skip()

	const numNodes = 6
	genesisPath := utils.GenerateGenesisAuths(t, numNodes)

	tomlConfig := config.Default()
	tomlConfig.Init.Genesis = genesisPath
	nodes := node.MakeNodes(t, numNodes, tomlConfig)
	ctx, cancel := context.WithCancel(context.Background())
	nodes.InitAndStartTest(ctx, t, cancel)

	numRounds := 10
	for i := 1; i < numRounds+1; i++ {
		const getFinalizedHeadByRoundTimeout = time.Second
		const retryWait = time.Second
		fin, err := compareFinalizedHeadsWithRetry(ctx, nodes,
			uint64(i), getFinalizedHeadByRoundTimeout, retryWait)
		require.NoError(t, err)
		t.Logf("finalised hash in round %d: %s", i, fin)
	}
}

func TestStress_Grandpa_NineAuthorities(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping TestStress_Grandpa_NineAuthorities")
	}

	const numNodes = 9
	genesisPath := libutils.GetGssmrGenesisRawPathTest(t)

	tomlConfig := config.LogGrandpa()
	tomlConfig.Init.Genesis = genesisPath
	nodes := node.MakeNodes(t, numNodes, tomlConfig)
	ctx, cancel := context.WithCancel(context.Background())
	nodes.InitAndStartTest(ctx, t, cancel)

	numRounds := 3
	for i := 1; i < numRounds+1; i++ {
		const getFinalizedHeadByRoundTimeout = time.Second
		const retryWait = time.Second
		fin, err := compareFinalizedHeadsWithRetry(ctx, nodes,
			uint64(i), getFinalizedHeadByRoundTimeout, retryWait)
		require.NoError(t, err)
		t.Logf("finalised hash in round %d: %s", i, fin)
	}
}

func TestStress_Grandpa_CatchUp(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping TestStress_Grandpa_CatchUp")
	}

	const numNodes = 6
	genesisPath := utils.GenerateGenesisAuths(t, numNodes)

	tomlConfig := config.Default()
	tomlConfig.Init.Genesis = genesisPath
	nodes := node.MakeNodes(t, numNodes, tomlConfig)
	ctx, cancel := context.WithCancel(context.Background())
	nodes.InitAndStartTest(ctx, t, cancel)

	time.Sleep(time.Second * 70) // let some rounds run

	node := node.New(t, tomlConfig, node.SetIndex(numNodes-1))
	node.InitAndStartTest(ctx, t, cancel)
	nodes = append(nodes, node)

	numRounds := 10
	for i := 1; i < numRounds+1; i++ {
		const getFinalizedHeadByRoundTimeout = time.Second
		const retryWait = time.Second
		fin, err := compareFinalizedHeadsWithRetry(ctx, nodes, uint64(i),
			getFinalizedHeadByRoundTimeout, retryWait)
		require.NoError(t, err)
		t.Logf("finalised hash in round %d: %s", i, fin)
	}
}
