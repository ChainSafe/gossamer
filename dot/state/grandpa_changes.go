// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package state

import (
	"fmt"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
)

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

// appliedBefore compares the effective number between two pending changes
// and returns true if:
// - the current pending change is applied before the target
// - the target is nil and the current contains a value
func (p *pendingChange) appliedBefore(target *pendingChange) bool {
	if p != nil && target != nil {
		return p.effectiveNumber() < target.effectiveNumber()
	}

	return p != nil
}

type orderedPendingChanges []*pendingChange

func (o orderedPendingChanges) Len() int      { return len(o) }
func (o orderedPendingChanges) Swap(i, j int) { o[i], o[j] = o[j], o[i] }

// Less order by effective number and then by block number
func (o orderedPendingChanges) Less(i, j int) bool {
	return o[i].effectiveNumber() < o[j].effectiveNumber() &&
		o[i].announcingHeader.Number < o[j].announcingHeader.Number
}

type pendingChangeNode struct {
	change *pendingChange
	nodes  []*pendingChangeNode
}

func (c *pendingChangeNode) importScheduledChange(blockHash common.Hash, blockNumber uint, pendingChange *pendingChange,
	isDescendantOf isDescendantOfFunc) (imported bool, err error) {
	announcingHash := c.change.announcingHeader.Hash()

	if blockHash.Equal(announcingHash) {
		return false, errDuplicateHashes
	}

	if blockNumber <= c.change.announcingHeader.Number {
		return false, nil
	}

	for _, childrenNodes := range c.nodes {
		imported, err := childrenNodes.importScheduledChange(blockHash, blockNumber, pendingChange, isDescendantOf)
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
