// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package state

import (
	"math/big"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/trie"
	"github.com/ChainSafe/gossamer/pkg/scale"

	"github.com/stretchr/testify/require"
)

var sampleBlockBody = *types.NewBody([]types.Extrinsic{[]byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}})

var testGenesisHeader = &types.Header{
	Number:    big.NewInt(0),
	StateRoot: trie.EmptyHash,
	Digest:    types.NewDigest(),
}

func newTestBlockState(t *testing.T, header *types.Header) *BlockState {
	db := NewInMemoryDB(t)
	if header == nil {
		header = testGenesisHeader
	}

	bs, err := NewBlockStateFromGenesis(db, header)
	require.NoError(t, err)
	return bs
}

func TestSetAndGetHeader(t *testing.T) {
	bs := newTestBlockState(t, nil)

	header := &types.Header{
		Number:    big.NewInt(0),
		StateRoot: trie.EmptyHash,
		Digest:    types.NewDigest(),
	}

	err := bs.SetHeader(header)
	require.NoError(t, err)

	res, err := bs.GetHeader(header.Hash())
	require.NoError(t, err)
	require.Equal(t, header, res)
}

func TestHasHeader(t *testing.T) {
	bs := newTestBlockState(t, nil)

	header := &types.Header{
		Number:    big.NewInt(0),
		StateRoot: trie.EmptyHash,
		Digest:    types.NewDigest(),
	}

	err := bs.SetHeader(header)
	require.NoError(t, err)

	has, err := bs.HasHeader(header.Hash())
	require.NoError(t, err)
	require.Equal(t, true, has)
}

func TestGetBlockByNumber(t *testing.T) {
	bs := newTestBlockState(t, testGenesisHeader)

	blockHeader := &types.Header{
		ParentHash: testGenesisHeader.Hash(),
		Number:     big.NewInt(1),
		Digest:     types.NewDigest(),
	}

	block := &types.Block{
		Header: *blockHeader,
		Body:   sampleBlockBody,
	}

	err := bs.AddBlock(block)
	require.NoError(t, err)

	retBlock, err := bs.GetBlockByNumber(blockHeader.Number)
	require.NoError(t, err)
	require.Equal(t, block, retBlock, "Could not validate returned retBlock as expected")
}

func TestAddBlock(t *testing.T) {
	bs := newTestBlockState(t, testGenesisHeader)

	// Create header
	header0 := &types.Header{
		Number:     big.NewInt(1),
		Digest:     types.NewDigest(),
		ParentHash: testGenesisHeader.Hash(),
	}
	// Create blockHash
	blockHash0 := header0.Hash()
	block0 := &types.Block{
		Header: *header0,
		Body:   sampleBlockBody,
	}

	// Add the block0 to the DB
	err := bs.AddBlock(block0)
	require.NoError(t, err)

	// Create header & blockData for block 2
	header1 := &types.Header{
		Number:     big.NewInt(2),
		Digest:     types.NewDigest(),
		ParentHash: blockHash0,
	}
	blockHash1 := header1.Hash()

	block1 := &types.Block{
		Header: *header1,
		Body:   sampleBlockBody,
	}

	// Add the block1 to the DB
	err = bs.AddBlock(block1)
	require.NoError(t, err)

	// Get the blocks & check if it's the same as the added blocks
	retBlock, err := bs.GetBlockByHash(blockHash0)
	require.NoError(t, err)

	require.Equal(t, block0, retBlock, "Could not validate returned block0 as expected")

	retBlock, err = bs.GetBlockByHash(blockHash1)
	require.NoError(t, err)

	// this will panic if not successful, so catch and fail it so
	func() {
		hash := retBlock.Header.Hash()
		defer func() {
			if r := recover(); r != nil {
				t.Fatal("got panic when processing retBlock.Header.Hash() ", r)
			}
		}()
		require.False(t, hash.IsEmpty())
	}()

	require.Equal(t, block1, retBlock, "Could not validate returned block1 as expected")

	// Check if latestBlock is set correctly
	require.Equal(t, block1.Header.Hash(), bs.BestBlockHash(), "Latest Header Block Check Fail")
}

func TestGetSlotForBlock(t *testing.T) {
	bs := newTestBlockState(t, testGenesisHeader)
	expectedSlot := uint64(77)

	babeHeader := types.NewBabeDigest()
	err := babeHeader.Set(*types.NewBabePrimaryPreDigest(0, expectedSlot, [32]byte{}, [64]byte{}))
	require.NoError(t, err)
	data, err := scale.Marshal(babeHeader)
	require.NoError(t, err)
	preDigest := types.NewBABEPreRuntimeDigest(data)

	digest := types.NewDigest()
	err = digest.Add(*preDigest)
	require.NoError(t, err)
	block := &types.Block{
		Header: types.Header{
			ParentHash: testGenesisHeader.Hash(),
			Number:     big.NewInt(int64(1)),
			Digest:     digest,
		},
		Body: types.Body{},
	}

	err = bs.AddBlock(block)
	require.NoError(t, err)

	res, err := bs.GetSlotForBlock(block.Header.Hash())
	require.NoError(t, err)
	require.Equal(t, expectedSlot, res)
}

func TestIsBlockOnCurrentChain(t *testing.T) {
	bs := newTestBlockState(t, testGenesisHeader)
	currChain, branchChains := AddBlocksToState(t, bs, 3, false)

	for _, header := range currChain {
		onChain, err := bs.isBlockOnCurrentChain(header)
		require.NoError(t, err)

		if !onChain {
			t.Fatalf("Fail: expected block %s to be on current chain", header.Hash())
		}
	}

	for _, header := range branchChains {
		onChain, err := bs.isBlockOnCurrentChain(header)
		require.NoError(t, err)

		if onChain {
			t.Fatalf("Fail: expected block %s not to be on current chain", header.Hash())
		}
	}
}

func TestAddBlock_BlockNumberToHash(t *testing.T) {
	bs := newTestBlockState(t, testGenesisHeader)
	currChain, branchChains := AddBlocksToState(t, bs, 8, false)

	bestHash := bs.BestBlockHash()
	bestHeader, err := bs.BestBlockHeader()
	require.NoError(t, err)

	var resBlock *types.Block
	for _, header := range currChain {
		resBlock, err = bs.GetBlockByNumber(header.Number)
		require.NoError(t, err)

		if resBlock.Header.Hash() != header.Hash() {
			t.Fatalf("Fail: got %s expected %s for block %d", resBlock.Header.Hash(), header.Hash(), header.Number)
		}
	}

	for _, header := range branchChains {
		resBlock, err = bs.GetBlockByNumber(header.Number)
		require.NoError(t, err)

		if resBlock.Header.Hash() == header.Hash() {
			t.Fatalf("Fail: should not have gotten block %s for branch block num=%d", header.Hash(), header.Number)
		}
	}

	newBlock := &types.Block{
		Header: types.Header{
			ParentHash: bestHash,
			Number:     big.NewInt(0).Add(bestHeader.Number, big.NewInt(1)),
		},
		Body: types.Body{},
	}

	err = bs.AddBlock(newBlock)
	require.NoError(t, err)

	resBlock, err = bs.GetBlockByNumber(newBlock.Header.Number)
	require.NoError(t, err)

	if resBlock.Header.Hash() != newBlock.Header.Hash() {
		t.Fatalf("Fail: got %s expected %s for block %d", resBlock.Header.Hash(), newBlock.Header.Hash(), newBlock.Header.Number)
	}
}

func TestFinalization_DeleteBlock(t *testing.T) {
	bs := newTestBlockState(t, testGenesisHeader)
	AddBlocksToState(t, bs, 5, false)

	btBefore := bs.bt.DeepCopy()
	before := bs.bt.GetAllBlocks()
	leaves := bs.Leaves()

	// pick block to finalise
	fin := leaves[len(leaves)-1]
	err := bs.SetFinalisedHash(fin, 1, 1)
	require.NoError(t, err)

	after := bs.bt.GetAllBlocks()

	isIn := func(arr []common.Hash, b common.Hash) bool {
		for _, a := range arr {
			if b == a {
				return true
			}
		}
		return false
	}

	// assert that every block except finalised has been deleted
	for _, b := range before {
		if b == fin {
			continue
		}

		if isIn(after, b) {
			continue
		}

		isFinalised, err := btBefore.IsDescendantOf(b, fin)
		require.NoError(t, err)

		has, err := bs.HasHeader(b)
		require.NoError(t, err)
		if isFinalised {
			require.True(t, has)
		} else {
			require.False(t, has)
		}

		has, err = bs.HasBlockBody(b)
		require.NoError(t, err)
		if isFinalised {
			require.True(t, has)
		} else {
			require.False(t, has)
		}
	}
}

func TestGetHashByNumber(t *testing.T) {
	bs := newTestBlockState(t, testGenesisHeader)

	res, err := bs.GetHashByNumber(big.NewInt(0))
	require.NoError(t, err)
	require.Equal(t, bs.genesisHash, res)

	header := &types.Header{
		Number:     big.NewInt(1),
		Digest:     types.NewDigest(),
		ParentHash: testGenesisHeader.Hash(),
	}

	block := &types.Block{
		Header: *header,
		Body:   sampleBlockBody,
	}

	err = bs.AddBlock(block)
	require.NoError(t, err)

	res, err = bs.GetHashByNumber(big.NewInt(1))
	require.NoError(t, err)
	require.Equal(t, header.Hash(), res)
}

func TestAddBlock_WithReOrg(t *testing.T) {
	t.Skip() // TODO: this should be fixed after state refactor PR
	bs := newTestBlockState(t, testGenesisHeader)

	header1a := &types.Header{
		Number:     big.NewInt(1),
		Digest:     types.NewDigest(),
		ParentHash: testGenesisHeader.Hash(),
	}

	blockbody1a := types.NewBody([]types.Extrinsic{[]byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}})
	block1a := &types.Block{
		Header: *header1a,
		Body:   *blockbody1a,
	}

	err := bs.AddBlock(block1a)
	require.NoError(t, err)

	block1hash, err := bs.GetHashByNumber(big.NewInt(1))
	require.NoError(t, err)
	require.Equal(t, header1a.Hash(), block1hash)

	header1b := &types.Header{
		Number:         big.NewInt(1),
		Digest:         types.NewDigest(),
		ParentHash:     testGenesisHeader.Hash(),
		ExtrinsicsRoot: common.Hash{99},
	}

	block1b := &types.Block{
		Header: *header1b,
		Body:   sampleBlockBody,
	}

	err = bs.AddBlock(block1b)
	require.NoError(t, err)

	// should still be hash 1a since it arrived first
	block1hash, err = bs.GetHashByNumber(big.NewInt(1))
	require.NoError(t, err)
	require.Equal(t, header1a.Hash(), block1hash)

	header2b := &types.Header{
		Number:         big.NewInt(2),
		Digest:         types.NewDigest(),
		ParentHash:     header1b.Hash(),
		ExtrinsicsRoot: common.Hash{99},
	}

	block2b := &types.Block{
		Header: *header2b,
		Body:   sampleBlockBody,
	}

	err = bs.AddBlock(block2b)
	require.NoError(t, err)

	// should now be hash 1b since it's on the longer chain
	block1hash, err = bs.GetHashByNumber(big.NewInt(1))
	require.NoError(t, err)
	require.Equal(t, header1b.Hash(), block1hash)

	block2hash, err := bs.GetHashByNumber(big.NewInt(2))
	require.NoError(t, err)
	require.Equal(t, header2b.Hash(), block2hash)

	header2a := &types.Header{
		Number:     big.NewInt(2),
		Digest:     types.NewDigest(),
		ParentHash: header1a.Hash(),
	}

	block2a := &types.Block{
		Header: *header2a,
		Body:   sampleBlockBody,
	}

	err = bs.AddBlock(block2a)
	require.NoError(t, err)

	header3a := &types.Header{
		Number:     big.NewInt(3),
		Digest:     types.NewDigest(),
		ParentHash: header2a.Hash(),
	}

	block3a := &types.Block{
		Header: *header3a,
		Body:   sampleBlockBody,
	}

	err = bs.AddBlock(block3a)
	require.NoError(t, err)

	// should now be hash 1a since it's on the longer chain
	block1hash, err = bs.GetHashByNumber(big.NewInt(1))
	require.NoError(t, err)
	require.Equal(t, header1a.Hash(), block1hash)

	// should now be hash 2a since it's on the longer chain
	block2hash, err = bs.GetHashByNumber(big.NewInt(2))
	require.NoError(t, err)
	require.Equal(t, header2a.Hash(), block2hash)

	block3hash, err := bs.GetHashByNumber(big.NewInt(3))
	require.NoError(t, err)
	require.Equal(t, header3a.Hash(), block3hash)
}

func TestAddBlockToBlockTree(t *testing.T) {
	bs := newTestBlockState(t, testGenesisHeader)

	header := &types.Header{
		Number:     big.NewInt(1),
		Digest:     types.NewDigest(),
		ParentHash: testGenesisHeader.Hash(),
	}

	err := bs.setArrivalTime(header.Hash(), time.Now())
	require.NoError(t, err)

	err = bs.AddBlockToBlockTree(&types.Block{
		Header: *header,
		Body:   types.Body{},
	})
	require.NoError(t, err)
	require.Equal(t, bs.BestBlockHash(), header.Hash())
}

func TestNumberIsFinalised(t *testing.T) {
	bs := newTestBlockState(t, testGenesisHeader)
	fin, err := bs.NumberIsFinalised(big.NewInt(0))
	require.NoError(t, err)
	require.True(t, fin)

	fin, err = bs.NumberIsFinalised(big.NewInt(1))
	require.NoError(t, err)
	require.False(t, fin)

	digest := types.NewDigest()
	prd, err := types.NewBabeSecondaryPlainPreDigest(0, 1).ToPreRuntimeDigest()
	require.NoError(t, err)
	err = digest.Add(*prd)
	require.NoError(t, err)

	digest2 := types.NewDigest()
	prd, err = types.NewBabeSecondaryPlainPreDigest(0, 100).ToPreRuntimeDigest()
	require.NoError(t, err)
	err = digest2.Add(*prd)
	require.NoError(t, err)

	header1 := types.Header{
		Number:     big.NewInt(1),
		Digest:     digest,
		ParentHash: testGenesisHeader.Hash(),
	}

	header2 := types.Header{
		Number:     big.NewInt(2),
		Digest:     digest2,
		ParentHash: header1.Hash(),
	}

	err = bs.AddBlock(&types.Block{
		Header: header1,
		Body:   types.Body{},
	})
	require.NoError(t, err)

	err = bs.AddBlock(&types.Block{
		Header: header2,
		Body:   types.Body{},
	})
	require.NoError(t, err)
	err = bs.SetFinalisedHash(header2.Hash(), 1, 1)
	require.NoError(t, err)

	fin, err = bs.NumberIsFinalised(big.NewInt(0))
	require.NoError(t, err)
	require.True(t, fin)

	fin, err = bs.NumberIsFinalised(big.NewInt(1))
	require.NoError(t, err)
	require.True(t, fin)

	fin, err = bs.NumberIsFinalised(big.NewInt(2))
	require.NoError(t, err)
	require.True(t, fin)

	fin, err = bs.NumberIsFinalised(big.NewInt(100))
	require.NoError(t, err)
	require.False(t, fin)
}
