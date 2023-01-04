// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package dot

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/genesis"
	"github.com/ChainSafe/gossamer/lib/runtime/wasmer"
	"github.com/ChainSafe/gossamer/lib/trie"
	"github.com/ChainSafe/gossamer/lib/utils"
	"github.com/stretchr/testify/require"
)

func writeGenesisToTestJSON(t *testing.T, genesis genesis.Genesis) (filename string) {
	t.Helper()

	jsonData, err := json.Marshal(genesis)
	require.NoError(t, err)
	filename = filepath.Join(t.TempDir(), "genesis-test")
	err = os.WriteFile(filename, jsonData, os.ModePerm)
	require.NoError(t, err)
	return filename
}

func newTestGenesisWithTrieAndHeader(t *testing.T) (
	gen genesis.Genesis, genesisTrie trie.Trie, genesisHeader types.Header) {
	t.Helper()

	genesisPath := utils.GetWestendDevGensisPath(t)
	genPtr, err := genesis.NewGenesisFromJSONRaw(genesisPath)
	require.NoError(t, err)
	gen = *genPtr

	genesisTrie, err = wasmer.NewTrieFromGenesis(gen)
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
