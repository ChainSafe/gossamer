// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package stress

import (
	"context"
	"os"
	"testing"
	"time"

	libutils "github.com/ChainSafe/gossamer/lib/utils"
	"github.com/ChainSafe/gossamer/tests/utils"
	"github.com/stretchr/testify/require"
)

func TestStress_Grandpa_OneAuthority(t *testing.T) {
	numNodes := 1
	nodes, err := utils.InitializeAndStartNodes(t, numNodes, utils.GenesisDev, utils.ConfigDefault)
	require.NoError(t, err)

	defer func() {
		errList := utils.StopNodes(t, nodes)
		require.Len(t, errList, 0)
	}()

	time.Sleep(time.Second * 10)

	ctx := context.Background()

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

	nodes, err := utils.InitializeAndStartNodes(t, numNodes, genesisPath, utils.ConfigDefault)
	require.NoError(t, err)

	defer func() {
		errList := utils.StopNodes(t, nodes)
		require.Len(t, errList, 0)
	}()

	ctx := context.Background()

	numRounds := 5
	for i := 1; i < numRounds+1; i++ {
		const getFinalizedHeadByRoundTimeout = time.Second
		fin, err := compareFinalizedHeadsWithRetry(ctx,
			nodes, uint64(i), getFinalizedHeadByRoundTimeout)
		require.NoError(t, err)
		t.Logf("finalised hash in round %d: %s", i, fin)
	}
}

func TestStress_Grandpa_SixAuthorities(t *testing.T) {
	t.Skip()

	const numNodes = 6
	genesisPath := utils.GenerateGenesisAuths(t, numNodes)

	nodes, err := utils.InitializeAndStartNodes(t, numNodes, genesisPath, utils.ConfigDefault)
	require.NoError(t, err)

	defer func() {
		errList := utils.StopNodes(t, nodes)
		require.Len(t, errList, 0)
	}()

	ctx := context.Background()

	numRounds := 10
	for i := 1; i < numRounds+1; i++ {
		const getFinalizedHeadByRoundTimeout = time.Second
		fin, err := compareFinalizedHeadsWithRetry(ctx, nodes,
			uint64(i), getFinalizedHeadByRoundTimeout)
		require.NoError(t, err)
		t.Logf("finalised hash in round %d: %s", i, fin)
	}
}

func TestStress_Grandpa_NineAuthorities(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping TestStress_Grandpa_NineAuthorities")
	}

	utils.CreateConfigLogGrandpa()
	defer os.Remove(utils.ConfigLogGrandpa)

	numNodes := 9
	genesisPath := libutils.GetGssmrGenesisRawPathTest(t)
	nodes, err := utils.InitializeAndStartNodes(t, numNodes, genesisPath, utils.ConfigLogGrandpa)
	require.NoError(t, err)

	defer func() {
		errList := utils.StopNodes(t, nodes)
		require.Len(t, errList, 0)
	}()

	ctx := context.Background()

	numRounds := 3
	for i := 1; i < numRounds+1; i++ {
		const getFinalizedHeadByRoundTimeout = time.Second
		fin, err := compareFinalizedHeadsWithRetry(ctx, nodes,
			uint64(i), getFinalizedHeadByRoundTimeout)
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

	nodes, err := utils.InitializeAndStartNodes(t, numNodes-1, genesisPath, utils.ConfigDefault)
	require.NoError(t, err)

	defer func() {
		errList := utils.StopNodes(t, nodes)
		require.Len(t, errList, 0)
	}()

	time.Sleep(time.Second * 70) // let some rounds run

	basePath := t.TempDir()
	node, err := utils.RunGossamer(t, numNodes-1,
		basePath,
		genesisPath, utils.ConfigDefault,
		false, false)
	require.NoError(t, err)
	nodes = append(nodes, node)

	ctx := context.Background()

	numRounds := 10
	for i := 1; i < numRounds+1; i++ {
		const getFinalizedHeadByRoundTimeout = time.Second
		fin, err := compareFinalizedHeadsWithRetry(ctx, nodes, uint64(i), getFinalizedHeadByRoundTimeout)
		require.NoError(t, err)
		t.Logf("finalised hash in round %d: %s", i, fin)
	}
}
