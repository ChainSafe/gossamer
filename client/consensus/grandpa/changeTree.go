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
	nodes  []*pendingChangeNode // TODO change this to children
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

// GetPreOrder does a preorder traversal of the ChangeTree to get all pending changes
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

// FinalizationResult Result of finalizing a node (that could be a part of the tree or not).
// When the struct is nil its the unchanged case, when its not its been changed and contains an optional value
type FinalizationResult struct {
	value *PendingChange
}

// FinalizeWithDescendentIf Finalize a root in the tree by either finalizing the node itself or a
// node's descendent that's not in the tree, guaranteeing that the node
// being finalized isn't a descendent of (or equal to) any of the root's
// children. The given `predicate` is checked on the prospective finalized
// root and must pass for finalization to occur. The given function
// `is_descendent_of` should return `true` if the second hash (target) is a
// descendent of the first hash (base).
//
// TODO NOTE: I for now instead of a vdt I will just interpret the signature of Result<FinalizationResult<V>, Error<E>>
// TODO cont: as a pointer to a struct
func (ct *ChangeTree) FinalizeWithDescendentIf(hash *common.Hash, number uint, isDescendentOf IsDescendentOf, predicate predicate[*PendingChange]) (*FinalizationResult, error) {
	if ct.bestFinalizedNumber != nil {
		if number <= *ct.bestFinalizedNumber {
			return nil, fmt.Errorf("tried to import or finalize node that is an ancestor of a previously finalized node")
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
			return nil, err
		}
		if predicate(root.change) && root.change.canonHash == *hash || isDesc {
			for _, child := range root.nodes {
				isDesc, err := isDescendentOf(child.change.canonHash, *hash)
				if err != nil {
					return nil, err
				}
				if child.change.canonHeight <= number && child.change.canonHash == *hash || isDesc {
					return nil, fmt.Errorf("finalized descendent of Tree node without finalizing its ancestor(s) first")
				}
			}
			uintI := uint(i)
			position = &uintI
			break
		}
	}

	var nodeData *PendingChange
	if position != nil {
		node := ct.swapRemove(ct.Roots(), *position)
		ct.tree = node.nodes
		// TODO verify this is the correct number, pretty sure we store canon number but weird to use here
		// I feel like (maybe effective height)
		ct.bestFinalizedNumber = &node.change.canonHeight
		nodeData = node.change
	}

	// Retain only roots that are descendents of the finalized block (this
	// happens if the node has been properly finalized) or that are
	// ancestors (or equal) to the finalized block (in this case the node
	// wasn't finalized earlier presumably because the predicate didn't
	// pass).
	changed := false
	roots = ct.Roots()

	// NOTE: import does use the canon hash and height

	for i := 0; i < len(roots); i++ {
		root := roots[i]
		isDescA, err := isDescendentOf(*hash, root.change.canonHash)
		if err != nil {
			return nil, err
		}

		isDescB, err := isDescendentOf(root.change.canonHash, *hash)
		if err != nil {
			return nil, err
		}

		retain := root.change.canonHeight > number && isDescA ||
			root.change.canonHeight == number && root.change.canonHash == *hash || isDescB

		if retain {
			// TODO make sure this is ok
			ct.tree = append(ct.tree, root)
		} else {
			changed = true
		}

		ct.bestFinalizedNumber = &number

	}

	if nodeData != nil {
		// Ok(FinalizationResult::Changed(Some(data))),
		return &FinalizationResult{value: nodeData}, nil
	} else {
		if changed {
			// Ok(FinalizationResult::Changed(None)),
			return &FinalizationResult{}, nil
		} else {
			// (None, false) => Ok(FinalizationResult::Unchanged),
			return nil, nil
		}
	}
}

// Removes an element from the vector and returns it.
//
// The removed element is replaced by the last element of the vector.
//
// This does not preserve ordering, but is *O*(1).
//
// Panics if `index` is out of bounds.
func (ct *ChangeTree) swapRemove(roots []*pendingChangeNode, index uint) pendingChangeNode {
	if index >= uint(len(roots)) {
		panic("swap_remove index out of bounds")
	}

	val := pendingChangeNode{}
	if roots[index] != nil {
		val = *roots[index]
	} else {
		panic("nil pending change node")
	}

	lastElem := roots[len(roots)-1]

	newRoots := roots[:len(roots)-1]
	// This should be the case where last elem was removed
	if index == uint(len(newRoots)) {
		ct.tree = newRoots
		return val
	}
	newRoots[index] = lastElem
	ct.tree = newRoots
	return val
}
