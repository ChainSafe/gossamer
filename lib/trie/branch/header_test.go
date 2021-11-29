// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package branch

import (
	"testing"

	"github.com/ChainSafe/gossamer/lib/trie/encode"
	"github.com/stretchr/testify/assert"
)

func Test_Branch_Header(t *testing.T) {
	testCases := map[string]struct {
		branch     *Branch
		encoding   []byte
		wrappedErr error
		errMessage string
	}{
		"no key": {
			branch:   &Branch{},
			encoding: []byte{0x80},
		},
		"with value": {
			branch: &Branch{
				Value: []byte{},
			},
			encoding: []byte{0xc0},
		},
		"key of length 30": {
			branch: &Branch{
				Key: make([]byte, 30),
			},
			encoding: []byte{0x9e},
		},
		"key of length 62": {
			branch: &Branch{
				Key: make([]byte, 62),
			},
			encoding: []byte{0xbe},
		},
		"key of length 63": {
			branch: &Branch{
				Key: make([]byte, 63),
			},
			encoding: []byte{0xbf, 0x0},
		},
		"key of length 64": {
			branch: &Branch{
				Key: make([]byte, 64),
			},
			encoding: []byte{0xbf, 0x1},
		},
		"key too big": {
			branch: &Branch{
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

			encoding, err := testCase.branch.Header()

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
