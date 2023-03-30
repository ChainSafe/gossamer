// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package commands

import (
	"fmt"
	"strings"
	"testing"

	"github.com/dgraph-io/badger/v2"
	"github.com/stretchr/testify/require"
)

// TestPruneState test "gossamer prune-state"
func TestPruneState(t *testing.T) {
	t.Skip()
	// this fails due to being unable to call blockState.GetHighestFinalisedHash()
	// when initialising the blockstate need to regenerate the test database
	// and/or move this to the state package (which would make sense)

	iterateDB := func(db *badger.DB, cb func(*badger.Item)) {
		txn := db.NewTransaction(false)
		itr := txn.NewIterator(badger.DefaultIteratorOptions)

		for itr.Rewind(); itr.Valid(); itr.Next() {
			cb(itr.Item())
		}
	}

	var (
		inputDBPath   = "../../tests/data/db"
		configFile    = "../../tests/data/db/config.toml"
		prunedDBPath  = fmt.Sprintf("%s/%s", t.TempDir(), "pruned")
		storagePrefix = "storage"
	)

	inputDB, err := badger.Open(badger.DefaultOptions(inputDBPath).WithReadOnly(true))
	require.NoError(t, err)

	nonStorageKeys := make(map[string]interface{})
	var numStorageKeys int

	getKeysInputDB := func(item *badger.Item) {
		key := string(item.Key())
		if strings.HasPrefix(key, storagePrefix) {
			numStorageKeys++
			return
		}
		nonStorageKeys[key] = nil
	}
	iterateDB(inputDB, getKeysInputDB)

	err = inputDB.Close()
	require.NoError(t, err)

	t.Log("Total keys in input DB", numStorageKeys+len(nonStorageKeys), "storage keys", numStorageKeys)
	t.Log("pruned DB path", prunedDBPath)

	// Run Prune command
	PruneStateCmd.SetArgs([]string{"--config", configFile, "--retain-blocks", "5"})
	err = PruneStateCmd.Execute()
	require.NoError(t, err)

	prunedDB, err := badger.Open(badger.DefaultOptions(prunedDBPath))
	require.NoError(t, err)

	nonStorageKeysPruned := make(map[string]interface{})
	var numStorageKeysPruned int

	getKeysPrunedDB := func(item *badger.Item) {
		key := string(item.Key())
		if strings.HasPrefix(key, storagePrefix) {
			numStorageKeysPruned++
			return
		}
		nonStorageKeysPruned[key] = nil
	}
	iterateDB(prunedDB, getKeysPrunedDB)

	t.Log("Total keys in pruned DB", len(nonStorageKeysPruned)+numStorageKeysPruned, "storage keys", numStorageKeysPruned)
	require.Equal(t, len(nonStorageKeysPruned), len(nonStorageKeys))

	// Check all non storage keys are present.
	for k := range nonStorageKeys {
		_, ok := nonStorageKeysPruned[k]
		require.True(t, ok)
	}

	err = prunedDB.Close()
	require.NoError(t, err)
}
