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
		"empty branch": {
			branchToEncode: &Node{
				Children: make([]*Node, ChildrenCapacity),
			},
			branchDecoded: &Node{
				Key:      []byte{},
				Children: make([]*Node, ChildrenCapacity),
				Dirty:    true,
			},
		},
		"branch with key 5": {
			branchToEncode: &Node{
				Children: make([]*Node, ChildrenCapacity),
				Key:      []byte{5},
			},
			branchDecoded: &Node{
				Key:      []byte{5},
				Children: make([]*Node, ChildrenCapacity),
				Dirty:    true,
			},
		},
		"branch with two bytes key": {
			branchToEncode: &Node{
				Key:      []byte{0xf, 0xa}, // note: each byte cannot be larger than 0xf
				Children: make([]*Node, ChildrenCapacity),
			},
			branchDecoded: &Node{
				Key:      []byte{0xf, 0xa},
				Children: make([]*Node, ChildrenCapacity),
				Dirty:    true,
			},
		},
		"branch with child leaf inline": {
			branchToEncode: &Node{
				Key: []byte{5},
				Children: padRightChildren([]*Node{
					{
						Key:   []byte{9},
						Value: []byte{10},
					},
				}),
			},
			branchDecoded: &Node{
				Key:         []byte{5},
				Descendants: 1,
				Children: padRightChildren([]*Node{
					{
						Key:   []byte{9},
						Value: []byte{10},
						Dirty: true,
					},
				}),
				Dirty: true,
			},
		},
		"branch with child leaf hash": {
			branchToEncode: &Node{
				Key: []byte{5},
				Children: padRightChildren([]*Node{
					{
						Key: []byte{
							10, 11, 12, 13,
							14, 15, 16, 17,
							18, 19, 20, 21,
							14, 15, 16, 17,
							10, 11, 12, 13,
							14, 15, 16, 17,
						},
						Value: []byte{
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
				Key: []byte{5},
				Children: padRightChildren([]*Node{
					{
						HashDigest: []byte{
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
				Dirty:       true,
				Descendants: 1,
			},
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			buffer := bytes.NewBuffer(nil)

			err := testCase.branchToEncode.Encode(buffer)
			require.NoError(t, err)

			oneBuffer := make([]byte, 1)
			_, err = buffer.Read(oneBuffer)
			require.NoError(t, err)
			header := oneBuffer[0]

			resultBranch, err := decodeBranch(buffer, header)
			require.NoError(t, err)

			assert.Equal(t, testCase.branchDecoded, resultBranch)
		})
	}
}
