// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only
package modules

import (
	"testing"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/genesis"
	"github.com/ChainSafe/gossamer/lib/runtime"
	"github.com/ChainSafe/gossamer/lib/utils"
	"github.com/ChainSafe/gossamer/pkg/trie"
	"github.com/stretchr/testify/require"
)

func stringToHex(s string) (hex string) {
	return common.BytesToHex([]byte(s))
}

func makeChange(keyHex, valueHex string) [2]*string {
	return [2]*string{&keyHex, &valueHex}
}

func newWestendLocalGenesisWithTrieAndHeader(t *testing.T) (
	gen genesis.Genesis, genesisTrie *trie.InMemoryTrie, genesisHeader types.Header) {
	t.Helper()

	genesisPath := utils.GetWestendLocalRawGenesisPath(t)
	genesisPtr, err := genesis.NewGenesisFromJSONRaw(genesisPath)
	require.NoError(t, err)
	gen = *genesisPtr

	genesisTrie, err = runtime.NewInMemoryTrieFromGenesis(gen)
	require.NoError(t, err)

	parentHash := common.NewHash([]byte{0})
	stateRoot := genesisTrie.MustHash()
	extrinsicRoot := trie.EmptyHash
	const number = 0
	digest := types.NewDigest()
	genesisHeader = *types.NewHeader(parentHash,
		stateRoot, extrinsicRoot, number, digest)

	return gen, genesisTrie, genesisHeader
}
