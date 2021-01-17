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
	"fmt"
	"math/big"
	"math/rand"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/tests/utils"

	log "github.com/ChainSafe/log15"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	if utils.MODE != "stress" {
		_, _ = fmt.Fprintln(os.Stdout, "Skipping stress test")
		return
	}

	if utils.HOSTNAME == "" {
		utils.HOSTNAME = "localhost"
	}

	utils.CreateConfigNoBabe()
	utils.CreateConfigBabeMaxThreshold()
	utils.CreateDefaultConfig()

	// TODO: implement test log flag
	utils.SetLogLevel(log.LvlInfo)
	h := log.StreamHandler(os.Stdout, log.TerminalFormat())
	logger.SetHandler(log.LvlFilterHandler(log.LvlInfo, h))

	utils.GenerateGenesisThreeAuth()

	// Start all tests
	code := m.Run()

	os.Remove(utils.ConfigNoBABE)
	os.Remove(utils.ConfigBABEMaxThreshold)
	os.Remove(utils.ConfigDefault)
	os.Exit(code)
}

func TestRestartNode(t *testing.T) {
	numNodes := 1
	nodes, err := utils.InitNodes(numNodes, utils.ConfigDefault)
	require.NoError(t, err)

	err = utils.StartNodes(t, nodes)
	require.NoError(t, err)

	errList := utils.StopNodes(t, nodes)
	require.Len(t, errList, 0)

	err = utils.StartNodes(t, nodes)
	require.NoError(t, err)

	errList = utils.StopNodes(t, nodes)
	require.Len(t, errList, 0)
}

func TestSync_SingleBlockProducer(t *testing.T) {
	//time.Sleep(time.Second)
	numNodes := 9 // TODO: increase this when syncing improves
	utils.SetLogLevel(log.LvlInfo)

	// start block producing node first
	node, err := utils.RunGossamer(t, numNodes-1, utils.TestDir(t, utils.KeyList[numNodes-1]), utils.GenesisDefault, utils.ConfigBABEMaxThreshold, false)
	require.NoError(t, err)

	// wait and start rest of nodes - if they all start at the same time the first round usually doesn't complete since
	// all nodes vote for different blocks.
	time.Sleep(time.Second * 5)
	nodes, err := utils.InitializeAndStartNodes(t, numNodes-1, utils.GenesisDefault, utils.ConfigNoBABE)
	require.NoError(t, err)
	nodes = append(nodes, node)

	defer func() {
		errList := utils.StopNodes(t, nodes)
		require.Len(t, errList, 0)
	}()

	numCmps := 10
	for i := 0; i < numCmps; i++ {
		time.Sleep(time.Second)
		t.Log("comparing...", i)
		hashes, err := compareBlocksByNumberWithRetry(t, nodes, strconv.Itoa(i))
		if len(hashes) > 1 || len(hashes) == 0 {
			require.NoError(t, err, i)
			continue
		}

		// there will only be one key in the mapping
		for _, nodesWithHash := range hashes {
			// allow 1 node to potentially not have synced. this is due to the runtime bug :(
			require.GreaterOrEqual(t, len(nodesWithHash), numNodes-1)
		}
	}
}

func TestSync_Basic(t *testing.T) {
	nodes, err := utils.InitializeAndStartNodes(t, 3, utils.GenesisDefault, utils.ConfigDefault)
	require.NoError(t, err)

	defer func() {
		errList := utils.StopNodes(t, nodes)
		require.Len(t, errList, 0)
	}()

	err = compareChainHeadsWithRetry(t, nodes)
	require.NoError(t, err)
}

func TestSync_MultipleEpoch(t *testing.T) {
	t.Skip("skipping TestSync_MultipleEpoch")
	numNodes := 3 // TODO: increase this when syncing improves
	utils.SetLogLevel(log.LvlInfo)

	// wait and start rest of nodes - if they all start at the same time the first round usually doesn't complete since
	nodes, err := utils.InitializeAndStartNodes(t, numNodes, utils.GenesisDefault, utils.ConfigDefault)
	require.NoError(t, err)

	defer func() {
		errList := utils.StopNodes(t, nodes)
		require.Len(t, errList, 0)
	}()

	time.Sleep(time.Second * 10)

	slotDuration := utils.SlotDuration(t, nodes[0])
	epochLength := utils.EpochLength(t, nodes[0])

	// Wait for epoch to pass
	time.Sleep(time.Duration(uint64(slotDuration.Nanoseconds()) * epochLength))

	// Just checking that everythings operating as expected
	header := utils.GetChainHead(t, nodes[0])
	currentHeight := header.Number.Int64()
	for i := int64(0); i < currentHeight; i++ {
		t.Log("comparing...", i)
		_, err = compareBlocksByNumberWithRetry(t, nodes, strconv.Itoa(int(i)))
		require.NoError(t, err, i)
	}
}

func TestSync_SingleSyncingNode(t *testing.T) {
	// TODO: Fix this test and enable it.
	t.Skip("skipping TestSync_SingleSyncingNode")
	utils.SetLogLevel(log.LvlInfo)

	// start block producing node
	alice, err := utils.RunGossamer(t, 0, utils.TestDir(t, utils.KeyList[0]), utils.GenesisDefault, utils.ConfigBABEMaxThreshold, false)
	require.NoError(t, err)
	time.Sleep(time.Second * 15)

	// start syncing node
	bob, err := utils.RunGossamer(t, 1, utils.TestDir(t, utils.KeyList[1]), utils.GenesisDefault, utils.ConfigNoBABE, false)
	require.NoError(t, err)

	nodes := []*utils.Node{alice, bob}
	defer func() {
		errList := utils.StopNodes(t, nodes)
		require.Len(t, errList, 0)
	}()

	numCmps := 100
	for i := 0; i < numCmps; i++ {
		t.Log("comparing...", i)
		_, err = compareBlocksByNumberWithRetry(t, nodes, strconv.Itoa(i))
		require.NoError(t, err, i)
	}
}

func TestSync_ManyProducers(t *testing.T) {
	// TODO: this fails with runtime: out of memory
	// this means when each node is connected to 8 other nodes, too much memory is being used.
	t.Skip()

	numNodes := 9 // 9 block producers
	utils.SetLogLevel(log.LvlInfo)
	nodes, err := utils.InitializeAndStartNodes(t, numNodes, utils.GenesisDefault, utils.ConfigDefault)
	require.NoError(t, err)

	defer func() {
		errList := utils.StopNodes(t, nodes)
		require.Len(t, errList, 0)
	}()

	numCmps := 100
	for i := 0; i < numCmps; i++ {
		t.Log("comparing...", i)
		_, err = compareBlocksByNumberWithRetry(t, nodes, strconv.Itoa(i))
		require.NoError(t, err, i)
		time.Sleep(time.Second)
	}
}

func TestSync_Bench(t *testing.T) {
	utils.SetLogLevel(log.LvlInfo)
	numBlocks := 64

	// start block producing node
	alice, err := utils.RunGossamer(t, 0, utils.TestDir(t, utils.KeyList[0]), utils.GenesisDefault, utils.ConfigBABEMaxThreshold, false)
	require.NoError(t, err)

	for {
		header, err := utils.GetChainHeadWithError(t, alice) //nolint
		if err != nil {
			continue
		}

		if header.Number.Int64() >= int64(numBlocks) {
			break
		}

		time.Sleep(3 * time.Second)
	}

	err = utils.PauseBABE(t, alice)
	require.NoError(t, err)
	t.Log("BABE paused")

	// start syncing node
	bob, err := utils.RunGossamer(t, 1, utils.TestDir(t, utils.KeyList[1]), utils.GenesisDefault, utils.ConfigNoBABE, false)
	require.NoError(t, err)

	nodes := []*utils.Node{alice, bob}
	defer func() {
		errList := utils.StopNodes(t, nodes)
		require.Len(t, errList, 0)
	}()

	// see how long it takes to sync to block numBlocks
	last := big.NewInt(int64(numBlocks))
	start := time.Now()
	var end time.Time

	for {
		if time.Since(start) >= testTimeout {
			t.Fatal("did not sync")
		}

		head, err := utils.GetChainHeadWithError(t, bob) //nolint
		if err != nil {
			continue
		}

		if head.Number.Cmp(last) >= 0 {
			end = time.Now()
			break
		}
	}

	maxTime := time.Second * 85
	minBPS := float64(0.75)
	totalTime := end.Sub(start)
	bps := float64(numBlocks) / end.Sub(start).Seconds()
	t.Log("total sync time:", totalTime)
	t.Log("blocks per second:", bps)
	require.LessOrEqual(t, int64(totalTime), int64(maxTime))
	require.GreaterOrEqual(t, bps, minBPS)

	// assert block is correct
	t.Log("comparing block...", numBlocks)
	_, err = compareBlocksByNumberWithRetry(t, nodes, strconv.Itoa(numBlocks))
	require.NoError(t, err, numBlocks)
	time.Sleep(time.Second * 3)
}

func TestSync_Restart(t *testing.T) {
	// TODO: Fix this test and enable it.
	t.Skip("skipping TestSync_Restart")
	numNodes := 3
	utils.SetLogLevel(log.LvlInfo)

	// start block producing node first
	node, err := utils.RunGossamer(t, numNodes-1, utils.TestDir(t, utils.KeyList[numNodes-1]), utils.GenesisDefault, utils.ConfigBABEMaxThreshold, false)
	require.NoError(t, err)

	// wait and start rest of nodes
	time.Sleep(time.Second * 5)
	nodes, err := utils.InitializeAndStartNodes(t, numNodes-1, utils.GenesisDefault, utils.ConfigNoBABE)
	require.NoError(t, err)
	nodes = append(nodes, node)

	defer func() {
		errList := utils.StopNodes(t, nodes)
		require.Len(t, errList, 0)
	}()

	done := make(chan struct{})

	// randomly turn off and on nodes
	go func() {
		for {
			select {
			case <-time.After(time.Second * 10):
				idx := rand.Intn(numNodes)

				errList := utils.StopNodes(t, nodes[idx:idx+1])
				require.Len(t, errList, 0)

				time.Sleep(time.Second)

				err = utils.StartNodes(t, nodes[idx:idx+1])
				require.NoError(t, err)
			case <-done:
				return
			}
		}
	}()

	numCmps := 12
	for i := 0; i < numCmps; i++ {
		t.Log("comparing...", i)
		_, err = compareBlocksByNumberWithRetry(t, nodes, strconv.Itoa(i))
		require.NoError(t, err, i)
		time.Sleep(time.Second * 5)
	}
	close(done)
}
