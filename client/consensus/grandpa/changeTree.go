// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"errors"
	"fmt"
	"github.com/ChainSafe/gossamer/lib/common"
)

/*
	TODO give a summary of how this works in context of grandpa
*/

// Represents a node in the ChangeTree
type pendingChangeNode struct {
	change *PendingChange
	nodes  []*pendingChangeNode
}

func (pcn *pendingChangeNode) importNode(hash common.Hash, number uint, change PendingChange, isDescendentOf IsDescendentOf) (bool, error) {

	announcingHash := pcn.change.canonHash
	if hash == announcingHash {
		return false, fmt.Errorf("%w: %s", errors.New("duplicated hashes"), hash)
	}

	isDescendant, err := isDescendentOf(announcingHash, hash)
	if err != nil {
		return false, fmt.Errorf("cannot check ancestry: %w", err)
	}

	if !isDescendant {
		return false, nil
	}

	if number <= pcn.change.canonHeight {
		return false, nil
	}

	for _, childrenNodes := range pcn.nodes {
		imported, err := childrenNodes.importNode(hash, number, change, isDescendentOf)
		if err != nil {
			return false, err
		}

		if imported {
			return true, nil
		}
	}
	childrenNode := &pendingChangeNode{
		change: &change,
	}
	pcn.nodes = append(pcn.nodes, childrenNode)
	return true, nil
}

// ChangeTree keeps track of the changes per fork allowing
// n forks in the same structure, this structure is intended
// to be an acyclic directed graph where the change nodes are
// placed by descendency order and number, you can ensure an
// node ancestry using the `isDescendantOfFunc`
type ChangeTree struct {
	tree  []*pendingChangeNode
	count uint
}

// NewChangeTree create an empty ChangeTree
func NewChangeTree() ChangeTree {
	return ChangeTree{}
}

// Import a new node into the tree.
//
// The given function `is_descendent_of` should return `true` if the second
// hash (target) is a descendent of the first hash (base).
//
// This method assumes that nodes in the same branch are imported in order.
//
// Returns `true` if the imported node is a root.
// WARNING: some users of this method (i.e. consensus epoch changes tree) currently silently
// rely on a **post-order DFS** traversal. If we are using instead a top-down traversal method
// then the `is_descendent_of` closure, when used after a warp-sync, may end up querying the
// backend for a block (the one corresponding to the root) that is not present and thus will
// return a wrong result.
func (ct *ChangeTree) Import(hash common.Hash, number uint, change PendingChange, isDescendentOf IsDescendentOf) (bool, error) {
	for _, root := range ct.tree {
		imported, err := root.importNode(hash, number, change, isDescendentOf)
		if err != nil {
			return false, err
		}

		if imported {
			logger.Debugf("changes on header %s (%d) imported successfully",
				hash, number)
			ct.count++
			return false, nil
		}
	}

	pendingChangeNode := &pendingChangeNode{
		change: &change,
	}

	ct.tree = append(ct.tree, pendingChangeNode)
	ct.count++
	return true, nil
}

// Roots returns the roots of each fork in the ChangeTree
// This is the equivalent of the slice in the outermost layer of the tree
func (ct *ChangeTree) Roots() []*pendingChangeNode {
	return ct.tree
}
