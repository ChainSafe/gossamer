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
	"errors"
	"io/ioutil"
	"math/big"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/dot/state"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/common/variadic"
	"github.com/ChainSafe/gossamer/lib/genesis"
	"github.com/ChainSafe/gossamer/lib/runtime"
	rtstorage "github.com/ChainSafe/gossamer/lib/runtime/storage"
	"github.com/ChainSafe/gossamer/lib/runtime/wasmer"
	"github.com/ChainSafe/gossamer/lib/scale"
	"github.com/ChainSafe/gossamer/lib/transaction"
	"github.com/ChainSafe/gossamer/lib/trie"

	"github.com/ChainSafe/chaindb"
	log "github.com/ChainSafe/log15"
	"github.com/stretchr/testify/require"
)

func newTestGenesisWithTrieAndHeader(t *testing.T) (*genesis.Genesis, *trie.Trie, *types.Header) {
	gen, err := genesis.NewGenesisFromJSONRaw("../../chain/gssmr/genesis.json")
	require.NoError(t, err)

	genTrie, err := genesis.NewTrieFromGenesis(gen)
	require.NoError(t, err)

	genesisHeader, err := types.NewHeader(common.NewHash([]byte{0}), big.NewInt(0), genTrie.MustHash(), trie.EmptyHash, types.Digest{})
	require.NoError(t, err)
	return gen, genTrie, genesisHeader
}

func newTestSyncer(t *testing.T) *Service {
	wasmer.DefaultTestLogLvl = 3

	cfg := &Config{}
	testDatadirPath, _ := ioutil.TempDir("/tmp", "test-datadir-*")
	stateSrvc := state.NewService(testDatadirPath, log.LvlInfo)
	stateSrvc.UseMemDB()

	gen, genTrie, genHeader := newTestGenesisWithTrieAndHeader(t)
	err := stateSrvc.Initialize(gen, genHeader, genTrie)
	require.NoError(t, err)

	err = stateSrvc.Start()
	require.NoError(t, err)

	if cfg.BlockState == nil {
		cfg.BlockState = stateSrvc.Block
	}

	if cfg.StorageState == nil {
		cfg.StorageState = stateSrvc.Storage
	}

	if cfg.Runtime == nil {
		// set state to genesis state
		genState, err := rtstorage.NewTrieState(genTrie) //nolint
		require.NoError(t, err)

		rtCfg := &wasmer.Config{}
		rtCfg.Storage = genState
		rtCfg.LogLvl = 3

		instance, err := wasmer.NewRuntimeFromGenesis(gen, rtCfg) //nolint
		require.NoError(t, err)
		cfg.Runtime = instance
	}

	if cfg.TransactionState == nil {
		cfg.TransactionState = stateSrvc.Transaction
	}

	if cfg.Verifier == nil {
		cfg.Verifier = &mockVerifier{}
	}

	if cfg.LogLvl == 0 {
		cfg.LogLvl = log.LvlDebug
	}

	syncer, err := NewService(cfg)
	require.NoError(t, err)
	return syncer
}

func TestHandleBlockResponse(t *testing.T) {
	if testing.Short() {
		t.Skip() // this test takes around 4min to run
	}

	syncer := newTestSyncer(t)
	syncer.highestSeenBlock = big.NewInt(132)

	responder := newTestSyncer(t)
	parent, err := responder.blockState.(*state.BlockState).BestBlockHeader()
	require.NoError(t, err)

	for i := 0; i < 130; i++ {
		block := buildBlock(t, responder.runtime, parent)
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
	syncer := newTestSyncer(t)
	syncer.highestSeenBlock = big.NewInt(20)

	parent, err := syncer.blockState.(*state.BlockState).BestBlockHeader()
	require.NoError(t, err)

	for i := 0; i < 4; i++ {
		block := buildBlock(t, syncer.runtime, parent)
		err = syncer.blockState.AddBlock(block)
		require.NoError(t, err)
		parent = block.Header
	}

	responder := newTestSyncer(t)

	parent, err = responder.blockState.(*state.BlockState).BestBlockHeader()
	require.NoError(t, err)

	for i := 0; i < 16; i++ {
		block := buildBlock(t, responder.runtime, parent)
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
	syncer := newTestSyncer(t)

	ext := []byte("nootwashere")
	tx := &transaction.ValidTransaction{
		Extrinsic: ext,
		Validity:  &transaction.Validity{Priority: 1},
	}

	syncer.transactionState.(*state.TransactionState).Push(tx)

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
	syncer := newTestSyncer(t)
	_, err := syncer.ProcessBlockData(nil)
	require.Equal(t, ErrNilBlockData, err)
}

func TestHandleBlockResponse_BlockData(t *testing.T) {
	syncer := newTestSyncer(t)

	parent, err := syncer.blockState.(*state.BlockState).BestBlockHeader()
	require.NoError(t, err)
	block := buildBlock(t, syncer.runtime, parent)

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

func buildBlock(t *testing.T, instance runtime.Instance, parent *types.Header) *types.Block {
	header := &types.Header{
		ParentHash: parent.Hash(),
		Number:     big.NewInt(0).Add(parent.Number, big.NewInt(1)),
		Digest:     types.Digest{},
	}

	err := instance.InitializeBlock(header)
	require.NoError(t, err)

	idata := types.NewInherentsData()
	err = idata.SetInt64Inherent(types.Timstap0, uint64(time.Now().Unix()))
	require.NoError(t, err)

	err = idata.SetInt64Inherent(types.Babeslot, 1)
	require.NoError(t, err)

	err = idata.SetBigIntInherent(types.Finalnum, big.NewInt(0))
	require.NoError(t, err)

	ienc, err := idata.Encode()
	require.NoError(t, err)

	// Call BlockBuilder_inherent_extrinsics which returns the inherents as extrinsics
	inherentExts, err := instance.InherentExtrinsics(ienc)
	require.NoError(t, err)

	// decode inherent extrinsics
	exts, err := scale.Decode(inherentExts, [][]byte{})
	require.NoError(t, err)

	// apply each inherent extrinsic
	for _, ext := range exts.([][]byte) {
		in, err := scale.Encode(ext) //nolint
		require.NoError(t, err)

		ret, err := instance.ApplyExtrinsic(in)
		require.NoError(t, err)
		require.Equal(t, ret, []byte{0, 0})
	}

	res, err := instance.FinalizeBlock()
	require.NoError(t, err)
	res.Number = header.Number

	return &types.Block{
		Header: res,
		Body:   types.NewBody(inherentExts),
	}
}

func TestSyncer_ExecuteBlock(t *testing.T) {
	syncer := newTestSyncer(t)

	parent, err := syncer.blockState.(*state.BlockState).BestBlockHeader()
	require.NoError(t, err)
	block := buildBlock(t, syncer.runtime, parent)

	// reset parentState
	parentState, err := syncer.storageState.TrieState(&parent.StateRoot)
	require.NoError(t, err)
	syncer.runtime.SetContextStorage(parentState)

	_, err = syncer.runtime.ExecuteBlock(block)
	require.NoError(t, err)
}

func TestSyncer_HandleRuntimeChanges(t *testing.T) {
	syncer := newTestSyncer(t)

	_, err := runtime.GetRuntimeBlob(runtime.POLKADOT_RUNTIME_FP, runtime.POLKADOT_RUNTIME_URL)
	require.NoError(t, err)

	testRuntime, err := ioutil.ReadFile(runtime.POLKADOT_RUNTIME_FP)
	require.NoError(t, err)

	ts, err := syncer.storageState.TrieState(nil)
	require.NoError(t, err)

	ts.Set(common.CodeKey, testRuntime)
	err = syncer.handleRuntimeChanges(ts)
	require.NoError(t, err)
}

func TestSyncer_HandleJustification(t *testing.T) {
	syncer := newTestSyncer(t)

	header := &types.Header{
		Number: big.NewInt(1),
	}

	just := []byte("testjustification")

	syncer.handleJustification(header, just)

	res, err := syncer.blockState.GetJustification(header.Hash())
	require.NoError(t, err)
	require.Equal(t, just, res)
}
