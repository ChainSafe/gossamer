// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package node

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func testForSliceModif(t *testing.T, original, copied []byte) {
	t.Helper()
	if !reflect.DeepEqual(original, copied) || len(copied) == 0 {
		// cannot test for modification
		return
	}
	original[0]++
	assert.NotEqual(t, copied, original)
}

func Test_Node_Copy(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		node         *Node
		settings     CopySettings
		expectedNode *Node
	}{
		"empty branch": {
			node: &Node{
				Children: make([]*Node, ChildrenCapacity),
			},
			expectedNode: &Node{
				Children: make([]*Node, ChildrenCapacity),
			},
		},
		"non empty branch": {
			node: &Node{
				Key:      []byte{1, 2},
				SubValue: []byte{3, 4},
				Children: padRightChildren([]*Node{
					nil, nil, {
						Key:      []byte{9},
						SubValue: []byte{1},
					},
				}),
				Dirty:       true,
				MerkleValue: []byte{5},
			},
			settings: DefaultCopySettings,
			expectedNode: &Node{
				Key:      []byte{1, 2},
				SubValue: []byte{3, 4},
				Children: padRightChildren([]*Node{
					nil, nil, {
						Key:      []byte{9},
						SubValue: []byte{1},
					},
				}),
				Dirty: true,
			},
		},
		"branch with children copied": {
			node: &Node{
				Children: padRightChildren([]*Node{
					nil, nil, {
						Key:      []byte{9},
						SubValue: []byte{1},
					},
				}),
			},
			settings: CopySettings{
				CopyChildren: true,
			},
			expectedNode: &Node{
				Children: padRightChildren([]*Node{
					nil, nil, {
						Key:      []byte{9},
						SubValue: []byte{1},
					},
				}),
			},
		},
		"deep copy branch": {
			node: &Node{
				Key:      []byte{1, 2},
				SubValue: []byte{3, 4},
				Children: padRightChildren([]*Node{
					nil, nil, {
						Key:      []byte{9},
						SubValue: []byte{1},
					},
				}),
				Dirty:       true,
				MerkleValue: []byte{5},
			},
			settings: DeepCopySettings,
			expectedNode: &Node{
				Key:      []byte{1, 2},
				SubValue: []byte{3, 4},
				Children: padRightChildren([]*Node{
					nil, nil, {
						Key:      []byte{9},
						SubValue: []byte{1},
					},
				}),
				Dirty:       true,
				MerkleValue: []byte{5},
			},
		},
		"non empty leaf": {
			node: &Node{
				Key:         []byte{1, 2},
				SubValue:    []byte{3, 4},
				Dirty:       true,
				MerkleValue: []byte{5},
			},
			settings: DefaultCopySettings,
			expectedNode: &Node{
				Key:      []byte{1, 2},
				SubValue: []byte{3, 4},
				Dirty:    true,
			},
		},
		"deep copy leaf": {
			node: &Node{
				Key:         []byte{1, 2},
				SubValue:    []byte{3, 4},
				Dirty:       true,
				MerkleValue: []byte{5},
			},
			settings: DeepCopySettings,
			expectedNode: &Node{
				Key:         []byte{1, 2},
				SubValue:    []byte{3, 4},
				Dirty:       true,
				MerkleValue: []byte{5},
			},
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			nodeCopy := testCase.node.Copy(testCase.settings)

			assert.Equal(t, testCase.expectedNode, nodeCopy)
			testForSliceModif(t, testCase.node.Key, nodeCopy.Key)
			testForSliceModif(t, testCase.node.SubValue, nodeCopy.SubValue)
			testForSliceModif(t, testCase.node.MerkleValue, nodeCopy.MerkleValue)

			if testCase.node.Kind() == Branch {
				testCase.node.Children[15] = &Node{Key: []byte("modified")}
				assert.NotEqual(t, nodeCopy.Children, testCase.node.Children)
			}
		})
	}
}
