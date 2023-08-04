// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package triedb

import (
	"bytes"
	"io"
	"testing"

	"github.com/ChainSafe/gossamer/internal/trie/node"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/ChainSafe/gossamer/pkg/trie/triedb/nibble"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_decodeLeaf(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		reader           io.Reader
		variant          node.Variant
		partialKeyLength uint16
		leaf             *Node
		errWrapped       error
		errMessage       string
	}{
		"key_decoding_error": {
			reader: bytes.NewBuffer([]byte{
				// missing key data byte
			}),
			variant:          node.LeafVariant,
			partialKeyLength: 1,
			errWrapped:       io.EOF,
			errMessage:       "cannot decode key: reading from reader: EOF",
		},
		"value_decoding_error": {
			reader: bytes.NewBuffer(concatByteSlices([][]byte{
				{9},        // key data
				{255, 255}, // bad storage value data
			})),
			variant:          node.LeafVariant,
			partialKeyLength: 2,
			errWrapped:       ErrDecodeStorageValue,
			errMessage:       "cannot decode storage value: unknown prefix for compact uint: 255",
		},
		"missing_storage_value_data": {
			reader: bytes.NewBuffer([]byte{
				9, // key data
				// missing storage value data
			}),
			variant:          node.LeafVariant,
			partialKeyLength: 2,
			errWrapped:       ErrDecodeStorageValue,
			errMessage:       "cannot decode storage value: reading byte: EOF",
		},
		"empty_storage_value_data": {
			reader: bytes.NewBuffer(concatByteSlices([][]byte{
				{9},                               // key data
				scaleEncodeByteSlice(t, []byte{}), // results to []byte{0}
			})),
			variant:          node.LeafVariant,
			partialKeyLength: 2,
			leaf: &Node{
				Type:    Leaf,
				Partial: *nibble.NewNibbleSliceWithPadding([]byte{9}, 0),
				Value:   &NodeValue{[]byte{}, false},
			},
		},
		"success": {
			reader: bytes.NewBuffer(concatByteSlices([][]byte{
				{9},                                // key data
				scaleEncodeBytes(t, 1, 2, 3, 4, 5), // storage value data
			})),
			variant:          node.LeafVariant,
			partialKeyLength: 2,
			leaf: &Node{
				Type:    Leaf,
				Partial: *nibble.NewNibbleSliceWithPadding([]byte{9}, 0),
				Value:   &NodeValue{[]byte{1, 2, 3, 4, 5}, false},
			},
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			leaf, err := decodeLeaf(testCase.reader, testCase.variant, testCase.partialKeyLength)

			assert.ErrorIs(t, err, testCase.errWrapped)
			if err != nil {
				assert.EqualError(t, err, testCase.errMessage)
			}
			assert.Equal(t, testCase.leaf, leaf)
		})
	}
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

func scaleEncodeByteSlice(t *testing.T, b []byte) (encoded []byte) {
	encoded, err := scale.Marshal(b)
	require.NoError(t, err)
	return encoded
}

func scaleEncodeBytes(t *testing.T, b ...byte) (encoded []byte) {
	return scaleEncodeByteSlice(t, b)
}
