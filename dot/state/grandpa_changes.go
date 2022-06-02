// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package state

import (
	"fmt"
	"sort"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
)

type conditionFunc[T any] func(T) (bool, error)
type isDescendantOfFunc func(parent, child common.Hash) (bool, error)

type pendingChange struct {
	bestFinalizedNumber uint32
	delay               uint32
	nextAuthorities     []types.Authority
	announcingHeader    *types.Header
}

func (p pendingChange) String() string {
	return fmt.Sprintf("announcing header: %s (%d), delay: %d, next authorities: %d",
		p.announcingHeader.Hash(), p.announcingHeader.Number, p.delay, len(p.nextAuthorities))
}

func (p *pendingChange) effectiveNumber() uint {
	return p.announcingHeader.Number + uint(p.delay)
}

type orderedPendingChanges []*pendingChange

func (oc *orderedPendingChanges) Len() int { return len(*oc) }

// findApplicable try to retrieve an applicable change from the slice of forced changes
func (oc orderedPendingChanges) findApplicable(importedHash common.Hash, importedNumber uint,
	isDescendatOf isDescendantOfFunc) (*pendingChange, error) {

	return oc.lookupChangeWhere(func(forced *pendingChange) (bool, error) {
		announcingHash := forced.announcingHeader.Hash()
		effectiveNumber := forced.effectiveNumber()

		if importedHash.Equal(announcingHash) && effectiveNumber == importedNumber {
			return true, nil
		}

		isDescendant, err := isDescendatOf(announcingHash, importedHash)
		if err != nil {
			return false, fmt.Errorf("cannot check ancestry: %w", err)
		}

		return isDescendant && effectiveNumber == importedNumber, nil
	})

}

// lookupChangeWhere return the first pending change which satisfy the condition
func (oc orderedPendingChanges) lookupChangeWhere(condition conditionFunc[*pendingChange]) (
	pendingChange *pendingChange, err error) {
	for _, change := range oc {
		ok, err := condition(change)
		if err != nil {
			return nil, fmt.Errorf("failed while applying condition: %w", err)
		}

		if ok {
			return change, nil
		}
	}

	return nil, nil //nolint:nilnil
}

// importChange only tracks the pending change if and only if it is the
// unique forced change in its fork, otherwise will return an error
func (oc *orderedPendingChanges) importChange(pendingChange *pendingChange, isDescendantOf isDescendantOfFunc) error {
	announcingHeader := pendingChange.announcingHeader.Hash()

	for _, change := range *oc {
		changeBlockHash := change.announcingHeader.Hash()

		if changeBlockHash.Equal(announcingHeader) {
			return errDuplicateHashes
		}

		isDescendant, err := isDescendantOf(changeBlockHash, announcingHeader)
		if err != nil {
			return fmt.Errorf("cannot verify ancestry: %w", err)
		}

		if isDescendant {
			return errAlreadyHasForcedChanges
		}
	}

	// Use a binary search to include the pending change in the right position
	// of a slice ordered by the effective number and by announcing header number
	idxToInsert := sort.Search(oc.Len(), func(i int) bool {
		return (*oc)[i].effectiveNumber() >= pendingChange.effectiveNumber() &&
			(*oc)[i].announcingHeader.Number >= pendingChange.announcingHeader.Number
	})

	*oc = append(*oc, pendingChange)
	copy((*oc)[idxToInsert+1:], (*oc)[idxToInsert:])
	(*oc)[idxToInsert] = pendingChange
	return nil
}

// pruneChanges will remove changes whose are not descendant of the hash argument
// this function updates the current state of the change tree
func (oc *orderedPendingChanges) pruneChanges(hash common.Hash, isDescendantOf isDescendantOfFunc) error {
	onBranchForcedChanges := make([]*pendingChange, 0, oc.Len())

	for _, forcedChange := range *oc {
		isDescendant, err := isDescendantOf(hash, forcedChange.announcingHeader.Hash())
		if err != nil {
			return fmt.Errorf("cannot verify ancestry: %w", err)
		}

		if isDescendant {
			onBranchForcedChanges = append(onBranchForcedChanges, forcedChange)
		}
	}

	*oc = make(orderedPendingChanges, len(onBranchForcedChanges))
	copy(*oc, onBranchForcedChanges)
	return nil
}

type pendingChangeNode struct {
	change *pendingChange
	nodes  []*pendingChangeNode
}

func (c *pendingChangeNode) importNode(blockHash common.Hash, blockNumber uint, pendingChange *pendingChange,
	isDescendantOf isDescendantOfFunc) (imported bool, err error) {
	announcingHash := c.change.announcingHeader.Hash()

	if blockHash.Equal(announcingHash) {
		return false, errDuplicateHashes
	}

	if blockNumber <= c.change.announcingHeader.Number {
		return false, nil
	}

	for _, childrenNodes := range c.nodes {
		imported, err := childrenNodes.importNode(blockHash, blockNumber, pendingChange, isDescendantOf)
		if err != nil {
			return false, err
		}

		if imported {
			return true, nil
		}
	}

	isDescendant, err := isDescendantOf(announcingHash, blockHash)
	if err != nil {
		return false, fmt.Errorf("cannot define ancestry: %w", err)
	}

	if !isDescendant {
		return false, nil
	}

	pendingChangeNode := &pendingChangeNode{change: pendingChange, nodes: []*pendingChangeNode{}}
	c.nodes = append(c.nodes, pendingChangeNode)
	return true, nil
}

// changeTree keeps track of the changes per fork allowing
// n forks in the same structure, this structure is intended
// to be an acyclic directed graph where the change nodes are
// placed by descendency order and number, you can ensure an
// node ancestry using the `isDescendantOfFunc`
type changeTree []*pendingChangeNode

func (ct changeTree) Len() int { return len(ct) }
func (ct *changeTree) importChange(pendingChange *pendingChange, isDescendantOf isDescendantOfFunc) error {
	for _, root := range *ct {
		imported, err := root.importNode(pendingChange.announcingHeader.Hash(),
			pendingChange.announcingHeader.Number, pendingChange, isDescendantOf)

		if err != nil {
			return err
		}

		if imported {
			logger.Debugf("changes on header %s (%d) imported successfully",
				pendingChange.announcingHeader.Hash(), pendingChange.announcingHeader.Number)
			return nil
		}
	}

	pendingChangeNode := &pendingChangeNode{
		change: pendingChange,
		nodes:  []*pendingChangeNode{},
	}

	*ct = append(*ct, pendingChangeNode)
	return nil
}

// lookupChangesWhere returns the first change which satisfy the
// condition whithout modify the current state of the change tree
func (ct changeTree) lookupChangeWhere(condition conditionFunc[*pendingChangeNode]) (
	changeNode *pendingChangeNode, err error) {
	for _, root := range ct {
		ok, err := condition(root)
		if err != nil {
			return nil, fmt.Errorf("failed while applying condition: %w", err)
		}

		if ok {
			return root, nil
		}
	}

	return nil, nil //nolint:nilnil
}

// findApplicable try to retrieve an applicable change
// from the tree, if it finds a change node then it will update the
// tree roots with the change node's children otherwise it will
// prune nodes that does not belongs to the same chain as `hash` argument
func (ct *changeTree) findApplicable(hash common.Hash, number uint,
	isDescendantOf isDescendantOfFunc) (changeNode *pendingChangeNode, err error) {

	changeNode, err = ct.findApplicableChange(hash, number, isDescendantOf)
	if err != nil {
		return nil, fmt.Errorf("cannot find applicable change: %w", err)
	}

	if changeNode == nil {
		err := ct.pruneChanges(hash, isDescendantOf)
		if err != nil {
			return nil, fmt.Errorf("cannot prune changes: %w", err)
		}
	} else {
		*ct = make([]*pendingChangeNode, len(changeNode.nodes))
		copy(*ct, changeNode.nodes)
	}

	return changeNode, nil
}

// findApplicableChange iterates through the change tree
// roots looking for the change node which:
// 1. contains the same hash as the one we're looking for.
// 2. contains a lower or equal effective number as the one we're looking for.
// 3. does not contains pending changes to be applied.
func (ct changeTree) findApplicableChange(hash common.Hash, number uint,
	isDescendantOf isDescendantOfFunc) (changeNode *pendingChangeNode, err error) {
	return ct.lookupChangeWhere(func(pcn *pendingChangeNode) (bool, error) {
		if pcn.change.effectiveNumber() > number {
			return false, nil
		}

		changeNodeHash := pcn.change.announcingHeader.Hash()
		if !hash.Equal(changeNodeHash) {
			isDescendant, err := isDescendantOf(changeNodeHash, hash)
			if err != nil {
				return false, fmt.Errorf("cannot verify ancestry: %w", err)
			}

			if !isDescendant {
				return false, nil
			}
		}

		// the changes must be applied in order, so we need to check if our finalized header
		// is ahead of any children, if it is that means some previous change was not applied
		for _, child := range pcn.nodes {
			isDescendant, err := isDescendantOf(child.change.announcingHeader.Hash(), hash)
			if err != nil {
				return false, fmt.Errorf("cannot verify ancestry: %w", err)
			}

			if child.change.announcingHeader.Number <= number && isDescendant {
				return false, errUnfinalizedAncestor
			}
		}

		return true, nil
	})
}

// pruneChanges will remove changes whose are not descendant of the hash argument
// this function updates the current state of the change tree
func (ct changeTree) pruneChanges(hash common.Hash, isDescendantOf isDescendantOfFunc) error {
	onBranchChanges := []*pendingChangeNode{}

	for _, root := range ct {
		scheduledChangeHash := root.change.announcingHeader.Hash()

		isDescendant, err := isDescendantOf(hash, scheduledChangeHash)
		if err != nil {
			return fmt.Errorf("cannot verify ancestry: %w", err)
		}

		if isDescendant {
			onBranchChanges = append(onBranchChanges, root)
		}
	}

	ct = make([]*pendingChangeNode, len(onBranchChanges))
	copy(ct, onBranchChanges)
	return nil
}
