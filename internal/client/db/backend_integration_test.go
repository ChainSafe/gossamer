// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

//go:build integration

package db

import (
	"testing"

	p_blockchain "github.com/ChainSafe/gossamer/internal/primitives/blockchain"
	"github.com/ChainSafe/gossamer/internal/primitives/core/hash"
	"github.com/stretchr/testify/require"
)

func TestBackend_Integration(t *testing.T) {
	t.Run("tree_route_regression", func(t *testing.T) {
		// NOTE: this is a test for a regression introduced in #3665, the result
		// of tree_route would be erroneously computed, since it was taking into
		// account the ancestor in CachedHeaderMetadata for the comparison.
		// in this test we simulate the same behavior with the side-effect
		// triggering the issue being eviction of a previously fetched record
		// from the cache, therefore this test is dependent on the LRU cache
		// size for header metadata, which is currently set to 5000 elements.
		backend := NewTestBackend(t, BlocksPruningSome(10000), 10000)
		blockchain := backend.blockchain

		genesis := insertHeader(t, backend, 0, hash.H256(""), nil, hash.H256(""))

		parent := genesis
		for i := uint64(1); i <= 100; i++ {
			parent = insertHeader(t, backend, i, parent, nil, hash.H256(""))
		}
		block100 := parent

		for i := uint64(101); i <= 7000; i++ {
			parent = insertHeader(t, backend, i, parent, nil, hash.H256(""))
		}
		block7000 := parent

		// This will cause the ancestor of block100 to be set to genesis as a side-effect.
		_, err := p_blockchain.LowestCommonAncestor(blockchain, genesis, block100)
		require.NoError(t, err)

		// While traversing the tree we will have to do 6900 calls to
		// HeaderMetadata, which will make sure we will exhaust our cache
		// which only takes 5000 elements. In particular, the CachedHeaderMetadata struct for
		// block #100 will be evicted and will get a new value (with ancestor set to its parent).
		treeRoute, err := p_blockchain.NewTreeRoute(blockchain, block100, block7000)
		require.NoError(t, err)

		require.Empty(t, treeRoute.Retracted())
	})
}
