package statemachine

import (
	"fmt"
	"testing"

	"github.com/ChainSafe/gossamer/internal/primitives/core/hash"
	"github.com/ChainSafe/gossamer/internal/primitives/core/offchain"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime"
	"github.com/stretchr/testify/require"
)

func TestOverlayedStorageWorks(t *testing.T) {
	overlayed := NewOverlayedChanges[hash.H256, runtime.BlakeTwo256]()

	key := string([]byte{42, 69, 169, 142})

	value, has := overlayed.Storage(key)
	require.False(t, has)
	require.Nil(t, value)

	overlayed.StartTransaction()

	overlayed.SetStorage(StorageKey(key), []byte{1, 2, 3})
	value, has = overlayed.Storage(key)
	require.True(t, has)
	require.Equal(t, []byte{1, 2, 3}, value)

	require.NoError(t, overlayed.CommitTransaction())

	overlayed.SetStorage(StorageKey(key), []byte{1, 2, 3})
	value, has = overlayed.Storage(key)
	require.True(t, has)
	require.Equal(t, []byte{1, 2, 3}, value)

	overlayed.StartTransaction()

	overlayed.SetStorage(StorageKey(key), nil)
	value, has = overlayed.Storage(key)
	require.True(t, has)
	require.Nil(t, value)

	require.NoError(t, overlayed.RollbackTransaction())
	value, has = overlayed.Storage(key)
	require.True(t, has)
	require.Equal(t, []byte{1, 2, 3}, value)

	overlayed.SetStorage(StorageKey(key), nil)
	value, has = overlayed.Storage(key)
	require.True(t, has)
	require.Nil(t, value)
}

func TestOffchainOverlayedStorageTransactionsWorks(t *testing.T) {
	overlayed := NewOverlayedChanges[hash.H256, runtime.BlakeTwo256]()
	key := string([]byte{42, 69, 169, 142})

	checkOffchainContent(t, *overlayed, 0, []keyValue{})

	overlayed.StartTransaction()

	overlayed.SetOffchainStorage(StorageKey(key), []byte{1, 2, 3})

	checkOffchainContent(t, *overlayed, 1, []keyValue{{key: key, value: []byte{1, 2, 3}}})

	require.NoError(t, overlayed.CommitTransaction())
	checkOffchainContent(t, *overlayed, 0, []keyValue{{key: key, value: []byte{1, 2, 3}}})

	overlayed.StartTransaction()

	overlayed.SetOffchainStorage(StorageKey(key), []byte{})
	checkOffchainContent(t, *overlayed, 1, []keyValue{{key: key, value: []byte{}}})

	overlayed.SetOffchainStorage(StorageKey(key), nil)
	checkOffchainContent(t, *overlayed, 1, []keyValue{{key: key, value: nil}})

	require.NoError(t, overlayed.RollbackTransaction())
	checkOffchainContent(t, *overlayed, 0, []keyValue{{key: key, value: []byte{1, 2, 3}}})

	overlayed.SetOffchainStorage(StorageKey(key), nil)
	checkOffchainContent(t, *overlayed, 0, []keyValue{{key: key, value: nil}})
}

type keyValue struct {
	key   string
	value []byte
}

type offchainKeyValue struct {
	key   string
	value offchain.OffchainOverlayedChange
}

func checkOffchainContent(t *testing.T, state OverlayedChanges[hash.H256, runtime.BlakeTwo256], nbcommit int, expected []keyValue) {
	cloned := state.Clone()

	for i := 0; i < nbcommit; i++ {
		require.NoError(t, cloned.CommitTransaction())
	}

	var offchainData []offchainKeyValue
	for k, v := range cloned.offchainDrainCommited() {
		offchainData = append(offchainData, offchainKeyValue{key: k, value: v})
	}

	var toCheck []offchainKeyValue
	for _, kv := range expected {
		var change OffchainOverlayedChange
		if kv.value != nil {
			change = OffchainOverlayedChangeSetValue(kv.value)
		} else {
			change = OffchainOverlayedChangeRemove{}
		}
		key := fmt.Sprintf("%s%s", offchain.StoragePrefix, kv.key)
		toCheck = append(toCheck, offchainKeyValue{key: key, value: change})
	}

	require.Equal(t, toCheck, offchainData)
}
