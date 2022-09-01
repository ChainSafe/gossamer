// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package genesis

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/trie"
	"github.com/ChainSafe/gossamer/lib/utils"
	"github.com/stretchr/testify/require"
)

// CreateTestGenesisJSONFile writes a genesis file using the fields given to
// the current test temporary directory.
func CreateTestGenesisJSONFile(t *testing.T, fields Fields) (filename string) {
	rawGenesis := &Genesis{
		Name:    "test",
		Genesis: fields,
	}
	jsonData, err := json.Marshal(rawGenesis)
	require.NoError(t, err)
	filename = filepath.Join(t.TempDir(), "genesis-test")
	err = os.WriteFile(filename, jsonData, os.ModePerm)
	require.NoError(t, err)
	return filename
}

// NewTestGenesisWithTrieAndHeader generates genesis, genesis trie and genesis header
func NewTestGenesisWithTrieAndHeader(t *testing.T) (*Genesis, *trie.Trie, *types.Header) {
	genesisPath := utils.GetGssmrV3SubstrateGenesisRawPathTest(t)
	gen, err := NewGenesisFromJSONRaw(genesisPath)
	require.NoError(t, err)

	tr, h := newGenesisTrieAndHeader(t, gen)
	return gen, tr, h
}

// NewDevGenesisWithTrieAndHeader generates test dev genesis, genesis trie and genesis header
func NewDevGenesisWithTrieAndHeader(t *testing.T) (*Genesis, *trie.Trie, *types.Header) {
	genesisPath := utils.GetDevV3SubstrateGenesisPath(t)

	gen, err := NewGenesisFromJSONRaw(genesisPath)
	require.NoError(t, err)

	tr, h := newGenesisTrieAndHeader(t, gen)
	return gen, tr, h
}

func newGenesisTrieAndHeader(t *testing.T, gen *Genesis) (*trie.Trie, *types.Header) {
	genTrie, err := NewTrieFromGenesis(gen)
	require.NoError(t, err)

	genesisHeader, err := types.NewHeader(common.NewHash([]byte{0}),
		genTrie.MustHash(), trie.EmptyHash, 0, types.NewDigest())
	require.NoError(t, err)

	return genTrie, genesisHeader
}
