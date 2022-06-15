// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package proof

import (
	"errors"
	"testing"

	"github.com/ChainSafe/gossamer/internal/trie/node"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func Test_Generate(t *testing.T) {
	t.Parallel()

	errTest := errors.New("test error")

	testCases := map[string]struct {
		rootHash          common.Hash
		fullKey           []byte // nibbles
		databaseBuilder   func(ctrl *gomock.Controller) Database
		encodedProofNodes [][]byte
		errWrapped        error
		errMessage        string
	}{
		"failed loading trie": {
			rootHash: common.Hash{1},
			databaseBuilder: func(ctrl *gomock.Controller) Database {
				mockDatabase := NewMockDatabase(ctrl)
				mockDatabase.EXPECT().Get(common.Hash{1}.ToBytes()).
					Return(nil, errTest)
				return mockDatabase
			},
			errWrapped: errTest,
			errMessage: "cannot load trie: " +
				"failed to find root key " +
				"0x0100000000000000000000000000000000000000000000000000000000000000: " +
				"test error",
		},
		"walk error": {
			rootHash: common.Hash{1},
			databaseBuilder: func(ctrl *gomock.Controller) Database {
				mockDatabase := NewMockDatabase(ctrl)
				encodedRoot := encodeNode(t, node.Node{
					Key:   []byte{1},
					Value: []byte{2},
				})
				mockDatabase.EXPECT().Get(common.Hash{1}.ToBytes()).
					Return(encodedRoot, nil)
				return mockDatabase
			},
			fullKey:    []byte{1},
			errWrapped: ErrKeyNotFound,
			errMessage: "cannot find node at key 0x01 in trie: key not found",
		},
		"leaf root": {
			rootHash: common.Hash{1},
			databaseBuilder: func(ctrl *gomock.Controller) Database {
				mockDatabase := NewMockDatabase(ctrl)
				encodedRoot := encodeNode(t, node.Node{
					Key:   []byte{1},
					Value: []byte{2},
				})
				mockDatabase.EXPECT().Get(common.Hash{1}.ToBytes()).
					Return(encodedRoot, nil)
				return mockDatabase
			},
			encodedProofNodes: [][]byte{
				encodeNode(t, node.Node{
					Key:   []byte{1},
					Value: []byte{2},
				}),
			},
		},
		"branch root": {
			rootHash: common.Hash{1},
			databaseBuilder: func(ctrl *gomock.Controller) Database {
				mockDatabase := NewMockDatabase(ctrl)
				encodedRoot := encodeNode(t, node.Node{
					Key:   []byte{1},
					Value: []byte{2},
					Children: padRightChildren([]*node.Node{
						nil, nil,
						{
							Key:   []byte{3},
							Value: []byte{4},
						},
					}),
				})
				mockDatabase.EXPECT().Get(common.Hash{1}.ToBytes()).
					Return(encodedRoot, nil)
				return mockDatabase
			},
			encodedProofNodes: [][]byte{
				encodeNode(t, node.Node{
					Key:   []byte{1},
					Value: []byte{2},
					Children: padRightChildren([]*node.Node{
						nil, nil,
						{
							Key:   []byte{3},
							Value: []byte{4},
						},
					}),
				}),
			},
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)

			database := testCase.databaseBuilder(ctrl)

			encodedProofNodes, err := Generate(testCase.rootHash,
				testCase.fullKey, database)

			assert.ErrorIs(t, err, testCase.errWrapped)
			if testCase.errWrapped != nil {
				assert.EqualError(t, err, testCase.errMessage)
			}
			assert.Equal(t, testCase.encodedProofNodes, encodedProofNodes)
		})
	}
}

func Test_walk(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		parent            *node.Node
		fullKey           []byte // nibbles
		encodedProofNodes [][]byte
		errWrapped        error
		errMessage        string
	}{
		"nil parent and empty full key": {},
		"nil parent and non empty full key": {
			fullKey:    []byte{1},
			errWrapped: ErrKeyNotFound,
			errMessage: "key not found",
		},
		"parent encode and hash error": {
			parent: &node.Node{
				Key:   make([]byte, int(^uint16(0))+63),
				Value: []byte{1},
			},
			errWrapped: node.ErrPartialKeyTooBig,
			errMessage: "encode node: " +
				"cannot encode header: partial key length cannot " +
				"be larger than or equal to 2^16: 65535",
		},
		"parent leaf and empty full key": {
			parent: &node.Node{
				Key:   []byte{1, 2},
				Value: []byte{1},
			},
			encodedProofNodes: [][]byte{{0b0100_0000 | 2, 0x12, 0x04, 0x01}},
		},
		"parent leaf and shorter full key": {
			parent: &node.Node{
				Key:   []byte{1, 2},
				Value: []byte{1},
			},
			fullKey:    []byte{1},
			errWrapped: ErrKeyNotFound,
			errMessage: "key not found",
		},
		"parent leaf and mismatching full key": {
			parent: &node.Node{
				Key:   []byte{1, 2},
				Value: []byte{1},
			},
			fullKey:    []byte{1, 3},
			errWrapped: ErrKeyNotFound,
			errMessage: "key not found",
		},
		"parent leaf and longer full key": {
			parent: &node.Node{
				Key:   []byte{1, 2},
				Value: []byte{1},
			},
			fullKey:    []byte{1, 2, 3},
			errWrapped: ErrKeyNotFound,
			errMessage: "key not found",
		},
		"branch and empty search key": {
			parent: &node.Node{
				Key:   []byte{1, 2},
				Value: []byte{3},
				Children: padRightChildren([]*node.Node{
					{
						Key:   []byte{4},
						Value: []byte{5},
					},
				}),
			},
			encodedProofNodes: [][]byte{
				{
					0b1100_0000 | 2,          // Node variant and partial key length
					0x12,                     // partial key
					0b0000_0001, 0b0000_0000, // children bitmap
					0x4, 0x3, // scale encoded value
					0x10,
					0b0100_0000 | 1, // Node variant and partial key length
					0x4, 0x4,        // partial key
					0x5, // leaf value
				}},
		},
		"branch and shorter full key": {
			parent: &node.Node{
				Key:   []byte{1, 2},
				Value: []byte{3},
				Children: padRightChildren([]*node.Node{
					{
						Key:   []byte{4},
						Value: []byte{5},
					},
				}),
			},
			fullKey:    []byte{1},
			errWrapped: ErrKeyNotFound,
			errMessage: "key not found",
		},
		"branch and mismatching full key": {
			parent: &node.Node{
				Key:   []byte{1, 2},
				Value: []byte{3},
				Children: padRightChildren([]*node.Node{
					{
						Key:   []byte{4},
						Value: []byte{5},
					},
				}),
			},
			fullKey:    []byte{1, 3},
			errWrapped: ErrKeyNotFound,
			errMessage: "key not found",
		},
		"branch and matching search key": {
			parent: &node.Node{
				Key:   []byte{1, 2},
				Value: []byte{3},
				Children: padRightChildren([]*node.Node{
					{
						Key:   []byte{4},
						Value: []byte{5},
					},
				}),
			},
			fullKey: []byte{1, 2},
			encodedProofNodes: [][]byte{
				{
					0b1100_0000 | 2,          // Node variant and partial key length
					0x12,                     // partial key
					0b0000_0001, 0b0000_0000, // children bitmap
					0x4, 0x3, // scale encoded value
					0x10,
					0b0100_0000 | 1, // Node variant and partial key length
					0x4, 0x4,        // partial key
					0x5, // leaf value
				}},
		},
		"key not found at deeper level": {
			parent: &node.Node{
				Key:   []byte{1, 2},
				Value: []byte{3},
				Children: padRightChildren([]*node.Node{
					{
						Key:   []byte{4, 5},
						Value: []byte{5},
					},
				}),
			},
			fullKey:    []byte{1, 2, 0x04, 4},
			errWrapped: ErrKeyNotFound,
			errMessage: "key not found",
		},
		"found leaf at deeper level": {
			parent: &node.Node{
				Key:   []byte{1, 2},
				Value: []byte{3},
				Children: padRightChildren([]*node.Node{
					{
						Key:   []byte{4},
						Value: []byte{5},
					},
				}),
			},
			fullKey: []byte{1, 2, 0x04},
			encodedProofNodes: [][]byte{
				{
					0b1100_0000 | 2,          // Node variant and partial key length
					0x12,                     // partial key
					0b0000_0001, 0b0000_0000, // children bitmap
					0x4, 0x3, // scale encoded value
					0x10,
					0b0100_0000 | 1, // Node variant and partial key length
					0x4, 0x4,        // partial key
					0x5, // leaf value
				}},
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			encodedProofNodes, err := walk(testCase.parent, testCase.fullKey)

			assert.ErrorIs(t, err, testCase.errWrapped)
			if testCase.errWrapped != nil {
				assert.EqualError(t, err, testCase.errMessage)
			}
			assert.Equal(t, testCase.encodedProofNodes, encodedProofNodes)
		})
	}
}

func Test_lenCommonPrefix(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		a      []byte
		b      []byte
		length int
	}{
		"nil slices": {},
		"empty slices": {
			a: []byte{},
			b: []byte{},
		},
		"fully different": {
			a: []byte{1, 2, 3},
			b: []byte{4, 5, 6},
		},
		"fully same": {
			a:      []byte{1, 2, 3},
			b:      []byte{1, 2, 3},
			length: 3,
		},
		"different and common prefix": {
			a:      []byte{1, 2, 3, 4},
			b:      []byte{1, 2, 4, 4},
			length: 2,
		},
		"first bigger than second": {
			a:      []byte{1, 2, 3},
			b:      []byte{1, 2},
			length: 2,
		},
		"first smaller than second": {
			a:      []byte{1, 2},
			b:      []byte{1, 2, 3},
			length: 2,
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			length := lenCommonPrefix(testCase.a, testCase.b)

			assert.Equal(t, testCase.length, length)
		})
	}
}
