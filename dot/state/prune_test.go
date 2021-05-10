package state

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dgraph-io/badger/v2"
	"github.com/stretchr/testify/require"
)

func iterateDB(db *badger.DB, cb func(*badger.Item)) {
	txn := db.NewTransaction(false)
	itr := txn.NewIterator(badger.DefaultIteratorOptions)

	for itr.Rewind(); itr.Valid(); itr.Next() {
		cb(itr.Item())
	}
}

func runPruneCmd(t *testing.T, inDBPath, prunedDBPath string) {
	currPath, err := os.Getwd()
	require.NoError(t, err)

	cmd := exec.Command(filepath.Join(currPath, "../..", "bin/gossamer"), "prune-state",
		"--basepath", inDBPath,
		"--pruned-db-path", prunedDBPath,
		"--bloom-size", "256",
		"--retain-block", "5")

	logger.Info("running prune command...", "cmd", cmd)

	_, err = cmd.CombinedOutput()
	require.NoError(t, err)
}

func TestPruneState(t *testing.T) {
	const (
		bloomSize      = 256
		retainBlockNum = 5
	)

	inDBPath := "../../tests/data/db"

	pruner, err := NewPruner(inDBPath, bloomSize, retainBlockNum)
	require.NoError(t, err)

	err = pruner.SetBloomFilter()
	require.NoError(t, err)

	nonStorageKeys := make(map[string]interface{})
	storageKeys := make(map[string]interface{})
	itr := pruner.InputDB.NewIterator()
	for itr.Next() {
		key := string(itr.Key())
		if !strings.HasPrefix(key, StoragePrefix) {
			nonStorageKeys[key] = nil
			continue
		}
		storageKeys[key] = nil
	}
	t.Log("Total keys in input DB", len(storageKeys)+len(nonStorageKeys), "storage keys", len(storageKeys))

	// close the input DB because `prune-state` cmd will open it.
	err = pruner.InputDB.Close()
	require.NoError(t, err)

	prunedDBPath := fmt.Sprintf("%s/%s", t.TempDir(), "badger")
	t.Log("pruned DB path", prunedDBPath)

	runPruneCmd(t, inDBPath, prunedDBPath)

	prunedDB, err := badger.Open(badger.DefaultOptions(prunedDBPath))
	require.NoError(t, err)

	storageKeysPruned := make(map[string]interface{})
	nonStorageKeysPruned := make(map[string]interface{})
	getKeysPrunedDB := func(item *badger.Item) {
		key := string(item.Key())
		if strings.HasPrefix(key, StoragePrefix) {
			key = strings.TrimPrefix(key, StoragePrefix)
			storageKeysPruned[key] = nil
			return
		}
		nonStorageKeysPruned[key] = nil
	}
	iterateDB(prunedDB, getKeysPrunedDB)
	t.Log("Total keys in pruned DB", len(storageKeysPruned)+len(nonStorageKeysPruned), "storage keys", len(storageKeysPruned))
	require.Equal(t, len(nonStorageKeysPruned), len(nonStorageKeys))

	// Check all non storage keys are present.
	for k := range nonStorageKeys {
		_, ok := nonStorageKeysPruned[k]
		require.True(t, ok)
	}

	// Check required storage keys exist.
	for k := range storageKeys {
		ok := pruner.bloom.contain([]byte(k))
		if ok {
			_, found := storageKeysPruned[k]
			require.True(t, found)
		}
	}

	err = prunedDB.Close()
	require.NoError(t, err)
}
