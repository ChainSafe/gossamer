// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package node

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_Node_StorageValueEqual(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		node     Node
		subValue []byte
		equal    bool
	}{
		"nil_node_subvalue_and_nil_subvalue": {
			equal: true,
		},
		"empty_node_subvalue_and_empty_subvalue": {
			node:     Node{StorageValue: []byte{}},
			subValue: []byte{},
			equal:    true,
		},
		"nil_node_subvalue_and_empty_subvalue": {
			subValue: []byte{},
		},
		"empty_node_subvalue_and_nil_subvalue": {
			node: Node{StorageValue: []byte{}},
		},
		"equal_non_empty_values": {
			node:     Node{StorageValue: []byte{1, 2}},
			subValue: []byte{1, 2},
			equal:    true,
		},
		"not_equal_non_empty_values": {
			node:     Node{StorageValue: []byte{1, 2}},
			subValue: []byte{1, 3},
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			node := testCase.node

			equal := node.StorageValueEqual(testCase.subValue)

			assert.Equal(t, testCase.equal, equal)
		})
	}
}
