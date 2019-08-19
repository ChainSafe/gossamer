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

	"github.com/ChainSafe/gossamer/common"
	"github.com/ChainSafe/gossamer/core"
	log "github.com/ChainSafe/log15"
	"github.com/disiqueira/gotree"
)

type Hash = common.Hash

// node is an element in the BlockTree
type node struct {
	hash     Hash     // Block hash
	number   *big.Int // Block number
	children []*node  // Nodes of children blocks
}

// BlockTree represents the current state with all possible blocks
type BlockTree struct {
	head            *node
	finalizedBlocks []*node
}

func NewBlockTreeFromGenesis(genesis core.Block) *BlockTree {
	head := &node{
		hash:     genesis.Hash,
		number:   genesis.BlockNumber,
		children: []*node{},
	}
	return &BlockTree{
		head:            head,
		finalizedBlocks: []*node{},
	}
}

// AddBlock inserts the block as child of its parent node
func (bt *BlockTree) AddBlock(block core.Block) {
	parent := bt.GetNode(block.PreviousHash)
	// Check if it already exists
	// TODO: Can shortcut this by checking DB
	n := bt.GetNode(block.Hash)
	if n != nil {
		log.Debug("Attempted to add block to tree that already exists", "hash", n.hash)
		return
	}
	n = &node{
		hash: block.Hash,
		number: block.BlockNumber,
		children: []*node{},
	}
	log.Debug("Adding child to parent", "parent", parent, "child", n)
	parent.addChild(n)
}

//helper used to find node by hash in tree DFS
func (bt *BlockTree) GetNode(h Hash) *node {
	if bt.head.hash == h {
		return bt.head
	}

	for _, child := range bt.head.children {
		if n := child.getNode(h); n != nil {
			return n
		}
	}

	return nil
}

// String utilizes gotree to create a printable tree
func (bt *BlockTree) String() string {
	tree := gotree.New(bt.head.String())

	for _, child := range bt.head.children {
		sub := tree.Add(child.String())
		child.createTree(sub)
	}

	return tree.Print()
}

// addChild appends node to list of children
func (n *node) addChild(node *node) {
	n.children = append(n.children, node)
}

func (n *node) String() string {
	return n.hash.String()
}

// createTree adds all the nodes children to the existing printable tree
func (n *node) createTree(tree gotree.Tree) {
	log.Debug("Getting tree", "node", n)
	for _, child := range n.children {
		tree.Add(child.String())
	}
}

func (n *node) getNode(h common.Hash) *node {
	if n.hash == h {
		return n
	} else if len(n.children) == 0 {
		return nil
	} else {
		for _, child := range n.children {
			if n := child.getNode(h); n != nil {
				return n
			}
		}
	}
	return nil
}

// TODO: Fix or remove
//finds node by hash returns stack containing path to that node
//need alternate with no return value to save space when not nessessary
func Chain(h Hash, BT BlockTree) []node {
	return nil //SubChain(BT.head.hash, h)
}

//LongestPath returns path to leftmost deepest leaf in BlockTree BT
func (bt *BlockTree) LongestPath ([]node, *big.Int) BlockTree {
	//dl := bt.DeepestLeaf()
	return BlockTree{} //bt.SubChain(bt.head, dl)

}

//returns leftmost deepest leaf in BlockTree BT
func (bt *BlockTree) DeepestLeaf() []node {
	return nil //bt.head.deepestLeaf()
}
//
//// DeepestLeaf returns leftmost deepest leaf in BlockTree BT
//func (n node) deepestLeaf() []node{
//	// TODO:
//	lvs := leaves(n)
//	lens := []int
//	for _, l := range lvs {
//		lens = append(lens, subChainLength(n, l))
//	}
//
//	max := lens[0]
//	maxIndex := 0
//	for _, i := range lens {
//		if i > max {
//			max = i
//			maxIndex = _
//		}
//	}
//
//	return l[maxIndex]
//}
//
//func subChainLength(start node, end node) int {
//	return len(subChain(start, end))
//}
//
func (bt *BlockTree) SubChain(start Hash, end Hash) []node {
	//verify that end is descendant of start
	//if (isDecendantOf(start, end)) {
	//	s := findNode(start, bt)
	//	e := findNode(end, bt)
	//	return subChain(s,e,[]node)
	//}
	return nil
}
//
//func subChain(start node, end node, chain []node ) []node {
//	for _, n := range start.children {
//		if (start == end) {
//			return chain
//		}
//		if isDecendantOf(n.Hash, end.Hash) {
//			chain = append(chain, n)
//			SubChain(n, end, chain)
//		}
//	}
//
//	return nil
//}
//
//

//
////returns node by hash given hash and blocktree
//func findNode(h Hash, BT BlockTree) node {
//	//TODO: verify that block with given hash exists in DB
//	return findNode(h, BT.head)
//}
//
////returns children of block given hash and blocktree
//func getChildren(h Hash, BT BlockTree) []node {
//	//TODO: verify that block with given hash exists in DB
//	node = findNode(h, BT)
//	return getChildren(node)
//}
//
////finds children of node
//func getChildren(n node) []node {
//	return n.children
//}
//
////returns hashes of blocks that are leaves on BT
////can probably memoize this and store if we find
////ourselves retrieving it a lot
//func leaves(BT BlockTree) []Hash {
//	//TODO: verify that block with given hash exists in DB
//	return leaves(BT.head, []node)
//}
//
//func leaves(n node, l []node) []node {
//	if len(n.children) == 0 {
//		leaves.append(n)
//	} else {
//		for _, c := range n.children {
//			l = append(leaves(c, c.children), l...)
//		}
//	}
//	return l
//}
//
////leaves of tree starting from node containing block with hash of h
//func leaves(h Hash) []Hash {
//	//TODO: verify that block with given hash exists in DB
//	l := findNode(h, []node)
//	return leaves(l)
//
//}
//
////stub to add block to DB by Hash
//func inputBlock(h Hash, b Block) {
//	return true
//}
//
//
////importing blocks to blocktree
//func addBlock(b Block, bt BlockTree) bool {
//	//TODO: verify that parent exists in the DB
//	//TODO: verify that block is not duplicate of block in DB
//	if (blockExists(b.previousHash) && !blockExists(b.Hash)) {
//		//if above two are true add block hash to tree
//		n := node{hash: b.Hash, number: b.BlockNumber}
//		parent = findNode(b.previousHash)
//		parent.children = append(parent.children, n)
//
//		//TODO: add block to db
//		if inputBlock(b.hash, b) {
//			return true
//		}
//	}
//
//	return false
//}
//
//
//func isDecendantOf(parent Hash, child Hash, bt BlockTree) bool {
//	//TODO: verify that parent and child exist in the DB
//	if (blockExists(parent) && blockExists(child)) {
//		//get node
//		p := findNode(parent, bt)
//		//check if node exists as descendant
//		if findNode(p, child) {
//			return true
//		}
//	}
//	// if node doesn't exist as a part of the tree with head parent,
//	// it is not a decendant of that block.
//	return false
//}
