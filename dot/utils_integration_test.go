// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

//go:build integration

package dot

import (
	"testing"

	westend_dev "github.com/ChainSafe/gossamer/chain/westend-dev"

	"github.com/ChainSafe/gossamer/lib/genesis"
	"github.com/ChainSafe/gossamer/lib/trie"
	"github.com/stretchr/testify/require"
)

func TestTrieSnapshot(t *testing.T) {
	config := westend_dev.DefaultConfig()

	genRawFile := NewTestGenesisRawFile(t, config)
	config.BasePath = t.TempDir()

	genRaw, err := genesis.NewGenesisFromJSONRaw(genRawFile)
	require.NoError(t, err)

	tri := trie.NewEmptyTrie()
	key := []byte("key")
	value := []byte("value")

	for k, v := range genRaw.Genesis.Raw["top"] {
		val := []byte(v)
		tri.Put([]byte(k), val)
	}

	deepCopyTrie := tri.DeepCopy()

	// Take Snapshot of the trie.
	newTrie := tri.Snapshot()

	// Get the Trie root hash for all the 3 tries.
	tHash, err := tri.Hash()
	require.NoError(t, err)

	dcTrieHash, err := deepCopyTrie.Hash()
	require.NoError(t, err)

	newTrieHash, err := newTrie.Hash()
	require.NoError(t, err)

	// Root hash for the 3 tries should be equal.
	require.Equal(t, tHash, dcTrieHash)
	require.Equal(t, tHash, newTrieHash)

	// Modify the current trie.
	value[0] = 'w'
	newTrie.Put(key, value)

	// Get the updated root hash of all tries.
	tHash, err = tri.Hash()
	require.NoError(t, err)

	dcTrieHash, err = deepCopyTrie.Hash()
	require.NoError(t, err)

	newTrieHash, err = newTrie.Hash()
	require.NoError(t, err)

	// Only the current trie should have a different root hash since it is updated.
	require.NotEqual(t, newTrieHash, dcTrieHash)
	require.NotEqual(t, newTrieHash, tHash)
	require.Equal(t, dcTrieHash, tHash)
}
