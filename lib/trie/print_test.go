// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package trie

import (
	"testing"

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
				root: &Node{
					PartialKey:   []byte{1, 2, 3},
					StorageValue: []byte{3, 4, 5},
					Generation:   1,
				},
			},
			s: `Leaf
├── Generation: 1
├── Dirty: false
├── Key: 0x010203
├── Storage value: 0x030405
└── Merkle value: nil`,
		},
		"branch root": {
			trie: Trie{
				root: &Node{
					PartialKey:   nil,
					StorageValue: []byte{1, 2},
					Descendants:  2,
					Children: []*Node{
						{
							PartialKey:   []byte{1, 2, 3},
							StorageValue: []byte{3, 4, 5},
							Generation:   2,
						},
						nil, nil,
						{
							PartialKey:   []byte{1, 2, 3},
							StorageValue: []byte{3, 4, 5},
							Generation:   3,
						},
					},
				},
			},
			s: `Branch
├── Generation: 0
├── Dirty: false
├── Key: nil
├── Storage value: 0x0102
├── Descendants: 2
├── Merkle value: nil
├── Child 0
|   └── Leaf
|       ├── Generation: 2
|       ├── Dirty: false
|       ├── Key: 0x010203
|       ├── Storage value: 0x030405
|       └── Merkle value: nil
└── Child 3
    └── Leaf
        ├── Generation: 3
        ├── Dirty: false
        ├── Key: 0x010203
        ├── Storage value: 0x030405
        └── Merkle value: nil`,
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
