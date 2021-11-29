// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package leaf

import (
	"bytes"
	"io"
	"testing"

	"github.com/ChainSafe/gossamer/lib/trie/decode"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func scaleEncodeBytes(t *testing.T, b ...byte) (encoded []byte) {
	encoded, err := scale.Marshal(b)
	require.NoError(t, err)
	return encoded
}

func concatByteSlices(slices [][]byte) (concatenated []byte) {
	length := 0
	for i := range slices {
		length += len(slices[i])
	}
	concatenated = make([]byte, 0, length)
	for _, slice := range slices {
		concatenated = append(concatenated, slice...)
	}
	return concatenated
}

func Test_Decode(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		reader     io.Reader
		header     byte
		leaf       *Leaf
		errWrapped error
		errMessage string
	}{
		"no data with header 1": {
			reader:     bytes.NewBuffer(nil),
			header:     1,
			errWrapped: ErrNodeTypeIsNotALeaf,
			errMessage: "node type is not a leaf: 0",
		},
		"key decoding error": {
			reader: bytes.NewBuffer([]byte{
				// missing key data byte
			}),
			header:     65, // node type 1 and key length 1
			errWrapped: decode.ErrReadKeyData,
			errMessage: "cannot decode key: cannot read key data: EOF",
		},
		"value decoding error": {
			reader: bytes.NewBuffer([]byte{
				9, // key data
				// missing value data
			}),
			header:     65, // node type 1 and key length 1
			errWrapped: ErrDecodeValue,
			errMessage: "cannot decode value: EOF",
		},
		"zero value": {
			reader: bytes.NewBuffer([]byte{
				9, // key data
				0, // missing value data
			}),
			header: 65, // node type 1 and key length 1
			leaf: &Leaf{
				Key:   []byte{9},
				Dirty: true,
			},
		},
		"success": {
			reader: bytes.NewBuffer(
				concatByteSlices([][]byte{
					{9},                                // key data
					scaleEncodeBytes(t, 1, 2, 3, 4, 5), // value data
				}),
			),
			header: 65, // node type 1 and key length 1
			leaf: &Leaf{
				Key:   []byte{9},
				Value: []byte{1, 2, 3, 4, 5},
				Dirty: true,
			},
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			leaf, err := Decode(testCase.reader, testCase.header)

			assert.ErrorIs(t, err, testCase.errWrapped)
			if err != nil {
				assert.EqualError(t, err, testCase.errMessage)
			}
			assert.Equal(t, testCase.leaf, leaf)
		})
	}
}
