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
	"github.com/ChainSafe/gossamer/core/consensus"
)

type Block struct {
	SlotNumber		*big.Int
	PreviousHash	Hash
	VrfOutput		VRFOutput
	Transactions 	[]Transaction
	Signature		Signature
	BlockNumber		*big.Int
	Hash			Hash		

}


type Hash [32]byte

//contains block and block header and pointers to children
type Node struct{
	hash			Hash
	number			*big.Int	
	children 		[]Node
}

//contains nodes, finalized roots
type BlockTree struct {
	head 			Node
	finalizedBlocks []Node
}


//finds node by hash returns stack containing path to that node
//need alternate with no return value to save space when not nessessary 
func Chain(h Hash, BT BlockTree) []Node {
	return SubChain(BT.head.hash, h)

}

//returns leftmost path to deepest leaf in BlockTree BT
func LongestPath(BT BlockTree) ([]Node, *big.Int) {
	dl := DeepestLeaf(BT)
	return SubChain(BT.head, dl)

}

//returns leftmost deepest leaf in BlockTree BT
func DeepestLeaf(BT BlockTree) []Node {
	return DeepestLeaf(BT.head)
}

//returns leftmost deepest leaf in BlockTree BT
func DeepestLeaf(head Node) {
	l = leaves(head)
	lens = []Int
	for _, n := range l {
		append(lens, subChainLength(head, n))
	}

	max := lens[0]
	maxIndex := 0
	for _, i := range lens {
		if i > max {
			max = i
			maxIndex = _
		}
	}

	return l[maxIndex]
}

func subChainLength(start Node, end Node) Int {
	return len(subChain(start, end))
}

func SubChain(start Hash, end Hash, BT BlockTree) []Node {
	//verify that end is descendant of start
	if (isDecendantOf(start, end)) {
		s := findNode(start, BT)
		e := findNode(end, BT)
		return subChain(s,e,[]Node)
	}
	return nil
}

func subChain(start Node, end Node, chain []Node ) []Node {
	for _, n := range start.children {
		if (start == end) {
			return chain			
		}
		if isDecendantOf(n.Hash, end.Hash) {
			chain = append(chain, n)
			SubChain(n, end, chain)
		}
	}

	return nil
}


//helper used to find node by hash in tree DFS
func findNode(h Hash, root Node) Node {
	if ( root.hash == h ) {
		return root
	}
	
	for _, n := range root.children {
		if findNode(n.hash, n) != nil {
			return n
		}

	}
	return nil

}

//returns node by hash given hash and blocktree
func findNode(h Hash, BT BlockTree) Node {
	//TODO: verify that block with given hash exists in DB
	return findNode(h, BT.head)
}

//returns children of block given hash and blocktree
func getChildren(h Hash, BT BlockTree) []Node {
	//TODO: verify that block with given hash exists in DB
	node = findNode(h, BT)
	return getChildren(node)
}

//finds children of node
func getChildren(n Node) []Node {
	return n.children
}

//returns hashes of blocks that are leaves on BT
//can probably memoize this and store if we find
//ourselves retrieving it a lot
func leaves(BT BlockTree) []Hash {
	//TODO: verify that block with given hash exists in DB
	return leaves(BT.head, []Node)
}

func leaves(n Node, l []Node) []Node {
	if len(n.children) == 0 {
		leaves.append(n)
	} else {
		for _, c := range n.children {
			l = append(leaves(c, c.children), l...)
		}
	}
	return l
}

//leaves of tree starting from node containing block with hash of h
func leaves(h Hash) []Hash {
	//TODO: verify that block with given hash exists in DB
	l := findNode(h, []Node)
	return leaves(l)

}


//stub to retrieve block by hash from db
func retrieveBlock(h Hash) {
	return true
}

//stub to verify that block exists in db
func blockExists(h Hash) {
	return true
}

//stub to add block to DB by Hash
func inputBlock(h Hash, b Block) {
	return true
}


//importing blocks to blocktree
func addBlock(b Block, bt BlockTree) bool {
	//TODO: verify that parent exists in the DB
	//TODO: verify that block is not duplicate of block in DB
	if (blockExists(b.previousHash) && !blockExists(b.Hash)) {
		//if above two are true add block hash to tree
		n := Node{hash: b.Hash, number: b.BlockNumber}
		parent = findNode(b.previousHash)
		parent.children = append(parent.children, n)

		//TODO: add block to db
		if inputBlock(b.hash, b) {
			return true
		}
	}

	return false
}


func isDecendantOf(parent Hash, child Hash, bt BlockTree) bool {
	//TODO: verify that parent and child exist in the DB
	if (blockExists(parent) && blockExists(child)) {
		//get node
		p := findNode(parent, bt)
		//check if node exists as descendant
		if findNode(p, child) {
			return true
		}
	} 
	// if node doesn't exist as a part of the tree with head parent, 
	// it is not a decendant of that block. 
	return false
} 






