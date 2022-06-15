// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package proof

import (
	"testing"

	"github.com/ChainSafe/gossamer/internal/trie/node"
	"github.com/ChainSafe/gossamer/lib/trie"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_Verify(t *testing.T) {
	t.Parallel()

	leafA := node.Node{
		Key:   []byte{1},
		Value: []byte{1},
	}

	longValue := []byte{
		0, 1, 2, 3, 4, 5, 6, 7, 8, 9,
		0, 1, 2, 3, 4, 5, 6, 7, 8, 9,
		0, 1, 2, 3, 4, 5, 6, 7, 8, 9,
		0, 1, 2, 3, 4, 5, 6, 7, 8, 9,
	}
	// leafB is a leaf encoding to more than 32 bytes
	leafB := node.Node{
		Key:   []byte{2},
		Value: longValue,
	}
	require.Greater(t, len(encodeNode(t, leafB)), 32)

	branch := node.Node{
		Key:   []byte{3, 4},
		Value: []byte{1},
		Children: padRightChildren([]*node.Node{
			&leafB,
			nil,
			&leafA,
			&leafB,
		}),
	}
	require.Greater(t, len(encodeNode(t, branch)), 32)

	testCases := map[string]struct {
		encodedProofNodes [][]byte
		rootHash          []byte
		keyLE             []byte
		value             []byte
		errWrapped        error
		errMessage        string
	}{
		"failed building proof trie": {
			rootHash:   []byte{1, 2, 3},
			errWrapped: ErrEmptyProof,
			errMessage: "building trie from proof encoded nodes: " +
				"proof slice empty: for Merkle root hash 0x010203",
		},
		"value not found": {
			encodedProofNodes: [][]byte{
				encodeNode(t, branch),
				encodeNode(t, leafB),
				// TODO Note leaf A is small enough to be inlined in branch
			},
			rootHash:   blake2bNode(t, branch),
			keyLE:      []byte{1, 1}, // nil child of branch
			errWrapped: ErrKeyNotFoundInProofTrie,
			errMessage: "key not found in proof trie: " +
				"0x0101 in proof trie for root hash " +
				"0xe92124f2c4d180493adb4c58250cbd8c5da9c4e3810d8f832e95b1c332de6103",
		},
		"key found with nil search value": {
			encodedProofNodes: [][]byte{
				encodeNode(t, branch),
				encodeNode(t, leafB),
				// Note leaf A is small enough to be inlined in branch
			},
			rootHash: blake2bNode(t, branch),
			keyLE:    []byte{0x34, 0x21}, // inlined short leaf of branch
		},
		"key found with mismatching value": {
			encodedProofNodes: [][]byte{
				encodeNode(t, branch),
				encodeNode(t, leafB),
				// Note leaf A is small enough to be inlined in branch
			},
			rootHash:   blake2bNode(t, branch),
			keyLE:      []byte{0x34, 0x21}, // inlined short leaf of branch
			value:      []byte{2},
			errWrapped: ErrValueMismatchProofTrie,
			errMessage: "value found in proof trie does not match: " +
				"expected value 0x02 but got value 0x01 from proof trie",
		},
		"key found with matching value": {
			encodedProofNodes: [][]byte{
				encodeNode(t, branch),
				encodeNode(t, leafB),
				// Note leaf A is small enough to be inlined in branch
			},
			rootHash: blake2bNode(t, branch),
			keyLE:    []byte{0x34, 0x32}, // large hash-referenced leaf of branch
			value:    longValue,
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			err := Verify(testCase.encodedProofNodes, testCase.rootHash, testCase.keyLE, testCase.value)

			assert.ErrorIs(t, err, testCase.errWrapped)
			if testCase.errWrapped != nil {
				assert.EqualError(t, err, testCase.errMessage)
			}
		})
	}
}

func Test_buildTrie(t *testing.T) {
	t.Parallel()

	leafAShort := node.Node{
		Key:   []byte{1},
		Value: []byte{2},
	}
	leafAShortEncoded := encodeNode(t, leafAShort)
	require.LessOrEqual(t, len(leafAShortEncoded), 32)

	longValue := []byte{
		0, 1, 2, 3, 4, 5, 6, 7, 8, 9,
		0, 1, 2, 3, 4, 5, 6, 7, 8, 9,
		0, 1, 2, 3, 4, 5, 6, 7, 8, 9,
		0, 1, 2, 3, 4, 5, 6, 7, 8, 9,
	}

	leafBLarge := node.Node{
		Key:   []byte{2},
		Value: longValue,
	}
	leafBLargeEncoded := encodeNode(t, leafBLarge)
	require.Greater(t, len(leafBLargeEncoded), 32)

	leafCLarge := node.Node{
		Key:   []byte{3},
		Value: longValue,
	}
	leafCLargeEncoded := encodeNode(t, leafCLarge)
	require.Greater(t, len(leafCLargeEncoded), 32)

	testCases := map[string]struct {
		encodedProofNodes [][]byte
		rootHash          []byte
		expectedTrie      *trie.Trie
		errWrapped        error
		errMessage        string
	}{
		"no proof node": {
			errWrapped: ErrEmptyProof,
			rootHash:   []byte{1},
			errMessage: "proof slice empty: for Merkle root hash 0x01",
		},
		"proof node decoding error": {
			encodedProofNodes: [][]byte{
				{1, 2, 3},
			},
			rootHash:   []byte{3},
			errWrapped: node.ErrUnknownNodeType,
			errMessage: "decoding node at index 0: " +
				"unknown node type: 0 (node encoded is 0x010203)",
		},
		"root proof encoding smaller than 32 bytes": {
			encodedProofNodes: [][]byte{
				encodeNode(t, leafAShort),
			},
			rootHash: blake2bNode(t, leafAShort),
			expectedTrie: trie.NewTrie(&node.Node{
				Key:        leafAShort.Key,
				Value:      leafAShort.Value,
				Encoding:   leafAShortEncoded,
				HashDigest: blake2bNode(t, leafAShort),
				Dirty:      true,
			}),
		},
		"root proof encoding larger than 32 bytes": {
			encodedProofNodes: [][]byte{
				leafBLargeEncoded,
			},
			rootHash: blake2bNode(t, leafBLarge),
			expectedTrie: trie.NewTrie(&node.Node{
				Key:        leafBLarge.Key,
				Value:      leafBLarge.Value,
				Encoding:   leafBLargeEncoded,
				HashDigest: blake2bNode(t, leafBLarge),
				Dirty:      true,
			}),
		},
		"discard unused node": {
			encodedProofNodes: [][]byte{
				leafAShortEncoded,
				leafBLargeEncoded,
			},
			rootHash: blake2bNode(t, leafAShort),
			expectedTrie: trie.NewTrie(&node.Node{
				Key:        leafAShort.Key,
				Value:      leafAShort.Value,
				Encoding:   leafAShortEncoded,
				HashDigest: blake2bNode(t, leafAShort),
				Dirty:      true,
			}),
		},
		"multiple unordered nodes": {
			encodedProofNodes: [][]byte{
				leafBLargeEncoded, // chilren 1 and 3
				encodeNode(t, node.Node{ // root
					Key: []byte{1},
					Children: padRightChildren([]*node.Node{
						&leafAShort, // inlined
						&leafBLarge, // referenced by Merkle value hash
						&leafCLarge, // referenced by Merkle value hash
						&leafBLarge, // referenced by Merkle value hash
					}),
				}),
				leafCLargeEncoded, // children 2
			},
			rootHash: blake2bNode(t, node.Node{
				Key: []byte{1},
				Children: padRightChildren([]*node.Node{
					&leafAShort,
					&leafBLarge,
					&leafCLarge,
					&leafBLarge,
				}),
			}),
			expectedTrie: trie.NewTrie(&node.Node{
				Key:         []byte{1},
				Descendants: 4,
				Dirty:       true,
				Children: padRightChildren([]*node.Node{
					{
						Key:   leafAShort.Key,
						Value: leafAShort.Value,
						Dirty: true,
					},
					{
						Key:        leafBLarge.Key,
						Value:      leafBLarge.Value,
						Encoding:   leafBLargeEncoded,
						HashDigest: blake2bNode(t, leafBLarge),
						Dirty:      true,
					},
					{
						Key:        leafCLarge.Key,
						Value:      leafCLarge.Value,
						Encoding:   leafCLargeEncoded,
						HashDigest: blake2bNode(t, leafCLarge),
						Dirty:      true,
					},
					{
						Key:        leafBLarge.Key,
						Value:      leafBLarge.Value,
						Encoding:   leafBLargeEncoded,
						HashDigest: blake2bNode(t, leafBLarge),
						Dirty:      true,
					},
				}),
				Encoding: encodeNode(t, node.Node{
					Key: []byte{1},
					Children: padRightChildren([]*node.Node{
						&leafAShort,
						&leafBLarge,
						&leafCLarge,
						&leafBLarge,
					}),
				}),
				HashDigest: blake2bNode(t, node.Node{
					Key: []byte{1},
					Children: padRightChildren([]*node.Node{
						&leafAShort,
						&leafBLarge,
						&leafCLarge,
						&leafBLarge,
					}),
				}),
			}),
		},
		"root not found": {
			encodedProofNodes: [][]byte{
				encodeNode(t, node.Node{
					Key:   []byte{1},
					Value: []byte{2},
				}),
			},
			rootHash:   []byte{3},
			errWrapped: ErrRootNodeNotFound,
			errMessage: "root node not found in proof: " +
				"for Merkle root hash 0x03 in proof Merkle value(s) 0x41010402",
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			trie, err := buildTrie(testCase.encodedProofNodes, testCase.rootHash)

			assert.ErrorIs(t, err, testCase.errWrapped)
			if testCase.errWrapped != nil {
				assert.EqualError(t, err, testCase.errMessage)
			}

			if testCase.expectedTrie != nil {
				require.NotNil(t, trie)
				require.Equal(t, testCase.expectedTrie.String(), trie.String())
			}
			assert.Equal(t, testCase.expectedTrie, trie)
		})
	}
}

func Test_loadProof(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		proofHashToNode map[string]*node.Node
		node            *node.Node
		expectedNode    *node.Node
	}{
		"leaf node": {
			node: &node.Node{
				Key:   []byte{1},
				Value: []byte{2},
			},
			expectedNode: &node.Node{
				Key:   []byte{1},
				Value: []byte{2},
			},
		},
		"branch node": {
			node: &node.Node{
				Key: []byte{1},
				Children: padRightChildren([]*node.Node{
					nil,
					{ // hash not found in proof map
						HashDigest: []byte{1},
					},
					{ // hash found in proof map
						HashDigest: []byte{2},
					},
				}),
			},
			proofHashToNode: map[string]*node.Node{
				"0x02": {
					Key: []byte{2},
					Children: padRightChildren([]*node.Node{
						{ // hash found in proof map
							HashDigest: []byte{3},
						},
					}),
				},
				"0x03": {
					Key:   []byte{3},
					Value: []byte{1},
				},
			},
			expectedNode: &node.Node{
				Key: []byte{1},
				Children: padRightChildren([]*node.Node{
					nil,
					{ // hash not found in proof map
						HashDigest: []byte{1},
					},
					{ // hash found in proof map
						Key: []byte{2},
						Children: padRightChildren([]*node.Node{
							{ // hash found in proof map
								Key:   []byte{3},
								Value: []byte{1},
							},
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

			loadProof(testCase.proofHashToNode, testCase.node)

			assert.Equal(t, testCase.expectedNode, testCase.node)
		})
	}
}

func Test_bytesToString(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		b []byte
		s string
	}{
		"nil slice": {
			s: "nil",
		},
		"empty slice": {
			b: []byte{},
			s: "0x",
		},
		"small slice": {
			b: []byte{1, 2, 3},
			s: "0x010203",
		},
		"big slice": {
			b: []byte{
				0, 1, 2, 3, 4, 5, 6, 7, 8, 9,
				0, 1, 2, 3, 4, 5, 6, 7, 8, 9,
				0, 1, 2, 3, 4, 5, 6, 7, 8, 9,
			},
			s: "0x0001020304050607...0203040506070809",
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			s := bytesToString(testCase.b)

			assert.Equal(t, testCase.s, s)
		})
	}
}
