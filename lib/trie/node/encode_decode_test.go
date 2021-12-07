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
		branchToEncode *Branch
		branchDecoded  *Branch
	}{
		"empty branch": {
			branchToEncode: new(Branch),
			branchDecoded: &Branch{
				Key:   []byte{},
				Dirty: true,
			},
		},
		"branch with key 5": {
			branchToEncode: &Branch{
				Key: []byte{5},
			},
			branchDecoded: &Branch{
				Key:   []byte{5},
				Dirty: true,
			},
		},
		"branch with two bytes key": {
			branchToEncode: &Branch{
				Key: []byte{0xf, 0xa}, // note: each byte cannot be larger than 0xf
			},
			branchDecoded: &Branch{
				Key:   []byte{0xf, 0xa},
				Dirty: true,
			},
		},
		"branch with child": {
			branchToEncode: &Branch{
				Key: []byte{5},
				Children: [16]Node{
					&Leaf{
						Key:   []byte{9},
						Value: []byte{10},
					},
				},
			},
			branchDecoded: &Branch{
				Key: []byte{5},
				Children: [16]Node{
					&Leaf{
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

			oneBuffer := make([]byte, 1)
			_, err = buffer.Read(oneBuffer)
			require.NoError(t, err)
			header := oneBuffer[0]

			resultBranch, err := DecodeBranch(buffer, header)
			require.NoError(t, err)

			assert.Equal(t, testCase.branchDecoded, resultBranch)
		})
	}
}
