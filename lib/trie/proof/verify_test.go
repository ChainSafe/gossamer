// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package proof

import (
	"testing"

	"github.com/ChainSafe/gossamer/internal/trie/node"
	"github.com/ChainSafe/gossamer/lib/trie"
	"github.com/ChainSafe/gossamer/lib/trie/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_Verify(t *testing.T) {
	t.Parallel()

	leafA := node.Node{
		PartialKey:   []byte{1},
		StorageValue: []byte{1},
	}

	// leafB is a leaf encoding to more than 32 bytes
	leafB := node.Node{
		PartialKey:   []byte{2},
		StorageValue: generateBytes(t, 40),
	}
	assertLongEncoding(t, leafB)

	branch := node.Node{
		PartialKey:   []byte{3, 4},
		StorageValue: []byte{1},
		Children: padRightChildren([]*node.Node{
			&leafB,
			nil,
			&leafA,
			&leafB,
		}),
	}
	assertLongEncoding(t, branch)

	testCases := map[string]struct {
		encodedProofNodes [][]byte
		rootHash          []byte
		keyLE             []byte
		value             []byte
		errWrapped        error
		errMessage        string
	}{
		"failed_building_proof_trie": {
			rootHash:   []byte{1, 2, 3},
			errWrapped: ErrEmptyProof,
			errMessage: "building trie from proof encoded nodes: " +
				"proof slice empty: for Merkle root hash 0x010203",
		},
		"value_not_found": {
			encodedProofNodes: [][]byte{
				encodeNode(t, branch),
				encodeNode(t, leafB),
				// Note leaf A is small enough to be inlined in branch
			},
			rootHash:   blake2bNode(t, branch),
			keyLE:      []byte{1, 1}, // nil child of branch
			errWrapped: ErrKeyNotFoundInProofTrie,
			errMessage: "key not found in proof trie: " +
				"0x0101 in proof trie for root hash " +
				"0xec4bb0acfcf778ae8746d3ac3325fc73c3d9b376eb5f8d638dbf5eb462f5e703",
		},
		"key_found_with_nil_search_value": {
			encodedProofNodes: [][]byte{
				encodeNode(t, branch),
				encodeNode(t, leafB),
				// Note leaf A is small enough to be inlined in branch
			},
			rootHash: blake2bNode(t, branch),
			keyLE:    []byte{0x34, 0x21}, // inlined short leaf of branch
		},
		"key_found_with_mismatching_value": {
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
		"key_found_with_matching_value": {
			encodedProofNodes: [][]byte{
				encodeNode(t, branch),
				encodeNode(t, leafB),
				// Note leaf A is small enough to be inlined in branch
			},
			rootHash: blake2bNode(t, branch),
			keyLE:    []byte{0x34, 0x32}, // large hash-referenced leaf of branch
			value:    generateBytes(t, 40),
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			err := Verify(testCase.encodedProofNodes, testCase.rootHash, testCase.keyLE, testCase.value, nil)

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
		PartialKey:   []byte{1},
		StorageValue: []byte{2},
	}
	assertShortEncoding(t, leafAShort)

	leafBLarge := node.Node{
		PartialKey:   []byte{2},
		StorageValue: generateBytes(t, 40),
	}
	assertLongEncoding(t, leafBLarge)

	leafCLarge := node.Node{
		PartialKey:   []byte{3},
		StorageValue: generateBytes(t, 40),
	}
	assertLongEncoding(t, leafCLarge)

	type testCase struct {
		encodedProofNodes [][]byte
		rootHash          []byte
		expectedTrie      *trie.Trie
		db                db.Database
		errWrapped        error
		errMessage        string
	}

	testCases := map[string]testCase{
		"no_proof_node": {
			errWrapped: ErrEmptyProof,
			rootHash:   []byte{1},
			errMessage: "proof slice empty: for Merkle root hash 0x01",
		},
		"root_node_decoding_error": {
			encodedProofNodes: [][]byte{
				getBadNodeEncoding(),
			},
			rootHash:   blake2b(t, getBadNodeEncoding()),
			errWrapped: node.ErrVariantUnknown,
			errMessage: "decoding root node: decoding header: " +
				"decoding header byte: node variant is unknown: " +
				"for header byte 00000011",
		},
		"root_proof_encoding_smaller_than_32_bytes": func() testCase {
			encodedProofNodes := [][]byte{
				encodeNode(t, leafAShort),
			}

			proofDB, err := db.NewMemoryDBFromProof(encodedProofNodes)
			assert.NoError(t, err)

			return testCase{
				encodedProofNodes: encodedProofNodes,
				rootHash:          blake2bNode(t, leafAShort),
				db:                proofDB,
				expectedTrie: trie.NewTrie(&node.Node{
					PartialKey:   leafAShort.PartialKey,
					StorageValue: leafAShort.StorageValue,
					Dirty:        true,
				}, proofDB),
			}
		}(),
		"root_proof_encoding_larger_than_32_bytes": func() testCase {
			encodedProofNodes := [][]byte{
				encodeNode(t, leafBLarge),
			}

			proofDB, err := db.NewMemoryDBFromProof(encodedProofNodes)
			assert.NoError(t, err)

			return testCase{
				encodedProofNodes: encodedProofNodes,
				rootHash:          blake2bNode(t, leafBLarge),
				db:                proofDB,
				expectedTrie: trie.NewTrie(&node.Node{
					PartialKey:   leafBLarge.PartialKey,
					StorageValue: leafBLarge.StorageValue,
					Dirty:        true,
				}, proofDB),
			}
		}(),
		"discard_unused_node": func() testCase {
			encodedProofNodes := [][]byte{
				encodeNode(t, leafAShort),
				encodeNode(t, leafBLarge),
			}

			proofDB, err := db.NewMemoryDBFromProof(encodedProofNodes)
			assert.NoError(t, err)

			return testCase{
				encodedProofNodes: encodedProofNodes,
				rootHash:          blake2bNode(t, leafAShort),
				db:                proofDB,
				expectedTrie: trie.NewTrie(&node.Node{
					PartialKey:   leafAShort.PartialKey,
					StorageValue: leafAShort.StorageValue,
					Dirty:        true,
				}, proofDB),
			}
		}(),
		"multiple_unordered_nodes": func() testCase {
			encodedProofNodes := [][]byte{
				encodeNode(t, leafBLarge), // chilren 1 and 3
				encodeNode(t, node.Node{ // root
					PartialKey: []byte{1},
					Children: padRightChildren([]*node.Node{
						&leafAShort, // inlined
						&leafBLarge, // referenced by Merkle value hash
						&leafCLarge, // referenced by Merkle value hash
						&leafBLarge, // referenced by Merkle value hash
					}),
				}),
				encodeNode(t, leafCLarge), // children 2
			}

			proofDB, err := db.NewMemoryDBFromProof(encodedProofNodes)
			assert.NoError(t, err)

			return testCase{
				encodedProofNodes: encodedProofNodes,
				rootHash: blake2bNode(t, node.Node{
					PartialKey: []byte{1},
					Children: padRightChildren([]*node.Node{
						&leafAShort,
						&leafBLarge,
						&leafCLarge,
						&leafBLarge,
					}),
				}),
				db: proofDB,
				expectedTrie: trie.NewTrie(&node.Node{
					PartialKey:  []byte{1},
					Descendants: 4,
					Dirty:       true,
					Children: padRightChildren([]*node.Node{
						{
							PartialKey:   leafAShort.PartialKey,
							StorageValue: leafAShort.StorageValue,
							Dirty:        true,
						},
						{
							PartialKey:   leafBLarge.PartialKey,
							StorageValue: leafBLarge.StorageValue,
							Dirty:        true,
						},
						{
							PartialKey:   leafCLarge.PartialKey,
							StorageValue: leafCLarge.StorageValue,
							Dirty:        true,
						},
						{
							PartialKey:   leafBLarge.PartialKey,
							StorageValue: leafBLarge.StorageValue,
							Dirty:        true,
						},
					}),
				}, proofDB),
			}
		}(),
		"load_proof_decoding_error": {
			encodedProofNodes: [][]byte{
				getBadNodeEncoding(),
				// root with one child pointing to hash of bad encoding above.
				concatBytes([][]byte{
					{0b1000_0000 | 0b0000_0001}, // branch with key size 1
					{1},                         // key
					{0b0000_0001, 0b0000_0000},  // children bitmap
					scaleEncode(t, blake2b(t, getBadNodeEncoding())), // child hash
				}),
			},
			rootHash: blake2b(t, concatBytes([][]byte{
				{0b1000_0000 | 0b0000_0001}, // branch with key size 1
				{1},                         // key
				{0b0000_0001, 0b0000_0000},  // children bitmap
				scaleEncode(t, blake2b(t, getBadNodeEncoding())), // child hash
			})),
			errWrapped: node.ErrVariantUnknown,
			errMessage: "loading proof: decoding child node for hash digest " +
				"0xa111753e9152fe5204e77af20c46f055cb90f56212249d6cbf265395e689a8ed: " +
				"decoding header: decoding header byte: " +
				"node variant is unknown: for header byte 00000011",
		},
		"root_not_found": {
			encodedProofNodes: [][]byte{
				encodeNode(t, node.Node{
					PartialKey:   []byte{1},
					StorageValue: []byte{2},
				}),
			},
			rootHash:   []byte{3},
			errWrapped: ErrRootNodeNotFound,
			errMessage: "root node not found in proof: " +
				"for root hash 0x03 in proof hash digests " +
				"0x60516d0bb6e1bbfb1293f1b276ea9505e9f4a4e7d98f620d05115e0b85274ae1",
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			trie, err := buildTrie(testCase.encodedProofNodes, testCase.rootHash, testCase.db)

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

	largeValue := generateBytes(t, 40)

	leafLarge := node.Node{
		PartialKey:   []byte{3},
		StorageValue: largeValue,
	}
	assertLongEncoding(t, leafLarge)

	testCases := map[string]struct {
		merkleValueToEncoding map[string][]byte
		node                  *node.Node
		expectedNode          *node.Node
		errWrapped            error
		errMessage            string
	}{
		"leaf_node": {
			node: &node.Node{
				PartialKey:   []byte{1},
				StorageValue: []byte{2},
			},
			expectedNode: &node.Node{
				PartialKey:   []byte{1},
				StorageValue: []byte{2},
			},
		},
		"branch_node_with_child_hash_not_found": {
			node: &node.Node{
				PartialKey:   []byte{1},
				StorageValue: []byte{2},
				Descendants:  1,
				Dirty:        true,
				Children: padRightChildren([]*node.Node{
					{MerkleValue: []byte{3}},
				}),
			},
			merkleValueToEncoding: map[string][]byte{},
			expectedNode: &node.Node{
				PartialKey:   []byte{1},
				StorageValue: []byte{2},
				Dirty:        true,
			},
		},
		"branch_node_with_child_hash_found": {
			node: &node.Node{
				PartialKey:   []byte{1},
				StorageValue: []byte{2},
				Descendants:  1,
				Dirty:        true,
				Children: padRightChildren([]*node.Node{
					{MerkleValue: []byte{2}},
				}),
			},
			merkleValueToEncoding: map[string][]byte{
				string([]byte{2}): encodeNode(t, node.Node{
					PartialKey:   []byte{3},
					StorageValue: []byte{1},
				}),
			},
			expectedNode: &node.Node{
				PartialKey:   []byte{1},
				StorageValue: []byte{2},
				Descendants:  1,
				Dirty:        true,
				Children: padRightChildren([]*node.Node{
					{
						PartialKey:   []byte{3},
						StorageValue: []byte{1},
						Dirty:        true,
					},
				}),
			},
		},
		"branch_node_with_one_child_hash_found_and_one_not_found": {
			node: &node.Node{
				PartialKey:   []byte{1},
				StorageValue: []byte{2},
				Descendants:  2,
				Dirty:        true,
				Children: padRightChildren([]*node.Node{
					{MerkleValue: []byte{2}}, // found
					{MerkleValue: []byte{3}}, // not found
				}),
			},
			merkleValueToEncoding: map[string][]byte{
				string([]byte{2}): encodeNode(t, node.Node{
					PartialKey:   []byte{3},
					StorageValue: []byte{1},
				}),
			},
			expectedNode: &node.Node{
				PartialKey:   []byte{1},
				StorageValue: []byte{2},
				Descendants:  1,
				Dirty:        true,
				Children: padRightChildren([]*node.Node{
					{
						PartialKey:   []byte{3},
						StorageValue: []byte{1},
						Dirty:        true,
					},
				}),
			},
		},
		"branch_node_with_branch_child_hash": {
			node: &node.Node{
				PartialKey:   []byte{1},
				StorageValue: []byte{2},
				Descendants:  2,
				Dirty:        true,
				Children: padRightChildren([]*node.Node{
					{MerkleValue: []byte{2}},
				}),
			},
			merkleValueToEncoding: map[string][]byte{
				string([]byte{2}): encodeNode(t, node.Node{
					PartialKey:   []byte{3},
					StorageValue: []byte{1},
					Children: padRightChildren([]*node.Node{
						{PartialKey: []byte{4}, StorageValue: []byte{2}},
					}),
				}),
			},
			expectedNode: &node.Node{
				PartialKey:   []byte{1},
				StorageValue: []byte{2},
				Descendants:  3,
				Dirty:        true,
				Children: padRightChildren([]*node.Node{
					{
						PartialKey:   []byte{3},
						StorageValue: []byte{1},
						Dirty:        true,
						Descendants:  1,
						Children: padRightChildren([]*node.Node{
							{
								PartialKey:   []byte{4},
								StorageValue: []byte{2},
								Dirty:        true,
							},
						}),
					},
				}),
			},
		},
		"child_decoding_error": {
			node: &node.Node{
				PartialKey:   []byte{1},
				StorageValue: []byte{2},
				Descendants:  1,
				Dirty:        true,
				Children: padRightChildren([]*node.Node{
					{MerkleValue: []byte{2}},
				}),
			},
			merkleValueToEncoding: map[string][]byte{
				string([]byte{2}): getBadNodeEncoding(),
			},
			expectedNode: &node.Node{
				PartialKey:   []byte{1},
				StorageValue: []byte{2},
				Descendants:  1,
				Dirty:        true,
				Children: padRightChildren([]*node.Node{
					{MerkleValue: []byte{2}},
				}),
			},
			errWrapped: node.ErrVariantUnknown,
			errMessage: "decoding child node for hash digest 0x02: " +
				"decoding header: decoding header byte: node variant is unknown: " +
				"for header byte 00000011",
		},
		"grand_child": {
			node: &node.Node{
				PartialKey:   []byte{1},
				StorageValue: []byte{1},
				Descendants:  1,
				Dirty:        true,
				Children: padRightChildren([]*node.Node{
					{MerkleValue: []byte{2}},
				}),
			},
			merkleValueToEncoding: map[string][]byte{
				string([]byte{2}): encodeNode(t, node.Node{
					PartialKey:   []byte{2},
					StorageValue: []byte{2},
					Descendants:  1,
					Dirty:        true,
					Children: padRightChildren([]*node.Node{
						&leafLarge, // encoded to hash
					}),
				}),
				string(blake2bNode(t, leafLarge)): encodeNode(t, leafLarge),
			},
			expectedNode: &node.Node{
				PartialKey:   []byte{1},
				StorageValue: []byte{1},
				Descendants:  2,
				Dirty:        true,
				Children: padRightChildren([]*node.Node{
					{
						PartialKey:   []byte{2},
						StorageValue: []byte{2},
						Descendants:  1,
						Dirty:        true,
						Children: padRightChildren([]*node.Node{
							{
								PartialKey:   leafLarge.PartialKey,
								StorageValue: leafLarge.StorageValue,
								Dirty:        true,
							},
						}),
					},
				}),
			},
		},
		"grand_child_load_proof_error": {
			node: &node.Node{
				PartialKey:   []byte{1},
				StorageValue: []byte{1},
				Descendants:  1,
				Dirty:        true,
				Children: padRightChildren([]*node.Node{
					{MerkleValue: []byte{2}},
				}),
			},
			merkleValueToEncoding: map[string][]byte{
				string([]byte{2}): encodeNode(t, node.Node{
					PartialKey:   []byte{2},
					StorageValue: []byte{2},
					Descendants:  1,
					Dirty:        true,
					Children: padRightChildren([]*node.Node{
						&leafLarge, // encoded to hash
					}),
				}),
				string(blake2bNode(t, leafLarge)): getBadNodeEncoding(),
			},
			expectedNode: &node.Node{
				PartialKey:   []byte{1},
				StorageValue: []byte{1},
				Descendants:  2,
				Dirty:        true,
				Children: padRightChildren([]*node.Node{
					{
						PartialKey:   []byte{2},
						StorageValue: []byte{2},
						Descendants:  1,
						Dirty:        true,
						Children: padRightChildren([]*node.Node{
							{
								MerkleValue: blake2bNode(t, leafLarge),
							},
						}),
					},
				}),
			},
			errWrapped: node.ErrVariantUnknown,
			errMessage: "decoding child node for hash digest " +
				"0x6888b9403129c11350c6054b46875292c0ffedcfd581e66b79bdf350b775ebf2: " +
				"decoding header: decoding header byte: node variant is unknown: " +
				"for header byte 00000011",
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			err := loadProof(testCase.merkleValueToEncoding, testCase.node)

			assert.ErrorIs(t, err, testCase.errWrapped)
			if testCase.errWrapped != nil {
				assert.EqualError(t, err, testCase.errMessage)
			}

			assert.Equal(t, testCase.expectedNode.String(), testCase.node.String())
		})
	}
}

func Test_bytesToString(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		b []byte
		s string
	}{
		"nil_slice": {
			s: "nil",
		},
		"empty_slice": {
			b: []byte{},
			s: "0x",
		},
		"small_slice": {
			b: []byte{1, 2, 3},
			s: "0x010203",
		},
		"big_slice": {
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
