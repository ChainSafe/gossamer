package blocktree

import (
	"math/big"

	"github.com/ChainSafe/gossamer/common"
	"github.com/disiqueira/gotree"
	"github.com/prometheus/common/log"
)

// node is an element in the BlockTree
type node struct {
	hash     common.Hash     // Block hash
	number   *big.Int // Block number
	children []*node  // Nodes of children blocks
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