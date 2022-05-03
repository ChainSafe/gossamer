// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package node

import (
	"bytes"
	"io"
	"testing"

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
		n          Node
		errWrapped error
		errMessage string
	}{
		"no data": {
			reader:     bytes.NewReader(nil),
			errWrapped: ErrReadHeaderByte,
			errMessage: "cannot read header byte: EOF",
		},
		"unknown node type": {
			reader:     bytes.NewReader([]byte{0}),
			errWrapped: ErrUnknownNodeType,
			errMessage: "unknown node type: 0",
		},
		"leaf decoding error": {
			reader: bytes.NewReader([]byte{
				65, // node type 1 (leaf) and key length 1
				// missing key data byte
			}),
			errWrapped: ErrReadKeyData,
			errMessage: "cannot decode leaf: cannot decode key: cannot read key data: EOF",
		},
		"leaf success": {
			reader: bytes.NewReader(
				append(
					[]byte{
						65, // node type 1 (leaf) and key length 1
						9,  // key data
					},
					scaleEncodeBytes(t, 1, 2, 3)...,
				),
			),
			n: &Leaf{
				Key:   []byte{9},
				Value: []byte{1, 2, 3},
				Dirty: true,
			},
		},
		"branch decoding error": {
			reader: bytes.NewReader([]byte{
				129, // node type 2 (branch without value) and key length 1
				// missing key data byte
			}),
			errWrapped: ErrReadKeyData,
			errMessage: "cannot decode branch: cannot decode key: cannot read key data: EOF",
		},
		"branch success": {
			reader: bytes.NewReader(
				[]byte{
					129,  // node type 2 (branch without value) and key length 1
					9,    // key data
					0, 0, // no children bitmap
				},
			),
			n: &Branch{
				Key:   []byte{9},
				Dirty: true,
			},
		},
		"branch with two inlined children": {
			reader: bytes.NewReader(
				[]byte{
					158, // node type 2 (branch w/o value) and key length 30
					// Key data start
					195, 101, 195, 207, 89, 214,
					113, 235, 114, 218, 14, 122,
					65, 19, 196, 16, 2, 80, 95,
					14, 123, 144, 18, 9, 107,
					65, 196, 235, 58, 175,
					// Key data end
					148, 127, 110, 164, 41, 8, 0, 0, 104, 95, 15, 31, 5,
					21, 244, 98, 205, 207, 132, 224, 241, 214, 4, 93, 252,
					187, 32, 134, 92, 74, 43, 127, 1, 0, 0,
				},
			),
			n: &Branch{
				Key: []byte{
					12, 3, 6, 5, 12, 3,
					12, 15, 5, 9, 13, 6,
					7, 1, 14, 11, 7, 2,
					13, 10, 0, 14, 7, 10,
					4, 1, 1, 3, 12, 4,
				},
				Children: [16]Node{
					nil, nil, nil, nil,
					&Leaf{
						Key: []byte{
							14, 7, 11, 9, 0, 1,
							2, 0, 9, 6, 11, 4,
							1, 12, 4, 14, 11,
							3, 10, 10, 15, 9,
							4, 7, 15, 6, 14,
							10, 4, 2, 9,
						},
						Value: []byte{0, 0},
						Dirty: true,
					},
					nil, nil, nil, nil,
					&Leaf{
						Key: []byte{
							15, 1, 15, 0, 5, 1,
							5, 15, 4, 6, 2, 12,
							13, 12, 15, 8, 4,
							14, 0, 15, 1, 13,
							6, 0, 4, 5, 13,
							15, 12, 11, 11,
						},
						Value: []byte{
							134, 92, 74, 43,
							127, 1, 0, 0,
						},
						Dirty: true,
					},
					nil, nil, nil, nil, nil, nil,
				},
				Dirty: true,
			},
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			n, err := Decode(testCase.reader)

			assert.ErrorIs(t, err, testCase.errWrapped)
			if err != nil {
				assert.EqualError(t, err, testCase.errMessage)
			}
			assert.Equal(t, testCase.n, n)
		})
	}
}

func Test_decodeBranch(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		reader     io.Reader
		header     byte
		branch     *Branch
		errWrapped error
		errMessage string
	}{
		"no data with header 1": {
			reader:     bytes.NewBuffer(nil),
			header:     65,
			errWrapped: ErrNodeTypeIsNotABranch,
			errMessage: "node type is not a branch: 1",
		},
		"key decoding error": {
			reader: bytes.NewBuffer([]byte{
				// missing key data byte
			}),
			header:     129, // node type 2 (branch without value) and key length 1
			errWrapped: ErrReadKeyData,
			errMessage: "cannot decode key: cannot read key data: EOF",
		},
		"children bitmap read error": {
			reader: bytes.NewBuffer([]byte{
				9, // key data
				// missing children bitmap 2 bytes
			}),
			header:     129, // node type 2 (branch without value) and key length 1
			errWrapped: ErrReadChildrenBitmap,
			errMessage: "cannot read children bitmap: EOF",
		},
		"children decoding error": {
			reader: bytes.NewBuffer([]byte{
				9,    // key data
				0, 4, // children bitmap
				// missing children scale encoded data
			}),
			header:     129, // node type 2 (branch without value) and key length 1
			errWrapped: ErrDecodeChildHash,
			errMessage: "cannot decode child hash: at index 10: EOF",
		},
		"success node type 2": {
			reader: bytes.NewBuffer(
				concatByteSlices([][]byte{
					{
						9,    // key data
						0, 4, // children bitmap
					},
					scaleEncodeBytes(t, 1, 2, 3, 4, 5), // child hash
				}),
			),
			header: 129, // node type 2 (branch without value) and key length 1
			branch: &Branch{
				Key: []byte{9},
				Children: [16]Node{
					nil, nil, nil, nil, nil,
					nil, nil, nil, nil, nil,
					&Leaf{
						HashDigest: []byte{1, 2, 3, 4, 5},
					},
				},
				Dirty: true,
			},
		},
		"value decoding error for node type 3": {
			reader: bytes.NewBuffer(
				concatByteSlices([][]byte{
					{9},    // key data
					{0, 4}, // children bitmap
					// missing encoded branch value
				}),
			),
			header:     193, // node type 3 (branch with value) and key length 1
			errWrapped: ErrDecodeValue,
			errMessage: "cannot decode value: EOF",
		},
		"success node type 3": {
			reader: bytes.NewBuffer(
				concatByteSlices([][]byte{
					{9},                                // key data
					{0, 4},                             // children bitmap
					scaleEncodeBytes(t, 7, 8, 9),       // branch value
					scaleEncodeBytes(t, 1, 2, 3, 4, 5), // child hash
				}),
			),
			header: 193, // node type 3 (branch with value) and key length 1
			branch: &Branch{
				Key:   []byte{9},
				Value: []byte{7, 8, 9},
				Children: [16]Node{
					nil, nil, nil, nil, nil,
					nil, nil, nil, nil, nil,
					&Leaf{
						HashDigest: []byte{1, 2, 3, 4, 5},
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

			branch, err := decodeBranch(testCase.reader, testCase.header)

			assert.ErrorIs(t, err, testCase.errWrapped)
			if err != nil {
				assert.EqualError(t, err, testCase.errMessage)
			}
			assert.Equal(t, testCase.branch, branch)
		})
	}
}

func Test_decodeLeaf(t *testing.T) {
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
			header:     65, // node type 1 (leaf) and key length 1
			errWrapped: ErrReadKeyData,
			errMessage: "cannot decode key: cannot read key data: EOF",
		},
		"value decoding error": {
			reader: bytes.NewBuffer([]byte{
				9,        // key data
				255, 255, // bad value data
			}),
			header:     65, // node type 1 (leaf) and key length 1
			errWrapped: ErrDecodeValue,
			errMessage: "cannot decode value: could not decode invalid integer",
		},
		"zero value": {
			reader: bytes.NewBuffer([]byte{
				9, // key data
				// missing value data
			}),
			header: 65, // node type 1 (leaf) and key length 1
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
			header: 65, // node type 1 (leaf) and key length 1
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

			leaf, err := decodeLeaf(testCase.reader, testCase.header)

			assert.ErrorIs(t, err, testCase.errWrapped)
			if err != nil {
				assert.EqualError(t, err, testCase.errMessage)
			}
			assert.Equal(t, testCase.leaf, leaf)
		})
	}
}
