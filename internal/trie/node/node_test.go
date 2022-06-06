// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package node

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_Node_String(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		node *Node
		s    string
	}{
		"empty leaf": {
			node: &Node{},
			s: `Leaf
├── Generation: 0
├── Dirty: false
├── Key: nil
├── Value: nil
├── Calculated encoding: nil
└── Calculated digest: nil`,
		},
		"leaf with value smaller than 1024": {
			node: &Node{
				Key:   []byte{1, 2},
				Value: []byte{3, 4},
				Dirty: true,
			},
			s: `Leaf
├── Generation: 0
├── Dirty: true
├── Key: 0x0102
├── Value: 0x0304
├── Calculated encoding: nil
└── Calculated digest: nil`,
		},
		"leaf with value higher than 1024": {
			node: &Node{
				Key:   []byte{1, 2},
				Value: make([]byte, 1025),
				Dirty: true,
			},
			s: `Leaf
├── Generation: 0
├── Dirty: true
├── Key: 0x0102
├── Value: 0x0000000000000000...0000000000000000
├── Calculated encoding: nil
└── Calculated digest: nil`,
		},
		"branch with value smaller than 1024": {
			node: &Node{
				Key:         []byte{1, 2},
				Value:       []byte{3, 4},
				Dirty:       true,
				Descendants: 3,
				Children: []*Node{
					nil, nil, nil,
					{},
					nil, nil, nil,
					{
						Descendants: 1,
						Children:    padRightChildren([]*Node{{}}),
					},
					nil, nil, nil,
					{},
					nil, nil, nil, nil,
				},
			},
			s: `Branch
├── Generation: 0
├── Dirty: true
├── Key: 0x0102
├── Value: 0x0304
├── Descendants: 3
├── Calculated encoding: nil
├── Calculated digest: nil
├── Child 3
|   └── Leaf
|       ├── Generation: 0
|       ├── Dirty: false
|       ├── Key: nil
|       ├── Value: nil
|       ├── Calculated encoding: nil
|       └── Calculated digest: nil
├── Child 7
|   └── Branch
|       ├── Generation: 0
|       ├── Dirty: false
|       ├── Key: nil
|       ├── Value: nil
|       ├── Descendants: 1
|       ├── Calculated encoding: nil
|       ├── Calculated digest: nil
|       └── Child 0
|           └── Leaf
|               ├── Generation: 0
|               ├── Dirty: false
|               ├── Key: nil
|               ├── Value: nil
|               ├── Calculated encoding: nil
|               └── Calculated digest: nil
└── Child 11
    └── Leaf
        ├── Generation: 0
        ├── Dirty: false
        ├── Key: nil
        ├── Value: nil
        ├── Calculated encoding: nil
        └── Calculated digest: nil`,
		},
		"branch with value higher than 1024": {
			node: &Node{
				Key:         []byte{1, 2},
				Value:       make([]byte, 1025),
				Dirty:       true,
				Descendants: 3,
				Children: []*Node{
					nil, nil, nil,
					{},
					nil, nil, nil,
					{
						Descendants: 1,
						Children:    padRightChildren([]*Node{{}}),
					},
					nil, nil, nil,
					{},
					nil, nil, nil, nil,
				},
			},
			s: `Branch
├── Generation: 0
├── Dirty: true
├── Key: 0x0102
├── Value: 0x0000000000000000...0000000000000000
├── Descendants: 3
├── Calculated encoding: nil
├── Calculated digest: nil
├── Child 3
|   └── Leaf
|       ├── Generation: 0
|       ├── Dirty: false
|       ├── Key: nil
|       ├── Value: nil
|       ├── Calculated encoding: nil
|       └── Calculated digest: nil
├── Child 7
|   └── Branch
|       ├── Generation: 0
|       ├── Dirty: false
|       ├── Key: nil
|       ├── Value: nil
|       ├── Descendants: 1
|       ├── Calculated encoding: nil
|       ├── Calculated digest: nil
|       └── Child 0
|           └── Leaf
|               ├── Generation: 0
|               ├── Dirty: false
|               ├── Key: nil
|               ├── Value: nil
|               ├── Calculated encoding: nil
|               └── Calculated digest: nil
└── Child 11
    └── Leaf
        ├── Generation: 0
        ├── Dirty: false
        ├── Key: nil
        ├── Value: nil
        ├── Calculated encoding: nil
        └── Calculated digest: nil`,
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			s := testCase.node.String()

			assert.Equal(t, testCase.s, s)
		})
	}
}
