// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package leaf

import (
	"testing"

	"github.com/ChainSafe/gossamer/lib/trie/encode"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func Test_Leaf_encodeHeader(t *testing.T) {
	testCases := map[string]struct {
		leaf       *Leaf
		writes     []writeCall
		errWrapped error
		errMessage string
	}{
		"no key": {
			leaf: &Leaf{},
			writes: []writeCall{
				{written: []byte{0x40}},
			},
		},
		"key of length 30": {
			leaf: &Leaf{
				Key: make([]byte, 30),
			},
			writes: []writeCall{
				{written: []byte{0x5e}},
			},
		},
		"short key write error": {
			leaf: &Leaf{
				Key: make([]byte, 30),
			},
			writes: []writeCall{
				{
					written: []byte{0x5e},
					err:     errTest,
				},
			},
			errWrapped: errTest,
			errMessage: errTest.Error(),
		},
		"key of length 62": {
			leaf: &Leaf{
				Key: make([]byte, 62),
			},
			writes: []writeCall{
				{written: []byte{0x7e}},
			},
		},
		"key of length 63": {
			leaf: &Leaf{
				Key: make([]byte, 63),
			},
			writes: []writeCall{
				{written: []byte{0x7f}},
				{written: []byte{0x0}},
			},
		},
		"key of length 64": {
			leaf: &Leaf{
				Key: make([]byte, 64),
			},
			writes: []writeCall{
				{written: []byte{0x7f}},
				{written: []byte{0x1}},
			},
		},
		"long key first byte write error": {
			leaf: &Leaf{
				Key: make([]byte, 63),
			},
			writes: []writeCall{
				{
					written: []byte{0x7f},
					err:     errTest,
				},
			},
			errWrapped: errTest,
			errMessage: errTest.Error(),
		},
		"key too big": {
			leaf: &Leaf{
				Key: make([]byte, 65535+63),
			},
			writes: []writeCall{
				{written: []byte{0x7f}},
			},
			errWrapped: encode.ErrPartialKeyTooBig,
			errMessage: "partial key length cannot be larger than or equal to 2^16: 65535",
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)

			writer := NewMockWriter(ctrl)
			var previousCall *gomock.Call
			for _, write := range testCase.writes {
				call := writer.EXPECT().
					Write(write.written).
					Return(write.n, write.err)

				if previousCall != nil {
					call.After(previousCall)
				}
				previousCall = call
			}

			err := testCase.leaf.encodeHeader(writer)

			assert.ErrorIs(t, err, testCase.errWrapped)
			if testCase.errWrapped != nil {
				assert.EqualError(t, err, testCase.errMessage)
			}
		})
	}
}
