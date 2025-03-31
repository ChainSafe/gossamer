// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package overlayedchanges

import (
	"bytes"
	"testing"

	"github.com/ChainSafe/gossamer/internal/primitives/core/hash"
	"github.com/ChainSafe/gossamer/internal/primitives/core/offchain"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime"
	"github.com/ChainSafe/gossamer/internal/primitives/state-machine/backend"
	"github.com/ChainSafe/gossamer/internal/primitives/storage"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/stretchr/testify/require"
)

func TestOffchainOverlayedChangesClone(t *testing.T) {
	offchainOverlayedChange := NewOffchainOverlayedChanges()

	offchainOverlayedChange.Set([]byte("prefix"), []byte("key"), []byte{1, 2, 3})

	clone := offchainOverlayedChange.Clone()

	require.Equal(t, offchainOverlayedChange, clone)

	clone.Set([]byte("prefix"), []byte("key"), []byte{1, 2, 3, 4})

	require.NotEqual(t, offchainOverlayedChange, clone)
}

func TestOverlayedStorageWorks(t *testing.T) {
	overlayed := NewOverlayedChanges[hash.H256, runtime.BlakeTwo256]()

	key := string([]byte{42, 69, 169, 142})

	value, has := overlayed.Storage(key)
	require.False(t, has)
	require.Nil(t, value)

	overlayed.StartTransaction()

	overlayed.SetStorage(backend.StorageKey(key), []byte{1, 2, 3})
	value, has = overlayed.Storage(key)
	require.True(t, has)
	require.Equal(t, []byte{1, 2, 3}, value)

	require.NoError(t, overlayed.CommitTransaction())

	overlayed.SetStorage(backend.StorageKey(key), []byte{1, 2, 3})
	value, has = overlayed.Storage(key)
	require.True(t, has)
	require.Equal(t, []byte{1, 2, 3}, value)

	overlayed.StartTransaction()

	overlayed.SetStorage(backend.StorageKey(key), nil)
	value, has = overlayed.Storage(key)
	require.True(t, has)
	require.Nil(t, value)

	require.NoError(t, overlayed.RollbackTransaction())
	value, has = overlayed.Storage(key)
	require.True(t, has)
	require.Equal(t, []byte{1, 2, 3}, value)

	overlayed.SetStorage(backend.StorageKey(key), nil)
	value, has = overlayed.Storage(key)
	require.True(t, has)
	require.Nil(t, value)
}

func TestOffchainOverlayedStorageTransactionsWorks(t *testing.T) {
	overlayed := NewOverlayedChanges[hash.H256, runtime.BlakeTwo256]()
	key := string([]byte{42, 69, 169, 142})

	checkOffchainContent(t, *overlayed, 0, []keyValue{})

	overlayed.StartTransaction()

	overlayed.SetOffchainStorage(backend.StorageKey(key), []byte{1, 2, 3})

	checkOffchainContent(t, *overlayed, 1, []keyValue{{key: key, value: []byte{1, 2, 3}}})

	require.NoError(t, overlayed.CommitTransaction())
	checkOffchainContent(t, *overlayed, 0, []keyValue{{key: key, value: []byte{1, 2, 3}}})

	overlayed.StartTransaction()

	overlayed.SetOffchainStorage(backend.StorageKey(key), []byte{})
	checkOffchainContent(t, *overlayed, 1, []keyValue{{key: key, value: []byte{}}})

	overlayed.SetOffchainStorage(backend.StorageKey(key), nil)
	checkOffchainContent(t, *overlayed, 1, []keyValue{{key: key, value: nil}})

	require.NoError(t, overlayed.RollbackTransaction())
	checkOffchainContent(t, *overlayed, 0, []keyValue{{key: key, value: []byte{1, 2, 3}}})

	overlayed.SetOffchainStorage(backend.StorageKey(key), nil)
	checkOffchainContent(t, *overlayed, 0, []keyValue{{key: key, value: nil}})
}

func TestOverlayedStorageRootWorks(t *testing.T) {
	stateVersion := storage.StateVersionV1

	initial := map[string][]byte{
		"doe":          []byte("reindeer"),
		"dog":          []byte("puppyXXX"),
		"dogglesworth": []byte("catXXX"),
		"doug":         []byte("notadog"),
	}

	b := backend.NewMemoryDBTrieBackendFromMap[hash.H256, runtime.BlakeTwo256](initial, stateVersion)
	overlayed := NewOverlayedChanges[hash.H256, runtime.BlakeTwo256]()

	overlayed.StartTransaction()
	overlayed.SetStorage(backend.StorageKey("dog"), []byte("puppy"))
	overlayed.SetStorage(backend.StorageKey("dogglesworth"), []byte("catYYY"))
	overlayed.SetStorage(backend.StorageKey("doug"), []byte{})
	require.NoError(t, overlayed.CommitTransaction())

	overlayed.StartTransaction()
	overlayed.SetStorage(backend.StorageKey("dogglesworth"), []byte("cat"))
	overlayed.SetStorage(backend.StorageKey("doug"), nil)

	root := common.MustHexToBytes("0x39245109cef3758c2eed2ccba8d9b370a917850af3824bc8348d505df2c298fa")
	overlayedRoot, _ := overlayed.StorageRoot(b, stateVersion)

	encodedRoot := overlayedRoot.MustMarshalSCALE()
	require.Equal(t, root, encodedRoot)

	overlayed.SetStorage(backend.StorageKey("doug2"), []byte("yes"))

	root = common.MustHexToBytes("0x5c0a4e35cb967de785e1cb8743e6f24b6ff6d45155317f2078f6eb3fc4ff3e3d")
	overlayedRoot, _ = overlayed.StorageRoot(b, stateVersion)

	encodedRoot = overlayedRoot.MustMarshalSCALE()
	require.Equal(t, root, encodedRoot)
}

func TestOverlayedChildStorageRootWorks(t *testing.T) {
	/*stateVersion := storage.StateVersionV1
	childInfo := storage.NewDefaultChildInfo([]byte("Child1"))
	b := backend.NewMemoryDBTrieBackend[hash.H256, runtime.BlakeTwo256]()
	overlayed := NewOverlayedChanges[hash.H256, runtime.BlakeTwo256]()

	overlayed.StartTransaction()
	overlayed.SetChildStorage(storage.NewDefaultChildInfo([]byte("Child1")), []byte{20}, []byte{20})
	overlayed.SetChildStorage(storage.NewDefaultChildInfo([]byte("Child1")), []byte{30}, []byte{30})
	overlayed.SetChildStorage(storage.NewDefaultChildInfo([]byte("Child1")), []byte{40}, []byte{40})
	require.NoError(t, overlayed.CommitTransaction())*/
}

type keyValue struct {
	key   string
	value []byte
}

type offchainKeyValue struct {
	key   backend.StorageKey
	value offchain.OffchainOverlayedChange
}

func checkOffchainContent(
	t *testing.T,
	state OverlayedChanges[hash.H256, runtime.BlakeTwo256],
	nbcommit int,
	expected []keyValue,
) {
	cloned := state.Clone()

	for range nbcommit {
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
		key := bytes.Join([][]byte{offchain.StoragePrefix, []byte(kv.key)}, []byte{})
		toCheck = append(toCheck, offchainKeyValue{key: key, value: change})
	}

	require.Equal(t, toCheck, offchainData)
}
