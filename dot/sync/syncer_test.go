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
	"fmt"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"math/big"
	"os"
	"testing"

	"github.com/ChainSafe/chaindb"
	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/dot/state"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common/optional"
	"github.com/ChainSafe/gossamer/lib/common/variadic"
	"github.com/ChainSafe/gossamer/lib/runtime"
	"github.com/ChainSafe/gossamer/lib/transaction"

	log "github.com/ChainSafe/log15"
	"github.com/stretchr/testify/require"
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

func TestHandleBlockResponse(t *testing.T) {
	if testing.Short() {
		t.Skip() // this test takes around 4min to run
	}

	syncer := NewTestSyncer(t, false)
	syncer.highestSeenBlock = big.NewInt(132)

	responder := NewTestSyncer(t, false)
	parent, err := responder.blockState.(*state.BlockState).BestBlockHeaderVdt()
	require.NoError(t, err)

	rt, err := responder.blockState.GetRuntime(nil)
	require.NoError(t, err)

	for i := 0; i < 130; i++ {
		block := BuildBlockVdt(t, rt, parent, nil)
		err = responder.blockState.AddBlockVdt(block)
		require.NoError(t, err)
		parent = &block.Header
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

	parent, err := syncer.blockState.(*state.BlockState).BestBlockHeaderVdt()
	require.NoError(t, err)

	rt, err := syncer.blockState.GetRuntime(nil)
	require.NoError(t, err)

	for i := 0; i < 4; i++ {
		block := BuildBlockVdt(t, rt, parent, nil)
		err = syncer.blockState.AddBlockVdt(block)
		require.NoError(t, err)
		parent = &block.Header
	}

	responder := NewTestSyncer(t, false)

	parent, err = responder.blockState.(*state.BlockState).BestBlockHeaderVdt()
	require.NoError(t, err)

	rt, err = responder.blockState.GetRuntime(nil)
	require.NoError(t, err)

	for i := 0; i < 16; i++ {
		block := BuildBlockVdt(t, rt, parent, nil)
		err = responder.blockState.AddBlockVdt(block)
		require.NoError(t, err)
		parent = &block.Header
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

	bd := &types.BlockDataVdt{
		Body: body,
	}

	msg := &network.BlockResponseMessageNew{
		BlockData: []*types.BlockDataVdt{bd},
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

	parent2, err := syncer.blockState.(*state.BlockState).BestBlockHeader()
	require.NoError(t, err)

	enc, err := parent2.Encode()
	require.NoError(t, err)


	rt, err := syncer.blockState.GetRuntime(nil)
	require.NoError(t, err)

	blockO := BuildBlock(t, rt, parent2, nil)

	fmt.Println("Original block digest in test")
	fmt.Println(blockO.Header.Digest)


	//require.Equal(t, blockO.Header.StateRoot, block.Header.StateRoot)


	bdOld := []*types.BlockData{{
		Hash:          blockO.Header.Hash(),
		Header:        blockO.Header.AsOptional(),
		Body:          blockO.Body.AsOptional(),
		Receipt:       nil,
		MessageQueue:  nil,
		Justification: nil,
	}}
	msgOld := &network.BlockResponseMessage{
		BlockData: bdOld,
	}

	_, err = syncer.ProcessBlockDataOld(msgOld.BlockData)

	parent, err := syncer.blockState.(*state.BlockState).BestBlockHeaderVdt()
	require.NoError(t, err)

	encVdt, err := scale.Marshal(*parent)
	require.NoError(t, err)
	require.Equal(t, enc, encVdt)

	fmt.Println("")

	block := BuildBlockVdt(t, rt, parent, nil)

	fmt.Println("Vdt block digest in test")
	fmt.Println(block.Header.Digest)

	bd := []*types.BlockDataVdt{{
		Hash:          block.Header.Hash(),
		Header:        &block.Header,
		Body:          &block.Body,
		Receipt:       nil,
		MessageQueue:  nil,
		Justification: nil,
	}}
	msg := &network.BlockResponseMessageNew{
		BlockData: bd,
	}

	_, err = syncer.ProcessBlockData(msg.BlockData)
	require.Nil(t, err)
}

func TestSyncer_ExecuteBlock(t *testing.T) {
	syncer := NewTestSyncer(t, false)

	parent, err := syncer.blockState.(*state.BlockState).BestBlockHeaderVdt()
	require.NoError(t, err)

	rt, err := syncer.blockState.GetRuntime(nil)
	require.NoError(t, err)

	block := BuildBlockVdt(t, rt, parent, nil)

	// reset parentState
	parentState, err := syncer.storageState.TrieState(&parent.StateRoot)
	require.NoError(t, err)
	rt.SetContextStorage(parentState)

	_, err = rt.ExecuteBlockVdt(block)
	require.NoError(t, err)
}

func TestSyncer_HandleJustification(t *testing.T) {
	syncer := NewTestSyncer(t, false)

	d := types.NewBabeSecondaryPlainPreDigest(0, 1).ToPreRuntimeDigest()
	//header := &types.Header{
	//	ParentHash: syncer.blockState.(*state.BlockState).GenesisHash(),
	//	Number:     big.NewInt(1),
	//	Digest:     types.Digest{d},
	//}
	digest := types.NewDigestVdt()
	digest.Add(*d)
	header := &types.HeaderVdt{
		ParentHash: syncer.blockState.(*state.BlockState).GenesisHash(),
		Number:     big.NewInt(1),
		Digest:     digest,
	}

	just := []byte("testjustification")

	err := syncer.blockState.AddBlockVdt(&types.BlockVdt{
		Header: *header,
		Body:   types.Body{},
	})
	require.NoError(t, err)

	syncer.handleJustificationVdt(header, just)

	res, err := syncer.blockState.GetJustification(header.Hash())
	require.NoError(t, err)
	require.Equal(t, just, res)
}

func TestSyncer_ProcessJustification(t *testing.T) {
	syncer := NewTestSyncer(t, false)

	parent, err := syncer.blockState.(*state.BlockState).BestBlockHeaderVdt()
	require.NoError(t, err)

	rt, err := syncer.blockState.GetRuntime(nil)
	require.NoError(t, err)

	block := BuildBlockVdt(t, rt, parent, nil)
	digest := types.NewDigestVdt()
	digest.Add(*types.NewBabeSecondaryPlainPreDigest(0, 1).ToPreRuntimeDigest())
	block.Header.Digest = digest

	err = syncer.blockState.(*state.BlockState).AddBlockVdt(block)
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
