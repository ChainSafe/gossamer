// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package node

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_Node_SetDirty(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		node     Node
		expected Node
	}{
		"not_dirty_to_dirty": {
			node: Node{
				MerkleValue: []byte{1},
			},
			expected: Node{Dirty: true},
		},
		"dirty_to_dirty": {
			node: Node{
				MerkleValue: []byte{1},
				Dirty:       true,
			},
			expected: Node{Dirty: true},
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			testCase.node.SetDirty()

			assert.Equal(t, testCase.expected, testCase.node)
		})
	}
}

func Test_Node_SetClean(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		node     Node
		expected Node
	}{
		"not_dirty_to_not_dirty": {
			node: Node{
				MerkleValue: []byte{1},
			},
			expected: Node{
				MerkleValue: []byte{1},
			},
		},
		"dirty_to_not_dirty": {
			node: Node{
				MerkleValue: []byte{1},
				Dirty:       true,
			},
			expected: Node{
				MerkleValue: []byte{1},
			},
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			testCase.node.SetClean()

			assert.Equal(t, testCase.expected, testCase.node)
		})
	}
}
