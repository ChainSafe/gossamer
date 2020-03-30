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

package state

import (
	"math/big"
	"reflect"
	"testing"

	"github.com/ChainSafe/gossamer/dot/core/types"
	"github.com/ChainSafe/gossamer/lib/blocktree"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/database"
	"github.com/ChainSafe/gossamer/lib/trie"

	"github.com/stretchr/testify/require"
)

func newTestBlockState(header *types.Header) *BlockState {
	db := database.NewMemDatabase()
	blockDb := NewBlockDB(db)

	if header == nil {
		return &BlockState{
			db: blockDb,
		}
	}

	return &BlockState{
		db: blockDb,
		bt: blocktree.NewBlockTreeFromGenesis(header, db),
	}
}

func TestSetAndGetHeader(t *testing.T) {
	bs := newTestBlockState(nil)

	header := &types.Header{
		Number:    big.NewInt(0),
		StateRoot: trie.EmptyHash,
		Digest:    [][]byte{},
	}

	err := bs.SetHeader(header)
	if err != nil {
		t.Fatal(err)
	}

	res, err := bs.GetHeader(header.Hash())
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(res, header) {
		t.Fatalf("Fail: got %v expected %v", res, header)
	}
}

func TestGetBlockByNumber(t *testing.T) {
	genesisHeader := &types.Header{
		Number: big.NewInt(0),
	}

	bs := newTestBlockState(genesisHeader)

	blockHeader := &types.Header{
		ParentHash: genesisHeader.Hash(),
		Number:     big.NewInt(1),
		Digest:     [][]byte{},
	}

	block := &types.Block{
		Header: blockHeader,
		Body:   &types.Body{0, 1, 2, 3, 4, 5, 6, 7, 8, 9},
	}

	// AddBlock also sets mapping [blockNumber : hash] in DB
	err := bs.AddBlock(block)
	require.Nil(t, err)

	retBlock, err := bs.GetBlockByNumber(blockHeader.Number)
	require.Nil(t, err)

	require.Equal(t, block, retBlock, "Could not validate returned retBlock as expected")

}

func TestAddBlock(t *testing.T) {
	genesisHeader := &types.Header{
		Number: big.NewInt(0),
	}

	bs := newTestBlockState(genesisHeader)

	// Create header
	header0 := &types.Header{
		Number:     big.NewInt(0),
		Digest:     [][]byte{},
		ParentHash: genesisHeader.Hash(),
	}
	// Create blockHash
	blockHash0 := header0.Hash()
	// BlockBody with fake extrinsics
	blockBody0 := types.Body{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}

	block0 := &types.Block{
		Header: header0,
		Body:   &blockBody0,
	}

	// Add the block0 to the DB
	err := bs.AddBlock(block0)
	require.Nil(t, err)

	// Create header & blockData for block 1
	header1 := &types.Header{
		Number:     big.NewInt(1),
		Digest:     [][]byte{},
		ParentHash: blockHash0,
	}
	blockHash1 := header1.Hash()

	// Create Block with fake extrinsics
	blockBody1 := types.Body{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}

	block1 := &types.Block{
		Header: header1,
		Body:   &blockBody1,
	}

	// Add the block1 to the DB
	err = bs.AddBlock(block1)
	require.Nil(t, err)

	// Get the blocks & check if it's the same as the added blocks
	retBlock, err := bs.GetBlockByHash(blockHash0)
	require.Nil(t, err)

	require.Equal(t, block0, retBlock, "Could not validate returned block0 as expected")

	retBlock, err = bs.GetBlockByHash(blockHash1)
	require.Nil(t, err)

	// this will panic if not successful, so catch and fail it so
	func() {
		hash := retBlock.Header.Hash()
		defer func() {
			if r := recover(); r != nil {
				t.Fatal("got panic when processing retBlock.Header.Hash() ", r)
			}
		}()
		require.NotEqual(t, hash, common.Hash{})
	}()

	require.Equal(t, block1, retBlock, "Could not validate returned block1 as expected")

	// Check if latestBlock is set correctly
	require.Equal(t, block1.Header.Hash(), bs.BestBlockHash(), "Latest Header Block Check Fail")
}

func TestGetSlotForBlock(t *testing.T) {
	genesisHeader := &types.Header{
		Number:    big.NewInt(0),
		StateRoot: trie.EmptyHash,
	}

	bs := newTestBlockState(genesisHeader)

	preDigest, err := common.HexToBytes("0x014241424538e93dcef2efc275b72b4fa748332dc4c9f13be1125909cf90c8e9109c45da16b04bc5fdf9fe06a4f35e4ae4ed7e251ff9ee3d0d840c8237c9fb9057442dbf00f210d697a7b4959f792a81b948ff88937e30bf9709a8ab1314f71284da89a40000000000000000001100000000000000")
	if err != nil {
		t.Fatal(err)
	}

	expectedSlot := uint64(17)

	block := &types.Block{
		Header: &types.Header{
			ParentHash: genesisHeader.Hash(),
			Number:     big.NewInt(int64(1)),
			Digest:     [][]byte{preDigest},
		},
		Body: &types.Body{},
	}

	err = bs.AddBlock(block)
	if err != nil {
		t.Fatal(err)
	}

	res, err := bs.GetSlotForBlock(block.Header.Hash())
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(res, expectedSlot) {
		t.Fatalf("Fail: got %x expected %x", res, expectedSlot)
	}
}

func TestIsBlockOnCurrentChain(t *testing.T) {
	genesisHeader := &types.Header{
		Number:    big.NewInt(0),
		StateRoot: trie.EmptyHash,
	}

	bs := newTestBlockState(genesisHeader)
	currChain, branchChains := addBlocksToState(bs, 8)

	for _, header := range currChain {
		onChain, err := bs.isBlockOnCurrentChain(header)
		if err != nil {
			t.Fatal(err)
		}
		if !onChain {
			t.Fatalf("Fail: expected block %s to be on current chain", header.Hash())
		}
	}

	for _, header := range branchChains {
		onChain, err := bs.isBlockOnCurrentChain(header)
		if err != nil {
			t.Fatal(err)
		}
		if onChain {
			t.Fatalf("Fail: expected block %s not to be on current chain", header.Hash())
		}
	}
}

func TestAddBlock_BlockNumberToHash(t *testing.T) {
	genesisHeader := &types.Header{
		Number:    big.NewInt(0),
		StateRoot: trie.EmptyHash,
	}

	bs := newTestBlockState(genesisHeader)
	currChain, branchChains := addBlocksToState(bs, 8)

	bestHash := bs.BestBlockHash()
	bestHeader, err := bs.BestBlockHeader()
	if err != nil {
		t.Fatal(err)
	}

	var resBlock *types.Block
	for _, header := range currChain {
		resBlock, err = bs.GetBlockByNumber(header.Number)
		if err != nil {
			t.Fatal(err)
		}

		if resBlock.Header.Hash() != header.Hash() {
			t.Fatalf("Fail: got %s expected %s for block %d", resBlock.Header.Hash(), header.Hash(), header.Number)
		}
	}

	for _, header := range branchChains {
		resBlock, err = bs.GetBlockByNumber(header.Number)
		if err != nil {
			t.Fatal(err)
		}

		if resBlock.Header.Hash() == header.Hash() {
			t.Fatalf("Fail: should not have gotten block %s for branch block num=%d", header.Hash(), header.Number)
		}
	}

	newBlock := &types.Block{
		Header: &types.Header{
			ParentHash: bestHash,
			Number:     big.NewInt(0).Add(bestHeader.Number, big.NewInt(1)),
		},
		Body: &types.Body{},
	}

	err = bs.AddBlock(newBlock)
	if err != nil {
		t.Fatal(err)
	}

	resBlock, err = bs.GetBlockByNumber(newBlock.Header.Number)
	if err != nil {
		t.Fatal(err)
	}

	if resBlock.Header.Hash() != newBlock.Header.Hash() {
		t.Fatalf("Fail: got %s expected %s for block %d", resBlock.Header.Hash(), newBlock.Header.Hash(), newBlock.Header.Number)
	}
}
