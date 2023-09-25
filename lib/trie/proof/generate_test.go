// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package proof

import (
	"testing"

	"github.com/ChainSafe/gossamer/internal/trie/codec"
	"github.com/ChainSafe/gossamer/internal/trie/node"
	"github.com/ChainSafe/gossamer/lib/trie"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_Generate(t *testing.T) {
	t.Parallel()

	someHash := make([]byte, 32)
	for i := range someHash {
		someHash[i] = byte(i)
	}

	largeValue := generateBytes(t, 50)
	assertLongEncoding(t, node.Node{StorageValue: largeValue})

	testCases := map[string]struct {
		rootHash          []byte
		fullKeysNibbles   [][]byte
		trie              *trie.Trie
		encodedProofNodes [][]byte
		errWrapped        error
		errMessage        string
	}{
		"walk_error": {
			rootHash:        someHash,
			fullKeysNibbles: [][]byte{{1}},
			trie: trie.NewTrie(&node.Node{
				PartialKey:   []byte{1},
				StorageValue: []byte{2},
			}, nil),
			errWrapped: ErrKeyNotFound,
			errMessage: "walking to node at key 0x01: key not found",
		},
		"leaf_root": {
			rootHash:        someHash,
			fullKeysNibbles: [][]byte{{}},
			trie: trie.NewTrie(&node.Node{
				PartialKey:   []byte{1},
				StorageValue: []byte{2},
			}, nil),
			encodedProofNodes: [][]byte{
				encodeNode(t, node.Node{
					PartialKey:   []byte{1},
					StorageValue: []byte{2},
				}),
			},
		},
		"branch_root": {
			rootHash:        someHash,
			fullKeysNibbles: [][]byte{{}},
			trie: trie.NewTrie(&node.Node{
				PartialKey:   []byte{1},
				StorageValue: []byte{2},
				Children: padRightChildren([]*node.Node{
					nil, nil,
					{
						PartialKey:   []byte{3},
						StorageValue: []byte{4},
					},
				}),
			}, nil),
			encodedProofNodes: [][]byte{
				encodeNode(t, node.Node{
					PartialKey:   []byte{1},
					StorageValue: []byte{2},
					Children: padRightChildren([]*node.Node{
						nil, nil,
						{
							PartialKey:   []byte{3},
							StorageValue: []byte{4},
						},
					}),
				}),
			},
		},
		"target_leaf_of_branch": {
			rootHash: someHash,
			fullKeysNibbles: [][]byte{
				{1, 2, 3, 4},
			},
			trie: trie.NewTrie(&node.Node{
				PartialKey:   []byte{1, 2},
				StorageValue: []byte{2},
				Children: padRightChildren([]*node.Node{
					nil, nil, nil,
					{ // full key 1, 2, 3, 4
						PartialKey:   []byte{4},
						StorageValue: largeValue,
					},
				}),
			}, nil),
			encodedProofNodes: [][]byte{
				encodeNode(t, node.Node{
					PartialKey:   []byte{1, 2},
					StorageValue: []byte{2},
					Children: padRightChildren([]*node.Node{
						nil, nil, nil,
						{
							PartialKey:   []byte{4},
							StorageValue: largeValue,
						},
					}),
				}),
				encodeNode(t, node.Node{
					PartialKey:   []byte{4},
					StorageValue: largeValue,
				}),
			},
		},
		"deduplicate_proof_nodes": {
			rootHash: someHash,
			fullKeysNibbles: [][]byte{
				{1, 2, 3, 4},
				{1, 2, 4, 4},
				{1, 2, 5, 5},
			},
			trie: trie.NewTrie(&node.Node{
				PartialKey:   []byte{1, 2},
				StorageValue: []byte{2},
				Children: padRightChildren([]*node.Node{
					nil, nil, nil,
					{ // full key 1, 2, 3, 4
						PartialKey:   []byte{4},
						StorageValue: largeValue,
					},
					{ // full key 1, 2, 4, 4
						PartialKey:   []byte{4},
						StorageValue: largeValue,
					},
					{ // full key 1, 2, 5, 5
						PartialKey:   []byte{5},
						StorageValue: largeValue,
					},
				}),
			}, nil),
			encodedProofNodes: [][]byte{
				encodeNode(t, node.Node{
					PartialKey:   []byte{1, 2},
					StorageValue: []byte{2},
					Children: padRightChildren([]*node.Node{
						nil, nil, nil,
						{ // full key 1, 2, 3, 4
							PartialKey:   []byte{4},
							StorageValue: largeValue,
						},
						{ // full key 1, 2, 4, 4
							PartialKey:   []byte{4},
							StorageValue: largeValue,
						},
						{ // full key 1, 2, 5, 5
							PartialKey:   []byte{5},
							StorageValue: largeValue,
						},
					}),
				}),
				encodeNode(t, node.Node{
					PartialKey:   []byte{4},
					StorageValue: largeValue,
				}),
				encodeNode(t, node.Node{
					PartialKey:   []byte{5},
					StorageValue: largeValue,
				}),
			},
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			fullKeysLE := make([][]byte, len(testCase.fullKeysNibbles))
			for i, fullKeyNibbles := range testCase.fullKeysNibbles {
				fullKeysLE[i] = codec.NibblesToKeyLE(fullKeyNibbles)
			}

			encodedProofNodes, err := Generate(testCase.trie, fullKeysLE)

			assert.ErrorIs(t, err, testCase.errWrapped)
			if testCase.errWrapped != nil {
				assert.EqualError(t, err, testCase.errMessage)
			}
			assert.Equal(t, testCase.encodedProofNodes, encodedProofNodes)
		})
	}
}

func Test_walkRoot(t *testing.T) {
	t.Parallel()

	largeValue := generateBytes(t, 40)
	assertLongEncoding(t, node.Node{StorageValue: largeValue})

	testCases := map[string]struct {
		parent            *node.Node
		fullKey           []byte // nibbles
		encodedProofNodes [][]byte
		errWrapped        error
		errMessage        string
	}{
		"nil_parent_and_empty_full_key": {},
		"nil_parent_and_non_empty_full_key": {
			fullKey:    []byte{1},
			errWrapped: ErrKeyNotFound,
			errMessage: "key not found",
		},
		// The parent encode error cannot be triggered here
		// since it can only be caused by a buffer.Write error.
		"parent_leaf_and_empty_full_key": {
			parent: &node.Node{
				PartialKey:   []byte{1, 2},
				StorageValue: []byte{1},
			},
			encodedProofNodes: [][]byte{encodeNode(t, node.Node{
				PartialKey:   []byte{1, 2},
				StorageValue: []byte{1},
			})},
		},
		"parent_leaf_and_shorter_full_key": {
			parent: &node.Node{
				PartialKey:   []byte{1, 2},
				StorageValue: []byte{1},
			},
			fullKey:    []byte{1},
			errWrapped: ErrKeyNotFound,
			errMessage: "key not found",
		},
		"parent_leaf_and_mismatching_full_key": {
			parent: &node.Node{
				PartialKey:   []byte{1, 2},
				StorageValue: []byte{1},
			},
			fullKey:    []byte{1, 3},
			errWrapped: ErrKeyNotFound,
			errMessage: "key not found",
		},
		"parent_leaf_and_longer_full_key": {
			parent: &node.Node{
				PartialKey:   []byte{1, 2},
				StorageValue: []byte{1},
			},
			fullKey:    []byte{1, 2, 3},
			errWrapped: ErrKeyNotFound,
			errMessage: "key not found",
		},
		"branch_and_empty_search_key": {
			parent: &node.Node{
				PartialKey:   []byte{1, 2},
				StorageValue: []byte{3},
				Children: padRightChildren([]*node.Node{
					{
						PartialKey:   []byte{4},
						StorageValue: []byte{5},
					},
				}),
			},
			encodedProofNodes: [][]byte{
				encodeNode(t, node.Node{
					PartialKey:   []byte{1, 2},
					StorageValue: []byte{3},
					Children: padRightChildren([]*node.Node{
						{
							PartialKey:   []byte{4},
							StorageValue: []byte{5},
						},
					}),
				}),
			},
		},
		"branch_and_shorter_full_key": {
			parent: &node.Node{
				PartialKey:   []byte{1, 2},
				StorageValue: []byte{3},
				Children: padRightChildren([]*node.Node{
					{
						PartialKey:   []byte{4},
						StorageValue: []byte{5},
					},
				}),
			},
			fullKey:    []byte{1},
			errWrapped: ErrKeyNotFound,
			errMessage: "key not found",
		},
		"branch_and_mismatching_full_key": {
			parent: &node.Node{
				PartialKey:   []byte{1, 2},
				StorageValue: []byte{3},
				Children: padRightChildren([]*node.Node{
					{
						PartialKey:   []byte{4},
						StorageValue: []byte{5},
					},
				}),
			},
			fullKey:    []byte{1, 3},
			errWrapped: ErrKeyNotFound,
			errMessage: "key not found",
		},
		"branch_and_matching_search_key": {
			parent: &node.Node{
				PartialKey:   []byte{1, 2},
				StorageValue: []byte{3},
				Children: padRightChildren([]*node.Node{
					{
						PartialKey:   []byte{4},
						StorageValue: []byte{5},
					},
				}),
			},
			fullKey: []byte{1, 2},
			encodedProofNodes: [][]byte{
				encodeNode(t, node.Node{
					PartialKey:   []byte{1, 2},
					StorageValue: []byte{3},
					Children: padRightChildren([]*node.Node{
						{
							PartialKey:   []byte{4},
							StorageValue: []byte{5},
						},
					}),
				}),
			},
		},
		"branch_and_matching_search_key_for_small_leaf_encoding": {
			parent: &node.Node{
				PartialKey:   []byte{1, 2},
				StorageValue: []byte{3},
				Children: padRightChildren([]*node.Node{
					{ // full key 1, 2, 0, 1, 2
						PartialKey:   []byte{1, 2},
						StorageValue: []byte{3},
					},
				}),
			},
			fullKey: []byte{1, 2, 0, 1, 2},
			encodedProofNodes: [][]byte{
				encodeNode(t, node.Node{
					PartialKey:   []byte{1, 2},
					StorageValue: []byte{3},
					Children: padRightChildren([]*node.Node{
						{ // full key 1, 2, 0, 1, 2
							PartialKey:   []byte{1, 2},
							StorageValue: []byte{3},
						},
					}),
				}),
				// Note the leaf encoding is not added since its encoding
				// is less than 32 bytes.
			},
		},
		"branch_and_matching_search_key_for_large_leaf_encoding": {
			parent: &node.Node{
				PartialKey:   []byte{1, 2},
				StorageValue: []byte{3},
				Children: padRightChildren([]*node.Node{
					{ // full key 1, 2, 0, 1, 2
						PartialKey:   []byte{1, 2},
						StorageValue: largeValue,
					},
				}),
			},
			fullKey: []byte{1, 2, 0, 1, 2},
			encodedProofNodes: [][]byte{
				encodeNode(t, node.Node{
					PartialKey:   []byte{1, 2},
					StorageValue: []byte{3},
					Children: padRightChildren([]*node.Node{
						{ // full key 1, 2, 0, 1, 2
							PartialKey:   []byte{1, 2},
							StorageValue: largeValue,
						},
					}),
				}),
				encodeNode(t, node.Node{
					PartialKey:   []byte{1, 2},
					StorageValue: largeValue,
				}),
			},
		},
		"key_not_found_at_deeper_level": {
			parent: &node.Node{
				PartialKey:   []byte{1, 2},
				StorageValue: []byte{3},
				Children: padRightChildren([]*node.Node{
					{
						PartialKey:   []byte{4, 5},
						StorageValue: []byte{5},
					},
				}),
			},
			fullKey:    []byte{1, 2, 0x04, 4},
			errWrapped: ErrKeyNotFound,
			errMessage: "key not found",
		},
		"found_leaf_at_deeper_level": {
			parent: &node.Node{
				PartialKey:   []byte{1, 2},
				StorageValue: []byte{3},
				Children: padRightChildren([]*node.Node{
					{
						PartialKey:   []byte{4},
						StorageValue: []byte{5},
					},
				}),
			},
			fullKey: []byte{1, 2, 0x04},
			encodedProofNodes: [][]byte{
				encodeNode(t, node.Node{
					PartialKey:   []byte{1, 2},
					StorageValue: []byte{3},
					Children: padRightChildren([]*node.Node{
						{
							PartialKey:   []byte{4},
							StorageValue: []byte{5},
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

			encodedProofNodes, err := walkRoot(testCase.parent, testCase.fullKey)

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

	largeValue := generateBytes(t, 40)
	assertLongEncoding(t, node.Node{StorageValue: largeValue})

	testCases := map[string]struct {
		parent            *node.Node
		fullKey           []byte // nibbles
		encodedProofNodes [][]byte
		errWrapped        error
		errMessage        string
	}{
		"nil_parent_and_empty_full_key": {},
		"nil_parent_and_non_empty_full_key": {
			fullKey:    []byte{1},
			errWrapped: ErrKeyNotFound,
			errMessage: "key not found",
		},
		// The parent encode error cannot be triggered here
		// since it can only be caused by a buffer.Write error.
		"parent_leaf_and_empty_full_key": {
			parent: &node.Node{
				PartialKey:   []byte{1, 2},
				StorageValue: largeValue,
			},
			encodedProofNodes: [][]byte{encodeNode(t, node.Node{
				PartialKey:   []byte{1, 2},
				StorageValue: largeValue,
			})},
		},
		"parent_leaf_and_shorter_full_key": {
			parent: &node.Node{
				PartialKey:   []byte{1, 2},
				StorageValue: []byte{1},
			},
			fullKey:    []byte{1},
			errWrapped: ErrKeyNotFound,
			errMessage: "key not found",
		},
		"parent_leaf_and_mismatching_full_key": {
			parent: &node.Node{
				PartialKey:   []byte{1, 2},
				StorageValue: []byte{1},
			},
			fullKey:    []byte{1, 3},
			errWrapped: ErrKeyNotFound,
			errMessage: "key not found",
		},
		"parent_leaf_and_longer_full_key": {
			parent: &node.Node{
				PartialKey:   []byte{1, 2},
				StorageValue: []byte{1},
			},
			fullKey:    []byte{1, 2, 3},
			errWrapped: ErrKeyNotFound,
			errMessage: "key not found",
		},
		"branch_and_empty_search_key": {
			parent: &node.Node{
				PartialKey:   []byte{1, 2},
				StorageValue: largeValue,
				Children: padRightChildren([]*node.Node{
					{
						PartialKey:   []byte{4},
						StorageValue: []byte{5},
					},
				}),
			},
			encodedProofNodes: [][]byte{
				encodeNode(t, node.Node{
					PartialKey:   []byte{1, 2},
					StorageValue: largeValue,
					Children: padRightChildren([]*node.Node{
						{
							PartialKey:   []byte{4},
							StorageValue: []byte{5},
						},
					}),
				}),
			},
		},
		"branch_and_shorter_full_key": {
			parent: &node.Node{
				PartialKey:   []byte{1, 2},
				StorageValue: []byte{3},
				Children: padRightChildren([]*node.Node{
					{
						PartialKey:   []byte{4},
						StorageValue: []byte{5},
					},
				}),
			},
			fullKey:    []byte{1},
			errWrapped: ErrKeyNotFound,
			errMessage: "key not found",
		},
		"branch_and_mismatching_full_key": {
			parent: &node.Node{
				PartialKey:   []byte{1, 2},
				StorageValue: []byte{3},
				Children: padRightChildren([]*node.Node{
					{
						PartialKey:   []byte{4},
						StorageValue: []byte{5},
					},
				}),
			},
			fullKey:    []byte{1, 3},
			errWrapped: ErrKeyNotFound,
			errMessage: "key not found",
		},
		"branch_and_matching_search_key": {
			parent: &node.Node{
				PartialKey:   []byte{1, 2},
				StorageValue: []byte{3},
				Children: padRightChildren([]*node.Node{
					{
						PartialKey:   []byte{4},
						StorageValue: largeValue,
					},
				}),
			},
			fullKey: []byte{1, 2},
			encodedProofNodes: [][]byte{
				encodeNode(t, node.Node{
					PartialKey:   []byte{1, 2},
					StorageValue: []byte{3},
					Children: padRightChildren([]*node.Node{
						{
							PartialKey:   []byte{4},
							StorageValue: largeValue,
						},
					}),
				}),
			},
		},
		"branch_and_matching_search_key_for_small_leaf_encoding": {
			parent: &node.Node{
				PartialKey:   []byte{1, 2},
				StorageValue: largeValue,
				Children: padRightChildren([]*node.Node{
					{ // full key 1, 2, 0, 1, 2
						PartialKey:   []byte{1, 2},
						StorageValue: []byte{3},
					},
				}),
			},
			fullKey: []byte{1, 2, 0, 1, 2},
			encodedProofNodes: [][]byte{
				encodeNode(t, node.Node{
					PartialKey:   []byte{1, 2},
					StorageValue: largeValue,
					Children: padRightChildren([]*node.Node{
						{ // full key 1, 2, 0, 1, 2
							PartialKey:   []byte{1, 2},
							StorageValue: []byte{3},
						},
					}),
				}),
				// Note the leaf encoding is not added since its encoding
				// is less than 32 bytes.
			},
		},
		"branch_and_matching_search_key_for_large_leaf_encoding": {
			parent: &node.Node{
				PartialKey:   []byte{1, 2},
				StorageValue: []byte{3},
				Children: padRightChildren([]*node.Node{
					{ // full key 1, 2, 0, 1, 2
						PartialKey:   []byte{1, 2},
						StorageValue: largeValue,
					},
				}),
			},
			fullKey: []byte{1, 2, 0, 1, 2},
			encodedProofNodes: [][]byte{
				encodeNode(t, node.Node{
					PartialKey:   []byte{1, 2},
					StorageValue: []byte{3},
					Children: padRightChildren([]*node.Node{
						{ // full key 1, 2, 0, 1, 2
							PartialKey:   []byte{1, 2},
							StorageValue: largeValue,
						},
					}),
				}),
				encodeNode(t, node.Node{
					PartialKey:   []byte{1, 2},
					StorageValue: largeValue,
				}),
			},
		},
		"key_not_found_at_deeper_level": {
			parent: &node.Node{
				PartialKey:   []byte{1, 2},
				StorageValue: []byte{3},
				Children: padRightChildren([]*node.Node{
					{
						PartialKey:   []byte{4, 5},
						StorageValue: []byte{5},
					},
				}),
			},
			fullKey:    []byte{1, 2, 0x04, 4},
			errWrapped: ErrKeyNotFound,
			errMessage: "key not found",
		},
		"found_leaf_at_deeper_level": {
			parent: &node.Node{
				PartialKey:   []byte{1, 2},
				StorageValue: []byte{3},
				Children: padRightChildren([]*node.Node{
					{
						PartialKey:   []byte{4},
						StorageValue: largeValue,
					},
				}),
			},
			fullKey: []byte{1, 2, 0x04},
			encodedProofNodes: [][]byte{
				encodeNode(t, node.Node{
					PartialKey:   []byte{1, 2},
					StorageValue: []byte{3},
					Children: padRightChildren([]*node.Node{
						{
							PartialKey:   []byte{4},
							StorageValue: largeValue,
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
		"nil_slices": {},
		"empty_slices": {
			a: []byte{},
			b: []byte{},
		},
		"fully_different": {
			a: []byte{1, 2, 3},
			b: []byte{4, 5, 6},
		},
		"fully_same": {
			a:      []byte{1, 2, 3},
			b:      []byte{1, 2, 3},
			length: 3,
		},
		"different_and_common_prefix": {
			a:      []byte{1, 2, 3, 4},
			b:      []byte{1, 2, 4, 4},
			length: 2,
		},
		"first_bigger_than_second": {
			a:      []byte{1, 2, 3},
			b:      []byte{1, 2},
			length: 2,
		},
		"first_smaller_than_second": {
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

// Note on the performance of walk:
// It was tried to optimise appending to the encodedProofNodes
// slice by:
// 1. appending to the same slice *[][]byte passed as argument
// 2. appending the upper node to the deeper nodes slice
// In both cases, the performance difference is very small
// so the code is kept to this inefficient-looking append,
// which is in the end quite performant still.
func Benchmark_walkRoot(b *testing.B) {
	trie := trie.NewEmptyTrie()

	// Build a deep trie.
	const trieDepth = 1000
	for i := 0; i < trieDepth; i++ {
		keySize := 1 + i
		key := make([]byte, keySize)
		const trieValueSize = 10
		value := make([]byte, trieValueSize)

		trie.Put(key, value)
	}

	longestKeyLE := make([]byte, trieDepth)
	longestKeyNibbles := codec.KeyLEToNibbles(longestKeyLE)

	rootNode := trie.RootNode()
	encodedProofNodes, err := walkRoot(rootNode, longestKeyNibbles)
	require.NoError(b, err)
	require.Equal(b, len(encodedProofNodes), trieDepth)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = walkRoot(rootNode, longestKeyNibbles)
	}
}
