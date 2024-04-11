// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package codec

import (
	"bytes"
	"io"
	"testing"

	"github.com/ChainSafe/gossamer/lib/common"
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

func Test_Decode(t *testing.T) {
	t.Parallel()

	hashedValue, err := common.Blake2bHash([]byte("test"))
	assert.NoError(t, err)

	testCases := map[string]struct {
		reader     io.Reader
		n          Node
		errWrapped error
		errMessage string
	}{
		"no_data": {
			reader:     bytes.NewReader(nil),
			errWrapped: io.EOF,
			errMessage: "decoding header: reading header byte: EOF",
		},
		"unknown_node_variant": {
			reader:     bytes.NewReader([]byte{0b0000_1000}),
			errWrapped: ErrVariantUnknown,
			errMessage: "decoding header: decoding header byte: node variant is unknown: for header byte 00001000",
		},
		"empty_node": {
			reader: bytes.NewReader([]byte{emptyVariant.bits}),
			n:      Empty{},
		},
		"leaf_decoding_error": {
			reader: bytes.NewReader([]byte{
				leafVariant.bits | 1, // key length 1
				// missing key data byte
			}),
			errWrapped: io.EOF,
			errMessage: "cannot decode key: " +
				"reading from reader: EOF",
		},
		"leaf_success": {
			reader: bytes.NewReader(bytes.Join([][]byte{
				{leafVariant.bits | 1}, // partial key length 1
				{9},                    // key data
				scaleEncodeBytes(t, 1, 2, 3),
			}, nil)),
			n: Leaf{
				PartialKey: []byte{9},
				Value:      NewInlineValue([]byte{1, 2, 3}),
			},
		},
		"branch_decoding_error": {
			reader: bytes.NewReader([]byte{
				branchVariant.bits | 1, // key length 1
				// missing key data byte
			}),
			errWrapped: io.EOF,
			errMessage: "cannot decode key: " +
				"reading from reader: EOF",
		},
		"branch_success": {
			reader: bytes.NewReader(bytes.Join([][]byte{
				{branchVariant.bits | 1},   // partial key length 1
				{9},                        // key data
				{0b0000_0000, 0b0000_0000}, // no children bitmap
			}, nil)),
			n: Branch{
				PartialKey: []byte{9},
			},
		},
		"leaf_with_hashed_value_success": {
			reader: bytes.NewReader(bytes.Join([][]byte{
				{leafWithHashedValueVariant.bits | 1}, // partial key length 1
				{9},                                   // key data
				hashedValue.ToBytes(),
			}, nil)),
			n: Leaf{
				PartialKey: []byte{9},
				Value:      NewHashedValue(hashedValue.ToBytes()),
			},
		},
		"leaf_with_hashed_value_fail_too_short": {
			reader: bytes.NewReader(bytes.Join([][]byte{
				{leafWithHashedValueVariant.bits | 1}, // partial key length 1
				{9},                                   // key data
				{0b0000_0000},                         // less than 32bytes
			}, nil)),
			errWrapped: ErrDecodeHashedValueTooShort,
			errMessage: "cannot decode leaf: hashed storage value too short: expected 32, got: 1",
		},
		"branch_with_hashed_value_success": {
			reader: bytes.NewReader(bytes.Join([][]byte{
				{branchWithHashedValueVariant.bits | 1}, // partial key length 1
				{9},                                     // key data
				{0b0000_0000, 0b0000_0000},              // no children bitmap
				hashedValue.ToBytes(),
			}, nil)),
			n: Branch{
				PartialKey: []byte{9},
				Value:      NewHashedValue(hashedValue.ToBytes()),
			},
		},
		"branch_with_hashed_value_fail_too_short": {
			reader: bytes.NewReader(bytes.Join([][]byte{
				{branchWithHashedValueVariant.bits | 1}, // partial key length 1
				{9},                                     // key data
				{0b0000_0000, 0b0000_0000},              // no children bitmap
				{0b0000_0000},
			}, nil)),
			errWrapped: ErrDecodeHashedValueTooShort,
			errMessage: "cannot decode branch: hashed storage value too short: expected 32, got: 1",
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
		reader      io.Reader
		nodeVariant variant
		partialKey  []byte
		branch      Branch
		errWrapped  error
		errMessage  string
	}{
		"children_bitmap_read_error": {
			reader: bytes.NewBuffer([]byte{
				// missing children bitmap 2 bytes
			}),
			nodeVariant: branchVariant,
			errWrapped:  ErrReadChildrenBitmap,
			errMessage:  "cannot read children bitmap: EOF",
		},
		"children_decoding_error": {
			reader: bytes.NewBuffer([]byte{
				0b0000_0000, 0b0000_0100, // children bitmap
				// missing children scale encoded data
			}),
			nodeVariant: branchVariant,
			partialKey:  []byte{1},
			errWrapped:  ErrDecodeChildHash,
			errMessage:  "cannot decode child hash: at index 10: decoding uint: reading byte: EOF",
		},
		"success_for_branch_variant": {
			reader: bytes.NewBuffer(
				bytes.Join([][]byte{
					{0b0000_0000, 0b0000_0100}, // children bitmap
					scaleEncodedChildHash,
				}, nil),
			),
			nodeVariant: branchVariant,
			partialKey:  []byte{1},
			branch: Branch{
				PartialKey: []byte{1},
				Children: [ChildrenCapacity]MerkleValue{
					nil, nil, nil, nil, nil,
					nil, nil, nil, nil, nil,
					HashedNode{
						Data: childHash,
					},
				},
			},
		},
		"value_decoding_error_for_branch_with_value_variant": {
			reader: bytes.NewBuffer(
				bytes.Join([][]byte{
					{0b0000_0000, 0b0000_0100}, // children bitmap
					// missing encoded branch storage value
				}, nil),
			),
			nodeVariant: branchWithValueVariant,
			partialKey:  []byte{1},
			errWrapped:  ErrDecodeStorageValue,
			errMessage:  "cannot decode storage value: decoding uint: reading byte: EOF",
		},
		"success_for_branch_with_value": {
			reader: bytes.NewBuffer(bytes.Join([][]byte{
				{0b0000_0000, 0b0000_0100},   // children bitmap
				scaleEncodeBytes(t, 7, 8, 9), // branch storage value
				scaleEncodedChildHash,
			}, nil)),
			nodeVariant: branchWithValueVariant,
			partialKey:  []byte{1},
			branch: Branch{
				PartialKey: []byte{1},
				Value:      NewInlineValue([]byte{7, 8, 9}),
				Children: [ChildrenCapacity]MerkleValue{
					nil, nil, nil, nil, nil,
					nil, nil, nil, nil, nil,
					HashedNode{
						Data: childHash,
					},
				},
			},
		},
		"branch_with_inlined_node_decoding_error": {
			reader: bytes.NewBuffer(bytes.Join([][]byte{
				{0b0000_0001, 0b0000_0000}, // children bitmap
				scaleEncodeBytes(t, 1),     // branch storage value
				{0},                        // garbage inlined node
			}, nil)),
			nodeVariant: branchWithValueVariant,
			partialKey:  []byte{1},
			branch: Branch{
				PartialKey: []byte{1},
				Value:      NewInlineValue([]byte{1}),
				Children: [ChildrenCapacity]MerkleValue{
					InlineNode{
						Data: []byte{},
					},
				},
			},
		},
		"branch_with_inlined_branch_and_leaf": {
			reader: bytes.NewBuffer(bytes.Join([][]byte{
				{0b0000_0011, 0b0000_0000}, // children bitmap
				// top level inlined leaf less than 32 bytes
				scaleEncodeByteSlice(t, bytes.Join([][]byte{
					{leafVariant.bits | 1}, // partial key length of 1
					{2},                    // key data
					scaleEncodeBytes(t, 2), // storage value data
				}, nil)),
				// top level inlined branch less than 32 bytes
				scaleEncodeByteSlice(t, bytes.Join([][]byte{
					{branchWithValueVariant.bits | 1}, // partial key length of 1
					{3},                               // key data
					{0b0000_0001, 0b0000_0000},        // children bitmap
					scaleEncodeBytes(t, 3),            // branch storage value
					// bottom level leaf
					scaleEncodeByteSlice(t, bytes.Join([][]byte{
						{leafVariant.bits | 1}, // partial key length of 1
						{4},                    // key data
						scaleEncodeBytes(t, 4), // storage value data
					}, nil)),
				}, nil)),
			}, nil)),
			nodeVariant: branchVariant,
			partialKey:  []byte{1},
			branch: Branch{
				PartialKey: []byte{1},
				Children: [ChildrenCapacity]MerkleValue{
					InlineNode{
						Data: bytes.Join([][]byte{
							{leafVariant.bits | 1}, // partial key length of 1
							{2},                    // key data
							scaleEncodeBytes(t, 2), // storage value data
						}, nil),
					},
					InlineNode{
						Data: bytes.Join([][]byte{
							{branchWithValueVariant.bits | 1}, // partial key length of 1
							{3},                               // key data
							{0b0000_0001, 0b0000_0000},        // children bitmap
							scaleEncodeBytes(t, 3),            // branch storage value
							// bottom level leaf
							scaleEncodeByteSlice(t, bytes.Join([][]byte{
								{leafVariant.bits | 1}, // partial key length of 1
								{4},                    // key data
								scaleEncodeBytes(t, 4), // storage value data
							}, nil)),
						}, nil),
					},
				},
			},
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			branch, err := decodeBranch(testCase.reader,
				testCase.nodeVariant, testCase.partialKey)

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
		variant    variant
		partialKey []byte
		leaf       Leaf
		errWrapped error
		errMessage string
	}{
		"value_decoding_error": {
			reader: bytes.NewBuffer(bytes.Join([][]byte{
				{255, 255}, // bad storage value data
			}, nil)),
			variant:    leafVariant,
			partialKey: []byte{9},
			errWrapped: ErrDecodeStorageValue,
			errMessage: "cannot decode storage value: decoding uint: unknown prefix for compact uint: 255",
		},
		"missing_storage_value_data": {
			reader: bytes.NewBuffer([]byte{
				// missing storage value data
			}),
			variant:    leafVariant,
			partialKey: []byte{9},
			errWrapped: ErrDecodeStorageValue,
			errMessage: "cannot decode storage value: decoding uint: reading byte: EOF",
		},
		"empty_storage_value_data": {
			reader: bytes.NewBuffer(bytes.Join([][]byte{
				scaleEncodeByteSlice(t, []byte{}), // results to []byte{0}
			}, nil)),
			variant:    leafVariant,
			partialKey: []byte{9},
			leaf: Leaf{
				PartialKey: []byte{9},
				Value:      NewInlineValue([]byte{}),
			},
		},
		"success": {
			reader: bytes.NewBuffer(bytes.Join([][]byte{
				scaleEncodeBytes(t, 1, 2, 3, 4, 5), // storage value data
			}, nil)),
			variant:    leafVariant,
			partialKey: []byte{9},
			leaf: Leaf{
				PartialKey: []byte{9},
				Value:      NewInlineValue([]byte{1, 2, 3, 4, 5}),
			},
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			leaf, err := decodeLeaf(testCase.reader, testCase.variant, testCase.partialKey)

			assert.ErrorIs(t, err, testCase.errWrapped)
			if err != nil {
				assert.EqualError(t, err, testCase.errMessage)
			}
			assert.Equal(t, testCase.leaf, leaf)
		})
	}
}
