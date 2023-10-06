// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"errors"
	"fmt"

	"github.com/ChainSafe/gossamer/pkg/scale"
	"golang.org/x/exp/constraints"
)

/*
	The grandpa ChangeTree is a structure built to track pending changes across forks for the Grandpa Protocol.
	This structure is intended to represent an acyclic directed graph where the children are
    placed in descending order and number, you can ensure node ancestry using the `isDescendantOfFunc`.
*/

var (
	errDuplicateHashes     = errors.New("duplicated hashes")
	errUnfinalisedAncestor = errors.New("finalised descendent of Tree node without finalising its " +
		"ancestor(s) first")
	errRevert = errors.New("tried to import or finalise node that is an ancestor of " +
		"a previously finalised node")
)

// ChangeTree keeps track of changes across forks
type ChangeTree[H comparable, N constraints.Unsigned] struct {
	roots               []*pendingChangeNode[H, N]
	bestFinalizedNumber *N
}

// NewChangeTree create an empty ChangeTree
func NewChangeTree[H comparable, N constraints.Unsigned]() ChangeTree[H, N] {
	return ChangeTree[H, N]{}
}

// pendingChangeNode Represents a node in the ChangeTree
type pendingChangeNode[H comparable, N constraints.Unsigned] struct {
	change   *PendingChange[H, N]
	children []*pendingChangeNode[H, N]
}

// Roots returns the roots of each fork in the ChangeTree
// This is the equivalent of the slice in the outermost layer of the roots
func (ct *ChangeTree[H, N]) Roots() []*pendingChangeNode[H, N] { //skipcq: RVV-B0011
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
func (ct *ChangeTree[H, N]) Import(hash H,
	number N,
	change PendingChange[H, N],
	isDescendentOf IsDescendentOf[H]) (bool, error) {
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

	pendingChangeNode := &pendingChangeNode[H, N]{
		change: &change,
	}

	ct.roots = append(ct.roots, pendingChangeNode)
	return true, nil
}

// PendingChanges does a preorder traversal of the ChangeTree to get all pending changes
func (ct *ChangeTree[H, N]) PendingChanges() []PendingChange[H, N] {
	if len(ct.roots) == 0 {
		return nil
	}

	changes := make([]PendingChange[H, N], 0, len(ct.roots))

	for i := 0; i < len(ct.roots); i++ {
		getPreOrder(&changes, ct.roots[i])
	}

	return changes
}

// getPreOrderChangeNodes does a preorder traversal of the ChangeTree to get all pending changes
func (ct *ChangeTree[H, N]) getPreOrderChangeNodes() []*pendingChangeNode[H, N] {
	if len(ct.roots) == 0 {
		return nil
	}

	changes := &[]*pendingChangeNode[H, N]{}

	for i := 0; i < len(ct.roots); i++ {
		getPreOrderChangeNodes(changes, ct.roots[i])
	}

	return *changes
}

// FinalizesAnyWithDescendentIf Checks if any node in the tree is finalised by either finalising the
// node itself or a node's descendent that's not in the tree, guaranteeing
// that the node being finalised isn't a descendent of (or equal to) any of
// the node's children. Returns *true if the node being finalised is
// a root, *false if the node being finalised is not a root, and
// nil if no node in the tree is finalised. The given `Predicate` is
// checked on the prospective finalised root and must pass for finalisation
// to occur. The given function `is_descendent_of` should return `true` if
// the second hash (target) is a descendent of the first hash (base). func(T) bool
func (ct *ChangeTree[H, N]) FinalizesAnyWithDescendentIf(hash *H,
	number N,
	isDescendentOf IsDescendentOf[H],
	predicate func(*PendingChange[H, N]) bool) (*bool, error) {
	if ct.bestFinalizedNumber != nil {
		if number <= *ct.bestFinalizedNumber {
			return nil, errRevert
		}
	}

	roots := ct.Roots()

	nodes := ct.getPreOrderChangeNodes()

	// check if the given hash is equal or a descendent of any node in the
	// tree, if we find a valid node that passes the Predicate then we must
	// ensure that we're not finalising past any of its child nodes.
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
					return nil, errUnfinalisedAncestor
				}
			}

			isEqual := false
			for _, val := range roots {
				if val.change.canonHash == root.change.canonHash {
					isEqual = true
					return &isEqual, nil
				}
			}
			return &isEqual, nil
		}
	}

	return nil, nil
}

// FinalizationResult Result of finalising a node (that could be a part of the roots or not).
type FinalizationResult scale.VaryingDataType

// Set will set a VaryingDataTypeValue using the underlying VaryingDataType
func (fr *FinalizationResult) Set(val scale.VaryingDataTypeValue) (err error) {
	vdt := scale.VaryingDataType(*fr)
	err = vdt.Set(val)
	if err != nil {
		return
	}
	*fr = FinalizationResult(vdt)
	return
}

// Value will return value from underying VaryingDataType
func (fr *FinalizationResult) Value() (val scale.VaryingDataTypeValue, err error) {
	vdt := scale.VaryingDataType(*fr)
	return vdt.Value()
}

func newFinalizationResult[H comparable, N constraints.Unsigned]() FinalizationResult {
	vdt, err := scale.NewVaryingDataType(changed[H, N]{}, unchanged{})
	if err != nil {
		panic(err)
	}
	return FinalizationResult(vdt)
}

type changed[H comparable, N constraints.Unsigned] struct {
	value *PendingChange[H, N]
}

func (changed[H, N]) Index() uint {
	return 0
}

type unchanged struct{}

func (unchanged) Index() uint {
	return 1
}

// FinalizeWithDescendentIf Finalize a root in the roots by either finalising the node itself or a
// node's descendent that's not in the roots, guaranteeing that the node
// being finalised isn't a descendent of (or equal to) any of the root's
// children. The given `Predicate` is checked on the prospective finalised
// root and must pass for finalisation to occur. The given function
// `is_descendent_of` should return `true` if the second hash (target) is a
// descendent of the first hash (base).
func (ct *ChangeTree[H, N]) FinalizeWithDescendentIf(hash *H, //skipcq:   GO-R1005
	number N,
	isDescendentOf IsDescendentOf[H],
	predicate func(*PendingChange[H, N]) bool) (result FinalizationResult, err error) {
	if ct.bestFinalizedNumber != nil {
		if number <= *ct.bestFinalizedNumber {
			return result, errRevert
		}
	}

	roots := ct.Roots()

	// check if the given hash is equal or a descendent of any root, if we
	// find a valid root that passes the Predicate then we must ensure that
	// we're not finalising past any children node.
	var position *N
	for i, root := range roots {
		isDesc, err := isDescendentOf(root.change.canonHash, *hash)
		if err != nil {
			return result, err
		}

		if predicate(root.change) && (root.change.canonHash == *hash || isDesc) {
			for _, child := range root.children {
				isDesc, err := isDescendentOf(child.change.canonHash, *hash)
				if err != nil {
					return result, err
				}
				if child.change.canonHeight <= number && (child.change.canonHash == *hash || isDesc) {
					return result, errUnfinalisedAncestor
				}
			}
			uintI := N(i)
			position = &uintI
			break
		}
	}

	var nodeData *PendingChange[H, N]
	if position != nil {
		node := ct.swapRemove(ct.Roots(), *position)
		ct.roots = node.children
		ct.bestFinalizedNumber = &node.change.canonHeight
		nodeData = node.change
	}

	// Retain only roots that are descendents of the finalised block (this
	// happens if the node has been properly finalised) or that are
	// ancestors (or equal) to the finalised block (in this case the node
	// wasn't finalised earlier presumably because the Predicate didn't
	// pass).
	didChange := false
	roots = ct.Roots()

	ct.roots = []*pendingChangeNode[H, N]{}
	for _, root := range roots {
		retain := false
		if root.change.canonHeight > number {
			isDescA, err := isDescendentOf(*hash, root.change.canonHash)
			if err != nil {
				return result, err
			}

			if isDescA {
				retain = true
			}
		} else if root.change.canonHeight == number && root.change.canonHash == *hash {
			retain = true
		} else {
			isDescB, err := isDescendentOf(root.change.canonHash, *hash)
			if err != nil {
				return result, err
			}

			if isDescB {
				retain = true
			}
		}
		if retain {
			ct.roots = append(ct.roots, root)
		} else {
			didChange = true
		}

		ct.bestFinalizedNumber = &number
	}

	result = newFinalizationResult[H, N]()

	if nodeData != nil {
		err = result.Set(changed[H, N]{
			value: nodeData,
		})
		if err != nil {
			return result, err
		}
		return result, nil
	} else {
		if didChange {
			err = result.Set(changed[H, N]{})
			if err != nil {
				return result, err
			}
			return result, nil
		} else {
			err = result.Set(unchanged{})
			if err != nil {
				return result, err
			}
			return result, nil
		}
	}
}

func (pcn *pendingChangeNode[H, N]) importNode(hash H,
	number N,
	change PendingChange[H, N],
	isDescendentOf IsDescendentOf[H]) (bool, error) {
	announcingHash := pcn.change.canonHash
	if hash == announcingHash {
		return false, fmt.Errorf("%w: %v", errDuplicateHashes, hash)
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
	childrenNode := &pendingChangeNode[H, N]{
		change: &change,
	}
	pcn.children = append(pcn.children, childrenNode)
	return true, nil
}

func getPreOrder[H comparable, N constraints.Unsigned](changes *[]PendingChange[H, N], //skipcq: RVV-A0006
	changeNode *pendingChangeNode[H, N]) {
	if changeNode == nil {
		return
	}

	if changes != nil {
		tempChanges := *changes
		tempChanges = append(tempChanges, *changeNode.change)
		*changes = tempChanges
	} else {
		change := []PendingChange[H, N]{*changeNode.change}
		changes = &change
	}

	for i := 0; i < len(changeNode.children); i++ {
		getPreOrder(changes, changeNode.children[i])
	}
}

func getPreOrderChangeNodes[H comparable, N constraints.Unsigned]( //skipcq: RVV-A0006 //skipcq: RVV-B0001
	changes *[]*pendingChangeNode[H, N],
	changeNode *pendingChangeNode[H, N]) {
	if changeNode == nil {
		return
	}

	if changes != nil {
		tempChanges := *changes
		tempChanges = append(tempChanges, changeNode)
		*changes = tempChanges
	} else {
		change := []*pendingChangeNode[H, N]{changeNode}
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
func (ct *ChangeTree[H, N]) swapRemove(roots []*pendingChangeNode[H, N], index N) pendingChangeNode[H, N] {
	if index >= N(len(roots)) {
		panic("swap_remove index out of bounds")
	}

	val := pendingChangeNode[H, N]{}
	if roots[index] != nil {
		val = *roots[index]
	} else {
		panic("nil pending hashNumber node")
	}

	lastElem := roots[len(roots)-1]

	newRoots := roots[:len(roots)-1]
	// This should be the case where last elem was removed
	if index == N(len(newRoots)) {
		ct.roots = newRoots
		return val
	}
	newRoots[index] = lastElem
	ct.roots = newRoots
	return val
}

// Remove from the tree some nodes (and their subtrees) using a `filter` predicate.
//
// The `filter` is called over tree nodes and returns a filter action:
// - `Remove` if the node and its subtree should be removed;
// - `KeepNode` if we should maintain the node and keep processing the tree.
// - `KeepTree` if we should maintain the node and its entire subtree.
//
// An iterator over all the pruned nodes is returned.
func (_ *ChangeTree[H, N]) drainFilter() { //nolint //skipcq: SCC-U1000
	// TODO implement
}
