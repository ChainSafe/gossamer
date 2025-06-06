// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package forktree

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func testForkTree(t *testing.T) (ForkTree[string, uint64, uint32], func(string, string) (bool, error)) {
	t.Helper()
	tree := NewForkTree[string, uint64, uint32]()

	//     +---B-c-C---D---E
	//     |
	//     |   +---G
	//     |   |
	// 0---A---F---H---I
	//     |       |
	//     |       +---L-m-M---N
	//     |           |
	//     |           +---O
	//     +---J---K
	//
	// (where N is not a part of fork tree)
	//
	// NOTE: the tree will get automatically rebalance on import and won't be laid out like the
	// diagram above. the children will be ordered by subtree depth and the longest branches
	// will be on the leftmost side of the tree.
	var isDescendantOf = func(base, block string) (bool, error) {
		letters := []string{"B", "C", "D", "E", "F", "G", "H", "I", "J", "K", "L", "M", "N", "O"}
		// This is a trick to have lowercase blocks be direct parents of their
		// uppercase correspondent (A excluded)
		block = strings.ToUpper(block)
		switch {
		case base == "A":
			_, found := slices.BinarySearch(letters, block)
			return found, nil
		case base == "B" || base == "c":
			return block == "C" || block == "D" || block == "E", nil
		case base == "C":
			return block == "D" || block == "E", nil
		case base == "D":
			return block == "E", nil
		case base == "E":
			return false, nil
		case base == "F":
			return block == "G" || block == "H" || block == "I" || block == "L" || block == "M" || block == "N" || block == "O", nil
		case base == "G":
			return false, nil
		case base == "H":
			return block == "I" || block == "L" || block == "M" || block == "N" || block == "O", nil
		case base == "I":
			return false, nil
		case base == "J":
			return block == "K", nil
		case base == "K":
			return false, nil
		case base == "L":
			return block == "M" || block == "N" || block == "O", nil
		case base == "m":
			return block == "M" || block == "N", nil
		case base == "M":
			return block == "N", nil
		case base == "O":
			return false, nil
		case base == "0":
			return true, nil
		default:
			return false, nil
		}
	}

	_, err := tree.Import("A", 10, 1, isDescendantOf)
	require.NoError(t, err)
	_, err = tree.Import("B", 20, 2, isDescendantOf)
	require.NoError(t, err)
	_, err = tree.Import("C", 30, 3, isDescendantOf)
	require.NoError(t, err)
	_, err = tree.Import("D", 40, 4, isDescendantOf)
	require.NoError(t, err)
	_, err = tree.Import("E", 50, 5, isDescendantOf)
	require.NoError(t, err)
	_, err = tree.Import("F", 20, 2, isDescendantOf)
	require.NoError(t, err)
	_, err = tree.Import("G", 30, 3, isDescendantOf)
	require.NoError(t, err)
	_, err = tree.Import("H", 30, 3, isDescendantOf)
	require.NoError(t, err)
	_, err = tree.Import("I", 40, 4, isDescendantOf)
	require.NoError(t, err)
	_, err = tree.Import("L", 40, 4, isDescendantOf)
	require.NoError(t, err)
	_, err = tree.Import("M", 50, 5, isDescendantOf)
	require.NoError(t, err)
	_, err = tree.Import("O", 50, 5, isDescendantOf)
	require.NoError(t, err)
	_, err = tree.Import("J", 20, 2, isDescendantOf)
	require.NoError(t, err)
	_, err = tree.Import("K", 30, 3, isDescendantOf)
	require.NoError(t, err)

	return tree, isDescendantOf
}

func TestForkTree(t *testing.T) {
	t.Run("import_doesnt_revert", func(t *testing.T) {
		tree, isDescendentOf := testForkTree(t)

		tree.FinalizeRoot("A")

		require.NotNil(t, tree.bestFinalizedNumber)
		require.Equal(t, uint64(10), *tree.bestFinalizedNumber)

		_, err := tree.Import("A", 10, 1, isDescendentOf)
		require.Error(t, err)
	})

	t.Run("import_doesnt_add_duplicates", func(t *testing.T) {
		tree, isDescendentOf := testForkTree(t)

		_, err := tree.Import("A", 10, 1, isDescendentOf)
		require.ErrorIs(t, err, ErrDuplicate)

		_, err = tree.Import("I", 40, 4, isDescendentOf)
		require.ErrorIs(t, err, ErrDuplicate)

		_, err = tree.Import("G", 30, 3, isDescendentOf)
		require.ErrorIs(t, err, ErrDuplicate)

		_, err = tree.Import("K", 30, 3, isDescendentOf)
		require.ErrorIs(t, err, ErrDuplicate)
	})

	type hn struct {
		H string
		N uint64
	}
	t.Run("finalize_root_works", func(t *testing.T) {
		var finalizeA = func() ForkTree[string, uint64, uint32] {
			tree, _ := testForkTree(t)

			var actual []hn
			for _, node := range tree.Roots() {
				actual = append(actual, hn{H: node.Hash, N: node.Number})
			}
			require.Equal(t, []hn{{H: "A", N: 10}}, actual)

			require.Nil(t, tree.bestFinalizedNumber)

			// finalizing "A" opens up three possible forks
			tree.FinalizeRoot("A")

			actual = nil
			for _, node := range tree.Roots() {
				actual = append(actual, hn{H: node.Hash, N: node.Number})
			}
			require.Equal(t, []hn{{H: "B", N: 20}, {H: "F", N: 20}, {H: "J", N: 20}}, actual)

			return tree
		}

		{
			tree := finalizeA()

			// finalizing "B" will progress on its fork and remove any other competing forks
			tree.FinalizeRoot("B")

			var actual []hn
			for _, node := range tree.Roots() {
				actual = append(actual, hn{H: node.Hash, N: node.Number})
			}
			require.Equal(t, []hn{{H: "C", N: 30}}, actual)
			// all the other forks have been pruned
			require.Len(t, tree.roots, 1)
		}

		{
			tree := finalizeA()

			// finalizing "J" will progress on its fork and remove any other competing forks
			tree.FinalizeRoot("J")

			var actual []hn
			for _, node := range tree.Roots() {
				actual = append(actual, hn{H: node.Hash, N: node.Number})
			}
			require.Equal(t, []hn{{H: "K", N: 30}}, actual)
			// all the other forks have been pruned
			require.Len(t, tree.roots, 1)
		}
	})

	t.Run("finalize_works", func(t *testing.T) {
		tree, isDescendentOf := testForkTree(t)

		originalRoots := slices.Clone(tree.roots)

		// finalizing a block prior to any in the node doesn't change the tree
		res, err := tree.Finalize("0", 0, isDescendentOf)
		require.NoError(t, err)
		require.Equal(t, FinalizationResultUnchanged{}, res)

		require.Equal(t, originalRoots, tree.roots)

		// finalizing "A" opens up three possible forks
		res, err = tree.Finalize("A", 10, isDescendentOf)
		require.NoError(t, err)
		v := uint32(1)
		require.Equal(t, FinalizationResultChanged[uint32]{Value: &v}, res)

		var actual []hn
		for _, node := range tree.Roots() {
			actual = append(actual, hn{H: node.Hash, N: node.Number})
		}
		require.Equal(t, []hn{{H: "B", N: 20}, {H: "F", N: 20}, {H: "J", N: 20}}, actual)

		// finalizing anything lower than what we observed will fail
		require.NotNil(t, tree.bestFinalizedNumber)
		require.Equal(t, uint64(10), *tree.bestFinalizedNumber)

		_, err = tree.Finalize("Z", 10, isDescendentOf)
		require.ErrorIs(t, err, ErrRevert)

		// trying to finalize a node without finalizing its ancestors first will fail
		_, err = tree.Finalize("H", 30, isDescendentOf)
		require.ErrorIs(t, err, ErrUnfinalizedAncestor)

		// after finalizing "F" we can finalize "H"
		res, err = tree.Finalize("F", 20, isDescendentOf)
		require.NoError(t, err)
		v = uint32(2)
		require.Equal(t, FinalizationResultChanged[uint32]{Value: &v}, res)

		res, err = tree.Finalize("H", 30, isDescendentOf)
		require.NoError(t, err)
		v = uint32(3)
		require.Equal(t, FinalizationResultChanged[uint32]{Value: &v}, res)

		actual = nil
		for _, node := range tree.Roots() {
			actual = append(actual, hn{H: node.Hash, N: node.Number})
		}
		require.Equal(t, []hn{{H: "L", N: 40}, {H: "I", N: 40}}, actual)

		// finalizing a node from another fork that isn't part of the tree clears the tree
		res, err = tree.Finalize("Z", 50, isDescendentOf)
		require.NoError(t, err)
		require.Equal(t, FinalizationResultChanged[uint32]{Value: nil}, res)

		require.Empty(t, tree.roots)
	})

	t.Run("finalize_with_ancestor_works", func(t *testing.T) {
		tree, isDescendentOf := testForkTree(t)

		originalRoots := slices.Clone(tree.roots)

		// finalizing a block prior to any in the node doesn't change the tree
		res, err := tree.FinalizeWithAncestors("0", 0, isDescendentOf)
		require.NoError(t, err)
		require.Equal(t, FinalizationResultUnchanged{}, res)

		require.Equal(t, originalRoots, tree.roots)

		// finalizing "A" opens up three possible forks
		res, err = tree.FinalizeWithAncestors("A", 10, isDescendentOf)
		require.NoError(t, err)
		v := uint32(1)
		require.Equal(t, FinalizationResultChanged[uint32]{Value: &v}, res)

		var actual []hn
		for _, node := range tree.Roots() {
			actual = append(actual, hn{H: node.Hash, N: node.Number})
		}
		require.Equal(t, []hn{{H: "B", N: 20}, {H: "F", N: 20}, {H: "J", N: 20}}, actual)

		// finalizing H:
		// 1) removes roots that are not ancestors/descendants of H (B, J)
		// 2) opens root that is ancestor of H (F -> G+H)
		// 3) finalizes the just opened root H (H -> I + L)
		res, err = tree.FinalizeWithAncestors("H", 30, isDescendentOf)
		require.NoError(t, err)
		v = uint32(3)
		require.Equal(t, FinalizationResultChanged[uint32]{Value: &v}, res)

		actual = nil
		for _, node := range tree.Roots() {
			actual = append(actual, hn{H: node.Hash, N: node.Number})
		}
		require.Equal(t, []hn{{H: "L", N: 40}, {H: "I", N: 40}}, actual)

		require.NotNil(t, tree.bestFinalizedNumber)
		require.Equal(t, uint64(30), *tree.bestFinalizedNumber)

		// finalizing N (which is not a part of the tree):
		// 1) removes roots that are not ancestors/descendants of N (I)
		// 2) opens root that is ancestor of N (L -> M+O)
		// 3) removes roots that are not ancestors/descendants of N (O)
		// 4) opens root that is ancestor of N (M -> {})
		res, err = tree.FinalizeWithAncestors("N", 60, isDescendentOf)
		require.NoError(t, err)
		require.Equal(t, FinalizationResultChanged[uint32]{Value: nil}, res)

		actual = nil
		for _, node := range tree.Roots() {
			actual = append(actual, hn{H: node.Hash, N: node.Number})
		}
		require.Empty(t, actual)

		require.NotNil(t, tree.bestFinalizedNumber)
		require.Equal(t, uint64(60), *tree.bestFinalizedNumber)
	})

	t.Run("finalize_with_descendent_works", func(t *testing.T) {
		type Change struct {
			Effective uint64
		}
		tree := NewForkTree[string, uint64, Change]()
		isDescendentOf := func(base, block string) (bool, error) {
			// A0 #1 - (B #2) - (C #5) - D #10 - E #15 - (F #100)
			//                            \
			//                             - (G #100)
			//
			// A1 #1
			//
			// Nodes B, C, F and G  are not part of the tree.
			switch {
			case base == "A0":
				return block == "B" || block == "C" || block == "D" || block == "E" || block == "G", nil
			case base == "A1":
				return false, nil
			case base == "C":
				return block == "D", nil
			case base == "D":
				return block == "E" || block == "F" || block == "G", nil
			case base == "E":
				return block == "F", nil
			default:
				return false, nil
			}
		}

		isRoot, err := tree.Import("A0", 1, Change{Effective: 5}, isDescendentOf)
		require.NoError(t, err)
		require.True(t, isRoot)
		isRoot, err = tree.Import("A1", 1, Change{Effective: 5}, isDescendentOf)
		require.NoError(t, err)
		require.True(t, isRoot)
		isRoot, err = tree.Import("D", 10, Change{Effective: 10}, isDescendentOf)
		require.NoError(t, err)
		require.False(t, isRoot)
		isRoot, err = tree.Import("E", 15, Change{Effective: 50}, isDescendentOf)
		require.NoError(t, err)
		require.False(t, isRoot)

		res, err := tree.FinalizesAnyWithDescendentIf("B", 2, isDescendentOf, func(c Change) bool {
			return c.Effective <= 2
		})
		require.NoError(t, err)
		require.Nil(t, res)

		// finalizing "D" is not allowed since it is not a root.
		_, err = tree.FinalizeWithDescendentIf("D", 10, isDescendentOf, func(c Change) bool {
			return c.Effective <= 10
		})
		require.ErrorIs(t, err, ErrUnfinalizedAncestor)

		// finalizing "D" will finalize a block from the tree, but it can't be applied yet
		// since it is not a root change.
		res, err = tree.FinalizesAnyWithDescendentIf("D", 10, isDescendentOf, func(c Change) bool {
			return c.Effective == 10
		})
		require.NoError(t, err)
		require.NotNil(t, res)
		require.False(t, *res)

		// finalizing "E" is not allowed since there are not finalized ancestors.
		_, err = tree.FinalizesAnyWithDescendentIf("E", 15, isDescendentOf, func(c Change) bool {
			return c.Effective == 10
		})
		require.ErrorIs(t, err, ErrUnfinalizedAncestor)

		// finalizing "B" doesn't finalize "A0" since the predicate doesn't pass,
		// although it will clear out "A1" from the tree
		result, err := tree.FinalizeWithDescendentIf("B", 2, isDescendentOf, func(c Change) bool {
			return c.Effective <= 2
		})
		require.NoError(t, err)
		require.Equal(t, FinalizationResultChanged[Change]{Value: nil}, result)

		var actual []hn
		for _, node := range tree.Roots() {
			actual = append(actual, hn{H: node.Hash, N: node.Number})
		}
		require.Equal(t, []hn{{H: "A0", N: 1}}, actual)

		// finalizing "C" will finalize the node "A0" and prune it out of the tree
		res, err = tree.FinalizesAnyWithDescendentIf("C", 5, isDescendentOf, func(c Change) bool {
			return c.Effective <= 5
		})
		require.NoError(t, err)
		require.NotNil(t, res)
		require.True(t, *res)

		result, err = tree.FinalizeWithDescendentIf("C", 5, isDescendentOf, func(c Change) bool {
			return c.Effective <= 5
		})
		require.NoError(t, err)
		require.Equal(t, FinalizationResultChanged[Change]{Value: &Change{Effective: 5}}, result)

		actual = nil
		for _, node := range tree.Roots() {
			actual = append(actual, hn{H: node.Hash, N: node.Number})
		}
		require.Equal(t, []hn{{H: "D", N: 10}}, actual)

		// finalizing "F" will fail since it would finalize past "E" without finalizing "D" first
		_, err = tree.FinalizesAnyWithDescendentIf("F", 100, isDescendentOf, func(c Change) bool {
			return c.Effective <= 100
		})
		require.ErrorIs(t, err, ErrUnfinalizedAncestor)

		// it will work with "G" though since it is not in the same branch as "E"
		res, err = tree.FinalizesAnyWithDescendentIf("G", 100, isDescendentOf, func(c Change) bool {
			return c.Effective <= 100
		})
		require.NoError(t, err)
		require.NotNil(t, res)
		require.True(t, *res)

		result, err = tree.FinalizeWithDescendentIf("G", 100, isDescendentOf, func(c Change) bool {
			return c.Effective <= 100
		})
		require.NoError(t, err)
		require.Equal(t, FinalizationResultChanged[Change]{Value: &Change{Effective: 10}}, result)

		// "E" will be pruned out
		require.Empty(t, tree.roots)
	})

	t.Run("iter_iterates_in_preorder", func(t *testing.T) {
		tree, _ := testForkTree(t)

		actual := make([]hn, 0)
		for node := range tree.Iter() {
			actual = append(actual, hn{H: node.Hash, N: node.Number})
		}
		expected := []hn{
			{H: "A", N: 10},
			{H: "B", N: 20},
			{H: "C", N: 30},
			{H: "D", N: 40},
			{H: "E", N: 50},
			{H: "F", N: 20},
			{H: "H", N: 30},
			{H: "L", N: 40},
			{H: "M", N: 50},
			{H: "O", N: 50},
			{H: "I", N: 40},
			{H: "G", N: 30},
			{H: "J", N: 20},
			{H: "K", N: 30},
		}
		require.Equal(t, expected, actual)
	})

	t.Run("minimizes_calls_to_is_descendent_of", func(t *testing.T) {
		var nIsDescendentOfCalls uint

		isDescendentOf := func(base, block string) (bool, error) {
			nIsDescendentOfCalls++
			return true, nil
		}

		{
			// Deep tree where we want to call `finalizes_any_with_descendent_if`. The
			// search for the node should first check the predicate (which is cheaper) and
			// only then call `isDescendentOf`
			tree := NewForkTree[string, uint, uint]()
			letters := []string{"A", "B", "C", "D", "E", "F", "G", "H", "I", "J", "K"}

			for i, letter := range letters {
				_, err := tree.Import(letter, uint(i), uint(i), func(base, block string) (bool, error) {
					return true, nil
				})
				require.NoError(t, err)
			}

			// "L" is a descendent of "K", but the predicate will only pass for "K",
			// therefore only one call to `isDescendentOf` should be made
			res, err := tree.FinalizesAnyWithDescendentIf("L", 11, isDescendentOf, func(i uint) bool {
				return i == 10
			})
			require.NoError(t, err)
			require.NotNil(t, res)
			require.False(t, *res)

			require.Equal(t, uint(1), nIsDescendentOfCalls)
		}

		nIsDescendentOfCalls = 0

		{
			// Multiple roots in the tree where we want to call `finalize_with_descendent_if`.
			// The search for the root node should first check the predicate (which is cheaper)
			// and only then call `isDescendentOf`
			tree := NewForkTree[string, uint, uint]()
			letters := []string{"A", "B", "C", "D", "E", "F", "G", "H", "I", "J", "K"}

			for i, letter := range letters {
				_, err := tree.Import(letter, uint(i), uint(i), func(base, block string) (bool, error) {
					return false, nil
				})
				require.NoError(t, err)
			}

			// "L" is a descendent of "K", but the predicate will only pass for "K",
			// therefore only one call to `isDescendentOf` should be made
			res, err := tree.FinalizeWithDescendentIf("L", 11, isDescendentOf, func(i uint) bool {
				return i == 10
			})
			require.NoError(t, err)
			v := uint(10)
			require.Equal(t, FinalizationResultChanged[uint]{Value: &v}, res)

			require.Equal(t, uint(1), nIsDescendentOfCalls)
		}
	})

	t.Run("map_works", func(t *testing.T) {
		tree, _ := testForkTree(t)

		// Extend the single root fork-tree to also exercise the roots order during map.
		isDescendentOf := func(base, block string) (bool, error) {
			return false, nil
		}
		isRoot, err := tree.Import("A1", 10, 1, isDescendentOf)
		require.NoError(t, err)
		require.True(t, isRoot)
		isRoot, err = tree.Import("A2", 10, 1, isDescendentOf)
		require.NoError(t, err)
		require.True(t, isRoot)

		oldTree := tree.Clone()
		newTree := Map[string, uint64, uint32, string](tree, func(hash string, number uint64, data uint32) string {
			return hash
		})
		require.Equal(t, len(oldTree.Roots()), len(newTree.Roots()))

		// Check content and order
		for node := range newTree.Iter() {
			require.Equal(t, node.Hash, node.Data)
		}
		oldTreeHashes := make([]string, 0)
		for node := range oldTree.Iter() {
			oldTreeHashes = append(oldTreeHashes, node.Hash)
		}
		newTreeHashes := make([]string, 0)
		for node := range newTree.Iter() {
			newTreeHashes = append(newTreeHashes, node.Hash)
		}
		require.Equal(t, oldTreeHashes, newTreeHashes)
	})

	t.Run("prune_works_for_in_tree_hashes", func(t *testing.T) {
		tree, isDescendentOf := testForkTree(t)

		removed, err := tree.Prune("C", 30, isDescendentOf, func(height uint32) bool {
			return true
		})
		require.NoError(t, err)
		actual := make([]string, 0)
		for _, node := range tree.Roots() {
			actual = append(actual, node.Hash)
		}
		require.Equal(t, []string{"B"}, actual)

		actual = nil
		for node := range tree.Iter() {
			actual = append(actual, node.Hash)
		}
		require.Equal(t, []string{"B", "C", "D", "E"}, actual)

		removedHashes := make([]string, 0)
		for node := range removed {
			removedHashes = append(removedHashes, node.Hash)
		}
		require.Equal(t, []string{"A", "F", "H", "L", "M", "O", "I", "G", "J", "K"}, removedHashes)

		removed, err = tree.Prune("E", 50, isDescendentOf, func(height uint32) bool {
			return true
		})

		actual = nil
		for _, node := range tree.Roots() {
			actual = append(actual, node.Hash)
		}
		require.Equal(t, []string{"D"}, actual)

		actual = nil
		for node := range tree.Iter() {
			actual = append(actual, node.Hash)
		}
		require.Equal(t, []string{"D", "E"}, actual)

		removedHashes = nil
		for node := range removed {
			removedHashes = append(removedHashes, node.Hash)
		}
		require.Equal(t, []string{"B", "C"}, removedHashes)
	})

	t.Run("prune_works_for_out_of_tree_hashes", func(t *testing.T) {
		tree, isDescendentOf := testForkTree(t)

		removed, err := tree.Prune("c", 25, isDescendentOf, func(height uint32) bool {
			return true
		})
		require.NoError(t, err)

		actual := make([]string, 0)
		for _, node := range tree.Roots() {
			actual = append(actual, node.Hash)
		}
		require.Equal(t, []string{"B"}, actual)

		actual = nil
		for node := range tree.Iter() {
			actual = append(actual, node.Hash)
		}
		require.Equal(t, []string{"B", "C", "D", "E"}, actual)

		removedHashes := make([]string, 0)
		for node := range removed {
			removedHashes = append(removedHashes, node.Hash)
		}
		require.Equal(t, []string{"A", "F", "H", "L", "M", "O", "I", "G", "J", "K"}, removedHashes)
	})

	t.Run("prune_works_for_not_direct_ancestor", func(t *testing.T) {
		tree, isDescendentOf := testForkTree(t)

		// This is to re-root the tree not at the immediate ancestor, but the one just before.
		removed, err := tree.Prune("m", 45, isDescendentOf, func(height uint32) bool {
			return height == 3
		})
		require.NoError(t, err)

		actual := make([]string, 0)
		for _, node := range tree.Roots() {
			actual = append(actual, node.Hash)
		}
		require.Equal(t, []string{"H"}, actual)

		actual = nil
		for node := range tree.Iter() {
			actual = append(actual, node.Hash)
		}
		require.Equal(t, []string{"H", "L", "M"}, actual)

		removedHashes := make([]string, 0)
		for node := range removed {
			removedHashes = append(removedHashes, node.Hash)
		}
		require.Equal(t, []string{"O", "I", "A", "B", "C", "D", "E", "F", "G", "J", "K"}, removedHashes)
	})

	t.Run("prune_works_for_far_away_ancestor", func(t *testing.T) {
		tree, isDescendentOf := testForkTree(t)

		removed, err := tree.Prune("m", 45, isDescendentOf, func(height uint32) bool {
			return height == 2
		})
		require.NoError(t, err)

		actual := make([]string, 0)
		for _, node := range tree.Roots() {
			actual = append(actual, node.Hash)
		}
		require.Equal(t, []string{"F"}, actual)

		actual = nil
		for node := range tree.Iter() {
			actual = append(actual, node.Hash)
		}
		require.Equal(t, []string{"F", "H", "L", "M"}, actual)

		removedHashes := make([]string, 0)
		for node := range removed {
			removedHashes = append(removedHashes, node.Hash)
		}
		require.Equal(t, []string{"O", "I", "G", "A", "B", "C", "D", "E", "J", "K"}, removedHashes)
	})

	t.Run("find_node_backtracks_after_finding_highest_descending_node", func(t *testing.T) {
		tree := NewForkTree[string, int32, int32]()

		// A - B
		//  \
		//   — C
		isDescendentOf := func(base, block string) (bool, error) {
			switch base {
			case "A":
				return block == "B" || block == "C" || block == "D", nil
			case "B", "C":
				return block == "D", nil
			case "0":
				return true, nil
			default:
				return false, nil
			}
		}

		_, err := tree.Import("A", 1, 1, isDescendentOf)
		require.NoError(t, err)
		_, err = tree.Import("B", 2, 2, isDescendentOf)
		require.NoError(t, err)
		_, err = tree.Import("C", 2, 4, isDescendentOf)
		require.NoError(t, err)

		// when searching the tree we reach node `C`, but the
		// predicate doesn't pass. we should backtrack to `B`, but not to `A`,
		// since "B" fulfills the predicate.
		node, err := tree.FindNodeWhere("D", 3, isDescendentOf, func(data int32) bool {
			return data < 3
		})
		require.NoError(t, err)
		require.NotNil(t, node)

		require.Equal(t, "B", node.Hash)
	})

	t.Run("rebalance_works", func(t *testing.T) {
		tree, _ := testForkTree(t)

		// the tree is automatically rebalanced on import, therefore we should iterate in preorder
		// exploring the longest forks first. check the ascii art above to understand the expected
		// output below.
		actual := make([]string, 0)
		for node := range tree.Iter() {
			actual = append(actual, node.Hash)
		}
		require.Equal(t, []string{"A", "B", "C", "D", "E", "F", "H", "L", "M", "O", "I", "G", "J", "K"}, actual)

		// let's add a block "P" which is a descendent of block "O"
		isDescendentOf := func(base, block string) (bool, error) {
			if block == "P" {
				return base == "A" || base == "F" || base == "H" || base == "L" || base == "O", nil
			}
			return false, nil
		}

		_, err := tree.Import("P", 60, 6, isDescendentOf)
		require.NoError(t, err)

		// this should re-order the tree, since the branch "A -> B -> C -> D -> E" is no longer tied
		// with 5 blocks depth. additionally "O" should be visited before "M" now, since it has one
		// descendent "P" which makes that branch 6 blocks long.
		actual = nil
		for node := range tree.Iter() {
			actual = append(actual, node.Hash)
		}
		require.Equal(t, []string{"A", "F", "H", "L", "O", "P", "M", "I", "G", "B", "C", "D", "E", "J", "K"}, actual)
	})

	t.Run("drain_filter_works", func(t *testing.T) {
		tree, _ := testForkTree(t)

		filter := func(h string, n uint64, d uint32) FilterAction {
			switch h {
			case "A", "B", "F", "G":
				return FilterActionKeepNode
			case "C":
				return FilterActionKeepTree
			case "H", "J":
				return FilterActionRemove
			default:
				panic(fmt.Sprintf("Unexpected filtering for node: %s", h))
			}
		}

		removed := tree.DrainFilter(filter)

		actual := make([]string, 0)
		for node := range tree.Iter() {
			actual = append(actual, node.Hash)
		}
		require.Equal(t, []string{"A", "B", "C", "D", "E", "F", "G"}, actual)

		removedHashes := make([]string, 0)
		for node := range removed {
			removedHashes = append(removedHashes, node.Hash)
		}
		require.Equal(t, []string{"H", "L", "M", "O", "I", "J", "K"}, removedHashes)
	})

	t.Run("find_node_index_works", func(t *testing.T) {
		tree, isDescendentOf := testForkTree(t)

		path, err := tree.FindNodeIndexWhere("D", 40, isDescendentOf, func(data uint32) bool {
			return true
		})
		require.NoError(t, err)
		require.NotNil(t, path)
		require.Equal(t, []uint{0, 0, 0}, path)

		path, err = tree.FindNodeIndexWhere("O", 50, isDescendentOf, func(data uint32) bool {
			return true
		})
		require.NoError(t, err)
		require.NotNil(t, path)
		require.Equal(t, []uint{0, 1, 0, 0}, path)

		path, err = tree.FindNodeIndexWhere("N", 60, isDescendentOf, func(data uint32) bool {
			return true
		})
		require.NoError(t, err)
		require.NotNil(t, path)
		require.Equal(t, []uint{0, 1, 0, 0, 0}, path)
	})

	t.Run("find_node_index_with_predicate_works", func(t *testing.T) {
		isDescendentOf := func(parent, child rune) (bool, error) {
			switch parent {
			case 'A':
				return child == 'B' || child == 'C' || child == 'D' || child == 'E' || child == 'F', nil
			case 'B':
				return child == 'C' || child == 'D', nil
			case 'C':
				return child == 'D', nil
			case 'E':
				return child == 'F', nil
			case 'D', 'F':
				return false, nil
			default:
				return false, fmt.Errorf("TestError")
			}
		}

		// A(t) --- B(f) --- C(t) --- D(f)
		//      \-- E(t) --- F(f)
		tree := NewForkTree[rune, uint8, bool]()
		_, err := tree.Import('A', 1, true, isDescendentOf)
		require.NoError(t, err)
		_, err = tree.Import('B', 2, false, isDescendentOf)
		require.NoError(t, err)
		_, err = tree.Import('C', 3, true, isDescendentOf)
		require.NoError(t, err)
		_, err = tree.Import('D', 4, false, isDescendentOf)
		require.NoError(t, err)

		_, err = tree.Import('E', 2, true, isDescendentOf)
		require.NoError(t, err)
		_, err = tree.Import('F', 3, false, isDescendentOf)
		require.NoError(t, err)

		path, err := tree.FindNodeIndexWhere('D', 4, isDescendentOf, func(value bool) bool {
			return !value
		})
		require.NoError(t, err)
		require.NotNil(t, path)
		require.Equal(t, []uint{0, 0}, path)

		path, err = tree.FindNodeIndexWhere('D', 4, isDescendentOf, func(value bool) bool {
			return value
		})
		require.NoError(t, err)
		require.NotNil(t, path)
		require.Equal(t, []uint{0, 0, 0}, path)

		path, err = tree.FindNodeIndexWhere('F', 3, isDescendentOf, func(value bool) bool {
			return !value
		})
		require.NoError(t, err)
		require.Nil(t, path)

		path, err = tree.FindNodeIndexWhere('F', 3, isDescendentOf, func(value bool) bool {
			return value
		})
		require.NoError(t, err)
		require.NotNil(t, path)
		require.Equal(t, []uint{0, 1}, path)
	})

	t.Run("find_node_works", func(t *testing.T) {
		tree, isDescendentOf := testForkTree(t)

		node, err := tree.FindNodeWhere("B", 20, isDescendentOf, func(data uint32) bool {
			return true
		})
		require.NoError(t, err)
		require.NotNil(t, node)
		require.Equal(t, "A", node.Hash)
		require.Equal(t, uint64(10), node.Number)

		node, err = tree.FindNodeWhere("D", 40, isDescendentOf, func(data uint32) bool {
			return true
		})
		require.NoError(t, err)
		require.NotNil(t, node)
		require.Equal(t, "C", node.Hash)
		require.Equal(t, uint64(30), node.Number)

		node, err = tree.FindNodeWhere("O", 50, isDescendentOf, func(data uint32) bool {
			return true
		})
		require.NoError(t, err)
		require.NotNil(t, node)
		require.Equal(t, "L", node.Hash)
		require.Equal(t, uint64(40), node.Number)

		node, err = tree.FindNodeWhere("N", 60, isDescendentOf, func(data uint32) bool {
			return true
		})
		require.NoError(t, err)
		require.NotNil(t, node)
		require.Equal(t, "M", node.Hash)
		require.Equal(t, uint64(50), node.Number)
	})

	t.Run("post_order_traversal_requirement", func(t *testing.T) {
		tree, isDescendentOf := testForkTree(t)

		// Test for the post-order DFS traversal requirement as specified by the
		// `FindNodeIndexWhere` and `Import` comments.
		isDescendentOfForPostOrder := func(parent, child string) (bool, error) {
			if parent == "A" {
				return false, fmt.Errorf("TestError")
			}
			if parent == "K" && child == "Z" {
				return true, nil
			}
			return isDescendentOf(parent, child)
		}

		// Post order traversal requirement for `FindNodeIndexWhere`
		path, err := tree.FindNodeIndexWhere("N", 60, isDescendentOfForPostOrder, func(data uint32) bool {
			return true
		})
		require.NoError(t, err)
		require.NotNil(t, path)
		require.Equal(t, []uint{0, 1, 0, 0, 0}, path)

		// Post order traversal requirement for `Import`
		res, err := tree.Import("Z", 100, 10, isDescendentOfForPostOrder)
		require.NoError(t, err)
		require.False(t, res)
		actual := make([]string, 0)
		for node := range tree.Iter() {
			actual = append(actual, node.Hash)
		}
		require.Equal(t, []string{"A", "B", "C", "D", "E", "F", "H", "L", "M", "O", "I", "G", "J", "K", "Z"}, actual)
	})
}
