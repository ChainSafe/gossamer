// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package node

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_Branch_Encode_Decode(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		branchToEncode *Node
		branchDecoded  *Node
	}{
		"empty_branch": {
			branchToEncode: &Node{
				Children: make([]*Node, ChildrenCapacity),
			},
			branchDecoded: &Node{
				PartialKey: []byte{},
				Children:   make([]*Node, ChildrenCapacity),
			},
		},
		"branch_with_key_5": {
			branchToEncode: &Node{
				Children:   make([]*Node, ChildrenCapacity),
				PartialKey: []byte{5},
			},
			branchDecoded: &Node{
				PartialKey: []byte{5},
				Children:   make([]*Node, ChildrenCapacity),
			},
		},
		"branch_with_two_bytes_key": {
			branchToEncode: &Node{
				PartialKey: []byte{0xf, 0xa}, // note: each byte cannot be larger than 0xf
				Children:   make([]*Node, ChildrenCapacity),
			},
			branchDecoded: &Node{
				PartialKey: []byte{0xf, 0xa},
				Children:   make([]*Node, ChildrenCapacity),
			},
		},
		"branch_with_child_leaf_inline": {
			branchToEncode: &Node{
				PartialKey: []byte{5},
				Children: padRightChildren([]*Node{
					{
						PartialKey:   []byte{9},
						StorageValue: []byte{10},
					},
				}),
			},
			branchDecoded: &Node{
				PartialKey:  []byte{5},
				Descendants: 1,
				Children: padRightChildren([]*Node{
					{
						PartialKey:   []byte{9},
						StorageValue: []byte{10},
					},
				}),
			},
		},
		"branch_with_child_leaf_hash": {
			branchToEncode: &Node{
				PartialKey: []byte{5},
				Children: padRightChildren([]*Node{
					{
						PartialKey: []byte{
							10, 11, 12, 13,
							14, 15, 16, 17,
							18, 19, 20, 21,
							14, 15, 16, 17,
							10, 11, 12, 13,
							14, 15, 16, 17,
						},
						StorageValue: []byte{
							10, 11, 12, 13,
							14, 15, 16, 17,
							10, 11, 12, 13,
							14, 15, 16, 17,
							10, 11, 12, 13,
						},
					},
				}),
			},
			branchDecoded: &Node{
				PartialKey: []byte{5},
				Children: padRightChildren([]*Node{
					{
						MerkleValue: []byte{
							2, 18, 48, 30, 98,
							133, 244, 78, 70,
							161, 196, 105, 228,
							190, 159, 228, 199, 29,
							254, 212, 160, 55, 199,
							21, 186, 226, 204, 145,
							132, 5, 39, 204,
						},
					},
				}),
				Descendants: 1,
			},
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			buffer := bytes.NewBuffer(nil)

			err := testCase.branchToEncode.Encode(buffer, NoMaxInlineValueSize)
			require.NoError(t, err)

			nodeVariant, partialKeyLength, err := decodeHeader(buffer)
			require.NoError(t, err)

			resultBranch, err := decodeBranch(buffer, nodeVariant, partialKeyLength)
			require.NoError(t, err)

			assert.Equal(t, testCase.branchDecoded, resultBranch)
		})
	}
}
