// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package rpc

import (
	"testing"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/genesis"
	"github.com/ChainSafe/gossamer/lib/runtime/wasmer"
	"github.com/ChainSafe/gossamer/lib/trie"
	"github.com/ChainSafe/gossamer/lib/utils"
	"github.com/stretchr/testify/require"
)

func newTestGenesisWithTrieAndHeader(t *testing.T) (
	gen *genesis.Genesis, genesisTrie *trie.Trie, genesisHeader *types.Header) {
	genesisPath := utils.GetGssmrV3SubstrateGenesisRawPathTest(t)
	gen, err := genesis.NewGenesisFromJSONRaw(genesisPath)
	require.NoError(t, err)

	genesisTrie, err = wasmer.NewTrieFromGenesis(*gen)
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
