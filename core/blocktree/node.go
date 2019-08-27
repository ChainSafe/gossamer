package blocktree

import (
	"fmt"
	"math/big"

	"github.com/ChainSafe/gossamer/common"
	"github.com/disiqueira/gotree"
)

// node is an element in the BlockTree
type node struct {
	hash     common.Hash     // Block hash
	parent *node
	number   *big.Int // Block number
	children []*node  // Nodes of children blocks
	depth *big.Int // Depth within the tree
}

// addChild appends node to n's list of children
func (n *node) addChild(node *node) {
	n.children = append(n.children, node)
}

// String returns stringified hash and depth of node
func (n *node) String() string {
	return fmt.Sprintf("{h: %s, d: %s}", n.hash.String(), n.depth)
}

// createTree adds all the nodes children to the existing printable tree.
// Note: this is strictly for BlockTree.String()
func (n *node) createTree(tree gotree.Tree) {
	for _, child := range n.children {
		sub := tree.Add(child.String())
		child.createTree(sub)
	}
}

// getNode recursively searches for a given hash
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

// isDescendantOf traverses the tree following all possible paths until it determines if n is a descendant of parent
func (n *node) isDecendantOf(parent *node) bool {
	// TODO: This might be improved by using parent in node struct and searching child -> parent
	// TODO: verify that parent and child exist in the DB
	// NOTE: here we assume the nodes exist
	if n.hash == parent.hash {
		return true
	} else if len(parent.children) == 0 {
		return false
	} else {
		for _, child := range parent.children {
			 if n.isDecendantOf(child) == true {
			 	return true
			 }
		}
	}
	return false
}