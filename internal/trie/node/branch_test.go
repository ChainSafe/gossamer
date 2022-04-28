// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package node

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_Branch_String(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		branch *Branch
		s      string
	}{
		"empty branch": {
			branch: &Branch{},
			s: `Branch
├── Generation: 0
├── Dirty: false
├── Key: nil
├── Value: nil
├── Descendants: 0
├── Calculated encoding: nil
└── Calculated digest: nil`,
		},
		"branch with value smaller than 1024": {
			branch: &Branch{
				Key:         []byte{1, 2},
				Value:       []byte{3, 4},
				Dirty:       true,
				Descendants: 3,
				Children: [16]Node{
					nil, nil, nil,
					&Leaf{},
					nil, nil, nil,
					&Branch{},
					nil, nil, nil,
					&Leaf{},
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
|       ├── Descendants: 0
|       ├── Calculated encoding: nil
|       └── Calculated digest: nil
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
			branch: &Branch{
				Key:         []byte{1, 2},
				Value:       make([]byte, 1025),
				Dirty:       true,
				Descendants: 3,
				Children: [16]Node{
					nil, nil, nil,
					&Leaf{},
					nil, nil, nil,
					&Branch{},
					nil, nil, nil,
					&Leaf{},
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
|       ├── Descendants: 0
|       ├── Calculated encoding: nil
|       └── Calculated digest: nil
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

			s := testCase.branch.String()

			assert.Equal(t, testCase.s, s)
		})
	}
}
