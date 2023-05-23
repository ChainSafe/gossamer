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
	tree                []*pendingChangeNode
	count               uint
	bestFinalizedNumber *uint
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

func (ct *ChangeTree) GetPreOrder() []PendingChange {
	if len(ct.tree) == 0 {
		return nil
	}

	changes := &[]PendingChange{}

	// this is basically a preorder search with rotating roots
	for i := 0; i < len(ct.tree); i++ {
		getPreOrder(changes, ct.tree[i])
	}

	return *changes
}

// FinalizeWithDescendentIf Finalize a root in the tree by either finalizing the node itself or a
// node's descendent that's not in the tree, guaranteeing that the node
// being finalized isn't a descendent of (or equal to) any of the root's
// children. The given `predicate` is checked on the prospective finalized
// root and must pass for finalization to occur. The given function
// `is_descendent_of` should return `true` if the second hash (target) is a
// descendent of the first hash (base).
func (ct *ChangeTree) FinalizeWithDescendentIf(hash *common.Hash, number uint, isDescendentOf IsDescendentOf, predicate predicate[*PendingChange]) error {
	if ct.bestFinalizedNumber != nil {
		if number <= *ct.bestFinalizedNumber {
			return fmt.Errorf("tried to import or finalize node that is an ancestor of a previously finalized node")
		}
	}

	roots := ct.Roots()

	// check if the given hash is equal or a descendent of any root, if we
	// find a valid root that passes the predicate then we must ensure that
	// we're not finalizing past any children node.
	var position *uint
	for i := 0; i < len(roots); i++ {
		root := roots[i]
		isDesc, err := isDescendentOf(root.change.canonHash, *hash)
		if err != nil {
			return err
		}
		if predicate(root.change) && root.change.canonHash == *hash || isDesc {
			for _, child := range root.nodes {
				isDesc, err := isDescendentOf(child.change.canonHash, *hash)
				if err != nil {
					return err
				}
				if child.change.canonHeight <= number && child.change.canonHash == *hash || isDesc {
					return fmt.Errorf("finalized descendent of Tree node without finalizing its ancestor(s) first")
				}
			}
			uintI := uint(i)
			position = &uintI
			break
		}
	}
}

func getPreOrder(changes *[]PendingChange, changeNode *pendingChangeNode) {
	if changeNode == nil {
		return
	}

	if changes != nil {
		tempChanges := *changes
		tempChanges = append(tempChanges, *changeNode.change)
		*changes = tempChanges
	} else {
		change := []PendingChange{*changeNode.change}
		changes = &change
	}

	for i := 0; i < len(changeNode.nodes); i++ {
		getPreOrder(changes, changeNode.nodes[i])
	}
}
