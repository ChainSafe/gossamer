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
	"time"

	"github.com/ChainSafe/gossamer/polkadb"

	"github.com/ChainSafe/gossamer/common"
	"github.com/ChainSafe/gossamer/core/types"
	log "github.com/ChainSafe/log15"
	"github.com/disiqueira/gotree"
)

type Hash = common.Hash

// BlockTree represents the current state with all possible blocks
type BlockTree struct {
	head            *Node
	leaves          leafMap
	finalizedBlocks []*Node
	Db              *polkadb.BlockDB
}

// NewBlockTreeFromGenesis initializes a blocktree with a genesis block.
func NewBlockTreeFromGenesis(genesis types.Block, db *polkadb.BlockDB) *BlockTree {
	head := &Node{
		Hash:        genesis.Header.Hash,
		Number:      genesis.Header.Number,
		parent:      nil,
		children:    []*Node{},
		depth:       big.NewInt(0),
		ArrivalTime: uint64(time.Now().UnixNano()), // set arrival time of genesis in nanoseconds since unix epoch
	}
	return &BlockTree{
		head:            head,
		finalizedBlocks: []*Node{},
		leaves:          leafMap{head.Hash: head},
		Db:              db,
	}
}

// AddBlock inserts the block as child of its parent Node
// Note: Assumes block has no children
func (bt *BlockTree) AddBlock(block types.Block) {
	parent := bt.GetNode(block.Header.ParentHash)
	// Check if it already exists
	// TODO: Can shortcut this by checking DB
	// TODO: Write blockData to db
	// TODO: Create getter functions to check if blockNum is greater than best block stored

	n := bt.GetNode(block.Header.Hash)
	if n != nil {
		log.Debug("Attempted to add block to tree that already exists", "Hash", n.Hash)
		return
	}

	depth := big.NewInt(0)
	depth.Add(parent.depth, big.NewInt(1))

	n = &Node{
		Hash:        block.Header.Hash,
		Number:      block.Header.Number,
		parent:      parent,
		children:    []*Node{},
		depth:       depth,
		ArrivalTime: uint64(time.Now().UnixNano()),
	}
	parent.addChild(n)

	bt.leaves.Replace(parent, n)
}

// GetNode finds and returns a Node based on its Hash. Returns nil if not found.
func (bt *BlockTree) GetNode(h Hash) *Node {
	if bt.head.Hash == h {
		return bt.head
	}

	for _, child := range bt.head.children {
		if n := child.getNode(h); n != nil {
			return n
		}
	}

	return nil
}

// GetNodeFromBlockNumber finds and returns a node from its number
func (bt *BlockTree) GetNodeFromBlockNumber(b *big.Int) *Node {
	if bt.head.Number.Cmp(b) != 0 {
		return bt.head
	}

	for _, child := range bt.head.children {
		if n := child.getNodeFromBlockNumber(b); n != nil {
			return n
		}
	}

	return nil

}

// String utilizes github.com/disiqueira/gotree to create a printable tree
func (bt *BlockTree) String() string {
	// Construct tree
	tree := gotree.New(bt.head.String())
	for _, child := range bt.head.children {
		sub := tree.Add(child.String())
		child.createTree(sub)
	}

	// Format leaves
	var leaves string
	for k := range bt.leaves {
		leaves = leaves + fmt.Sprintf("0x%X ", k)
	}

	metadata := fmt.Sprintf("Leaves: %v", leaves)

	return fmt.Sprintf("%s\n%s\n", metadata, tree.Print())
}

// LongestPath returns the path from the root to leftmost deepest leaf in BlockTree BT
func (bt *BlockTree) LongestPath() []*Node {
	dl := bt.DeepestLeaf()
	var path []*Node
	for curr := dl; ; curr = curr.parent {
		path = append([]*Node{curr}, path...)
		if curr.parent == nil {
			return path
		}
	}
}

//
func (bt *BlockTree) SubChain(start Hash, end Hash) []*Node {
	sn := bt.GetNode(start)
	en := bt.GetNode(end)
	return sn.subChain(en)
}

// DeepestLeaf returns leftmost deepest leaf in BlockTree BT
func (bt *BlockTree) DeepestLeaf() *Node {
	return bt.leaves.DeepestLeaf()
}

// computes the slot for a block from genesis
// helper for now, there's a better way to do this
func (bt *BlockTree) ComputeSlotForNode(n *Node, sd uint64) uint64 {
	gt := bt.head.ArrivalTime
	nt := n.ArrivalTime

	sp := uint64(0)
	for gt < nt {
		gt += sd
		sp += 1
	}

	return sp
}
