// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package stress

import (
	"context"
	"fmt"
	"math/big"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	gsrpc "github.com/centrifuge/go-substrate-rpc-client/v3"
	"github.com/centrifuge/go-substrate-rpc-client/v3/signature"
	"github.com/centrifuge/go-substrate-rpc-client/v3/types"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	gosstypes "github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/lib/common"
	libutils "github.com/ChainSafe/gossamer/lib/utils"
	"github.com/ChainSafe/gossamer/tests/utils"
	"github.com/ChainSafe/gossamer/tests/utils/config"
	"github.com/ChainSafe/gossamer/tests/utils/node"
	"github.com/ChainSafe/gossamer/tests/utils/rpc"
)

func TestMain(m *testing.M) {
	if utils.MODE != "stress" {
		fmt.Println("Skipping stress test")
		return
	}

	logLvl := log.Info
	if utils.LOGLEVEL != "" {
		var err error
		logLvl, err = log.ParseLevel(utils.LOGLEVEL)
		if err != nil {
			panic(fmt.Sprintf("Invalid log level: %s", err))
		}
	}

	utils.Logger.Patch(log.SetLevel(logLvl))
	logger.Patch(log.SetLevel(logLvl))

	// Start all tests
	code := m.Run()
	os.Exit(code)
}

func TestRestartNode(t *testing.T) {
	const numNodes = 1
	nodes := node.MakeNodes(t, numNodes)

	err := nodes.Init(context.Background())
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	waitErr := make(chan error)

	started, startErr := nodes.Start(ctx, waitErr)
	if startErr != nil {
		cancel()
		for i := 0; i < started; i++ {
			<-waitErr
		}
		close(waitErr)
		t.Fatalf("failed to start nodes: %s", startErr)
	}

	// Stop nodes
	cancel()
	for i := 0; i < started; i++ {
		<-waitErr
	}

	ctx, cancel = context.WithCancel(context.Background())

	started, startErr = nodes.Start(ctx, waitErr)
	if startErr != nil {
		cancel()
		for i := 0; i < started; i++ {
			<-waitErr
		}
		close(waitErr)
		t.Fatalf("failed to start nodes: %s", startErr)
	}

	// Stop nodes
	cancel()
	for i := 0; i < started; i++ {
		<-waitErr
	}
}

func TestSync_SingleBlockProducer(t *testing.T) {
	numNodes := 4
	utils.Logger.Patch(log.SetLevel(log.Info))

	// start block producing node first
	basePath := t.TempDir()
	genesisPath := libutils.GetDevGenesisSpecPathTest(t)
	configNoGrandpa := config.CreateNoGrandpa(t)
	babeLeadNode := node.New(t,
		node.SetIndex(numNodes-1),
		node.SetBasePath(basePath),
		node.SetGenesis(genesisPath),
		node.SetConfig(configNoGrandpa),
		node.SetBabeLead(true))

	ctx, cancel := context.WithCancel(context.Background())
	babeLeadNode.InitAndStartTest(ctx, t, cancel)

	configNoAuthority := config.CreateNotAuthority(t)

	// wait and start rest of nodes - if they all start at the same time the first round usually doesn't complete since
	// all nodes vote for different blocks.
	time.Sleep(time.Second * 15)

	nodes := node.MakeNodes(t, numNodes-1,
		node.SetGenesis(genesisPath),
		node.SetConfig(configNoAuthority))
	nodes.InitAndStartTest(ctx, t, cancel)
	nodes = append(nodes, babeLeadNode)

	time.Sleep(time.Second * 30)

	numCmps := 10

	for i := 0; i < numCmps; i++ {
		time.Sleep(3 * time.Second)
		t.Log("comparing...", i)

		const comparisonTimeout = 5 * time.Second
		compareCtx, cancel := context.WithTimeout(ctx, comparisonTimeout)

		hashes := compareBlocksByNumber(compareCtx, t, nodes, strconv.Itoa(i))

		cancel()

		// there will only be one key in the mapping
		for _, nodesWithHash := range hashes {
			// allow 1 node to potentially not have synced. this is due to the need to increase max peer count
			require.GreaterOrEqual(t, len(nodesWithHash), numNodes-1)
		}
	}
}

func TestSync_Basic(t *testing.T) {
	genesisPath := libutils.GetGssmrGenesisRawPathTest(t)

	config := config.CreateDefault(t)
	const numNodes = 3
	nodes := node.MakeNodes(t, numNodes,
		node.SetGenesis(genesisPath), node.SetConfig(config))

	ctx, cancel := context.WithCancel(context.Background())
	nodes.InitAndStartTest(ctx, t, cancel)

	const getChainHeadTimeout = time.Second

	err := compareChainHeadsWithRetry(ctx, nodes, getChainHeadTimeout)
	require.NoError(t, err)
}

func TestSync_MultipleEpoch(t *testing.T) {
	t.Skip("skipping TestSync_MultipleEpoch")
	genesisPath := libutils.GetGssmrGenesisRawPathTest(t)
	numNodes := 3
	utils.Logger.Patch(log.SetLevel(log.Info))

	// wait and start rest of nodes - if they all start at the same time the first round usually doesn't complete since
	config := config.CreateDefault(t)
	nodes := node.MakeNodes(t, numNodes,
		node.SetGenesis(genesisPath), node.SetConfig(config))

	ctx, cancel := context.WithCancel(context.Background())
	nodes.InitAndStartTest(ctx, t, cancel)

	time.Sleep(time.Second * 10)

	slotDurationCtx, cancel := context.WithTimeout(ctx, time.Second)
	slotDuration, err := rpc.SlotDuration(slotDurationCtx, nodes[0].GetRPCPort())
	cancel()
	require.NoError(t, err)

	epochLengthCtx, cancel := context.WithTimeout(ctx, time.Second)
	epochLength, err := rpc.EpochLength(epochLengthCtx, nodes[0].GetRPCPort())
	cancel()
	require.NoError(t, err)

	// Wait for epoch to pass
	time.Sleep(time.Duration(uint64(slotDuration.Nanoseconds()) * epochLength))

	// Just checking that everythings operating as expected
	getChainHeadCtx, cancel := context.WithTimeout(ctx, time.Second)
	header, err := rpc.GetChainHead(getChainHeadCtx, nodes[0].GetRPCPort())
	cancel()
	require.NoError(t, err)

	currentHeight := header.Number
	for i := uint(0); i < currentHeight; i++ {
		t.Log("comparing...", i)

		const compareTimeout = 5 * time.Second
		compareCtx, cancel := context.WithTimeout(ctx, compareTimeout)

		_ = compareBlocksByNumber(compareCtx, t, nodes, strconv.Itoa(int(i)))

		cancel()
	}
}

func TestSync_SingleSyncingNode(t *testing.T) {
	// TODO: Fix this test and enable it.
	t.Skip("skipping TestSync_SingleSyncingNode")
	utils.Logger.Patch(log.SetLevel(log.Info))

	ctx, cancel := context.WithCancel(context.Background())

	// start block producing node
	blockProducingNodebasePath := t.TempDir()
	genesisPath := libutils.GetDevGenesisSpecPathTest(t)
	configPath := config.CreateDefault(t)
	alice := node.New(t,
		node.SetIndex(0),
		node.SetBasePath(blockProducingNodebasePath),
		node.SetGenesis(genesisPath),
		node.SetConfig(configPath),
		node.SetBabeLead(true))

	alice.InitAndStartTest(ctx, t, cancel)

	time.Sleep(time.Second * 15)

	// start syncing node
	syncingNodeBasePath := t.TempDir()
	configPath = config.CreateNoBabe(t)
	bob := node.New(t,
		node.SetIndex(1),
		node.SetBasePath(syncingNodeBasePath),
		node.SetGenesis(genesisPath),
		node.SetConfig(configPath))

	bob.InitAndStartTest(ctx, t, cancel)

	nodes := node.Nodes{alice, bob}

	numCmps := 100
	for i := 0; i < numCmps; i++ {
		t.Log("comparing...", i)

		const compareTimeout = 5 * time.Second
		compareCtx, cancel := context.WithTimeout(ctx, compareTimeout)

		_ = compareBlocksByNumber(compareCtx, t, nodes, strconv.Itoa(i))

		cancel()
	}
}

func TestSync_Bench(t *testing.T) {
	utils.Logger.Patch(log.SetLevel(log.Info))
	const numBlocks uint = 64

	// start block producing node
	blockProducingNodebasePath := t.TempDir()
	genesisPath := libutils.GetDevGenesisSpecPathTest(t)
	configNoGrandpa := config.CreateNoGrandpa(t)
	alice := node.New(t,
		node.SetIndex(0),
		node.SetBasePath(blockProducingNodebasePath),
		node.SetGenesis(genesisPath),
		node.SetConfig(configNoGrandpa),
		node.SetBabeLead(true))

	ctx, cancel := context.WithCancel(context.Background())
	alice.InitAndStartTest(ctx, t, cancel)

	for {
		getChainHeadCtx, getChainCancel := context.WithTimeout(ctx, time.Second)
		header, err := rpc.GetChainHead(getChainHeadCtx, alice.GetRPCPort())
		getChainCancel()
		if err != nil {
			continue
		}

		if header.Number >= numBlocks {
			break
		}

		time.Sleep(3 * time.Second)
	}

	pauseBabeCtx, pauseBabeCancel := context.WithTimeout(ctx, time.Second)
	err := rpc.PauseBABE(pauseBabeCtx, alice.GetRPCPort())
	pauseBabeCancel()

	require.NoError(t, err)
	t.Log("BABE paused")

	// start syncing node
	syncingNodeBasePath := t.TempDir()
	configNoAuthority := config.CreateNotAuthority(t)
	bob := node.New(t,
		node.SetIndex(1),
		node.SetBasePath(syncingNodeBasePath),
		node.SetGenesis(genesisPath),
		node.SetConfig(configNoAuthority),
		node.SetBabeLead(true))

	bob.InitAndStartTest(ctx, t, cancel)
	require.NoError(t, err)

	nodes := node.Nodes{alice, bob}

	// see how long it takes to sync to block numBlocks
	last := numBlocks
	start := time.Now()
	var end time.Time

	for {
		if time.Since(start) >= testTimeout {
			t.Fatal("did not sync")
		}

		getChainHeadCtx, getChainHeadCancel := context.WithTimeout(ctx, time.Second)
		head, err := rpc.GetChainHead(getChainHeadCtx, bob.GetRPCPort())
		getChainHeadCancel()

		if err != nil {
			continue
		}

		if head.Number >= last {
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

	const compareTimeout = 5 * time.Second
	compareCtx, pauseBabeCancel := context.WithTimeout(ctx, compareTimeout)

	_ = compareBlocksByNumber(compareCtx, t, nodes, fmt.Sprint(numBlocks))

	pauseBabeCancel()

	time.Sleep(time.Second * 3)
}

func TestSync_Restart(t *testing.T) {
	// TODO: Fix this test and enable it.
	t.Skip("skipping TestSync_Restart")
	numNodes := 3
	utils.Logger.Patch(log.SetLevel(log.Info))

	mainCtx, mainCancel := context.WithCancel(context.Background())

	nodeCtxs := make([]context.Context, numNodes)
	nodeCancels := make([]context.CancelFunc, numNodes)
	nodeWaitErrs := make([]chan error, numNodes)
	for i := 0; i < numNodes; i++ {
		nodeCtxs[i], nodeCancels[i] = context.WithCancel(mainCtx)
		nodeWaitErrs[i] = make(chan error)
	}

	// Note we assume no runtime error in this test otherwise
	// it gets rather complex to handle runtime errors and stop
	// the test.

	// start block producing node first
	blockProducingNodeBasePath := t.TempDir()
	genesisPath := libutils.GetGssmrGenesisRawPathTest(t)
	configPath := config.CreateDefault(t)
	producingNode := node.New(t,
		node.SetIndex(numNodes-1),
		node.SetBasePath(blockProducingNodeBasePath),
		node.SetGenesis(genesisPath),
		node.SetConfig(configPath),
		node.SetBabeLead(true))

	err := producingNode.Init(mainCtx)
	require.NoError(t, err)

	err = producingNode.Start(nodeCtxs[0], nodeWaitErrs[0])
	t.Cleanup(func() {
		// note we need to use indexes since these
		// slice elements might change.
		nodeCancels[0]()
		<-nodeWaitErrs[0]
	})
	require.NoError(t, err)

	// wait and start rest of nodes
	time.Sleep(time.Second * 5)
	configPath = config.CreateNoBabe(t)
	nodes := node.MakeNodes(t, numNodes-1,
		node.SetGenesis(genesisPath), node.SetConfig(configPath))
	for i, node := range nodes {
		err := node.Init(mainCtx)
		require.NoError(t, err)

		err = node.Start(nodeCtxs[i+1], nodeWaitErrs[i+1])
		t.Cleanup(func() {
			// note we need to use indexes since these
			// slice elements might change.
			nodeCancels[i+1]()
			<-nodeWaitErrs[i+1]
		})
		require.NoError(t, err)
	}

	nodes = append(nodes, producingNode)

	// randomly turn off and on nodes
	onOffRoutineDone := make(chan struct{})
	go func() {
		defer close(onOffRoutineDone)
		for {
			select {
			case <-time.After(time.Second * 10):
				idx := rand.Intn(numNodes)

				// Stop node
				nodeCancels[idx]()
				<-nodeWaitErrs[idx]

				// Start node
				nodeCtxs[idx], nodeCancels[idx] = context.WithCancel(mainCtx)
				waitErr := make(chan error)
				err := nodes[idx].Start(nodeCtxs[idx], waitErr)
				nodeWaitErrs[idx] = waitErr
				if err != nil {
					assert.NoError(t, err) // cannot use require.NoError from a goroutine
					mainCancel()           // stop all operations
					return
				}
			case <-mainCtx.Done():
				return
			}
		}
	}()

	numCmps := 12
	for i := 0; i < numCmps; i++ {
		t.Log("comparing...", i)

		const compareTimeout = 5 * time.Second
		compareCtx, cancel := context.WithTimeout(mainCtx, compareTimeout)

		_ = compareBlocksByNumber(compareCtx, t, nodes, strconv.Itoa(i))

		cancel()

		time.Sleep(time.Second * 5)
	}

	mainCancel()
	<-onOffRoutineDone
}

func TestSync_SubmitExtrinsic(t *testing.T) {
	t.Skip()

	ctx, cancel := context.WithCancel(context.Background())

	// index of node to submit tx to
	idx := 0 // TODO: randomise this

	// start block producing node first
	blockProducingNodeBasePath := t.TempDir()
	genesisPath := libutils.GetDevGenesisSpecPathTest(t)
	configNoGrandpa := config.CreateNoGrandpa(t)
	producingNode := node.New(t,
		node.SetIndex(0),
		node.SetBasePath(blockProducingNodeBasePath),
		node.SetGenesis(genesisPath),
		node.SetConfig(configNoGrandpa),
		node.SetBabeLead(true))
	producingNode.InitAndStartTest(ctx, t, cancel)

	nodes := node.Nodes{producingNode}

	configNoAuthority := config.CreateNotAuthority(t)

	// Start rest of nodes
	basePath2 := t.TempDir()
	n := node.New(t,
		node.SetIndex(1),
		node.SetBasePath(basePath2),
		node.SetGenesis(genesisPath),
		node.SetConfig(configNoAuthority),
	)
	nodes = append(nodes, n)

	basePath3 := t.TempDir()
	n = node.New(t,
		node.SetIndex(2),
		node.SetBasePath(basePath3),
		node.SetGenesis(genesisPath),
		node.SetConfig(configNoAuthority),
	)
	nodes = append(nodes, n)

	// send tx to non-authority node
	api, err := gsrpc.NewSubstrateAPI(fmt.Sprintf("http://localhost:%s", nodes[idx].GetRPCPort()))
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

	// get starting header so that we can lookup blocks by number later
	getChainHeadCtx, getChainHeadCancel := context.WithTimeout(ctx, time.Second)
	prevHeader, err := rpc.GetChainHead(getChainHeadCtx, nodes[idx].GetRPCPort())
	getChainHeadCancel()
	require.NoError(t, err)

	// Send the extrinsic
	hash, err := api.RPC.Author.SubmitExtrinsic(ext)
	require.NoError(t, err)
	require.NotEqual(t, types.Hash{}, hash)

	time.Sleep(time.Second * 20)

	// wait until there's no more pending extrinsics
	for i := 0; i < maxRetries; i++ {
		getPendingExtsCtx, getPendingExtsCancel := context.WithTimeout(ctx, time.Second)
		exts := getPendingExtrinsics(getPendingExtsCtx, t, nodes[idx])
		getPendingExtsCancel()

		if len(exts) == 0 {
			break
		}

		time.Sleep(time.Second)
	}

	getChainHeadCtx, getChainHeadCancel = context.WithTimeout(ctx, time.Second)
	header, err := rpc.GetChainHead(getChainHeadCtx, nodes[idx].GetRPCPort())
	getChainHeadCancel()
	require.NoError(t, err)

	// search from child -> parent blocks for extrinsic
	var (
		resExts    []gosstypes.Extrinsic
		extInBlock uint
	)

	for i := 0; i < maxRetries; i++ {
		getBlockCtx, getBlockCancel := context.WithTimeout(ctx, time.Second)
		block, err := rpc.GetBlock(getBlockCtx, nodes[idx].GetRPCPort(), header.ParentHash)
		getBlockCancel()
		require.NoError(t, err)

		if block == nil {
			// couldn't get block, increment retry counter
			continue
		}

		header = &block.Header
		logger.Debugf("got block with header %s and body %v from node with key %s", header, block.Body, nodes[idx].GetKey())

		if block.Body != nil {
			resExts = block.Body

			logger.Debugf("extrinsics: %v", resExts)
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
		logger.Debugf("comparing extrinsic 0x%x against expected 0x%x", ext, extEnc)
		if strings.Compare(extEnc, common.BytesToHex(ext)) == 0 {
			included = true
		}
	}

	require.True(t, included)

	const compareTimeout = 5 * time.Second
	compareCtx, cancel := context.WithTimeout(ctx, compareTimeout)

	_ = compareBlocksByNumber(compareCtx, t, nodes, fmt.Sprint(extInBlock))

	cancel()
}

func Test_SubmitAndWatchExtrinsic(t *testing.T) {
	// start block producing node first
	blockProducingNodeBasePath := t.TempDir()
	genesisPath := libutils.GetDevGenesisSpecPathTest(t)
	configNoGrandpa := config.CreateNoGrandpa(t)
	producingNode := node.New(t,
		node.SetIndex(0),
		node.SetBasePath(blockProducingNodeBasePath),
		node.SetGenesis(genesisPath),
		node.SetConfig(configNoGrandpa),
		node.SetWebsocket(true),
		node.SetBabeLead(true),
	)
	ctx, cancel := context.WithCancel(context.Background())
	producingNode.InitAndStartTest(ctx, t, cancel)

	// send tx to non-authority node
	api, err := gsrpc.NewSubstrateAPI(fmt.Sprintf("ws://localhost:%s", producingNode.GetWSPort()))
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

	conn, _, err := websocket.DefaultDialer.Dial("ws://localhost:8546", nil)
	if err != nil {
		fmt.Println(err)
	}

	message := []byte(`{"id":1, "jsonrpc":"2.0", "method": "author_submitAndWatchExtrinsic", "params":["` + extEnc + `"]}`)

	err = conn.WriteMessage(websocket.TextMessage, message)
	require.NoError(t, err)

	var result []byte
	_, result, err = conn.ReadMessage()
	require.NoError(t, err)
	require.Equal(t, `{"jsonrpc":"2.0","result":1,"id":1}`+"\n", string(result))

	conn.SetReadDeadline(time.Now().Add(10 * time.Second))

	_, result, err = conn.ReadMessage()
	require.NoError(t, err)
	require.Equal(t, `{"jsonrpc":"2.0","method":"author_extrinsicUpdate",`+
		`"params":{"result":"ready","subscription":1}}`+"\n", string(result))

	_, result, err = conn.ReadMessage()
	require.NoError(t, err)
	require.Contains(t, string(result), `{"jsonrpc":"2.0",`+
		`"method":"author_extrinsicUpdate","params":{"result":{"inBlock":"`)

}

func TestSync_SubmitExtrinsicLoad(t *testing.T) {
	t.Skip()

	// Instantiate the API
	// send tx to non-authority node
	api, err := gsrpc.NewSubstrateAPI(fmt.Sprintf("ws://localhost:%s", "8546"))
	require.NoError(t, err)

	meta, err := api.RPC.State.GetMetadataLatest()
	require.NoError(t, err)

	// Create a call, transferring 12345 units to Bob
	bob, err := types.NewMultiAddressFromHexAccountID("0x8eaf04151687736326c9fea17e25fc5287613693c912909cb226aa4794f26a48")
	require.NoError(t, err)

	// 1 unit of transfer
	bal, ok := new(big.Int).SetString("1000", 10)
	require.True(t, ok, "failed to convert balance")

	c, err := types.NewCall(meta, "Balances.transfer", bob, types.NewUCompact(bal))
	require.NoError(t, err)

	// Create the extrinsic
	ext := types.NewExtrinsic(c)

	genesisHash, err := api.RPC.Chain.GetBlockHash(0)
	require.NoError(t, err)

	rv, err := api.RPC.State.GetRuntimeVersionLatest()
	require.NoError(t, err)

	alice := signature.TestKeyringPairAlice.PublicKey
	key, err := types.CreateStorageKey(meta, "System", "Account", alice)
	require.NoError(t, err)

	var accountInfo types.AccountInfo
	ok, err = api.RPC.State.GetStorageLatest(key, &accountInfo)
	require.NoError(t, err)
	require.True(t, ok)

	previous := accountInfo.Data.Free
	t.Logf("%#x has a balance of %v\n", alice, previous)
	t.Logf("You may leave this example running and transfer any value to %#x\n", alice)

	nonce := uint32(accountInfo.Nonce)
	for i := 0; i < 1000; i++ {
		t.Logf("nonce: %v", nonce)
		o := types.SignatureOptions{
			BlockHash:          genesisHash,
			Era:                types.ExtrinsicEra{IsMortalEra: false},
			GenesisHash:        genesisHash,
			Nonce:              types.NewUCompactFromUInt(uint64(nonce)),
			SpecVersion:        rv.SpecVersion,
			Tip:                types.NewUCompactFromUInt(0),
			TransactionVersion: rv.TransactionVersion,
		}

		nonce++
		// Sign the transaction using Alice's default account
		err = ext.Sign(signature.TestKeyringPairAlice, o)
		require.NoError(t, err)

		// Send the extrinsic
		hash, err := api.RPC.Author.SubmitExtrinsic(ext)
		require.NoError(t, err)
		require.NotEqual(t, types.Hash{}, hash)

		t.Logf("Balance transferred from Alice to Bob: %v\n", bal.String())
		// Output: Balance transferred from Alice to Bob: 100000000000000
	}
}

func TestStress_SecondarySlotProduction(t *testing.T) {
	rootPath, err := libutils.GetProjectRootPath()
	require.NoError(t, err)

	// genesis_two_auths_secondaryvrf_0_9_10.json has 2 authorities and block production by
	// secondary VRF slots enabled
	genesisTwoAuthsSecondaryVRF0_9_10 := filepath.Join(rootPath, "tests/utils/genesis_two_auths_secondaryvrf_0_9_10.json")

	testcases := []struct {
		description  string
		genesis      string
		allowedSlots gosstypes.AllowedSlots
	}{
		{
			description:  "with secondary vrf slots enabled",
			genesis:      genesisTwoAuthsSecondaryVRF0_9_10,
			allowedSlots: gosstypes.PrimaryAndSecondaryVRFSlots,
		},
	}
	const numNodes = 2
	for _, c := range testcases {
		t.Run(c.description, func(t *testing.T) {
			config := config.CreateDefault(t)

			nodes := node.MakeNodes(t, numNodes,
				node.SetGenesis(c.genesis), node.SetConfig(config))

			ctx, cancel := context.WithCancel(context.Background())
			nodes.InitAndStartTest(ctx, t, cancel)

			primaryCount := 0
			secondaryPlainCount := 0
			secondaryVRFCount := 0

			for i := 1; i < 10; i++ {
				fmt.Printf("%d iteration\n", i)

				getBlockHashCtx, cancel := context.WithTimeout(ctx, time.Second)
				hash, err := rpc.GetBlockHash(getBlockHashCtx, nodes[0].GetRPCPort(), fmt.Sprintf("%d", i))
				cancel()

				require.NoError(t, err)

				getBlockCtx, cancel := context.WithTimeout(ctx, time.Second)
				block, err := rpc.GetBlock(getBlockCtx, nodes[0].GetRPCPort(), hash)
				cancel()
				require.NoError(t, err)

				header := block.Header

				preDigestItem := header.Digest.Types[0]

				preDigest, ok := preDigestItem.Value().(gosstypes.PreRuntimeDigest)
				require.True(t, ok)

				babePreDigest, err := gosstypes.DecodeBabePreDigest(preDigest.Data)
				require.NoError(t, err)

				switch babePreDigest.(type) {
				case gosstypes.BabePrimaryPreDigest:
					primaryCount++
				case gosstypes.BabeSecondaryVRFPreDigest:
					secondaryVRFCount++
				case gosstypes.BabeSecondaryPlainPreDigest:
					secondaryPlainCount++
				}
				require.NotNil(t, babePreDigest)

				time.Sleep(10 * time.Second)
			}

			switch c.allowedSlots {
			case gosstypes.PrimaryAndSecondaryPlainSlots:
				assert.Greater(t, secondaryPlainCount, 0)
				assert.Empty(t, secondaryVRFCount)
			case gosstypes.PrimaryAndSecondaryVRFSlots:
				assert.Greater(t, secondaryVRFCount, 0)
				assert.Empty(t, secondaryPlainCount)
			case gosstypes.PrimarySlots:
				assert.Empty(t, secondaryPlainCount)
				assert.Empty(t, secondaryVRFCount)
			}
		})
	}
}
