// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package inmemory

import (
	"testing"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/trie/node"
	"github.com/stretchr/testify/assert"
)

func Test_InMemoryTrie_GenesisBlock(t *testing.T) {
	t.Parallel()

	withHash := func(header types.Header) types.Header {
		header.Hash()
		return header
	}

	testCases := map[string]struct {
		trie          InMemoryTrie
		genesisHeader types.Header
		errSentinel   error
		errMessage    string
	}{
		"empty_trie": {
			genesisHeader: withHash(types.Header{
				ParentHash:     common.Hash{0},
				StateRoot:      EmptyHash,
				ExtrinsicsRoot: EmptyHash,
				Digest:         types.NewDigest(),
			}),
		},
		"non_empty_trie": {
			trie: InMemoryTrie{
				root: &node.Node{
					PartialKey:   []byte{1, 2, 3},
					StorageValue: []byte{4, 5, 6},
				},
			},
			genesisHeader: withHash(types.Header{
				ParentHash: common.Hash{0},
				StateRoot: common.Hash{
					0x25, 0xc1, 0x86, 0xd4, 0x5b, 0xc9, 0x1d, 0x9f,
					0xf5, 0xfd, 0x29, 0xd3, 0x29, 0x8a, 0xa3, 0x63,
					0x83, 0xf3, 0x2d, 0x14, 0xa8, 0xbd, 0xde, 0xc9,
					0x7b, 0x57, 0x92, 0x78, 0x67, 0xfc, 0x8a, 0xfa},
				ExtrinsicsRoot: EmptyHash,
				Digest:         types.NewDigest(),
			}),
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			trie := testCase.trie

			genesisHeader, err := trie.GenesisBlock()

			assert.ErrorIs(t, err, testCase.errSentinel)
			if testCase.errSentinel != nil {
				assert.EqualError(t, err, testCase.errMessage)
			}
			assert.Equal(t, testCase.genesisHeader, genesisHeader)
		})
	}
}
