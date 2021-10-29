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
	"strings"
	"testing"
	"time"

	gosstypes "github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/tests/utils"
	log "github.com/ChainSafe/log15"
	gsrpc "github.com/centrifuge/go-substrate-rpc-client/v3"
	"github.com/centrifuge/go-substrate-rpc-client/v3/signature"
	"github.com/centrifuge/go-substrate-rpc-client/v3/types"
	"github.com/gorilla/websocket"
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
	utils.CreateDefaultConfig()
	utils.CreateConfigNoGrandpa()
	utils.CreateConfigNotAuthority()

	defer func() {
		os.Remove(utils.ConfigNoBABE)
		os.Remove(utils.ConfigDefault)
		os.Remove(utils.ConfigNoGrandpa)
		os.Remove(utils.ConfigNotAuthority)
	}()

	logLvl := log.LvlInfo
	if utils.LOGLEVEL != "" {
		var err error
		logLvl, err = log.LvlFromString(utils.LOGLEVEL)
		if err != nil {
			panic(fmt.Sprint("Invalid log level: ", err))
		}
	}

	utils.SetLogLevel(logLvl)
	h := log.StreamHandler(os.Stdout, log.TerminalFormat())
	logger.SetHandler(log.LvlFilterHandler(logLvl, h))

	utils.GenerateGenesisThreeAuth()

	// Start all tests
	code := m.Run()
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
	numNodes := 4
	utils.SetLogLevel(log.LvlInfo)

	// start block producing node first
	//nolint
	node, err := utils.RunGossamer(t, numNodes-1, utils.TestDir(t, utils.KeyList[numNodes-1]), utils.GenesisDev, utils.ConfigNoGrandpa, false, true)
	require.NoError(t, err)

	// wait and start rest of nodes - if they all start at the same time the first round usually doesn't complete since
	// all nodes vote for different blocks.
	time.Sleep(time.Second * 15)
	nodes, err := utils.InitializeAndStartNodes(t, numNodes-1, utils.GenesisDev, utils.ConfigNotAuthority)
	require.NoError(t, err)
	nodes = append(nodes, node)

	time.Sleep(time.Second * 20)

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
			// allow 1 node to potentially not have synced. this is due to the need to increase max peer count
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
	numNodes := 3
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
	alice, err := utils.RunGossamer(t, 0, utils.TestDir(t, utils.KeyList[0]), utils.GenesisDev, utils.ConfigDefault, false, true)
	require.NoError(t, err)
	time.Sleep(time.Second * 15)

	// start syncing node
	bob, err := utils.RunGossamer(t, 1, utils.TestDir(t, utils.KeyList[1]), utils.GenesisDev, utils.ConfigNoBABE, false, false)
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

func TestSync_Bench(t *testing.T) {
	utils.SetLogLevel(log.LvlInfo)
	numBlocks := 64

	// start block producing node
	alice, err := utils.RunGossamer(t, 0, utils.TestDir(t, utils.KeyList[1]), utils.GenesisDev, utils.ConfigNoGrandpa, false, true)
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
	bob, err := utils.RunGossamer(t, 1, utils.TestDir(t, utils.KeyList[0]), utils.GenesisDev, utils.ConfigNotAuthority, false, true)
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
	//nolint
	node, err := utils.RunGossamer(t, numNodes-1, utils.TestDir(t, utils.KeyList[numNodes-1]), utils.GenesisDefault, utils.ConfigDefault, false, true)
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

func TestSync_SubmitExtrinsic(t *testing.T) {
	t.Log("starting gossamer...")

	// index of node to submit tx to
	idx := 0 // TODO: randomise this

	// start block producing node first
	node, err := utils.RunGossamer(t, 0, utils.TestDir(t, utils.KeyList[0]), utils.GenesisDev, utils.ConfigNoGrandpa, false, true)
	require.NoError(t, err)
	nodes := []*utils.Node{node}

	// Start rest of nodes
	node, err = utils.RunGossamer(t, 1, utils.TestDir(t, utils.KeyList[1]), utils.GenesisDev, utils.ConfigNotAuthority, false, false)
	require.NoError(t, err)
	nodes = append(nodes, node)
	node, err = utils.RunGossamer(t, 2, utils.TestDir(t, utils.KeyList[2]), utils.GenesisDev, utils.ConfigNotAuthority, false, false)
	require.NoError(t, err)
	nodes = append(nodes, node)

	defer func() {
		t.Log("going to tear down gossamer...")
		errList := utils.StopNodes(t, nodes)
		require.Len(t, errList, 0)
	}()

	// send tx to non-authority node
	api, err := gsrpc.NewSubstrateAPI(fmt.Sprintf("http://localhost:%s", nodes[idx].RPCPort))
	require.NoError(t, err)

	meta, err := api.RPC.State.GetMetadataLatest()
	require.NoError(t, err)

	c, err := types.NewCall(meta, "System.remark", []byte{0xab})
	require.NoError(t, err)

	// Create the extrinsic
	ext := types.NewExtrinsic(c)

	genesisHash, err := api.RPC.Chain.GetBlockHash(0)
	require.NoError(t, err)

	rv, err := api.RPC.State.GetRuntimeVersionLatest()
	require.NoError(t, err)

	key, err := types.CreateStorageKey(meta, "System", "Account", signature.TestKeyringPairAlice.PublicKey, nil)
	require.NoError(t, err)

	var accInfo types.AccountInfo
	ok, err := api.RPC.State.GetStorageLatest(key, &accInfo)
	require.NoError(t, err)
	require.True(t, ok)

	o := types.SignatureOptions{
		BlockHash:          genesisHash,
		Era:                types.ExtrinsicEra{IsImmortalEra: true},
		GenesisHash:        genesisHash,
		Nonce:              types.NewUCompactFromUInt(uint64(accInfo.Nonce)),
		SpecVersion:        rv.SpecVersion,
		Tip:                types.NewUCompactFromUInt(0),
		TransactionVersion: rv.TransactionVersion,
	}

	// Sign the transaction using Alice's default account
	err = ext.Sign(signature.TestKeyringPairAlice, o)
	require.NoError(t, err)

	extEnc, err := types.EncodeToHexString(ext)
	require.NoError(t, err)

	prevHeader := utils.GetChainHead(t, nodes[idx]) // get starting header so that we can lookup blocks by number later

	// Send the extrinsic
	hash, err := api.RPC.Author.SubmitExtrinsic(ext)
	require.NoError(t, err)
	require.NotEqual(t, types.Hash{}, hash)

	time.Sleep(time.Second * 20)

	// wait until there's no more pending extrinsics
	for i := 0; i < maxRetries; i++ {
		exts := getPendingExtrinsics(t, nodes[idx])
		if len(exts) == 0 {
			break
		}

		time.Sleep(time.Second)
	}

	header := utils.GetChainHead(t, nodes[idx])

	// search from child -> parent blocks for extrinsic
	var (
		resExts    []gosstypes.Extrinsic
		extInBlock *big.Int
	)

	for i := 0; i < maxRetries; i++ {
		block := utils.GetBlock(t, nodes[idx], header.ParentHash)
		if block == nil {
			// couldn't get block, increment retry counter
			continue
		}

		header = &block.Header
		logger.Debug("got block from node", "header", header, "body", block.Body, "node", nodes[idx].Key)

		if block.Body != nil {
			resExts = block.Body

			logger.Debug("extrinsics", "exts", resExts)
			if len(resExts) >= 2 {
				extInBlock = block.Header.Number
				break
			}
		}

		if header.Hash() == prevHeader.Hash() {
			t.Fatal("could not find extrinsic in any blocks")
		}
	}

	var included bool
	for _, ext := range resExts {
		logger.Debug("comparing", "expected", extEnc, "in block", common.BytesToHex(ext))
		if strings.Compare(extEnc, common.BytesToHex(ext)) == 0 {
			included = true
		}
	}

	require.True(t, included)

	hashes, err := compareBlocksByNumberWithRetry(t, nodes, extInBlock.String())
	require.NoError(t, err, hashes)
}

func TestSync_SubmitAndWatchExtrinsic(t *testing.T) {
	t.Log("starting gossamer...")

	// index of node to submit tx to
	idx := 0 // TODO: randomise this

	// start block producing node first
	node, err := utils.RunGossamer(t, 0, utils.TestDir(t, utils.KeyList[0]), utils.GenesisDev, utils.ConfigNoGrandpa, true)
	require.NoError(t, err)
	nodes := []*utils.Node{node}

	// Start rest of nodes
	node, err = utils.RunGossamer(t, 1, utils.TestDir(t, utils.KeyList[1]), utils.GenesisDev, utils.ConfigNotAuthority, true)
	require.NoError(t, err)
	nodes = append(nodes, node)
	node, err = utils.RunGossamer(t, 2, utils.TestDir(t, utils.KeyList[2]), utils.GenesisDev, utils.ConfigNotAuthority, true)
	require.NoError(t, err)
	nodes = append(nodes, node)

	defer func() {
		t.Log("going to tear down gossamer...")
		errList := utils.StopNodes(t, nodes)
		require.Len(t, errList, 0)
	}()

	// send tx to non-authority node
	api, err := gsrpc.NewSubstrateAPI(fmt.Sprintf("ws://localhost:%s", nodes[idx].WSPort))
	require.NoError(t, err)

	meta, err := api.RPC.State.GetMetadataLatest()
	require.NoError(t, err)

	c, err := types.NewCall(meta, "System.remark", []byte{0xab})
	require.NoError(t, err)

	// Create the extrinsic
	ext := types.NewExtrinsic(c)

	genesisHash, err := api.RPC.Chain.GetBlockHash(0)
	require.NoError(t, err)

	rv, err := api.RPC.State.GetRuntimeVersionLatest()
	require.NoError(t, err)

	key, err := types.CreateStorageKey(meta, "System", "Account", signature.TestKeyringPairAlice.PublicKey, nil)
	require.NoError(t, err)

	var accInfo types.AccountInfo
	ok, err := api.RPC.State.GetStorageLatest(key, &accInfo)
	require.NoError(t, err)
	require.True(t, ok)

	o := types.SignatureOptions{
		BlockHash:          genesisHash,
		Era:                types.ExtrinsicEra{IsImmortalEra: true},
		GenesisHash:        genesisHash,
		Nonce:              types.NewUCompactFromUInt(uint64(accInfo.Nonce)),
		SpecVersion:        rv.SpecVersion,
		Tip:                types.NewUCompactFromUInt(0),
		TransactionVersion: rv.TransactionVersion,
	}

	// Sign the transaction using Alice's default account
	err = ext.Sign(signature.TestKeyringPairAlice, o)
	require.NoError(t, err)

	extEnc, err := types.EncodeToHexString(ext)
	require.NoError(t, err)

	prevHeader := utils.GetChainHead(t, nodes[idx]) // get starting header so that we can lookup blocks by number later

	// Send the extrinsic
	// hash, err := api.RPC.Author.SubmitExtrinsic(ext)
	// require.NoError(t, err)
	// require.NotEqual(t, hash, common.Hash{})

	hexExt, err := types.EncodeToHexString(ext)
	if err != nil {
		fmt.Println(err)
	}

	conn, _, err := websocket.DefaultDialer.Dial("ws://localhost:8546", nil)
	if err != nil {
		fmt.Println(err)
	}

	message := []byte(`{"id":1, "jsonrpc":"2.0", "method": "author_submitAndWatchExtrinsic", "params":["` + hexExt + `"]}`)

	//message := []byte(`{"id":1, "jsonrpc":"2.0", "method": "author_submitAndWatchExtrinsic", "params":["0xad018400d43593c715fdd31c61141abd04a99fd6822c8558854ccde39a5684e7a56da27d01f2bb38a08174047730a6163e4128cf2c728387dec88870e46212cc17d86163484cbd43b6dee15eb91467e46acb6cd6671b7694bddf34380efdb2fdc4ddfc3f8f000000000108abcd"]}`)
	err = conn.WriteMessage(websocket.TextMessage, message)
	if err != nil {
		fmt.Println(err)
	}

	_, message, err = conn.ReadMessage()
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println(string(message))
	}
	// send and watch extrinsic
	// extStatusSubscription, err := api.RPC.Author.SubmitAndWatchExtrinsic(ext)
	// require.NoError(t, err)

	// for extrinsicStatus := range extStatusSubscription.Chan() {
	// 	fmt.Printf("extrinsicStatus: %v\n", extrinsicStatus)
	// }

	time.Sleep(time.Second * 20)

	// wait until there's no more pending extrinsics
	for i := 0; i < maxRetries; i++ {
		exts := getPendingExtrinsics(t, nodes[idx])
		if len(exts) == 0 {
			break
		}

		time.Sleep(time.Second)
	}

	header := utils.GetChainHead(t, nodes[idx])

	// search from child -> parent blocks for extrinsic
	var (
		resExts    []gosstypes.Extrinsic
		extInBlock *big.Int
	)

	for i := 0; i < maxRetries; i++ {
		block := utils.GetBlock(t, nodes[idx], header.ParentHash)
		if block == nil {
			// couldn't get block, increment retry counter
			continue
		}

		header = &block.Header
		logger.Debug("got block from node", "header", header, "body", block.Body, "node", nodes[idx].Key)

		if block.Body != nil {
			resExts = block.Body

			logger.Debug("extrinsics", "exts", resExts)
			if len(resExts) >= 2 {
				extInBlock = block.Header.Number
				break
			}
		}

		if header.Hash() == prevHeader.Hash() {
			t.Fatal("could not find extrinsic in any blocks")
		}
	}

	var included bool
	for _, ext := range resExts {
		logger.Debug("comparing", "expected", extEnc, "in block", common.BytesToHex(ext))
		if strings.Compare(extEnc, common.BytesToHex(ext)) == 0 {
			included = true
		}
	}

	require.True(t, included)

	hashes, err := compareBlocksByNumberWithRetry(t, nodes, extInBlock.String())
	require.NoError(t, err, hashes)
}
