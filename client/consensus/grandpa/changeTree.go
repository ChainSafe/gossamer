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

var (
	errDuplicateHashes     = errors.New("duplicated hashes")
	errUnfinalizedAncestor = errors.New("finalized descendent of Tree node without finalizing its ancestor(s) first")
	errRevert              = errors.New("tried to import or finalize node that is an ancestor of a previously finalized node")
)

// ChangeTree keeps track of the changes per fork allowing
// n forks in the same structure. This structure is intended
// to represent an acyclic directed graph where the change children are
// placed by descendency order and number, you can ensure an
// node ancestry using the `isDescendantOfFunc`
type ChangeTree struct {
	roots               []*pendingChangeNode
	bestFinalizedNumber *uint
}

// NewChangeTree create an empty ChangeTree
func NewChangeTree() *ChangeTree {
	return &ChangeTree{}
}

// pendingChangeNode Represents a node in the ChangeTree
type pendingChangeNode struct {
	change   *PendingChange
	children []*pendingChangeNode
}

// Roots returns the roots of each fork in the ChangeTree
// This is the equivalent of the slice in the outermost layer of the roots
func (ct *ChangeTree) Roots() []*pendingChangeNode {
	return ct.roots
}

// Import a new node into the roots.
//
// The given function `is_descendent_of` should return `true` if the second
// hash (target) is a descendent of the first hash (base).
//
// This method assumes that children in the same branch are imported in order.
//
// Returns `true` if the imported node is a root.
// WARNING: some users of this method (i.e. consensus epoch changes roots) currently silently
// rely on a **post-order DFS** traversal. If we are using instead a top-down traversal method
// then the `is_descendent_of` closure, when used after a warp-sync, may end up querying the
// backend for a block (the one corresponding to the root) that is not present and thus will
// return a wrong result.
func (ct *ChangeTree) Import(hash common.Hash, number uint, change PendingChange, isDescendentOf IsDescendentOf) (bool, error) {
	for _, root := range ct.roots {
		imported, err := root.importNode(hash, number, change, isDescendentOf)
		if err != nil {
			return false, err
		}

		if imported {
			logger.Debugf("changes on header %s (%d) imported successfully",
				hash, number)
			return false, nil
		}
	}

	pendingChangeNode := &pendingChangeNode{
		change: &change,
	}

	ct.roots = append(ct.roots, pendingChangeNode)
	return true, nil
}

// GetPreOrder does a preorder traversal of the ChangeTree to get all pending changes
func (ct *ChangeTree) GetPreOrder() []PendingChange {
	if len(ct.roots) == 0 {
		return nil
	}

	changes := make([]PendingChange, 0, len(ct.roots))

	for i := 0; i < len(ct.roots); i++ {
		getPreOrder(&changes, ct.roots[i])
	}

	return changes
}

// GetPreOrderChangeNodes does a preorder traversal of the ChangeTree to get all pending changes
func (ct *ChangeTree) GetPreOrderChangeNodes() []*pendingChangeNode {
	if len(ct.roots) == 0 {
		return nil
	}

	changes := &[]*pendingChangeNode{}

	for i := 0; i < len(ct.roots); i++ {
		getPreOrderChangeNodes(changes, ct.roots[i])
	}

	return *changes
}

// FinalizeAnyWithDescendentIf Checks if any node in the tree is finalized by either finalizing the
// node itself or a node's descendent that's not in the tree, guaranteeing
// that the node being finalized isn't a descendent of (or equal to) any of
// the node's children. Returns *true if the node being finalized is
// a root, *false if the node being finalized is not a root, and
// nil if no node in the tree is finalized. The given `Predicate` is
// checked on the prospective finalized root and must pass for finalization
// to occur. The given function `is_descendent_of` should return `true` if
// the second hash (target) is a descendent of the first hash (base).
func (ct *ChangeTree) FinalizeAnyWithDescendentIf(hash *common.Hash, number uint, isDescendentOf IsDescendentOf, predicate Predicate[*PendingChange]) (*bool, error) {
	if ct.bestFinalizedNumber != nil {
		if number <= *ct.bestFinalizedNumber {
			return nil, errRevert
		}
	}

	roots := ct.Roots()

	nodes := ct.GetPreOrderChangeNodes()

	// check if the given hash is equal or a descendent of any node in the
	// tree, if we find a valid node that passes the Predicate then we must
	// ensure that we're not finalizing past any of its child nodes.
	for i := 0; i < len(nodes); i++ {
		root := nodes[i]
		isDesc, err := isDescendentOf(root.change.canonHash, *hash)
		if err != nil {
			return nil, err
		}

		if predicate(root.change) && (root.change.canonHash == *hash || isDesc) {
			children := root.children
			for _, child := range children {
				isChildDescOf, err := isDescendentOf(child.change.canonHash, *hash)
				if err != nil {
					return nil, err
				}

				if child.change.canonHeight <= number && (child.change.canonHash == *hash || isChildDescOf) {
					return nil, errUnfinalizedAncestor
				}
			}

			isEqual := false
			for _, val := range roots {
				if val.change.canonHash == root.change.canonHash {
					isEqual = true
					break
				}
			}
			return &isEqual, nil
		}
	}

	return nil, nil
}

// FinalizationResult Result of finalizing a node (that could be a part of the roots or not).
// When the struct is nil it's the unchanged case, when it's not its been changed and contains an optional value
type FinalizationResult struct {
	value *PendingChange
}

// FinalizeWithDescendentIf Finalize a root in the roots by either finalizing the node itself or a
// node's descendent that's not in the roots, guaranteeing that the node
// being finalized isn't a descendent of (or equal to) any of the root's
// children. The given `Predicate` is checked on the prospective finalized
// root and must pass for finalization to occur. The given function
// `is_descendent_of` should return `true` if the second hash (target) is a
// descendent of the first hash (base).
func (ct *ChangeTree) FinalizeWithDescendentIf(hash *common.Hash, number uint, isDescendentOf IsDescendentOf, predicate Predicate[*PendingChange]) (*FinalizationResult, error) {
	if ct.bestFinalizedNumber != nil {
		if number <= *ct.bestFinalizedNumber {
			return nil, errRevert
		}
	}

	roots := ct.Roots()

	// check if the given hash is equal or a descendent of any root, if we
	// find a valid root that passes the Predicate then we must ensure that
	// we're not finalizing past any children node.
	var position *uint
	for i, root := range roots {
		isDesc, err := isDescendentOf(root.change.canonHash, *hash)
		if err != nil {
			return nil, err
		}

		if predicate(root.change) && (root.change.canonHash == *hash || isDesc) {
			for _, child := range root.children {
				isDesc, err := isDescendentOf(child.change.canonHash, *hash)
				if err != nil {
					return nil, err
				}
				if child.change.canonHeight <= number && (child.change.canonHash == *hash || isDesc) {
					return nil, errUnfinalizedAncestor
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
		ct.roots = node.children
		ct.bestFinalizedNumber = &node.change.canonHeight
		nodeData = node.change
	}

	// Retain only roots that are descendents of the finalized block (this
	// happens if the node has been properly finalized) or that are
	// ancestors (or equal) to the finalized block (in this case the node
	// wasn't finalized earlier presumably because the Predicate didn't
	// pass).
	changed := false
	roots = ct.Roots()

	ct.roots = []*pendingChangeNode{}
	for _, root := range roots {
		retain := false
		if root.change.canonHeight > number {
			isDescA, err := isDescendentOf(*hash, root.change.canonHash)
			if err != nil {
				return nil, err
			}

			if isDescA {
				retain = true
			}
		} else if root.change.canonHeight == number && root.change.canonHash == *hash {
			retain = true
		} else {
			isDescB, err := isDescendentOf(root.change.canonHash, *hash)
			if err != nil {
				return nil, err
			}

			if isDescB {
				retain = true
			}
		}
		if retain {
			ct.roots = append(ct.roots, root)
		} else {
			changed = true
		}

		ct.bestFinalizedNumber = &number
	}

	if nodeData != nil {
		return &FinalizationResult{value: nodeData}, nil
	} else {
		if changed {
			return &FinalizationResult{}, nil
		} else {
			return nil, nil
		}
	}
}

func (pcn *pendingChangeNode) importNode(hash common.Hash, number uint, change PendingChange, isDescendentOf IsDescendentOf) (bool, error) {
	announcingHash := pcn.change.canonHash
	if hash == announcingHash {
		return false, fmt.Errorf("%w: %s", errDuplicateHashes, hash)
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

	for _, childrenNodes := range pcn.children {
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
	pcn.children = append(pcn.children, childrenNode)
	return true, nil
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

	for i := 0; i < len(changeNode.children); i++ {
		getPreOrder(changes, changeNode.children[i])
	}
}

func getPreOrderChangeNodes(changes *[]*pendingChangeNode, changeNode *pendingChangeNode) {
	if changeNode == nil {
		return
	}

	if changes != nil {
		tempChanges := *changes
		tempChanges = append(tempChanges, changeNode)
		*changes = tempChanges
	} else {
		change := []*pendingChangeNode{changeNode}
		changes = &change
	}

	for i := 0; i < len(changeNode.children); i++ {
		getPreOrderChangeNodes(changes, changeNode.children[i])
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
		ct.roots = newRoots
		return val
	}
	newRoots[index] = lastElem
	ct.roots = newRoots
	return val
}

func (ct *ChangeTree) DrainFilter() {
	// TODO implement
}
