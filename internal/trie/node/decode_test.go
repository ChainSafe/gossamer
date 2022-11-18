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
	return scaleEncodeByteSlice(t, b)
}

func scaleEncodeByteSlice(t *testing.T, b []byte) (encoded []byte) {
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
			errMessage: "decoding header: reading header byte: EOF",
		},
		"unknown node variant": {
			reader:     bytes.NewReader([]byte{0}),
			errWrapped: ErrVariantUnknown,
			errMessage: "decoding header: decoding header byte: node variant is unknown: for header byte 00000000",
		},
		"leaf decoding error": {
			reader: bytes.NewReader([]byte{
				leafVariant.bits | 1, // key length 1
				// missing key data byte
			}),
			errWrapped: io.EOF,
			errMessage: "cannot decode leaf: cannot decode key: " +
				"reading from reader: EOF",
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
				PartialKey:   []byte{9},
				StorageValue: []byte{1, 2, 3},
			},
		},
		"branch decoding error": {
			reader: bytes.NewReader([]byte{
				branchVariant.bits | 1, // key length 1
				// missing key data byte
			}),
			errWrapped: io.EOF,
			errMessage: "cannot decode branch: cannot decode key: " +
				"reading from reader: EOF",
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
				PartialKey: []byte{9},
				Children:   make([]*Node, ChildrenCapacity),
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

	const childHashLength = 32
	childHash := make([]byte, childHashLength)
	for i := range childHash {
		childHash[i] = byte(i)
	}
	scaleEncodedChildHash := scaleEncodeByteSlice(t, childHash)

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
			errMessage:       "cannot decode key: reading from reader: EOF",
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
			errMessage:       "cannot decode child hash: at index 10: reading byte: EOF",
		},
		"success for branch variant": {
			reader: bytes.NewBuffer(
				concatByteSlices([][]byte{
					{9},    // key data
					{0, 4}, // children bitmap
					scaleEncodedChildHash,
				}),
			),
			variant:          branchVariant.bits,
			partialKeyLength: 1,
			branch: &Node{
				PartialKey: []byte{9},
				Children: padRightChildren([]*Node{
					nil, nil, nil, nil, nil,
					nil, nil, nil, nil, nil,
					{
						MerkleValue: childHash,
					},
				}),
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
			errMessage:       "cannot decode value: reading byte: EOF",
		},
		"success for branch with value": {
			reader: bytes.NewBuffer(concatByteSlices([][]byte{
				{9},                          // key data
				{0, 4},                       // children bitmap
				scaleEncodeBytes(t, 7, 8, 9), // branch value
				scaleEncodedChildHash,
			})),
			variant:          branchWithValueVariant.bits,
			partialKeyLength: 1,
			branch: &Node{
				PartialKey:   []byte{9},
				StorageValue: []byte{7, 8, 9},
				Children: padRightChildren([]*Node{
					nil, nil, nil, nil, nil,
					nil, nil, nil, nil, nil,
					{
						MerkleValue: childHash,
					},
				}),
				Descendants: 1,
			},
		},
		"branch with inlined node decoding error": {
			reader: bytes.NewBuffer(concatByteSlices([][]byte{
				{1},                        // key data
				{0b0000_0001, 0b0000_0000}, // children bitmap
				scaleEncodeBytes(t, 1),     // branch value
				{0},                        // garbage inlined node
			})),
			variant:          branchWithValueVariant.bits,
			partialKeyLength: 1,
			errWrapped:       io.EOF,
			errMessage: "decoding inlined child at index 0: " +
				"decoding header: reading header byte: EOF",
		},
		"branch with inlined branch and leaf": {
			reader: bytes.NewBuffer(concatByteSlices([][]byte{
				{1},                        // key data
				{0b0000_0011, 0b0000_0000}, // children bitmap
				// top level inlined leaf less than 32 bytes
				scaleEncodeByteSlice(t, concatByteSlices([][]byte{
					{leafVariant.bits | 1}, // partial key length of 1
					{2},                    // key data
					scaleEncodeBytes(t, 2), // value data
				})),
				// top level inlined branch less than 32 bytes
				scaleEncodeByteSlice(t, concatByteSlices([][]byte{
					{branchWithValueVariant.bits | 1}, // partial key length of 1
					{3},                               // key data
					{0b0000_0001, 0b0000_0000},        // children bitmap
					scaleEncodeBytes(t, 3),            // branch value
					// bottom level leaf
					scaleEncodeByteSlice(t, concatByteSlices([][]byte{
						{leafVariant.bits | 1}, // partial key length of 1
						{4},                    // key data
						scaleEncodeBytes(t, 4), // value data
					})),
				})),
			})),
			variant:          branchVariant.bits,
			partialKeyLength: 1,
			branch: &Node{
				PartialKey:  []byte{1},
				Descendants: 3,
				Children: padRightChildren([]*Node{
					{PartialKey: []byte{2}, StorageValue: []byte{2}},
					{
						PartialKey:   []byte{3},
						StorageValue: []byte{3},
						Descendants:  1,
						Children: padRightChildren([]*Node{
							{PartialKey: []byte{4}, StorageValue: []byte{4}},
						}),
					},
				}),
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
			errMessage:       "cannot decode key: reading from reader: EOF",
		},
		"value decoding error": {
			reader: bytes.NewBuffer([]byte{
				9,        // key data
				255, 255, // bad value data
			}),
			variant:          leafVariant.bits,
			partialKeyLength: 1,
			errWrapped:       ErrDecodeValue,
			errMessage:       "cannot decode value: unknown prefix for compact uint: 255",
		},
		"missing value data": {
			reader: bytes.NewBuffer([]byte{
				9, // key data
				// missing value data
			}),
			variant:          leafVariant.bits,
			partialKeyLength: 1,
			leaf: &Node{
				PartialKey: []byte{9},
			},
		},
		"empty value data": {
			reader: bytes.NewBuffer(concatByteSlices([][]byte{
				{9}, // key data
				scaleEncodeByteSlice(t, nil),
			})),
			variant:          leafVariant.bits,
			partialKeyLength: 1,
			leaf: &Node{
				PartialKey: []byte{9},
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
				PartialKey:   []byte{9},
				StorageValue: []byte{1, 2, 3, 4, 5},
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
