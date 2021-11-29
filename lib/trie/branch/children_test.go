// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package branch

import (
	"testing"

	"github.com/ChainSafe/gossamer/lib/trie/leaf"
	"github.com/ChainSafe/gossamer/lib/trie/node"
	"github.com/stretchr/testify/assert"
)

func Test_Branch_ChildrenBitmap(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		branch *Branch
		bitmap uint16
	}{
		"no children": {
			branch: &Branch{},
		},
		"index 0": {
			branch: &Branch{
				Children: [16]node.Node{
					&leaf.Leaf{},
				},
			},
			bitmap: 1,
		},
		"index 0 and 4": {
			branch: &Branch{
				Children: [16]node.Node{
					&leaf.Leaf{},
					nil, nil, nil,
					&leaf.Leaf{},
				},
			},
			bitmap: 1<<4 + 1,
		},
		"index 0, 4 and 15": {
			branch: &Branch{
				Children: [16]node.Node{
					&leaf.Leaf{},
					nil, nil, nil,
					&leaf.Leaf{},
					nil, nil, nil, nil, nil,
					nil, nil, nil, nil, nil,
					&leaf.Leaf{},
				},
			},
			bitmap: 1<<15 + 1<<4 + 1,
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			bitmap := testCase.branch.ChildrenBitmap()

			assert.Equal(t, testCase.bitmap, bitmap)
		})
	}
}

func Test_Branch_NumChildren(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		branch *Branch
		count  int
	}{
		"zero": {
			branch: &Branch{},
		},
		"one": {
			branch: &Branch{
				Children: [16]node.Node{
					&leaf.Leaf{},
				},
			},
			count: 1,
		},
		"two": {
			branch: &Branch{
				Children: [16]node.Node{
					&leaf.Leaf{},
					nil, nil, nil,
					&leaf.Leaf{},
				},
			},
			count: 2,
		},
		"three": {
			branch: &Branch{
				Children: [16]node.Node{
					&leaf.Leaf{},
					nil, nil, nil,
					&leaf.Leaf{},
					nil, nil, nil, nil, nil,
					nil, nil, nil, nil, nil,
					&leaf.Leaf{},
				},
			},
			count: 3,
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			count := testCase.branch.NumChildren()

			assert.Equal(t, testCase.count, count)
		})
	}
}
