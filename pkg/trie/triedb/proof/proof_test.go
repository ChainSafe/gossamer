// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package proof

import (
	"fmt"
	"testing"

	"github.com/ChainSafe/gossamer/internal/primitives/core/hash"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime"
	"github.com/ChainSafe/gossamer/pkg/trie"
	"github.com/ChainSafe/gossamer/pkg/trie/triedb"
	"github.com/stretchr/testify/require"
)

func Test_GenerateAndVerify(t *testing.T) {
	testCases := map[string]struct {
		entries       []trie.Entry
		keys          []string
		expectedProof MerkleProof[hash.H256, runtime.BlakeTwo256]
	}{
		"leaf": {
			entries: []trie.Entry{
				{
					Key:   []byte("a"),
					Value: []byte("a"),
				},
			},
			keys: []string{"a"},
			expectedProof: MerkleProof[hash.H256, runtime.BlakeTwo256]{
				{66, 97, 0}, // 'a' node without value
			},
		},
		"branch_and_leaf": {
			entries: []trie.Entry{
				{
					Key:   []byte("a"),
					Value: []byte("a"),
				},
				{
					Key:   []byte("ab"),
					Value: []byte("ab"),
				},
			},
			keys: []string{"ab"},
			expectedProof: MerkleProof[hash.H256, runtime.BlakeTwo256]{
				{194, 97, 64, 0, 4, 97, 12, 65, 2, 0},
			},
		},
		"complex_trie": {
			entries: []trie.Entry{
				{
					Key:   []byte("pol"),
					Value: []byte("pol"),
				},
				{
					Key:   []byte("polka"),
					Value: []byte("polka"),
				},
				{
					Key:   []byte("polkadot"),
					Value: []byte("polkadot"),
				},
				{
					Key:   []byte("go"),
					Value: []byte("go"),
				},
				{
					Key:   []byte("golang"),
					Value: []byte("golang"),
				},
				{
					Key:   []byte("gossamer"),
					Value: []byte("gossamer"),
				},
			},
			keys: []string{"go"},
			expectedProof: MerkleProof[hash.H256, runtime.BlakeTwo256]{
				{
					128, 192, 0, 0, 128, 114, 166, 121, 79, 225, 146, 229,
					34, 68, 211, 54, 148, 205, 192, 58, 131, 95, 46, 239,
					201, 206, 94, 116, 179, 122, 33, 19, 156, 225, 190, 57, 57,
				},
				{
					131, 7, 111, 192, 0, 48, 71, 12, 97, 110, 103, 24, 103,
					111, 108, 97, 110, 103, 64, 75, 3, 115, 97, 109, 101,
					114, 32, 103, 111, 115, 115, 97, 109, 101, 114,
				},
			},
		},
		"complex_trie_multiple_keys": {
			entries: []trie.Entry{
				{
					Key:   []byte("pol"),
					Value: []byte("pol"),
				},
				{
					Key:   []byte("polka"),
					Value: []byte("polka"),
				},
				{
					Key:   []byte("polkadot"),
					Value: []byte("polkadot"),
				},
				{
					Key:   []byte("go"),
					Value: []byte("go"),
				},
				{
					Key:   []byte("golang"),
					Value: []byte("golang"),
				},
				{
					Key:   []byte("gossamer"),
					Value: []byte("gossamer"),
				},
			},
			keys: []string{"go", "polkadot"},
			expectedProof: MerkleProof[hash.H256, runtime.BlakeTwo256]{
				{
					128, 192, 0, 0, 0,
				},
				{
					131, 7, 111, 192, 0, 48, 71, 12, 97, 110, 103, 24,
					103, 111, 108, 97, 110, 103, 64, 75, 3, 115, 97,
					109, 101, 114, 32, 103, 111, 115, 115, 97, 109,
					101, 114,
				},
				{
					197, 0, 111, 108, 64, 0, 12, 112, 111, 108, 68,
					195, 11, 97, 64, 0, 20, 112, 111, 108, 107, 97,
					20, 69, 4, 111, 116, 0,
				},
			},
		},
	}

	for name, testCase := range testCases {
		trieVersions := []trie.TrieLayout{trie.V0, trie.V1}

		for _, trieVersion := range trieVersions {
			t.Run(fmt.Sprintf("%s_%s", name, trieVersion.String()), func(t *testing.T) {
				// Build trie
				inmemoryDB := NewMemoryDB(triedb.EmptyNode)
				triedb := triedb.NewEmptyTrieDB[hash.H256, runtime.BlakeTwo256](inmemoryDB)
				triedb.SetVersion(trieVersion)

				for _, entry := range testCase.entries {
					triedb.Put(entry.Key, entry.Value)
				}

				root := triedb.MustHash()

				// Generate proof
				proof, err := NewMerkleProof[hash.H256, runtime.BlakeTwo256](inmemoryDB, trieVersion, root, testCase.keys)
				require.NoError(t, err)
				require.Equal(t, testCase.expectedProof, proof)

				// Verify proof
				items := make([]proofItem, len(testCase.keys))
				for i, key := range testCase.keys {
					items[i] = proofItem{
						key:   []byte(key),
						value: triedb.Get([]byte(key)),
					}
				}
				err = proof.Verify(trieVersion, root.Bytes(), items)

				require.NoError(t, err)
			})
		}

	}
}
