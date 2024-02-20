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
	runtime "github.com/ChainSafe/gossamer/lib/runtime"
	"github.com/ChainSafe/gossamer/lib/utils"
	"github.com/ChainSafe/gossamer/pkg/trie"
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

func newWestendDevGenesisWithTrieAndHeader(t *testing.T) (
	gen genesis.Genesis, genesisTrie *trie.InMemoryTrie, genesisHeader types.Header) {
	t.Helper()

	genesisPath := utils.GetWestendDevRawGenesisPath(t)
	genPtr, err := genesis.NewGenesisFromJSONRaw(genesisPath)
	require.NoError(t, err)
	gen = *genPtr

	genesisTrie, err = runtime.NewInMemoryTrieFromGenesis(gen)
	require.NoError(t, err)

	parentHash := common.NewHash([]byte{0})

	// We are using state trie V0 since we are using the genesis trie where v0 is used
	stateRoot := trie.V0.MustHash(genesisTrie)

	extrinsicRoot := trie.EmptyHash
	const number = 0
	genesisHeader = *types.NewHeader(parentHash,
		stateRoot, extrinsicRoot, number, nil)

	return gen, genesisTrie, genesisHeader
}
