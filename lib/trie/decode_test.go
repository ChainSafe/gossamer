// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package trie

import (
	"bytes"
	"io"
	"testing"

	"github.com/ChainSafe/gossamer/lib/trie/branch"
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

func Test_decodeNode(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		reader     io.Reader
		n          node.Node
		errWrapped error
		errMessage string
	}{
		"no data": {
			reader:     bytes.NewBuffer(nil),
			errWrapped: ErrReadHeaderByte,
			errMessage: "cannot read header byte: EOF",
		},
		"unknown node type": {
			reader:     bytes.NewBuffer([]byte{0}),
			errWrapped: ErrUnknownNodeType,
			errMessage: "unknown node type: 0",
		},
		"leaf decoding error": {
			reader: bytes.NewBuffer([]byte{
				65, // node type 1 and key length 1
				// missing key data byte
			}),
			errWrapped: decode.ErrReadKeyData,
			errMessage: "cannot decode leaf: cannot decode key: cannot read key data: EOF",
		},
		"leaf success": {
			reader: bytes.NewBuffer(
				append(
					[]byte{
						65, // node type 1 and key length 1
						9,  // key data
					},
					scaleEncodeBytes(t, 1, 2, 3)...,
				),
			),
			n: &leaf.Leaf{
				Key:   []byte{9},
				Value: []byte{1, 2, 3},
				Dirty: true,
			},
		},
		"branch decoding error": {
			reader: bytes.NewBuffer([]byte{
				129, // node type 2 and key length 1
				// missing key data byte
			}),
			errWrapped: decode.ErrReadKeyData,
			errMessage: "cannot decode branch: cannot decode key: cannot read key data: EOF",
		},
		"branch success": {
			reader: bytes.NewBuffer(
				[]byte{
					129,  // node type 2 and key length 1
					9,    // key data
					0, 0, // no children bitmap
				},
			),
			n: &branch.Branch{
				Key:   []byte{9},
				Dirty: true,
			},
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			n, err := decodeNode(testCase.reader)

			assert.ErrorIs(t, err, testCase.errWrapped)
			if err != nil {
				assert.EqualError(t, err, testCase.errMessage)
			}
			assert.Equal(t, testCase.n, n)
		})
	}
}
