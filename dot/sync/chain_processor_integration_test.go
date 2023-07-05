//go:build integration

// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package sync

import (
	"errors"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/dot/state"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/babe/inherents"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/common/variadic"
	runtime "github.com/ChainSafe/gossamer/lib/runtime"
	"github.com/ChainSafe/gossamer/lib/transaction"
	"github.com/ChainSafe/gossamer/pkg/scale"

	"github.com/stretchr/testify/require"
)

func buildBlockWithSlotAndTimestamp(t *testing.T, instance runtime.Instance,
	parent *types.Header, currentSlot, timestamp uint64) *types.Block {
	t.Helper()

	digest := types.NewDigest()
	prd, err := types.NewBabeSecondaryPlainPreDigest(0, currentSlot).ToPreRuntimeDigest()
	require.NoError(t, err)
	err = digest.Add(*prd)
	require.NoError(t, err)
	header := &types.Header{
		ParentHash:     parent.Hash(),
		StateRoot:      common.Hash{},
		ExtrinsicsRoot: common.Hash{},
		Number:         parent.Number + 1,
		Digest:         digest,
	}

	err = instance.InitializeBlock(header)
	require.NoError(t, err)

	inherentData := types.NewInherentData()
	err = inherentData.SetInherent(types.Timstap0, timestamp)
	require.NoError(t, err)

	err = inherentData.SetInherent(types.Babeslot, currentSlot)
	require.NoError(t, err)

	parachainInherent := inherents.ParachainInherentData{
		ParentHeader: *parent,
	}

	err = inherentData.SetInherent(types.Parachn0, parachainInherent)
	require.NoError(t, err)

	err = inherentData.SetInherent(types.Newheads, []byte{0})
	require.NoError(t, err)

	encodedInherentData, err := inherentData.Encode()
	require.NoError(t, err)

	// Call BlockBuilder_inherent_extrinsics which returns the inherents as encoded extrinsics
	encodedInherentExtrinsics, err := instance.InherentExtrinsics(encodedInherentData)
	require.NoError(t, err)

	var inherentExtrinsics [][]byte
	err = scale.Unmarshal(encodedInherentExtrinsics, &inherentExtrinsics)
	require.NoError(t, err)

	for _, inherent := range inherentExtrinsics {
		encodedInherent, err := scale.Marshal(inherent)
		require.NoError(t, err)

		applyExtrinsicResult, err := instance.ApplyExtrinsic(encodedInherent)
		require.NoError(t, err)
		require.Equal(t, applyExtrinsicResult, []byte{0, 0})
	}

	finalisedHeader, err := instance.FinalizeBlock()
	require.NoError(t, err)

	body := types.Body(types.BytesArrayToExtrinsics(inherentExtrinsics))

	finalisedHeader.Number = header.Number
	finalisedHeader.Hash()

	return &types.Block{
		Header: *finalisedHeader,
		Body:   body,
	}
}

func buildAndAddBlocksToState(t *testing.T,
	runtime runtime.Instance, blockState *state.BlockState, amount uint) {

	t.Helper()

	parent, err := blockState.BestBlockHeader()
	require.NoError(t, err)

	babeConfig, err := runtime.BabeConfiguration()
	require.NoError(t, err)

	timestamp := uint64(time.Now().Unix())
	slotDuration := babeConfig.SlotDuration

	for i := uint(0); i < amount; i++ {
		// calculate the exact slot for each produced block
		currentSlot := timestamp / slotDuration

		block := buildBlockWithSlotAndTimestamp(t, runtime, parent, currentSlot, timestamp)
		err = blockState.AddBlock(block)
		require.NoError(t, err)
		parent = &block.Header

		// increase the timestamp by the slot duration
		// so we will get a different slot for the next block
		timestamp += slotDuration
	}

}

func TestChainProcessor_HandleBlockResponse_ValidChain(t *testing.T) {
	syncer := newTestSyncer(t)
	responder := newTestSyncer(t)

	bestBlockHash := responder.blockState.(*state.BlockState).BestBlockHash()
	runtimeInstance, err := responder.blockState.GetRuntime(bestBlockHash)
	require.NoError(t, err)

	buildAndAddBlocksToState(t, runtimeInstance,
		responder.blockState.(*state.BlockState), maxResponseSize*2)

	// syncer makes request for chain
	startNum := 1
	start, err := variadic.NewUint32OrHash(startNum)
	require.NoError(t, err)

	req := &network.BlockRequestMessage{
		RequestedData: network.RequestedDataHeader + network.RequestedDataBody,
		StartingBlock: *start,
	}

	// get response
	resp, err := responder.CreateBlockResponse(req)
	require.NoError(t, err)

	// process response
	for _, bd := range resp.BlockData {
		err = syncer.chainProcessor.(*chainProcessor).processBlockData(*bd)
		require.NoError(t, err)
	}

	// syncer makes request for chain again (block 129+)
	startNum = 129
	start, err = variadic.NewUint32OrHash(startNum)
	require.NoError(t, err)

	req = &network.BlockRequestMessage{
		RequestedData: network.RequestedDataHeader + network.RequestedDataBody,
		StartingBlock: *start,
	}

	// get response
	resp, err = responder.CreateBlockResponse(req)
	require.NoError(t, err)

	// process response
	for _, bd := range resp.BlockData {
		err = syncer.chainProcessor.(*chainProcessor).processBlockData(*bd)
		require.NoError(t, err)
	}
}

func TestChainProcessor_HandleBlockResponse_MissingBlocks(t *testing.T) {
	syncer := newTestSyncer(t)

	bestBlockHash := syncer.blockState.(*state.BlockState).BestBlockHash()
	syncerRuntime, err := syncer.blockState.GetRuntime(bestBlockHash)
	require.NoError(t, err)

	const syncerAmountOfBlocks = 4
	buildAndAddBlocksToState(t, syncerRuntime, syncer.blockState.(*state.BlockState), syncerAmountOfBlocks)

	responder := newTestSyncer(t)
	responderRuntime, err := responder.blockState.GetRuntime(bestBlockHash)
	require.NoError(t, err)

	const responderAmountOfBlocks = 16
	buildAndAddBlocksToState(t, responderRuntime, responder.blockState.(*state.BlockState), responderAmountOfBlocks)

	startNum := 15
	start, err := variadic.NewUint32OrHash(startNum)
	require.NoError(t, err)

	req := &network.BlockRequestMessage{
		RequestedData: 3,
		StartingBlock: *start,
	}

	// resp contains blocks 15 to 15 + maxResponseSize)
	resp, err := responder.CreateBlockResponse(req)
	require.NoError(t, err)

	for _, bd := range resp.BlockData {
		err = syncer.chainProcessor.(*chainProcessor).processBlockData(*bd)
		require.True(t, errors.Is(err, errFailedToGetParent))
	}
}

func TestChainProcessor_handleBody_ShouldRemoveIncludedExtrinsics(t *testing.T) {
	syncer := newTestSyncer(t)

	ext := []byte("nootwashere")
	tx := &transaction.ValidTransaction{
		Extrinsic: ext,
		Validity:  &transaction.Validity{Priority: 1},
	}

	_, err := syncer.chainProcessor.(*chainProcessor).transactionState.(*state.TransactionState).Push(tx)
	require.NoError(t, err)

	body := types.NewBody([]types.Extrinsic{ext})
	syncer.chainProcessor.(*chainProcessor).handleBody(body)

	inQueue := syncer.chainProcessor.(*chainProcessor).transactionState.(*state.TransactionState).Pop()
	require.Nil(t, inQueue, "queue should be empty")
}

func TestChainProcessor_HandleBlockResponse_BlockData(t *testing.T) {
	syncer := newTestSyncer(t)

	parent, err := syncer.blockState.(*state.BlockState).BestBlockHeader()
	require.NoError(t, err)

	runtimeInstance, err := syncer.blockState.GetRuntime(parent.Hash())
	require.NoError(t, err)

	babeConfig, err := runtimeInstance.BabeConfiguration()
	require.NoError(t, err)

	timestamp := uint64(time.Now().Unix())
	slotDuration := babeConfig.SlotDuration

	// calculate the exact slot for each produced block
	currentSlot := timestamp / slotDuration
	block := buildBlockWithSlotAndTimestamp(t, runtimeInstance, parent, currentSlot, timestamp)

	bd := []*types.BlockData{{
		Hash:          block.Header.Hash(),
		Header:        &block.Header,
		Body:          &block.Body,
		Receipt:       nil,
		MessageQueue:  nil,
		Justification: nil,
	}}
	msg := &network.BlockResponseMessage{
		BlockData: bd,
	}

	for _, bd := range msg.BlockData {
		err = syncer.chainProcessor.(*chainProcessor).processBlockData(*bd)
		require.NoError(t, err)
	}
}

func TestChainProcessor_ExecuteBlock(t *testing.T) {
	syncer := newTestSyncer(t)

	parent, err := syncer.blockState.(*state.BlockState).BestBlockHeader()
	require.NoError(t, err)

	bestBlockHash := syncer.blockState.(*state.BlockState).BestBlockHash()
	runtimeInstance, err := syncer.blockState.GetRuntime(bestBlockHash)
	require.NoError(t, err)

	babeConfig, err := runtimeInstance.BabeConfiguration()
	require.NoError(t, err)

	timestamp := uint64(time.Now().Unix())
	slotDuration := babeConfig.SlotDuration

	// calculate the exact slot for each produced block
	currentSlot := timestamp / slotDuration
	block := buildBlockWithSlotAndTimestamp(t, runtimeInstance, parent, currentSlot, timestamp)

	// reset parentState
	parentState, err := syncer.chainProcessor.(*chainProcessor).storageState.TrieState(&parent.StateRoot)
	require.NoError(t, err)
	runtimeInstance.SetContextStorage(parentState)

	_, err = runtimeInstance.ExecuteBlock(block)
	require.NoError(t, err)
}

func TestChainProcessor_HandleJustification(t *testing.T) {
	syncer := newTestSyncer(t)

	d, err := types.NewBabeSecondaryPlainPreDigest(0, 1).ToPreRuntimeDigest()
	require.NoError(t, err)
	digest := types.NewDigest()
	err = digest.Add(*d)
	require.NoError(t, err)

	header := &types.Header{
		ParentHash: syncer.blockState.(*state.BlockState).GenesisHash(),
		Number:     1,
		Digest:     digest,
	}

	just := []byte("testjustification")

	err = syncer.blockState.(*state.BlockState).AddBlock(&types.Block{
		Header: *header,
		Body:   types.Body{},
	})
	require.NoError(t, err)

	err = syncer.chainProcessor.(*chainProcessor).handleJustification(header, just)
	require.NoError(t, err)

	res, err := syncer.blockState.GetJustification(header.Hash())
	require.NoError(t, err)
	require.Equal(t, just, res)
}

func TestChainProcessor_processReadyBlocks_errFailedToGetParent(t *testing.T) {
	syncer := newTestSyncer(t)
	processor := syncer.chainProcessor.(*chainProcessor)
	go processor.processReadyBlocks()
	defer processor.cancel()

	header := &types.Header{
		Number: 1,
	}

	processor.readyBlocks.push(&types.BlockData{
		Header: header,
		Body:   &types.Body{},
	})

	time.Sleep(time.Millisecond * 100)
	require.True(t, processor.pendingBlocks.(*disjointBlockSet).hasBlock(header.Hash()))
}
