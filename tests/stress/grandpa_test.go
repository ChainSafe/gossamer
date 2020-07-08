// Copyright 2020 ChainSafe Systems (ON) Corp.
// This file is part of gossamer.
//
// The gossamer library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The gossamer library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the gossamer library. If not, see <http://www.gnu.org/licenses/>.

package stress

import (
	"io/ioutil"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/tests/utils"

	"github.com/stretchr/testify/require"
)

func TestStress_Grandpa_OneAuthority(t *testing.T) {
	numNodes = 1
	nodes, err := utils.InitializeAndStartNodes(t, numNodes, utils.GenesisOneAuth, utils.ConfigDefault)
	require.NoError(t, err)

	defer func() {
		errList := utils.TearDown(t, nodes)
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
	numNodes = 3
	nodes, err := utils.InitializeAndStartNodes(t, numNodes, utils.GenesisThreeAuths, utils.ConfigDefault)
	require.NoError(t, err)

	defer func() {
		errList := utils.TearDown(t, nodes)
		require.Len(t, errList, 0)
	}()

	time.Sleep(time.Second * 10)
	fin, err := compareFinalizedHeadsWithRetry(t, nodes, 1)
	require.NoError(t, err)
	t.Logf("finalized hash in round 1: %s", fin)

	time.Sleep(time.Second * 10)
	fin, err = compareFinalizedHeadsWithRetry(t, nodes, 2)
	require.NoError(t, err)
	t.Logf("finalized hash in round 2: %s", fin)
}

func TestStress_Grandpa_NineAuthorities(t *testing.T) {
	// short for now, remove when syncing is more stable
	if testing.Short() {
		t.Skip("skipping TestStress_Grandpa_NineAuthorities")
	}

	numNodes = 9

	// only log info from 1 node
	tmpdir, err := ioutil.TempDir("", "gossamer-stress-8")
	require.NoError(t, err)
	node, err := utils.RunGossamer(t, numNodes-1, tmpdir, utils.GenesisDefault, utils.ConfigLogGrandpa)
	require.NoError(t, err)

	// wait and start rest of nodes - if they all start at the same time the first round usually doesn't complete since
	// all nodes vote for different blocks.
	time.Sleep(time.Second * 3)
	nodes, err := utils.InitializeAndStartNodes(t, numNodes-1, utils.GenesisDefault, utils.ConfigLogNone)
	require.NoError(t, err)
	nodes = append(nodes, node)

	defer func() {
		errList := utils.TearDown(t, nodes)
		require.Len(t, errList, 0)
	}()

	numRounds := 2
	for i := 1; i < numRounds+1; i++ {
		// TODO: this is a long time for a round to complete; this is because syncing is inefficient
		// need to improve syncing protocol
		time.Sleep(time.Second * 10)
		fin, err := compareFinalizedHeadsWithRetry(t, nodes, uint64(i))
		require.NoError(t, err)
		t.Logf("finalized hash in round %d: %s", i, fin)
	}
}
