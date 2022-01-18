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
						hashDigest: []byte{1, 2, 3, 4, 5},
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
						hashDigest: []byte{1, 2, 3, 4, 5},
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
