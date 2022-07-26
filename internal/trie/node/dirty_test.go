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
		"not dirty to dirty": {
			node: Node{
				Encoding:   []byte{1},
				HashDigest: []byte{1},
			},
			expected: Node{Dirty: true},
		},
		"dirty to dirty": {
			node: Node{
				Encoding:   []byte{1},
				HashDigest: []byte{1},
				Dirty:      true,
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
		"not dirty to not dirty": {
			node: Node{
				Encoding:   []byte{1},
				HashDigest: []byte{1},
			},
			expected: Node{
				Encoding:   []byte{1},
				HashDigest: []byte{1},
			},
		},
		"dirty to not dirty": {
			node: Node{
				Encoding:   []byte{1},
				HashDigest: []byte{1},
				Dirty:      true,
			},
			expected: Node{
				Encoding:   []byte{1},
				HashDigest: []byte{1},
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
