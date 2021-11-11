// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: Apache-2.0

package stress

import (
	"os"
	"testing"
	"time"

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

	compareChainHeadsWithRetry(t, nodes)
	prev, _ := compareFinalizedHeads(t, nodes)

	time.Sleep(time.Second * 10)
	curr, _ := compareFinalizedHeads(t, nodes)
	require.NotEqual(t, prev, curr)
}

func TestStress_Grandpa_ThreeAuthorities(t *testing.T) {
	t.Skip()

	utils.GenerateGenesisThreeAuth()
	defer os.Remove(utils.GenesisThreeAuths)

	numNodes := 3
	nodes, err := utils.InitializeAndStartNodes(t, numNodes, utils.GenesisThreeAuths, utils.ConfigDefault)
	require.NoError(t, err)

	defer func() {
		errList := utils.StopNodes(t, nodes)
		require.Len(t, errList, 0)
	}()

	numRounds := 5
	for i := 1; i < numRounds+1; i++ {
		fin, err := compareFinalizedHeadsWithRetry(t, nodes, uint64(i))
		require.NoError(t, err)
		t.Logf("finalised hash in round %d: %s", i, fin)
	}
}

func TestStress_Grandpa_SixAuthorities(t *testing.T) {
	t.Skip()
	utils.GenerateGenesisSixAuth()
	defer os.Remove(utils.GenesisSixAuths)

	numNodes := 6
	nodes, err := utils.InitializeAndStartNodes(t, numNodes, utils.GenesisSixAuths, utils.ConfigDefault)
	require.NoError(t, err)

	defer func() {
		errList := utils.StopNodes(t, nodes)
		require.Len(t, errList, 0)
	}()

	numRounds := 10
	for i := 1; i < numRounds+1; i++ {
		fin, err := compareFinalizedHeadsWithRetry(t, nodes, uint64(i))
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
	nodes, err := utils.InitializeAndStartNodes(t, numNodes, utils.GenesisDefault, utils.ConfigLogGrandpa)
	require.NoError(t, err)

	defer func() {
		errList := utils.StopNodes(t, nodes)
		require.Len(t, errList, 0)
	}()

	numRounds := 3
	for i := 1; i < numRounds+1; i++ {
		fin, err := compareFinalizedHeadsWithRetry(t, nodes, uint64(i))
		require.NoError(t, err)
		t.Logf("finalised hash in round %d: %s", i, fin)
	}
}

func TestStress_Grandpa_CatchUp(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping TestStress_Grandpa_CatchUp")
	}

	utils.GenerateGenesisSixAuth()
	defer os.Remove(utils.GenesisSixAuths)

	numNodes := 6
	nodes, err := utils.InitializeAndStartNodes(t, numNodes-1, utils.GenesisSixAuths, utils.ConfigDefault)
	require.NoError(t, err)

	defer func() {
		errList := utils.StopNodes(t, nodes)
		require.Len(t, errList, 0)
	}()

	time.Sleep(time.Second * 70) // let some rounds run
	//nolint
	node, err := utils.RunGossamer(t, numNodes-1, utils.TestDir(t, utils.KeyList[numNodes-1]), utils.GenesisSixAuths, utils.ConfigDefault, false, false)
	require.NoError(t, err)
	nodes = append(nodes, node)

	numRounds := 10
	for i := 1; i < numRounds+1; i++ {
		fin, err := compareFinalizedHeadsWithRetry(t, nodes, uint64(i))
		require.NoError(t, err)
		t.Logf("finalised hash in round %d: %s", i, fin)
	}
}
