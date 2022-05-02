// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package node

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_Node_ChildrenBitmap(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		node   Node
		bitmap uint16
	}{
		"no children": {
			node: Node{},
		},
		"index 0": {
			node: Node{
				Children: []*Node{
					{},
				},
			},
			bitmap: 1,
		},
		"index 0 and 4": {
			node: Node{
				Children: []*Node{
					{},
					nil, nil, nil,
					{},
				},
			},
			bitmap: 1<<4 + 1,
		},
		"index 0, 4 and 15": {
			node: Node{
				Children: []*Node{
					{},
					nil, nil, nil,
					{},
					nil, nil, nil, nil, nil,
					nil, nil, nil, nil, nil,
					{},
				},
			},
			bitmap: 1<<15 + 1<<4 + 1,
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			bitmap := testCase.node.ChildrenBitmap()

			assert.Equal(t, testCase.bitmap, bitmap)
		})
	}
}

func Test_Node_NumChildren(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		node  Node
		count int
	}{
		"zero": {
			node: Node{},
		},
		"one": {
			node: Node{
				Children: []*Node{
					{},
				},
			},
			count: 1,
		},
		"two": {
			node: Node{
				Children: []*Node{
					{},
					nil, nil, nil,
					{},
				},
			},
			count: 2,
		},
		"three": {
			node: Node{
				Children: []*Node{
					{},
					nil, nil, nil,
					{},
					nil, nil, nil, nil, nil,
					nil, nil, nil, nil, nil,
					{},
				},
			},
			count: 3,
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			count := testCase.node.NumChildren()

			assert.Equal(t, testCase.count, count)
		})
	}
}
