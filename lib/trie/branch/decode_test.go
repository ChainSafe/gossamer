// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package branch

import (
	"bytes"
	"io"
	"testing"

	"github.com/ChainSafe/gossamer/lib/trie/decode"
	"github.com/ChainSafe/gossamer/lib/trie/leaf"
	"github.com/ChainSafe/gossamer/lib/trie/node"
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
		branch     *Branch
		errWrapped error
		errMessage string
	}{
		"no data with header 0": {
			reader:     bytes.NewBuffer(nil),
			errWrapped: ErrReadHeaderByte,
			errMessage: "cannot read header byte: EOF",
		},
		"no data with header 1": {
			reader:     bytes.NewBuffer(nil),
			header:     1,
			errWrapped: ErrNodeTypeIsNotABranch,
			errMessage: "node type is not a branch: 0",
		},
		"first byte as 0 header 0": {
			reader:     bytes.NewBuffer([]byte{0}),
			errWrapped: ErrNodeTypeIsNotABranch,
			errMessage: "node type is not a branch: 0",
		},
		"key decoding error": {
			reader: bytes.NewBuffer([]byte{
				129, // node type 2 and key length 1
				// missing key data byte
			}),
			errWrapped: decode.ErrReadKeyData,
			errMessage: "cannot decode key: cannot read key data: EOF",
		},
		"children bitmap read error": {
			reader: bytes.NewBuffer([]byte{
				129, // node type 2 and key length 1
				9,   // key data
				// missing children bitmap 2 bytes
			}),
			errWrapped: ErrReadChildrenBitmap,
			errMessage: "cannot read children bitmap: EOF",
		},
		"children decoding error": {
			reader: bytes.NewBuffer([]byte{
				129,  // node type 2 and key length 1
				9,    // key data
				0, 4, // children bitmap
				// missing children scale encoded data
			}),
			errWrapped: ErrDecodeChildHash,
			errMessage: "cannot decode child hash: at index 10: EOF",
		},
		"success node type 2": {
			reader: bytes.NewBuffer(
				concatByteSlices([][]byte{
					{
						129,  // node type 2 and key length 1
						9,    // key data
						0, 4, // children bitmap
					},
					scaleEncodeBytes(t, 1, 2, 3, 4, 5), // child hash
				}),
			),
			branch: &Branch{
				Key: []byte{9},
				Children: [16]node.Node{
					nil, nil, nil, nil, nil,
					nil, nil, nil, nil, nil,
					&leaf.Leaf{
						Hash: []byte{1, 2, 3, 4, 5},
					},
				},
				Dirty: true,
			},
		},
		"value decoding error for node type 3": {
			reader: bytes.NewBuffer(
				concatByteSlices([][]byte{
					{
						193, // node type 3 and key length 1
						9,   // key data
					},
					{0, 4}, // children bitmap
					// missing encoded branch value
				}),
			),
			errWrapped: ErrDecodeValue,
			errMessage: "cannot decode value: EOF",
		},
		"success node type 3": {
			reader: bytes.NewBuffer(
				concatByteSlices([][]byte{
					{
						193, // node type 3 and key length 1
						9,   // key data
					},
					{0, 4},                             // children bitmap
					scaleEncodeBytes(t, 7, 8, 9),       // branch value
					scaleEncodeBytes(t, 1, 2, 3, 4, 5), // child hash
				}),
			),
			branch: &Branch{
				Key:   []byte{9},
				Value: []byte{7, 8, 9},
				Children: [16]node.Node{
					nil, nil, nil, nil, nil,
					nil, nil, nil, nil, nil,
					&leaf.Leaf{
						Hash: []byte{1, 2, 3, 4, 5},
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

			branch, err := Decode(testCase.reader, testCase.header)

			assert.ErrorIs(t, err, testCase.errWrapped)
			if err != nil {
				assert.EqualError(t, err, testCase.errMessage)
			}
			assert.Equal(t, testCase.branch, branch)
		})
	}
}
