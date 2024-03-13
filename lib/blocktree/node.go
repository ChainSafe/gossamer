// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package blocktree

import (
	"fmt"
	"time"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/disiqueira/gotree"
)

// node is an element in the BlockTree
type node struct {
	hash        common.Hash // Block hash
	parent      *node       // Parent Node
	children    []*node     // Nodes of children blocks
	number      uint        // block number
	arrivalTime time.Time   // Arrival time of the block
	isPrimary   bool        // whether the block was authored in a primary slot or not
}

// addChild appends Node to n's list of children
func (n *node) addChild(node *node) {
	n.children = append(n.children, node)
}

// string returns stringified hash and number of node
func (n *node) string() string {
	return fmt.Sprintf("{hash: %s, number: %d, arrivalTime: %s}", n.hash.String(), n.number, n.arrivalTime)
}

// createTree adds all the nodes children to the existing printable tree.
// Note: this is strictly for BlockTree.String()
func (n *node) createTree(tree gotree.Tree) {
	for _, child := range n.children {
		sub := tree.Add(child.string())
		child.createTree(sub)
	}
}

// getNode recursively searches for a node with a given hash
func (n *node) getNode(h common.Hash) *node {
	if n == nil {
		return nil
	}

	if n.hash == h {
		return n
	}

	for _, child := range n.children {
		if n := child.getNode(h); n != nil {
			return n
		}
	}

	return nil
}

// getNodesWithNumber returns all descendent nodes with the desired number
func (n *node) getNodesWithNumber(number uint, hashes []common.Hash) []common.Hash {
	for _, child := range n.children {
		// number matches
		if child.number == number {
			hashes = append(hashes, child.hash)
		}

		// are deeper than desired number, return
		if child.number > number {
			return hashes
		}

		hashes = child.getNodesWithNumber(number, hashes)
	}

	return hashes
}

// isDescendantOf traverses the tree following all possible paths until it determines if n is a descendant of parent
func (n *node) isDescendantOf(parent *node) bool {
	if parent == nil || n == nil {
		return false
	}

	// NOTE: here we assume the nodes exists in tree
	if n.hash == parent.hash {
		return true
	} else if len(parent.children) == 0 {
		return false
	} else {
		for _, child := range parent.children {
			if n.isDescendantOf(child) {
				return true
			}
		}
	}
	return false
}

// getLeaves returns all nodes that are leaf nodes with the current node as its ancestor
func (n *node) getLeaves(leaves []*node) []*node {
	if n == nil {
		return leaves
	}

	if leaves == nil {
		leaves = []*node{}
	}

	if n.children == nil || len(n.children) == 0 {
		leaves = append(leaves, n)
	}

	for _, child := range n.children {
		leaves = child.getLeaves(leaves)
	}

	return leaves
}

// getAllDescendants returns an array of the node's hash and all its descendants's hashes
func (n *node) getAllDescendants(desc []Hash) []Hash {
	if n == nil {
		return desc
	}

	if desc == nil {
		desc = []Hash{}
	}

	desc = append(desc, n.hash)
	for _, child := range n.children {
		desc = child.getAllDescendants(desc)
	}

	return desc
}

// deepCopy returns a copy of the given node
func (n *node) deepCopy(parent *node) *node {
	nCopy := new(node)
	nCopy.hash = n.hash
	nCopy.arrivalTime = n.arrivalTime
	nCopy.number = n.number

	nCopy.children = make([]*node, len(n.children))
	for i, child := range n.children {
		nCopy.children[i] = child.deepCopy(n)
	}

	if n.parent != nil {
		nCopy.parent = parent
	}

	return nCopy
}

func (n *node) prune(finalised *node, pruned []Hash) []Hash {
	if finalised == nil {
		return pruned
	}

	if pruned == nil {
		pruned = []Hash{}
	}

	// if this is a descedent of the finalised block, keep it
	// all descendents of this block will also be descendents of the finalised block,
	// so don't need to check any of those
	if n.isDescendantOf(finalised) {
		return pruned
	}

	// if it's not an ancestor the finalised block, prune it
	if !finalised.isDescendantOf(n) {
		pruned = append(pruned, n.hash)
		n.parent.deleteChild(n)
	}

	// if this is an ancestor of the finalised block, keep it,
	// and check its children
	for _, child := range n.children {
		pruned = child.prune(finalised, pruned)
	}

	return pruned
}

func (n *node) deleteChild(toDelete *node) {
	for i, child := range n.children {
		if child.hash == toDelete.hash {
			n.children = append(n.children[:i], n.children[i+1:]...)
			return
		}
	}
}

func (n *node) primaryAncestorCount(count int) int {
	if n == nil {
		return count
	}

	if n.isPrimary && n.parent != nil {
		// if parent is nil, we're at the root node
		// we don't need to count it, as all blocks have the root as an ancestor
		count++
	}

	return n.parent.primaryAncestorCount(count)
}

func (n *node) hashesAtNumber(number uint, hashes []common.Hash) []common.Hash {
	// there is no need to go furthen in the node's children
	// since they have a greater number at least
	if number == n.number {
		hashes = append(hashes, n.hash)
		return hashes
	}

	// if the number is greater than current node,
	// then search among its children
	if number > n.number {
		for _, children := range n.children {
			hashes = children.hashesAtNumber(number, hashes)
		}

		return hashes
	}

	return hashes
}
