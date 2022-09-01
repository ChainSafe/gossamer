// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package babe

import (
	"testing"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/genesis"
	"github.com/ChainSafe/gossamer/lib/trie"
	"github.com/ChainSafe/gossamer/lib/utils"
	"github.com/stretchr/testify/require"
)

// newDevGenesisWithTrieAndHeader generates test dev genesis, genesis trie and genesis header
func newDevGenesisWithTrieAndHeader(t *testing.T) (gen *genesis.Genesis, genesisTrie *trie.Trie, header *types.Header) {
	genesisPath := utils.GetDevV3SubstrateGenesisPath(t)

	gen, err := genesis.NewGenesisFromJSONRaw(genesisPath)
	require.NoError(t, err)

	genesisTrie, err = genesis.NewTrieFromGenesis(gen)
	require.NoError(t, err)

	genesisHeader, err := types.NewHeader(common.NewHash([]byte{0}),
		genesisTrie.MustHash(), trie.EmptyHash, 0, types.NewDigest())
	require.NoError(t, err)

	return gen, genesisTrie, genesisHeader
}

func newTestGenesisWithTrieAndHeader(t *testing.T) (
	gen *genesis.Genesis, genesisTrie *trie.Trie, genesisHeader *types.Header) {
	genesisPath := utils.GetGssmrV3SubstrateGenesisRawPathTest(t)
	gen, err := genesis.NewGenesisFromJSONRaw(genesisPath)
	require.NoError(t, err)

	genesisTrie, err = genesis.NewTrieFromGenesis(gen)
	require.NoError(t, err)

	parentHash := common.NewHash([]byte{0})
	stateRoot := genesisTrie.MustHash()
	extrinsicRoot := trie.EmptyHash
	const number = 0
	digest := types.NewDigest()
	genesisHeader, err = types.NewHeader(parentHash,
		stateRoot, extrinsicRoot, number, digest)
	require.NoError(t, err)

	return gen, genesisTrie, genesisHeader
}
