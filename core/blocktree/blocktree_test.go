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

package blocktree

import (
	"math/big"
	"strconv"
	"testing"

	"github.com/ChainSafe/gossamer/common"
	"github.com/ChainSafe/gossamer/core/types"
	db "github.com/ChainSafe/gossamer/polkadb"
	log "github.com/ChainSafe/log15"
)

var zeroHash, _ = common.HexToHash("0x00")

func createGenesisBlock() types.Block {
	return types.Block{
		Header: types.BlockHeader{
			ParentHash: zeroHash,
			Number:     big.NewInt(0),
			Hash:       common.Hash{0x00},
			
		},
		Body: types.BlockBody{},
	}
}

func intToHashable(in int) string {
	if in < 0 {
		return ""
	}

	out := strconv.Itoa(in)
	if len(out)%2 != 0 {
		out = "0" + out
	}
	return "0x" + out
}

func createFlatTree(t *testing.T, depth int) *BlockTree {
	d := &db.BlockDB{
		Db: db.NewMemDatabase(),
	}

	bt := NewBlockTreeFromGenesis(createGenesisBlock(), d)

	previousHash := bt.head.Hash

	for i := 1; i <= depth; i++ {
		hash, err := common.HexToHash(intToHashable(i))

		if err != nil {
			t.Error(err)
		}

		block := types.Block{
			Header: types.BlockHeader{
				ParentHash: previousHash,
				Hash:       hash,
				Number:     big.NewInt(int64(i)),
			},
			Body: types.BlockBody{},
		}

		bt.AddBlock(block)
		previousHash = hash
	}

	return bt
}

func TestBlockTree_GetBlock(t *testing.T) {
	// Calls AddBlock
	bt := createFlatTree(t, 2)

	h, err := common.HexToHash(intToHashable(2))
	if err != nil {
		log.Error("failed to create Hash", "err", err)
	}

	n := bt.GetNode(h)

	if n.Number.Cmp(big.NewInt(2)) != 0 {
		t.Errorf("got: %s expected: %s", n.Number, big.NewInt(2))
	}

}

func TestBlockTree_AddBlock(t *testing.T) {
	bt := createFlatTree(t, 1)

	block := types.Block{
		Header: types.BlockHeader{
			ParentHash: common.Hash{0x01},
			Number:     nil,
			Hash:       common.Hash{0x02},
		},
		Body: types.BlockBody{},
	}

	bt.AddBlock(block)

	n := bt.GetNode(common.Hash{0x02})

	if bt.leaves[n.Hash] == nil {
		t.Errorf("expected %x to be a leaf", n.Hash)
	}

	oldHash := common.Hash{0x01}

	if bt.leaves[oldHash] != nil {
		t.Errorf("expected %x to no longer be a leaf", oldHash)
	}
}

func TestNode_isDecendantOf(t *testing.T) {
	// Create tree with depth 4 (with 4 nodes)
	bt := createFlatTree(t, 4)

	// Compute Hash of leaf and fetch node
	hashFour, err := common.HexToHash(intToHashable(4))
	if err != nil {
		t.Error(err)
	}

	// Check leaf is decendant of root
	leaf := bt.GetNode(hashFour)
	if !leaf.isDescendantOf(bt.head) {
		t.Error("failed to verify leaf is descendant of root")
	}

	// Verify the inverse relationship does not hold
	if bt.head.isDescendantOf(leaf) {
		t.Error("root should not be decendant of anything")
	}

}

func TestBlockTree_LongestPath(t *testing.T) {
	bt := createFlatTree(t, 3)

	// Insert a block to create a competing path
	extraBlock := types.Block{
		Header: types.BlockHeader{
			ParentHash: zeroHash,
			Number:     big.NewInt(1),
			Hash:       common.Hash{0xAB},
		},
		Body: types.BlockBody{},
	}

	bt.AddBlock(extraBlock)

	expectedPath := []*Node{
		bt.GetNode(common.Hash{0x00}),
		bt.GetNode(common.Hash{0x01}),
		bt.GetNode(common.Hash{0x02}),
		bt.GetNode(common.Hash{0x03}),
	}

	longestPath := bt.LongestPath()

	for i, n := range longestPath {
		if n.Hash != expectedPath[i].Hash {
			t.Errorf("expected Hash: 0x%X got: 0x%X\n", expectedPath[i].Hash, n.Hash)
		}
	}
}

func TestBlockTree_Subchain(t *testing.T) {
	bt := createFlatTree(t, 4)

	// Insert a block to create a competing path
	extraBlock := types.Block{
		Header: types.BlockHeader{
			ParentHash: zeroHash,
			Number:     big.NewInt(1),
			Hash:       common.Hash{0xAB},
		},
		Body: types.BlockBody{},
	}

	bt.AddBlock(extraBlock)

	expectedPath := []*Node{
		bt.GetNode(common.Hash{0x01}),
		bt.GetNode(common.Hash{0x02}),
		bt.GetNode(common.Hash{0x03}),
	}

	subChain := bt.SubChain(common.Hash{0x01}, common.Hash{0x03})

	for i, n := range subChain {
		if n.Hash != expectedPath[i].Hash {
			t.Errorf("expected Hash: 0x%X got: 0x%X\n", expectedPath[i].Hash, n.Hash)
		}
	}
}

func TestBlockTree_ComputeSlotForNode(t *testing.T) {
	bt := createFlatTree(t, 2)
	
	expectedSlotNumber := uint64(1)
	slotNumber := bt.ComputeSlotForNode(bt.GetNode(common.Hash{0x01}), 100000000)
	
	if (slotNumber != expectedSlotNumber) {
		t.Errorf("expected Slot Number: %d got: %d", expectedSlotNumber, slotNumber)	
	}
	
}


// TODO: Need to define leftmost (see BlockTree.LongestPath)
//func TestBlockTree_LongestPath_LeftMost(t *testing.T) {
//	bt := createFlatTree(t, 1)
//
//	// Insert a block to create a competing path
//	extraBlock := types.Block{
//		SlotNumber:   nil,
//		ParentHash: zeroHash,
//		Number:  big.NewInt(1),
//		Hash:         common.Hash{0xAB},
//	}
//
//	bt.AddBlock(extraBlock)
//
//	expectedPath := []*Node{
//		bt.GetNode(common.Hash{0x00}),
//		bt.GetNode(common.Hash{0xAB}),
//	}
//
//	longestPath := bt.LongestPath()
//
//	for i, n := range longestPath {
//		if n.Hash != expectedPath[i].Hash {
//			t.Errorf("expected Hash: 0x%X got: 0x%X\n", expectedPath[i].Hash, n.Hash)
//		}
//	}
//}
