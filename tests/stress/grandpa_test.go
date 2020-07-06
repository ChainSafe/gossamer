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
	fin := compareFinalizedHeadsWithRetry(t, nodes, 1)
	t.Logf("finalized hash in round 1: %s", fin)

	time.Sleep(time.Second * 10)
	fin = compareFinalizedHeadsWithRetry(t, nodes, 2)
	t.Logf("finalized hash in round 2: %s", fin)
}

func TestStress_Grandpa_NineAuthorities(t *testing.T) {
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

	numRounds := 3
	for i := 1; i < numRounds+1; i++ {
		// TODO: this is a long time for a round to complete; this is because syncing is inefficient
		// need to improve syncing protocol
		time.Sleep(time.Second * 10)
		fin := compareFinalizedHeadsWithRetry(t, nodes, uint64(i))
		t.Logf("finalized hash in round %d: %s", i, fin)
	}
}
