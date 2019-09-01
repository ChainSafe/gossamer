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
	"fmt"
	"math/big"
	"strconv"
	"testing"

	"github.com/ChainSafe/gossamer/common"
	"github.com/ChainSafe/gossamer/core"
	log "github.com/ChainSafe/log15"
)

var zeroHash, _ = common.HexToHash("0x00")

func createGenesisBlock() core.Block {
	return core.Block{
		SlotNumber:   nil,
		PreviousHash: zeroHash,
		//VrfOutput:    nil,
		//Transactions: nil,
		//Signature:    nil,
		BlockNumber: big.NewInt(0),
		Hash:        common.Hash{0x00},
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
	bt := NewBlockTreeFromGenesis(createGenesisBlock())

	previousHash := bt.head.hash

	for i := 1; i <= depth; i++ {
		hash, err := common.HexToHash(intToHashable(i))

		if err != nil {
			t.Error(err)
		}

		block := core.Block{
			PreviousHash: previousHash,
			Hash:         hash,
			BlockNumber:  big.NewInt(int64(i)),
		}

		bt.AddBlock(block)
		previousHash = hash
	}

	fmt.Println("CREATED NEW TREE")
	fmt.Println(bt.String())
	return bt
}

func TestBlockTree_GetBlock(t *testing.T) {
	// Calls AddBlock
	bt := createFlatTree(t, 2)

	h, err := common.HexToHash(intToHashable(2))
	if err != nil {
		log.Error("failed to create hash", "err", err)
	}

	n := bt.GetNode(h)

	if n.number.Cmp(big.NewInt(2)) != 0 {
		t.Errorf("got: %s expected: %s", n.number, big.NewInt(2))
	}

}

func TestBlockTree_AddBlock(t *testing.T) {
	bt := createFlatTree(t, 1)

	block := core.Block{
		SlotNumber:   nil,
		PreviousHash: common.Hash{0x01},
		BlockNumber:  nil,
		Hash:         common.Hash{0x02},
	}

	bt.AddBlock(block)

	n := bt.GetNode(common.Hash{0x02})

	if bt.leaves[n.hash] == nil {
		t.Errorf("expected %x to be a leaf", n.hash)
	}

	oldHash := common.Hash{0x01}

	if bt.leaves[oldHash] != nil {
		t.Errorf("expected %x to no longer be a leaf", oldHash)
	}
}

func TestNode_isDecendantOf(t *testing.T) {
	// Create tree with depth 4 (with 4 nodes)
	bt := createFlatTree(t, 4)

	// Compute hash of leaf and fetch node
	hashFour, err := common.HexToHash(intToHashable(4))
	if err != nil {
		t.Error(err)
	}

	// Check leaf is decendant of root
	leaf := bt.GetNode(hashFour)
	if !leaf.isDecendantOf(bt.head) {
		t.Error("failed to verify leaf is descendant of root")
	}

	// Verify the inverse relationship does not hold
	if bt.head.isDecendantOf(leaf) {
		t.Error("root should not be decendant of anything")
	}

}

func TestBlockTree_LongestPath(t *testing.T) {
	bt := createFlatTree(t, 3)

	// Insert a block to create a competing path
	extraBlock := core.Block{
		SlotNumber:   nil,
		PreviousHash: zeroHash,
		BlockNumber:  big.NewInt(1),
		Hash:         common.Hash{0xAB},
	}

	bt.AddBlock(extraBlock)

	expectedPath := []*node{
		bt.GetNode(common.Hash{0x00}),
		bt.GetNode(common.Hash{0x01}),
		bt.GetNode(common.Hash{0x02}),
	}

	longestPath := bt.LongestPath()

	for i, n := range longestPath {
		if n.hash != expectedPath[i].hash {
			t.Errorf("expected hash: %s got: %s\n", expectedPath[i].hash, n.hash)
		}
	}
}

// TODO: Need to define leftmost (see BlockTree.LongestPath)
func TestBlockTree_LongestPath_LeftMost(t *testing.T) {
	bt := createFlatTree(t, 1)

	// Insert a block to create a competing path
	extraBlock := core.Block{
		SlotNumber:   nil,
		PreviousHash: zeroHash,
		BlockNumber:  big.NewInt(1),
		Hash:         common.Hash{0xAB},
	}

	bt.AddBlock(extraBlock)

	fmt.Println(bt.String())

	expectedPath := []*node{
		bt.GetNode(common.Hash{0x00}),
		bt.GetNode(common.Hash{0xAB}),
	}

	longestPath := bt.LongestPath()

	for i, n := range longestPath {
		if n.hash != expectedPath[i].hash {
			t.Errorf("expected hash: %s got: %s\n", expectedPath[i].hash, n.hash)
		}
	}
}
