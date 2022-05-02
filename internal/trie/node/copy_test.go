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
		"empty leaf": {
			node:         &Node{Type: Leaf},
			settings:     DefaultCopySettings,
			expectedNode: &Node{Type: Leaf},
		},
		"non empty leaf": {
			node: &Node{
				Type:       Leaf,
				Key:        []byte{1, 2},
				Value:      []byte{3, 4},
				Dirty:      true,
				HashDigest: []byte{5},
				Encoding:   []byte{6},
			},
			settings: DefaultCopySettings,
			expectedNode: &Node{
				Type:  Leaf,
				Key:   []byte{1, 2},
				Value: []byte{3, 4},
				Dirty: true,
			},
		},
		"deep copy leaf": {
			node: &Node{
				Type:       Leaf,
				Key:        []byte{1, 2},
				Value:      []byte{3, 4},
				Dirty:      true,
				HashDigest: []byte{5},
				Encoding:   []byte{6},
			},
			settings: DeepCopySettings,
			expectedNode: &Node{
				Type:       Leaf,
				Key:        []byte{1, 2},
				Value:      []byte{3, 4},
				Dirty:      true,
				HashDigest: []byte{5},
				Encoding:   []byte{6},
			},
		},
		"empty branch": {
			node: &Node{
				Type:     Branch,
				Children: make([]*Node, ChildrenCapacity),
			},
			expectedNode: &Node{
				Type:     Branch,
				Children: make([]*Node, ChildrenCapacity),
			},
		},
		"non empty branch": {
			node: &Node{
				Type:  Branch,
				Key:   []byte{1, 2},
				Value: []byte{3, 4},
				Children: padRightChildren([]*Node{
					nil, nil, {
						Type: Leaf,
						Key:  []byte{9},
					},
				}),
				Dirty:      true,
				HashDigest: []byte{5},
				Encoding:   []byte{6},
			},
			settings: DefaultCopySettings,
			expectedNode: &Node{
				Type:  Branch,
				Key:   []byte{1, 2},
				Value: []byte{3, 4},
				Children: padRightChildren([]*Node{
					nil, nil, {
						Type: Leaf,
						Key:  []byte{9},
					},
				}),
				Dirty: true,
			},
		},
		"branch with children copied": {
			node: &Node{
				Type: Branch,
				Children: padRightChildren([]*Node{
					nil, nil, {
						Type: Leaf,
						Key:  []byte{9},
					},
				}),
			},
			settings: CopySettings{
				CopyChildren: true,
			},
			expectedNode: &Node{
				Type: Branch,
				Children: padRightChildren([]*Node{
					nil, nil, {
						Type: Leaf,
						Key:  []byte{9},
					},
				}),
			},
		},
		"deep copy branch": {
			node: &Node{
				Type:  Branch,
				Key:   []byte{1, 2},
				Value: []byte{3, 4},
				Children: padRightChildren([]*Node{
					nil, nil, {
						Type: Leaf,
						Key:  []byte{9},
					},
				}),
				Dirty:      true,
				HashDigest: []byte{5},
				Encoding:   []byte{6},
			},
			settings: DeepCopySettings,
			expectedNode: &Node{
				Type:  Branch,
				Key:   []byte{1, 2},
				Value: []byte{3, 4},
				Children: padRightChildren([]*Node{
					nil, nil, {
						Type: Leaf,
						Key:  []byte{9},
					},
				}),
				Dirty:      true,
				HashDigest: []byte{5},
				Encoding:   []byte{6},
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
			testForSliceModif(t, testCase.node.Value, nodeCopy.Value)
			testForSliceModif(t, testCase.node.HashDigest, nodeCopy.HashDigest)
			testForSliceModif(t, testCase.node.Encoding, nodeCopy.Encoding)

			if testCase.node.Children != nil { // branch
				testCase.node.Children[15] = &Node{Type: Leaf, Key: []byte("modified")}
				assert.NotEqual(t, nodeCopy.Children, testCase.node.Children)
			}
		})
	}
}
