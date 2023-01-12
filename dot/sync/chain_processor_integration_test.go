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
	"github.com/ChainSafe/gossamer/lib/babe"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/common/variadic"
	"github.com/ChainSafe/gossamer/lib/transaction"
	"github.com/ChainSafe/gossamer/pkg/scale"

	"github.com/stretchr/testify/require"
)

func buildBlockWithSlotAndTimestamp(t *testing.T, instance state.Runtime,
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

	idata := types.NewInherentData()
	err = idata.SetInherent(types.Timstap0, timestamp)
	require.NoError(t, err)

	err = idata.SetInherent(types.Babeslot, currentSlot)
	require.NoError(t, err)

	parachainInherent := babe.ParachainInherentData{
		ParentHeader: *parent,
	}

	err = idata.SetInherent(types.Parachn0, parachainInherent)
	require.NoError(t, err)

	err = idata.SetInherent(types.Newheads, []byte{0})
	require.NoError(t, err)

	ienc, err := idata.Encode()
	require.NoError(t, err)

	// Call BlockBuilder_inherent_extrinsics which returns the inherents as encoded extrinsics
	inherentExts, err := instance.InherentExtrinsics(ienc)
	require.NoError(t, err)

	// decode inherent extrinsics
	var inExts [][]byte
	err = scale.Unmarshal(inherentExts, &inExts)
	require.NoError(t, err)

	// apply each inherent extrinsic
	for _, inherent := range inExts {
		in, err := scale.Marshal(inherent)
		require.NoError(t, err)

		ret, err := instance.ApplyExtrinsic(in)
		require.NoError(t, err)
		require.Equal(t, ret, []byte{0, 0})
	}

	res, err := instance.FinalizeBlock()
	require.NoError(t, err)

	body := types.Body(types.BytesArrayToExtrinsics(inExts))

	res.Number = header.Number
	res.Hash()

	return &types.Block{
		Header: *res,
		Body:   body,
	}
}

// TODO: add test against latest gssmr runtime
// See https://github.com/ChainSafe/gossamer/issues/2703
func TestChainProcessor_HandleBlockResponse_ValidChain(t *testing.T) {
	syncer := newTestSyncer(t)
	responder := newTestSyncer(t)

	// get responder to build valid chain
	parent, err := responder.blockState.(*state.BlockState).BestBlockHeader()
	require.NoError(t, err)

	bestBlockHash := responder.blockState.(*state.BlockState).BestBlockHash()
	rt, err := responder.blockState.GetRuntime(bestBlockHash)
	require.NoError(t, err)

	babeCfg, err := rt.BabeConfiguration()
	require.NoError(t, err)

	timestamp := uint64(time.Now().Unix())
	slotDuration := babeCfg.SlotDuration

	for i := 0; i < maxResponseSize*2; i++ {
		// calcule the exact slot for each produced block
		currentSlot := timestamp / slotDuration

		block := buildBlockWithSlotAndTimestamp(t, rt, parent, currentSlot, timestamp)
		err = responder.blockState.(*state.BlockState).AddBlock(block)
		require.NoError(t, err)
		parent = &block.Header

		// increase the timestamp by the slot duration
		// so we will get a different slot for the next block
		timestamp += slotDuration
	}

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

// TODO: add test against latest gssmr runtime
// See https://github.com/ChainSafe/gossamer/issues/2703
func TestChainProcessor_HandleBlockResponse_MissingBlocks(t *testing.T) {
	syncer := newTestSyncer(t)

	parent, err := syncer.blockState.(*state.BlockState).BestBlockHeader()
	require.NoError(t, err)

	bestBlockHash := syncer.blockState.(*state.BlockState).BestBlockHash()
	rt, err := syncer.blockState.GetRuntime(bestBlockHash)
	require.NoError(t, err)

	babeCfg, err := rt.BabeConfiguration()
	require.NoError(t, err)

	timestamp := uint64(time.Now().Unix())
	slotDuration := babeCfg.SlotDuration

	for i := 0; i < 4; i++ {
		// calcule the exact slot for each produced block
		currentSlot := timestamp / slotDuration

		block := buildBlockWithSlotAndTimestamp(t, rt, parent, currentSlot, timestamp)
		err = syncer.blockState.(*state.BlockState).AddBlock(block)
		require.NoError(t, err)
		parent = &block.Header

		// increase the timestamp by the slot duration
		// so we will get a different slot for the next block
		timestamp += slotDuration
	}

	responder := newTestSyncer(t)

	parent, err = responder.blockState.(*state.BlockState).BestBlockHeader()
	require.NoError(t, err)

	rt, err = responder.blockState.GetRuntime(bestBlockHash)
	require.NoError(t, err)

	babeCfg, err = rt.BabeConfiguration()
	require.NoError(t, err)

	timestamp = uint64(time.Now().Unix())
	slotDuration = babeCfg.SlotDuration

	for i := 0; i < 16; i++ {
		// calcule the exact slot for each produced block
		currentSlot := timestamp / slotDuration

		block := buildBlockWithSlotAndTimestamp(t, rt, parent, currentSlot, timestamp)
		err = responder.blockState.(*state.BlockState).AddBlock(block)
		require.NoError(t, err)
		parent = &block.Header

		// increase the timestamp by the slot duration
		// so we will get a different slot for the next block
		timestamp += slotDuration
	}

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

// TODO: add test against latest gssmr runtime
// See https://github.com/ChainSafe/gossamer/issues/2703
func TestChainProcessor_HandleBlockResponse_BlockData(t *testing.T) {
	syncer := newTestSyncer(t)

	parent, err := syncer.blockState.(*state.BlockState).BestBlockHeader()
	require.NoError(t, err)

	rt, err := syncer.blockState.GetRuntime(parent.Hash())
	require.NoError(t, err)

	block := BuildBlock(t, rt, parent, nil)

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

// TODO: add test against latest gssmr runtime
// See https://github.com/ChainSafe/gossamer/issues/2703
func TestChainProcessor_ExecuteBlock(t *testing.T) {
	syncer := newTestSyncer(t)

	parent, err := syncer.blockState.(*state.BlockState).BestBlockHeader()
	require.NoError(t, err)

	bestBlockHash := syncer.blockState.(*state.BlockState).BestBlockHash()
	rt, err := syncer.blockState.GetRuntime(bestBlockHash)
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
