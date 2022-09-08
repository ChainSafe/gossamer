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

	const version = trie.V0

	keys := []string{
		"cat",
		"catapulta",
		"catapora",
		"dog",
		"doguinho",
	}

	testTrie := trie.NewEmptyTrie()

	for i, key := range keys {
		value := fmt.Sprintf("%x-%d", key, i)
		testTrie.Put([]byte(key), []byte(value), version)
	}

	rootHash, err := testTrie.Hash(version)
	require.NoError(t, err)

	database, err := chaindb.NewBadgerDB(&chaindb.Config{
		InMemory: true,
	})
	require.NoError(t, err)
	err = testTrie.Store(database)
	require.NoError(t, err)

	for i, key := range keys {
		fullKeys := [][]byte{[]byte(key)}
		proof, err := Generate(rootHash.ToBytes(), fullKeys, database, version)
		require.NoError(t, err)

		expectedValue := fmt.Sprintf("%x-%d", key, i)
		err = Verify(proof, rootHash.ToBytes(), []byte(key), []byte(expectedValue), version)
		require.NoError(t, err)
	}
}
