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
		n          *Node
		errWrapped error
		errMessage string
	}{
		"no data": {
			reader:     bytes.NewReader(nil),
			errWrapped: io.EOF,
			errMessage: "cannot decode header: cannot read header byte: EOF",
		},
		"unknown node variant": {
			reader:     bytes.NewReader([]byte{0}),
			errWrapped: ErrVariantUnknown,
			errMessage: "cannot decode header: cannot parse header byte: node variant is unknown: for header byte 00000000",
		},
		"leaf decoding error": {
			reader: bytes.NewReader([]byte{
				leafVariant.bits | 1, // key length 1
				// missing key data byte
			}),
			errWrapped: io.EOF,
			errMessage: "cannot decode leaf: cannot decode key: " +
				"cannot read from reader: EOF",
		},
		"leaf success": {
			reader: bytes.NewReader(
				append(
					[]byte{
						leafVariant.bits | 1, // key length 1
						9,                    // key data
					},
					scaleEncodeBytes(t, 1, 2, 3)...,
				),
			),
			n: &Node{
				Key:   []byte{9},
				Value: []byte{1, 2, 3},
				Dirty: true,
			},
		},
		"branch decoding error": {
			reader: bytes.NewReader([]byte{
				branchVariant.bits | 1, // key length 1
				// missing key data byte
			}),
			errWrapped: io.EOF,
			errMessage: "cannot decode branch: cannot decode key: " +
				"cannot read from reader: EOF",
		},
		"branch success": {
			reader: bytes.NewReader(
				[]byte{
					branchVariant.bits | 1, // key length 1
					9,                      // key data
					0, 0,                   // no children bitmap
				},
			),
			n: &Node{
				Key:      []byte{9},
				Children: make([]*Node, ChildrenCapacity),
				Dirty:    true,
			},
		},
		"branch with two inlined children": {
			reader: bytes.NewReader(
				[]byte{
					branchVariant.bits | 30, // key length 30
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
			n: &Node{
				Key: []byte{
					12, 3, 6, 5, 12, 3,
					12, 15, 5, 9, 13, 6,
					7, 1, 14, 11, 7, 2,
					13, 10, 0, 14, 7, 10,
					4, 1, 1, 3, 12, 4,
				},
				Descendants: 2,
				Children: []*Node{
					nil, nil, nil, nil,
					{
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
					{
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
		reader           io.Reader
		variant          byte
		partialKeyLength uint16
		branch           *Node
		errWrapped       error
		errMessage       string
	}{
		"key decoding error": {
			reader: bytes.NewBuffer([]byte{
				// missing key data byte
			}),
			variant:          branchVariant.bits,
			partialKeyLength: 1,
			errWrapped:       io.EOF,
			errMessage:       "cannot decode key: cannot read from reader: EOF",
		},
		"children bitmap read error": {
			reader: bytes.NewBuffer([]byte{
				9, // key data
				// missing children bitmap 2 bytes
			}),
			variant:          branchVariant.bits,
			partialKeyLength: 1,
			errWrapped:       ErrReadChildrenBitmap,
			errMessage:       "cannot read children bitmap: EOF",
		},
		"children decoding error": {
			reader: bytes.NewBuffer([]byte{
				9,    // key data
				0, 4, // children bitmap
				// missing children scale encoded data
			}),
			variant:          branchVariant.bits,
			partialKeyLength: 1,
			errWrapped:       ErrDecodeChildHash,
			errMessage:       "cannot decode child hash: at index 10: EOF",
		},
		"success for branch variant": {
			reader: bytes.NewBuffer(
				concatByteSlices([][]byte{
					{9},                                // key data
					{0, 4},                             // children bitmap
					scaleEncodeBytes(t, 1, 2, 3, 4, 5), // child hash
				}),
			),
			variant:          branchVariant.bits,
			partialKeyLength: 1,
			branch: &Node{
				Key: []byte{9},
				Children: padRightChildren([]*Node{
					nil, nil, nil, nil, nil,
					nil, nil, nil, nil, nil,
					{
						HashDigest: []byte{1, 2, 3, 4, 5},
					},
				}),
				Dirty:       true,
				Descendants: 1,
			},
		},
		"value decoding error for branch with value variant": {
			reader: bytes.NewBuffer(
				concatByteSlices([][]byte{
					{9},    // key data
					{0, 4}, // children bitmap
					// missing encoded branch value
				}),
			),
			variant:          branchWithValueVariant.bits,
			partialKeyLength: 1,
			errWrapped:       ErrDecodeValue,
			errMessage:       "cannot decode value: EOF",
		},
		"success for branch with value": {
			reader: bytes.NewBuffer(
				concatByteSlices([][]byte{
					{9},                                // key data
					{0, 4},                             // children bitmap
					scaleEncodeBytes(t, 7, 8, 9),       // branch value
					scaleEncodeBytes(t, 1, 2, 3, 4, 5), // child hash
				}),
			),
			variant:          branchWithValueVariant.bits,
			partialKeyLength: 1,
			branch: &Node{
				Key:   []byte{9},
				Value: []byte{7, 8, 9},
				Children: padRightChildren([]*Node{
					nil, nil, nil, nil, nil,
					nil, nil, nil, nil, nil,
					{
						HashDigest: []byte{1, 2, 3, 4, 5},
					},
				}),
				Dirty:       true,
				Descendants: 1,
			},
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			branch, err := decodeBranch(testCase.reader,
				testCase.variant, testCase.partialKeyLength)

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
		reader           io.Reader
		variant          byte
		partialKeyLength uint16
		leaf             *Node
		errWrapped       error
		errMessage       string
	}{
		"key decoding error": {
			reader: bytes.NewBuffer([]byte{
				// missing key data byte
			}),
			variant:          leafVariant.bits,
			partialKeyLength: 1,
			errWrapped:       io.EOF,
			errMessage:       "cannot decode key: cannot read from reader: EOF",
		},
		"value decoding error": {
			reader: bytes.NewBuffer([]byte{
				9,        // key data
				255, 255, // bad value data
			}),
			variant:          leafVariant.bits,
			partialKeyLength: 1,
			errWrapped:       ErrDecodeValue,
			errMessage:       "cannot decode value: could not decode invalid integer",
		},
		"zero value": {
			reader: bytes.NewBuffer([]byte{
				9, // key data
				// missing value data
			}),
			variant:          leafVariant.bits,
			partialKeyLength: 1,
			leaf: &Node{
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
			variant:          leafVariant.bits,
			partialKeyLength: 1,
			leaf: &Node{
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

			leaf, err := decodeLeaf(testCase.reader,
				testCase.partialKeyLength)

			assert.ErrorIs(t, err, testCase.errWrapped)
			if err != nil {
				assert.EqualError(t, err, testCase.errMessage)
			}
			assert.Equal(t, testCase.leaf, leaf)
		})
	}
}
