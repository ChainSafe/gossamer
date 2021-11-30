// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package blocktree

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNode_GetLeaves(t *testing.T) {
	var bt *BlockTree
	var branches []testBranch

	for {
		bt, branches = createTestBlockTree(t, testHeader, 5)
		if len(branches) > 0 && len(bt.getNode(branches[0].hash).children) > 0 {
			break
		}
	}

	testNode := bt.getNode(branches[0].hash).children[0]
	leaves := testNode.getLeaves(nil)

	expected := []*node{}
	for _, lf := range bt.leaves.toMap() {
		if lf.isDescendantOf(testNode) {
			expected = append(expected, lf)
		}
	}

	require.ElementsMatch(t, expected, leaves)
}

func TestNode_Prune(t *testing.T) {
	var bt *BlockTree
	var branches []testBranch

	for {
		bt, branches = createTestBlockTree(t, testHeader, 5)
		if len(branches) > 0 && len(bt.getNode(branches[0].hash).children) > 1 {
			break
		}
	}

	copy := bt.DeepCopy()

	// pick some block to finalise
	finalised := bt.root.children[0].children[0].children[0]
	pruned := bt.root.prune(finalised, nil)

	for _, prunedHash := range pruned {
		prunedNode := copy.getNode(prunedHash)
		if prunedNode.isDescendantOf(finalised) {
			t.Fatal("pruned node that's descendant of finalised node!!")
		}

		if finalised.isDescendantOf(prunedNode) {
			t.Fatal("pruned an ancestor of the finalised node!!")
		}
	}
}
