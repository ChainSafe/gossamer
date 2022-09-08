// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only
package modules

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

func stringToHex(s string) (hex string) {
	return common.BytesToHex([]byte(s))
}

func makeChange(keyHex, valueHex string) [2]*string {
	return [2]*string{&keyHex, &valueHex}
}

func newTestGenesisWithTrieAndHeader(t *testing.T) (
	gen genesis.Genesis, genesisTrie trie.Trie, genesisHeader types.Header,
	stateVersion trie.Version) {
	t.Helper()

	genesisPath := utils.GetGssmrV3SubstrateGenesisRawPathTest(t)
	genesisPtr, err := genesis.NewGenesisFromJSONRaw(genesisPath)
	require.NoError(t, err)
	gen = *genesisPtr

	genesisTrie, err = wasmer.NewTrieFromGenesis(gen)
	require.NoError(t, err)

	stateVersion, err = wasmer.StateVersionFromGenesis(gen)
	require.NoError(t, err)

	parentHash := common.NewHash([]byte{0})
	stateRoot := genesisTrie.MustHash(stateVersion)
	extrinsicRoot := trie.EmptyHash
	const number = 0
	digest := types.NewDigest()
	genesisHeaderPtr, err := types.NewHeader(parentHash,
		stateRoot, extrinsicRoot, number, digest)
	require.NoError(t, err)
	genesisHeader = *genesisHeaderPtr

	return gen, genesisTrie, genesisHeader, stateVersion
}
