// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package encodedecode_test

import (
	"bytes"
	"testing"

	"github.com/ChainSafe/gossamer/lib/trie/branch"
	"github.com/ChainSafe/gossamer/lib/trie/leaf"
	"github.com/ChainSafe/gossamer/lib/trie/node"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_Branch_Encode_Decode(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		branchToEncode *branch.Branch
		branchDecoded  *branch.Branch
	}{
		"empty branch": {
			branchToEncode: new(branch.Branch),
			branchDecoded: &branch.Branch{
				Key:   []byte{},
				Dirty: true,
			},
		},
		"branch with key 5": {
			branchToEncode: &branch.Branch{
				Key: []byte{5},
			},
			branchDecoded: &branch.Branch{
				Key:   []byte{5},
				Dirty: true,
			},
		},
		"branch with two bytes key": {
			branchToEncode: &branch.Branch{
				Key: []byte{0xf, 0xa}, // note: each byte cannot be larger than 0xf
			},
			branchDecoded: &branch.Branch{
				Key:   []byte{0xf, 0xa},
				Dirty: true,
			},
		},
		"branch with child": {
			branchToEncode: &branch.Branch{
				Key: []byte{5},
				Children: [16]node.Node{
					&leaf.Leaf{
						Key:   []byte{9},
						Value: []byte{10},
					},
				},
			},
			branchDecoded: &branch.Branch{
				Key: []byte{5},
				Children: [16]node.Node{
					&leaf.Leaf{
						Hash: []byte{0x41, 0x9, 0x4, 0xa},
					},
				},
				Dirty: true,
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

			const header = 0
			resultBranch, err := branch.Decode(buffer, header)
			require.NoError(t, err)

			assert.Equal(t, testCase.branchDecoded, resultBranch)
		})
	}
}
