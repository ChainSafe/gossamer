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
		"no_children": {
			node: Node{},
		},
		"index_0": {
			node: Node{
				Children: []*Node{
					{},
				},
			},
			bitmap: 1,
		},
		"index_0_and_4": {
			node: Node{
				Children: []*Node{
					{},
					nil, nil, nil,
					{},
				},
			},
			bitmap: 1<<4 + 1,
		},
		"index_0,_4_and_15": {
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

func Test_Node_HasChild(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		node Node
		has  bool
	}{
		"no_child": {},
		"one_child_at_index_0": {
			node: Node{
				Children: []*Node{
					{},
				},
			},
			has: true,
		},
		"one_child_at_index_1": {
			node: Node{
				Children: []*Node{
					nil,
					{},
				},
			},
			has: true,
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			has := testCase.node.HasChild()

			assert.Equal(t, testCase.has, has)
		})
	}
}
