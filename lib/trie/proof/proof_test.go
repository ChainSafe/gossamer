// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package proof

import (
	"fmt"
	"testing"

	"github.com/ChainSafe/chaindb"
	"github.com/ChainSafe/gossamer/lib/trie"
	"github.com/stretchr/testify/require"
)

func Test_Generate_Verify(t *testing.T) {
	t.Parallel()

	keys := []string{
		"cat",
		"catapulta",
		"catapora",
		"dog",
		"doguinho",
	}

	trie := trie.NewEmptyTrie()

	for i, key := range keys {
		value := fmt.Sprintf("%x-%d", key, i)
		trie.Put([]byte(key), []byte(value))
	}

	rootHash, err := trie.Hash()
	require.NoError(t, err)

	database, err := chaindb.NewBadgerDB(&chaindb.Config{
		InMemory: true,
	})
	require.NoError(t, err)
	err = trie.WriteDirty(database)
	require.NoError(t, err)

	for i, key := range keys {
		fullKeys := [][]byte{[]byte(key)}
		proof, err := Generate(rootHash.ToBytes(), fullKeys, database)
		require.NoError(t, err)

		expectedValue := fmt.Sprintf("%x-%d", key, i)
		err = Verify(proof, rootHash.ToBytes(), []byte(key), []byte(expectedValue))
		require.NoError(t, err)
	}
}
