// Copyright 2019 ChainSafe Systems (ON) Corp.
// This file is part of gossamer.
//
// The gossamer library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The gossamer library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the gossamer library. If not, see <http://www.gnu.org/licenses/>.

package blocktree

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNode_GetLeaves(t *testing.T) {
	var bt *BlockTree
	var branches []testBranch

	for {
		bt, branches = createTestBlockTree(testHeader, 5, nil)
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
		bt, branches = createTestBlockTree(testHeader, 5, nil)
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
