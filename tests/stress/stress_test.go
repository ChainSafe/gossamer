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
	"io/ioutil"
	"math/big"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/dot/rpc/modules"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/runtime/extrinsic"
	"github.com/ChainSafe/gossamer/tests/utils"

	log "github.com/ChainSafe/log15"
	scribble "github.com/nanobox-io/golang-scribble"
	"github.com/stretchr/testify/require"
)

var (
	numNodes  = 3
	getHeader = "chain_getHeader"
)

func TestMain(m *testing.M) {
	if utils.GOSSAMER_INTEGRATION_TEST_MODE != "stress" {
		_, _ = fmt.Fprintln(os.Stdout, "Going to skip stress test")
		return
	}

	_, _ = fmt.Fprintln(os.Stdout, "Going to start stress test")

	if utils.NETWORK_SIZE != "" {
		currentNetworkSize, err := strconv.Atoi(utils.NETWORK_SIZE)
		if err == nil {
			_, _ = fmt.Fprintln(os.Stdout, "Going to use custom network size", "currentNetworkSize", currentNetworkSize)
			numNodes = currentNetworkSize
		}
	}

	if utils.HOSTNAME == "" {
		_, _ = fmt.Fprintln(os.Stdout, "HOSTNAME is not set, skipping stress test")
		return
	}

	// Start all tests
	code := m.Run()
	os.Exit(code)
}

func getChainHead(t *testing.T, node *utils.Node) *types.Header {
	respBody, err := utils.PostRPC(t, getHeader, "http://"+utils.HOSTNAME+":"+node.RPCPort, "[]")
	require.NoError(t, err)

	header := new(modules.ChainBlockHeaderResponse)
	utils.DecodeRPC(t, respBody, header)

	parentHash, err := common.HexToHash(header.ParentHash)
	require.NoError(t, err)

	nb, err := common.HexToBytes(header.Number)
	require.NoError(t, err)
	number := big.NewInt(0).SetBytes(nb)

	stateRoot, err := common.HexToHash(header.StateRoot)
	require.NoError(t, err)

	extrinsicsRoot, err := common.HexToHash(header.ExtrinsicsRoot)
	require.NoError(t, err)

	digest := [][]byte{}

	for _, l := range header.Digest.Logs {
		d, err := common.HexToBytes(l)
		require.NoError(t, err)
		digest = append(digest, d)
	}

	h, err := types.NewHeader(parentHash, number, stateRoot, extrinsicsRoot, digest)
	require.NoError(t, err)
	return h
}

func TestStressSync(t *testing.T) {
	t.Log("going to start TestStressSync")
	nodes, err := utils.StartNodes(t, numNodes)
	require.NoError(t, err)

	tempDir, err := ioutil.TempDir("", "gossamer-stress-db")
	require.NoError(t, err)
	t.Log("going to start a JSON database to track all chains", "tempDir", tempDir)

	db, err := scribble.New(tempDir, nil)
	require.NoError(t, err)

	for _, node := range nodes {
		header := getChainHead(t, node)

		err = db.Write("blocks_"+node.Key, header.Number.String(), header)
		require.NoError(t, err)
	}

	//TODO: #803 cleanup optimization
	errList := utils.TearDown(t, nodes)
	require.Len(t, errList, 0)
}

func TestStress_IncludeData(t *testing.T) {
	nodes, err := utils.StartNodes(t, numNodes)
	require.NoError(t, err)

	time.Sleep(5 * time.Second)

	ext := extrinsic.NewIncludeDataExt([]byte("nootwashere"))
	tx, err := ext.Encode()
	require.NoError(t, err)
	t.Log(tx)

	for _, node := range nodes {
		header := getChainHead(t, node)
		t.Log(header)
	}

	//TODO: #803 cleanup optimization
	errList := utils.TearDown(t, nodes)
	require.Len(t, errList, 0)
}
