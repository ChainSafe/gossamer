package sync

import (
	"errors"
	"math/big"
	"testing"

	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/dot/state"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common/variadic"
	"github.com/ChainSafe/gossamer/lib/transaction"

	"github.com/ChainSafe/chaindb"

	"github.com/stretchr/testify/require"
)

func TestChainProcessor_HandleBlockResponse_ValidChain(t *testing.T) {
	syncer := newTestSyncer(t)
	responder := newTestSyncer(t)

	// get responder to build valid chain
	parent, err := responder.blockState.(*state.BlockState).BestBlockHeader()
	require.NoError(t, err)

	rt, err := responder.blockState.GetRuntime(nil)
	require.NoError(t, err)

	for i := 0; i < maxResponseSize*2; i++ {
		block := BuildBlock(t, rt, parent, nil)
		err = responder.blockState.AddBlock(block)
		require.NoError(t, err)
		parent = block.Header
	}

	// syncer makes request for chain
	startNum := 1
	start, err := variadic.NewUint64OrHash(startNum)
	require.NoError(t, err)

	req := &network.BlockRequestMessage{
		RequestedData: network.RequestedDataHeader + network.RequestedDataBody,
		StartingBlock: start,
	}

	// get response
	resp, err := responder.CreateBlockResponse(req)
	require.NoError(t, err)

	// process response
	for _, bd := range resp.BlockData {
		err = syncer.chainProcessor.(*chainProcessor).processBlockData(bd)
		require.NoError(t, err)
	}

	// syncer makes request for chain again (block 129+)
	startNum = 129
	start, err = variadic.NewUint64OrHash(startNum)
	require.NoError(t, err)

	req = &network.BlockRequestMessage{
		RequestedData: network.RequestedDataHeader + network.RequestedDataBody,
		StartingBlock: start,
	}

	// get response
	resp, err = responder.CreateBlockResponse(req)
	require.NoError(t, err)

	// process response
	for _, bd := range resp.BlockData {
		err = syncer.chainProcessor.(*chainProcessor).processBlockData(bd)
		require.NoError(t, err)
	}
}

func TestChainProcessor_HandleBlockResponse_MissingBlocks(t *testing.T) {
	syncer := newTestSyncer(t)

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

	responder := newTestSyncer(t)

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

	// resp contains blocks 15 to 15 + maxResponseSize)
	resp, err := responder.CreateBlockResponse(req)
	require.NoError(t, err)

	for _, bd := range resp.BlockData {
		err = syncer.chainProcessor.(*chainProcessor).processBlockData(bd)
		require.True(t, errors.Is(err, chaindb.ErrKeyNotFound))
	}
}

func TestChainProcessor_RemoveIncludedExtrinsics(t *testing.T) {
	syncer := newTestSyncer(t)

	ext := []byte("nootwashere")
	tx := &transaction.ValidTransaction{
		Extrinsic: ext,
		Validity:  &transaction.Validity{Priority: 1},
	}

	_, err := syncer.chainProcessor.(*chainProcessor).transactionState.(*state.TransactionState).Push(tx)
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

	for _, bd := range msg.BlockData {
		err = syncer.chainProcessor.(*chainProcessor).processBlockData(bd)
		require.NoError(t, err)
	}

	inQueue := syncer.chainProcessor.(*chainProcessor).transactionState.(*state.TransactionState).Pop()
	require.Nil(t, inQueue, "queue should be empty")
}

func TestChainProcessor_HandleBlockResponse_NoBlockData(t *testing.T) {
	syncer := newTestSyncer(t)
	err := syncer.chainProcessor.(*chainProcessor).processBlockData(nil)
	require.Equal(t, ErrNilBlockData, err)
}

func TestChainProcessor_HandleBlockResponse_BlockData(t *testing.T) {
	syncer := newTestSyncer(t)

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

	for _, bd := range msg.BlockData {
		err = syncer.chainProcessor.(*chainProcessor).processBlockData(bd)
		require.NoError(t, err)
	}
}

func TestChainProcessor_ExecuteBlock(t *testing.T) {
	syncer := newTestSyncer(t)

	parent, err := syncer.blockState.(*state.BlockState).BestBlockHeader()
	require.NoError(t, err)

	rt, err := syncer.blockState.GetRuntime(nil)
	require.NoError(t, err)

	block := BuildBlock(t, rt, parent, nil)

	// reset parentState
	parentState, err := syncer.chainProcessor.(*chainProcessor).storageState.TrieState(&parent.StateRoot)
	require.NoError(t, err)
	rt.SetContextStorage(parentState)

	_, err = rt.ExecuteBlock(block)
	require.NoError(t, err)
}

func TestChainProcessor_HandleJustification(t *testing.T) {
	syncer := newTestSyncer(t)

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

	syncer.chainProcessor.(*chainProcessor).handleJustification(header, just)

	res, err := syncer.blockState.GetJustification(header.Hash())
	require.NoError(t, err)
	require.Equal(t, just, res)
}
