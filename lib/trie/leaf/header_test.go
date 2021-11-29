// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package leaf

import (
	"testing"

	"github.com/ChainSafe/gossamer/lib/trie/encode"
	"github.com/stretchr/testify/assert"
)

func Test_Leaf_Header(t *testing.T) {
	testCases := map[string]struct {
		leaf       *Leaf
		encoding   []byte
		wrappedErr error
		errMessage string
	}{
		"no key": {
			leaf:     &Leaf{},
			encoding: []byte{0x40},
		},
		"key of length 30": {
			leaf: &Leaf{
				Key: make([]byte, 30),
			},
			encoding: []byte{0x5e},
		},
		"key of length 62": {
			leaf: &Leaf{
				Key: make([]byte, 62),
			},
			encoding: []byte{0x7e},
		},
		"key of length 63": {
			leaf: &Leaf{
				Key: make([]byte, 63),
			},
			encoding: []byte{0x7f, 0x0},
		},
		"key of length 64": {
			leaf: &Leaf{
				Key: make([]byte, 64),
			},
			encoding: []byte{0x7f, 0x1},
		},
		"key too big": {
			leaf: &Leaf{
				Key: make([]byte, 65535+63),
			},
			wrappedErr: encode.ErrPartialKeyTooBig,
			errMessage: "partial key length cannot be larger than or equal to 2^16: 65535",
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			encoding, err := testCase.leaf.Header()

			if testCase.wrappedErr != nil {
				assert.ErrorIs(t, err, testCase.wrappedErr)
				assert.EqualError(t, err, testCase.errMessage)
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, testCase.encoding, encoding)
		})
	}
}
