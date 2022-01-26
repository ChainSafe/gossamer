// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package trie

import (
	"testing"

	"github.com/ChainSafe/gossamer/internal/trie/node"
	"github.com/stretchr/testify/assert"
)

func Test_Trie_String(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		trie Trie
		s    string
	}{
		"empty trie": {
			s: "empty",
		},
		"leaf root": {
			trie: Trie{
				root: &node.Leaf{
					Key:        []byte{1, 2, 3},
					Value:      []byte{3, 4, 5},
					Generation: 1,
				},
			},
			s: `Leaf
├── Generation: 1
├── Dirty: false
├── Key: 0x010203
├── Value: 0x030405
├── Calculated encoding: nil
└── Calculated digest: nil`,
		},
		"branch root": {
			trie: Trie{
				root: &node.Branch{
					Key:   nil,
					Value: []byte{1, 2},
					Children: [16]node.Node{
						&node.Leaf{
							Key:        []byte{1, 2, 3},
							Value:      []byte{3, 4, 5},
							Generation: 2,
						},
						nil, nil,
						&node.Leaf{
							Key:        []byte{1, 2, 3},
							Value:      []byte{3, 4, 5},
							Generation: 3,
						},
					},
				},
			},
			s: `Branch
├── Generation: 0
├── Dirty: false
├── Key: nil
├── Value: 0x0102
├── Calculated encoding: nil
├── Calculated digest: nil
├── Child 0
|   └── Leaf
|       ├── Generation: 2
|       ├── Dirty: false
|       ├── Key: 0x010203
|       ├── Value: 0x030405
|       ├── Calculated encoding: nil
|       └── Calculated digest: nil
└── Child 3
    └── Leaf
        ├── Generation: 3
        ├── Dirty: false
        ├── Key: 0x010203
        ├── Value: 0x030405
        ├── Calculated encoding: nil
        └── Calculated digest: nil`,
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			s := testCase.trie.String()

			assert.Equal(t, testCase.s, s)
		})
	}
}
