// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package branch

import (
	"testing"

	"github.com/ChainSafe/gossamer/lib/trie/encode"
	gomock "github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func Test_Branch_encodeHeader(t *testing.T) {
	testCases := map[string]struct {
		branch     *Branch
		writes     []writeCall
		errWrapped error
		errMessage string
	}{
		"no key": {
			branch: &Branch{},
			writes: []writeCall{
				{written: []byte{0x80}},
			},
		},
		"with value": {
			branch: &Branch{
				Value: []byte{},
			},
			writes: []writeCall{
				{written: []byte{0xc0}},
			},
		},
		"key of length 30": {
			branch: &Branch{
				Key: make([]byte, 30),
			},
			writes: []writeCall{
				{written: []byte{0x9e}},
			},
		},
		"key of length 62": {
			branch: &Branch{
				Key: make([]byte, 62),
			},
			writes: []writeCall{
				{written: []byte{0xbe}},
			},
		},
		"key of length 63": {
			branch: &Branch{
				Key: make([]byte, 63),
			},
			writes: []writeCall{
				{written: []byte{0xbf}},
				{written: []byte{0x0}},
			},
		},
		"key of length 64": {
			branch: &Branch{
				Key: make([]byte, 64),
			},
			writes: []writeCall{
				{written: []byte{0xbf}},
				{written: []byte{0x1}},
			},
		},
		"key too big": {
			branch: &Branch{
				Key: make([]byte, 65535+63),
			},
			writes: []writeCall{
				{written: []byte{0xbf}},
			},
			errWrapped: encode.ErrPartialKeyTooBig,
			errMessage: "partial key length cannot be larger than or equal to 2^16: 65535",
		},
		"small key length write error": {
			branch: &Branch{},
			writes: []writeCall{
				{
					written: []byte{0x80},
					err:     errTest,
				},
			},
			errWrapped: errTest,
			errMessage: "test error",
		},
		"long key length write error": {
			branch: &Branch{
				Key: make([]byte, 64),
			},
			writes: []writeCall{
				{
					written: []byte{0xbf},
					err:     errTest,
				},
			},
			errWrapped: errTest,
			errMessage: "test error",
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

			err := testCase.branch.encodeHeader(writer)

			assert.ErrorIs(t, err, testCase.errWrapped)
			if testCase.errWrapped != nil {
				assert.EqualError(t, err, testCase.errMessage)
			}
		})
	}
}
