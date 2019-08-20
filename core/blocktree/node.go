package blocktree

import (
	"fmt"
	"math/big"

	"github.com/ChainSafe/gossamer/common"
	"github.com/disiqueira/gotree"
	log "github.com/ChainSafe/log15"
)

// node is an element in the BlockTree
type node struct {
	hash     common.Hash     // Block hash
	// TODO: Do we need this?
	// parentHash common.Hash
	number   *big.Int // Block number
	children []*node  // Nodes of children blocks
	depth *big.Int // Depth within the tree
}

// addChild appends node to list of children
func (n *node) addChild(node *node) {
	n.children = append(n.children, node)
}

func (n *node) String() string {
	return fmt.Sprintf("{h: %s, d: %s}", n.hash.String(), n.depth)
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

func (n *node) isDecendantOf(parent *node) bool {
	// TODO: This might be improved by adding parent hash to node struct and searching child -> parent
	// TODO: verify that parent and child exist in the DB
	// NOTE: here we assume the nodes exist
	if n.hash == parent.hash {
		return true
	} else if len(parent.children) == 0 {
		return false
	} else {
		for _, child := range parent.children {
			n.isDecendantOf(child)
		}
	}
	return false
}