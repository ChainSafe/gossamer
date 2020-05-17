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
	"bytes"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/dot/rpc/modules"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/common/optional"
	"github.com/ChainSafe/gossamer/lib/runtime/extrinsic"
	"github.com/ChainSafe/gossamer/lib/trie"
	"github.com/ChainSafe/gossamer/tests/utils"

	log "github.com/ChainSafe/log15"
	scribble "github.com/nanobox-io/golang-scribble"
	"github.com/stretchr/testify/require"
)

var (
	numNodes   = 3
	maxRetries = 8
)

func TestMain(m *testing.M) {
	if utils.GOSSAMER_INTEGRATION_TEST_MODE != "stress" {
		_, _ = fmt.Fprintln(os.Stdout, "Going to skip stress test")
		return
	}

	_, _ = fmt.Fprintln(os.Stdout, "Going to start stress test")

	if utils.NETWORK_SIZE != "" {
		var err error
		numNodes, err = strconv.Atoi(utils.NETWORK_SIZE)
		if err == nil {
			_, _ = fmt.Fprintf(os.Stdout, "Going to use custom network size %d\n", numNodes)
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
		header := utils.GetChainHead(t, node)

		err = db.Write("blocks_"+node.Key, header.Number.String(), header)
		require.NoError(t, err)
	}

	//TODO: #803 cleanup optimization
	errList := utils.TearDown(t, nodes)
	require.Len(t, errList, 0)
}

// submitExtrinsicAssertInclusion submits an extrinsic to a random node and asserts that the extrinsic was included in some block
// and that the nodes remain synced
func submitExtrinsicAssertInclusion(t *testing.T, nodes []*utils.Node, ext extrinsic.Extrinsic) {
	tx, err := ext.Encode()
	require.NoError(t, err)

	txStr := hex.EncodeToString(tx)
	log.Info("submitting transaction", "tx", txStr)

	// send extrinsic to random node
	idx := rand.Intn(len(nodes))
	prevHeader := utils.GetChainHead(t, nodes[idx]) // get starting header so that we can lookup blocks by number later
	respBody, err := utils.PostRPC(t, utils.AuthorSubmitExtrinsic, utils.NewEndpoint(utils.HOSTNAME, nodes[idx].RPCPort), "\"0x"+txStr+"\"")
	require.NoError(t, err)

	var hash modules.ExtrinsicHashResponse
	err = utils.DecodeRPC(t, respBody, &hash)
	require.Nil(t, err)

	log.Info("submitted transaction", "hash", hash)

	// wait for nodes to build block + sync, then get headers
	time.Sleep(time.Second * 5)
	var hashes map[common.Hash][]string
	for i := 0; i < maxRetries; i++ {
		hashes, err = utils.CompareChainHeads(t, nodes)
		if err == nil {
			break
		}

		time.Sleep(time.Second)
	}
	require.NoError(t, err, hashes)

	header := utils.GetChainHead(t, nodes[idx])
	log.Info("got header from node", "header", header, "hash", header.Hash(), "node", nodes[idx].Key)

	// search from child -> parent blocks for extrinsic
	time.Sleep(time.Second * 5)
	var resExts []types.Extrinsic
	i := 0
	for header.ExtrinsicsRoot == trie.EmptyHash && i != maxRetries {
		block := utils.GetBlock(t, nodes[idx], header.ParentHash)
		if block == nil {
			// couldn't get block, increment retry counter
			i++
			continue
		}

		header = block.Header
		log.Info("got block from node", "hash", header.Hash(), "node", nodes[idx].Key)
		log.Debug("got block from node", "header", header, "body", block.Body, "hash", header.Hash(), "node", nodes[idx].Key)

		if block.Body != nil && !bytes.Equal(*(block.Body), []byte{0}) {
			resExts, err = block.Body.AsExtrinsics()
			require.NoError(t, err, block.Body)
			break
		}

		if header.Hash() == prevHeader.Hash() {
			t.Fatal("could not find extrinsic in any blocks")
		}
	}

	// assert that the extrinsic included is the one we submitted
	require.Equal(t, 1, len(resExts), fmt.Sprintf("did not find extrinsic in block on node %s", nodes[idx].Key))
	require.Equal(t, resExts[0], types.Extrinsic(tx))

	// repeat sync check for sanity
	time.Sleep(time.Second * 5)
	for i = 0; i < maxRetries; i++ {
		hashes, err = utils.CompareChainHeads(t, nodes)
		if err == nil {
			break
		}

		time.Sleep(time.Second)
	}
	require.NoError(t, err, hashes)
}

func TestStress_IncludeData(t *testing.T) {
	nodes, err := utils.StartNodes(t, numNodes)
	require.NoError(t, err)

	time.Sleep(5 * time.Second)

	// create IncludeData extrnsic
	ext := extrinsic.NewIncludeDataExt([]byte("nootwashere"))
	submitExtrinsicAssertInclusion(t, nodes, ext)

	//TODO: #803 cleanup optimization
	errList := utils.TearDown(t, nodes)
	require.Len(t, errList, 0)
}

func TestStress_StorageChange(t *testing.T) {
	nodes, err := utils.StartNodes(t, numNodes)
	require.NoError(t, err)

	time.Sleep(5 * time.Second)

	// create IncludeData extrnsic
	key := []byte("noot")
	value := []byte("washere")
	ext := extrinsic.NewStorageChangeExt(key, optional.NewBytes(true, value))
	submitExtrinsicAssertInclusion(t, nodes, ext)

	time.Sleep(5 * time.Second)

	// for each node, check that storage was updated accordingly
	for _, node := range nodes {
		log.Info("getting storage from node", "node", node.Key)
		res := utils.GetStorage(t, node, key)

		// TODO: currently, around 2/3 nodes have the updated state, even if they all have the same
		// chain head. figure out why this is the case and fix it.
		idx := rand.Intn(len(nodes))
		if idx == node.Idx {
			// TODO: why does finalize_block modify the storage value?
			require.NotEqual(t, []byte{}, res)
			require.Equal(t, true, bytes.Contains(value, res[2:]))
		}
	}

	//TODO: #803 cleanup optimization
	errList := utils.TearDown(t, nodes)
	require.Len(t, errList, 0)
}
