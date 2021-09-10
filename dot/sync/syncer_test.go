// Copyright 2019 ChainSafe Systems (ON) Corp.
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

package sync

import (
	"io/ioutil"
	"math/big"
	"os"
	"testing"

	"github.com/stretchr/testify/mock"

	"github.com/ChainSafe/gossamer/dot/state"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/genesis"
	"github.com/ChainSafe/gossamer/lib/runtime"
	rtstorage "github.com/ChainSafe/gossamer/lib/runtime/storage"
	"github.com/ChainSafe/gossamer/lib/runtime/wasmer"
	"github.com/ChainSafe/gossamer/lib/trie"
	log "github.com/ChainSafe/log15"
	"github.com/stretchr/testify/require"

	syncmocks "github.com/ChainSafe/gossamer/dot/sync/mocks"
)

func TestMain(m *testing.M) {
	wasmFilePaths, err := runtime.GenerateRuntimeWasmFile()
	if err != nil {
		log.Error("failed to generate runtime wasm file", err)
		os.Exit(1)
	}

	// Start all tests
	code := m.Run()

	runtime.RemoveFiles(wasmFilePaths)
	os.Exit(code)
}

func newMockFinalityGadget() *syncmocks.FinalityGadget {
	m := new(syncmocks.FinalityGadget)
	// using []uint8 instead of []byte: https://github.com/stretchr/testify/pull/969
	m.On("VerifyBlockJustification", mock.AnythingOfType("common.Hash"), mock.AnythingOfType("[]uint8")).Return(nil)
	return m
}

func newMockVerifier() *syncmocks.MockVerifier {
	m := new(syncmocks.MockVerifier)
	m.On("VerifyBlock", mock.AnythingOfType("*types.Header")).Return(nil)
	return m
}

func newMockNetwork() *syncmocks.MockNetwork {
	m := new(syncmocks.MockNetwork)
	m.On("DoBlockRequest", mock.AnythingOfType("peer.ID"), mock.AnythingOfType("*BlockRequestMessage")).Return(nil, nil)
	return m
}

func newTestSyncer(t *testing.T, usePolkadotGenesis bool) *Service {
	wasmer.DefaultTestLogLvl = 3

	cfg := &Config{}
	testDatadirPath, _ := ioutil.TempDir("/tmp", "test-datadir-*")

	scfg := state.Config{
		Path:     testDatadirPath,
		LogLevel: log.LvlInfo,
	}
	stateSrvc := state.NewService(scfg)
	stateSrvc.UseMemDB()

	gen, genTrie, genHeader := newTestGenesisWithTrieAndHeader(t, usePolkadotGenesis)
	err := stateSrvc.Initialise(gen, genHeader, genTrie)
	require.NoError(t, err)

	err = stateSrvc.Start()
	require.NoError(t, err)

	if cfg.BlockState == nil {
		cfg.BlockState = stateSrvc.Block
	}

	if cfg.StorageState == nil {
		cfg.StorageState = stateSrvc.Storage
	}

	cfg.BlockImportHandler = new(syncmocks.MockBlockImportHandler)
	cfg.BlockImportHandler.(*syncmocks.MockBlockImportHandler).On("HandleBlockImport", mock.AnythingOfType("*types.Block"), mock.AnythingOfType("*storage.TrieState")).Return(nil)

	// initialize runtime
	genState, err := rtstorage.NewTrieState(genTrie) //nolint
	require.NoError(t, err)

	rtCfg := &wasmer.Config{}
	rtCfg.Storage = genState
	rtCfg.LogLvl = 3

	rtCfg.CodeHash, err = cfg.StorageState.LoadCodeHash(nil)
	require.NoError(t, err)

	instance, err := wasmer.NewRuntimeFromGenesis(gen, rtCfg)
	require.NoError(t, err)

	cfg.BlockState.StoreRuntime(cfg.BlockState.BestBlockHash(), instance)

	cfg.TransactionState = stateSrvc.Transaction
	cfg.BabeVerifier = newMockVerifier()
	cfg.LogLvl = log.LvlInfo
	cfg.FinalityGadget = newMockFinalityGadget()
	cfg.Network = newMockNetwork()

	syncer, err := NewService(cfg)
	require.NoError(t, err)
	return syncer
}

func newTestGenesisWithTrieAndHeader(t *testing.T, usePolkadotGenesis bool) (*genesis.Genesis, *trie.Trie, *types.Header) {
	fp := "../../chain/gssmr/genesis.json"
	if usePolkadotGenesis {
		fp = "../../chain/polkadot/genesis.json"
	}

	gen, err := genesis.NewGenesisFromJSONRaw(fp)
	require.NoError(t, err)

	genTrie, err := genesis.NewTrieFromGenesis(gen)
	require.NoError(t, err)

	genesisHeader, err := types.NewHeader(common.NewHash([]byte{0}), genTrie.MustHash(), trie.EmptyHash, big.NewInt(0), types.Digest{})
	require.NoError(t, err)
	return gen, genTrie, genesisHeader
}

/*

func TestHandleBlockResponse(t *testing.T) {
	if testing.Short() {
		t.Skip() // this test takes around 4min to run
	}

	syncer := NewTestSyncer(t, false)
	syncer.highestSeenBlock = big.NewInt(132)

	responder := NewTestSyncer(t, false)
	parent, err := responder.blockState.(*state.BlockState).BestBlockHeader()
	require.NoError(t, err)

	rt, err := responder.blockState.GetRuntime(nil)
	require.NoError(t, err)

	for i := 0; i < 130; i++ {
		block := BuildBlock(t, rt, parent, nil)
		err = responder.blockState.AddBlock(block)
		require.NoError(t, err)
		parent = block.Header
	}

	startNum := 1
	start, err := variadic.NewUint64OrHash(startNum)
	require.NoError(t, err)

	req := &network.BlockRequestMessage{
		RequestedData: 3,
		StartingBlock: start,
	}

	resp, err := responder.CreateBlockResponse(req)
	require.NoError(t, err)

	_, err = syncer.ProcessBlockData(resp.BlockData)
	require.NoError(t, err)

	resp2, err := responder.CreateBlockResponse(req)
	require.NoError(t, err)
	_, err = syncer.ProcessBlockData(resp2.BlockData)
	require.NoError(t, err)
	// response should contain blocks 13 to 20, and we should be synced
	require.True(t, syncer.synced)
}

func TestHandleBlockResponse_MissingBlocks(t *testing.T) {
	syncer := NewTestSyncer(t, false)
	syncer.highestSeenBlock = big.NewInt(20)

	parent, err := syncer.blockState.(*state.BlockState).BestBlockHeader()
	require.NoError(t, err)

	rt, err := syncer.blockState.GetRuntime(nil)
	require.NoError(t, err)

	for i := 0; i < 4; i++ {
		block := BuildBlock(t, rt, parent, nil)
		err = syncer.blockState.AddBlock(block)
		require.NoError(t, err)
		parent = block.Header
	}

	responder := NewTestSyncer(t, false)

	parent, err = responder.blockState.(*state.BlockState).BestBlockHeader()
	require.NoError(t, err)

	rt, err = responder.blockState.GetRuntime(nil)
	require.NoError(t, err)

	for i := 0; i < 16; i++ {
		block := BuildBlock(t, rt, parent, nil)
		err = responder.blockState.AddBlock(block)
		require.NoError(t, err)
		parent = block.Header
	}

	startNum := 15
	start, err := variadic.NewUint64OrHash(startNum)
	require.NoError(t, err)

	req := &network.BlockRequestMessage{
		RequestedData: 3,
		StartingBlock: start,
	}

	// resp contains blocks 16 + (16 + maxResponseSize)
	resp, err := responder.CreateBlockResponse(req)
	require.NoError(t, err)

	// request should start from block 5 (best block number + 1)
	syncer.synced = false
	_, err = syncer.ProcessBlockData(resp.BlockData)
	require.True(t, errors.Is(err, chaindb.ErrKeyNotFound))
}

func TestRemoveIncludedExtrinsics(t *testing.T) {
	syncer := NewTestSyncer(t, false)

	ext := []byte("nootwashere")
	tx := &transaction.ValidTransaction{
		Extrinsic: ext,
		Validity:  &transaction.Validity{Priority: 1},
	}

	_, err := syncer.transactionState.(*state.TransactionState).Push(tx)
	require.NoError(t, err)

	exts := []types.Extrinsic{ext}
	body, err := types.NewBodyFromExtrinsics(exts)
	require.NoError(t, err)

	bd := &types.BlockData{
		Body: body.AsOptional(),
	}

	msg := &network.BlockResponseMessage{
		BlockData: []*types.BlockData{bd},
	}

	_, err = syncer.ProcessBlockData(msg.BlockData)
	require.NoError(t, err)

	inQueue := syncer.transactionState.(*state.TransactionState).Pop()
	require.Nil(t, inQueue, "queue should be empty")
}

func TestHandleBlockResponse_NoBlockData(t *testing.T) {
	syncer := NewTestSyncer(t, false)
	_, err := syncer.ProcessBlockData(nil)
	require.Equal(t, ErrNilBlockData, err)
}

func TestHandleBlockResponse_BlockData(t *testing.T) {
	syncer := NewTestSyncer(t, false)

	parent, err := syncer.blockState.(*state.BlockState).BestBlockHeader()
	require.NoError(t, err)

	rt, err := syncer.blockState.GetRuntime(nil)
	require.NoError(t, err)

	block := BuildBlock(t, rt, parent, nil)

	bd := []*types.BlockData{{
		Hash:          block.Header.Hash(),
		Header:        block.Header.AsOptional(),
		Body:          block.Body.AsOptional(),
		Receipt:       nil,
		MessageQueue:  nil,
		Justification: nil,
	}}
	msg := &network.BlockResponseMessage{
		BlockData: bd,
	}

	_, err = syncer.ProcessBlockData(msg.BlockData)
	require.Nil(t, err)
}

func TestSyncer_ExecuteBlock(t *testing.T) {
	syncer := NewTestSyncer(t, false)

	parent, err := syncer.blockState.(*state.BlockState).BestBlockHeader()
	require.NoError(t, err)

	rt, err := syncer.blockState.GetRuntime(nil)
	require.NoError(t, err)

	block := BuildBlock(t, rt, parent, nil)

	// reset parentState
	parentState, err := syncer.storageState.TrieState(&parent.StateRoot)
	require.NoError(t, err)
	rt.SetContextStorage(parentState)

	_, err = rt.ExecuteBlock(block)
	require.NoError(t, err)
}

func TestSyncer_HandleJustification(t *testing.T) {
	syncer := NewTestSyncer(t, false)

	d := types.NewBabeSecondaryPlainPreDigest(0, 1).ToPreRuntimeDigest()
	header := &types.Header{
		ParentHash: syncer.blockState.(*state.BlockState).GenesisHash(),
		Number:     big.NewInt(1),
		Digest:     types.Digest{d},
	}

	just := []byte("testjustification")

	err := syncer.blockState.AddBlock(&types.Block{
		Header: header,
		Body:   &types.Body{},
	})
	require.NoError(t, err)

	syncer.handleJustification(header, just)

	res, err := syncer.blockState.GetJustification(header.Hash())
	require.NoError(t, err)
	require.Equal(t, just, res)
}

func TestSyncer_ProcessJustification(t *testing.T) {
	syncer := NewTestSyncer(t, false)

	parent, err := syncer.blockState.(*state.BlockState).BestBlockHeader()
	require.NoError(t, err)

	rt, err := syncer.blockState.GetRuntime(nil)
	require.NoError(t, err)

	block := BuildBlock(t, rt, parent, nil)
	block.Header.Digest = types.Digest{
		types.NewBabeSecondaryPlainPreDigest(0, 1).ToPreRuntimeDigest(),
	}

	err = syncer.blockState.(*state.BlockState).AddBlock(block)
	require.NoError(t, err)

	just := []byte("testjustification")

	data := []*types.BlockData{
		{
			Hash:          syncer.blockState.BestBlockHash(),
			Justification: optional.NewBytes(true, just),
		},
	}

	_, err = syncer.ProcessJustification(data)
	require.NoError(t, err)

	res, err := syncer.blockState.GetJustification(syncer.blockState.BestBlockHash())
	require.NoError(t, err)
	require.Equal(t, just, res)
}

*/
