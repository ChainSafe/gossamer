// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package overlayedchanges

import (
	"bytes"
	"iter"
	"testing"

	"github.com/ChainSafe/gossamer/internal/primitives/core/hash"
	"github.com/ChainSafe/gossamer/internal/primitives/core/offchain"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime"
	"github.com/ChainSafe/gossamer/internal/primitives/state-machine/backend"
	"github.com/ChainSafe/gossamer/internal/primitives/storage"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/scale"
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
	overlayed.SetStorage(StorageKey("dog"), []byte("puppy"))
	overlayed.SetStorage(StorageKey("dogglesworth"), []byte("catYYY"))
	overlayed.SetStorage(StorageKey("doug"), []byte{})
	require.NoError(t, overlayed.CommitTransaction())

	overlayed.StartTransaction()
	overlayed.SetStorage(StorageKey("dogglesworth"), []byte("cat"))
	overlayed.SetStorage(StorageKey("doug"), nil)

	root := common.MustHexToBytes("0x39245109cef3758c2eed2ccba8d9b370a917850af3824bc8348d505df2c298fa")
	overlayedRoot, _ := overlayed.StorageRoot(b, stateVersion)

	encodedRoot := scale.MustMarshal(overlayedRoot)
	require.Equal(t, root, encodedRoot)

	overlayed.SetStorage(StorageKey("doug2"), []byte("yes"))

	root = common.MustHexToBytes("0x5c0a4e35cb967de785e1cb8743e6f24b6ff6d45155317f2078f6eb3fc4ff3e3d")
	overlayedRoot, cached := overlayed.StorageRoot(b, stateVersion)
	require.False(t, cached)

	encodedRoot = scale.MustMarshal(overlayedRoot)
	require.Equal(t, root, encodedRoot)

	// Calling a second time should use it from the cache
	overlayedRoot, cached = overlayed.StorageRoot(b, stateVersion)
	require.True(t, cached)
	encodedRoot = scale.MustMarshal(overlayedRoot)
	require.Equal(t, root, encodedRoot)
}

func TestOverlayedChildStorageRootWorks(t *testing.T) {
	stateVersion := storage.StateVersionV1
	childInfo := storage.NewDefaultChildInfo([]byte("Child1"))
	b := backend.NewMemoryDBTrieBackend[hash.H256, runtime.BlakeTwo256]()
	overlay := NewOverlayedChanges[hash.H256, runtime.BlakeTwo256]()

	overlay.StartTransaction()
	overlay.SetChildStorage(childInfo, []byte{20}, []byte{20})
	overlay.SetChildStorage(childInfo, []byte{30}, []byte{30})
	overlay.SetChildStorage(childInfo, []byte{40}, []byte{40})
	require.NoError(t, overlay.CommitTransaction())

	overlay.SetChildStorage(childInfo, []byte{10}, []byte{10})
	overlay.SetChildStorage(childInfo, []byte{30}, nil)

	childRoot := common.MustHexToBytes("0xc02965e1df4dc5baf6977390ce67dab1d7a9b27a87c1afe27b50d29cc990e0f5")
	root := common.MustHexToBytes("0xeafb765909c3ed5afd92a0c564acf4620d0234b31702e8e8e9b48da72a748838")

	overlayedChildRoot, cached, err := overlay.ChildStorageRoot(childInfo, b, stateVersion)
	require.NoError(t, err)
	require.False(t, cached)
	encodedChildRoot := scale.MustMarshal(overlayedChildRoot)
	require.Equal(t, childRoot, encodedChildRoot)

	overlayedRoot, cached := overlay.StorageRoot(b, stateVersion)
	require.False(t, cached)
	encodedRoot := scale.MustMarshal(overlayedRoot)
	require.Equal(t, root, encodedRoot)

	// Calling a second time should use it from the cache
	overlayedChildRoot, cached, err = overlay.ChildStorageRoot(childInfo, b, stateVersion)
	require.NoError(t, err)
	require.True(t, cached)
	encodedChildRoot = scale.MustMarshal(overlayedChildRoot)
	require.Equal(t, childRoot, encodedChildRoot)
}

func TestExtinsicChangesAreCollected(t *testing.T) {
	overlay := NewOverlayedChanges[hash.H256, runtime.BlakeTwo256]()
	overlay.SetCollectExtrinsic(true)

	overlay.StartTransaction()
	overlay.SetStorage(StorageKey([]byte{100}), []byte{101})

	overlay.SetExtrinsicIndex(0)
	overlay.SetStorage(StorageKey([]byte{1}), []byte{2})

	overlay.SetExtrinsicIndex(1)
	overlay.SetStorage(StorageKey([]byte{3}), []byte{4})

	overlay.SetExtrinsicIndex(2)
	overlay.SetStorage(StorageKey([]byte{1}), []byte{6})

	assertExtrinsics(t, overlay.top, []byte{1}, []uint32{0, 2})
	assertExtrinsics(t, overlay.top, []byte{3}, []uint32{1})
	assertExtrinsics(t, overlay.top, []byte{100}, []uint32{NoExtrinsicIndex})

	overlay.StartTransaction()

	overlay.SetExtrinsicIndex(3)
	overlay.SetStorage(StorageKey([]byte{3}), []byte{7})

	overlay.SetExtrinsicIndex(4)
	overlay.SetStorage(StorageKey([]byte{1}), []byte{8})

	assertExtrinsics(t, overlay.top, []byte{1}, []uint32{0, 2, 4})
	assertExtrinsics(t, overlay.top, []byte{3}, []uint32{1, 3})
	assertExtrinsics(t, overlay.top, []byte{100}, []uint32{NoExtrinsicIndex})

	require.NoError(t, overlay.RollbackTransaction())

	assertExtrinsics(t, overlay.top, []byte{1}, []uint32{0, 2})
	assertExtrinsics(t, overlay.top, []byte{3}, []uint32{1})
	assertExtrinsics(t, overlay.top, []byte{100}, []uint32{NoExtrinsicIndex})
}

func TestNextStorageKeyChangeWorks(t *testing.T) {
	overlay := NewOverlayedChanges[hash.H256, runtime.BlakeTwo256]()

	overlay.StartTransaction()

	overlay.SetStorage(StorageKey([]byte{20}), []byte{20})
	overlay.SetStorage(StorageKey([]byte{30}), []byte{30})
	overlay.SetStorage(StorageKey([]byte{40}), []byte{40})
	require.NoError(t, overlay.CommitTransaction())

	overlay.SetStorage(StorageKey([]byte{10}), []byte{10})
	overlay.SetStorage(StorageKey([]byte{30}), nil)

	next, _ := iter.Pull2(overlay.IterAfter(StorageKey([]byte{5})))
	nextTo5key, nextTo5value, _ := next()

	require.Equal(t, StorageKey([]byte{10}), nextTo5key)
	require.Equal(t, StorageValue([]byte{10}), nextTo5value.Value())

	next, _ = iter.Pull2(overlay.IterAfter(StorageKey([]byte{10})))
	nextTo10key, nextTo10value, _ := next()

	require.Equal(t, StorageKey([]byte{20}), nextTo10key)
	require.Equal(t, StorageValue([]byte{20}), nextTo10value.Value())

	next, _ = iter.Pull2(overlay.IterAfter(StorageKey([]byte{20})))
	nextTo20key, nextTo20value, _ := next()

	require.Equal(t, StorageKey([]byte{30}), nextTo20key)
	require.Equal(t, StorageValue(nil), nextTo20value.Value())

	next, _ = iter.Pull2(overlay.IterAfter(StorageKey([]byte{30})))
	nextTo30key, nextTo30value, _ := next()

	require.Equal(t, StorageKey([]byte{40}), nextTo30key)
	require.Equal(t, StorageValue([]byte{40}), nextTo30value.Value())

	overlay.SetStorage(StorageKey([]byte{50}), StorageValue([]byte{50}))

	next, _ = iter.Pull2(overlay.IterAfter(StorageKey([]byte{40})))
	nextTo40key, nextTo40value, _ := next()

	require.Equal(t, StorageKey([]byte{50}), nextTo40key)
	require.Equal(t, StorageValue([]byte{50}), nextTo40value.Value())
}

func TestNextChildStorageKeyChangeWorks(t *testing.T) {
	childInfo := storage.NewDefaultChildInfo([]byte("Child1"))
	child := StorageKey(childInfo.StorageKey())
	overlay := NewOverlayedChanges[hash.H256, runtime.BlakeTwo256]()

	overlay.StartTransaction()
	overlay.SetChildStorage(childInfo, []byte{20}, []byte{20})
	overlay.SetChildStorage(childInfo, []byte{30}, []byte{30})
	overlay.SetChildStorage(childInfo, []byte{40}, []byte{40})

	require.NoError(t, overlay.CommitTransaction())
	overlay.SetChildStorage(childInfo, []byte{10}, []byte{10})
	overlay.SetChildStorage(childInfo, []byte{30}, nil)

	next, _ := iter.Pull2(overlay.ChildIterAfter(child, StorageKey([]byte{5})))
	nextTo5key, nextTo5value, _ := next()

	require.Equal(t, StorageKey([]byte{10}), nextTo5key)
	require.Equal(t, StorageValue([]byte{10}), nextTo5value.Value())

	next, _ = iter.Pull2(overlay.ChildIterAfter(child, StorageKey([]byte{10})))
	nextTo10key, nextTo10value, _ := next()

	require.Equal(t, StorageKey([]byte{20}), nextTo10key)
	require.Equal(t, StorageValue([]byte{20}), nextTo10value.Value())

	next, _ = iter.Pull2(overlay.ChildIterAfter(child, StorageKey([]byte{20})))
	nextTo20key, nextTo20value, _ := next()

	require.Equal(t, StorageKey([]byte{30}), nextTo20key)
	require.Equal(t, StorageValue(nil), nextTo20value.Value())

	next, _ = iter.Pull2(overlay.ChildIterAfter(child, StorageKey([]byte{30})))
	nextTo30key, nextTo30value, _ := next()

	require.Equal(t, StorageKey([]byte{40}), nextTo30key)
	require.Equal(t, StorageValue([]byte{40}), nextTo30value.Value())

	overlay.SetChildStorage(childInfo, []byte{50}, []byte{50})

	next, _ = iter.Pull2(overlay.ChildIterAfter(child, StorageKey([]byte{40})))
	nextTo40key, nextTo40value, _ := next()

	require.Equal(t, StorageKey([]byte{50}), nextTo40key)
	require.Equal(t, StorageValue([]byte{50}), nextTo40value.Value())
}

type keyValue struct {
	key   string
	value []byte
}

type offchainKeyValue struct {
	key   StorageKey
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

func assertExtrinsics(t *testing.T, overlay overlayedChangeSet, key []byte, expected []uint32) {
	var extrinsics []uint32

	entry, has := overlay.Get(string(key))
	require.True(t, has)

	entry.Extrinsics().Scan(func(ex uint32) bool {
		extrinsics = append(extrinsics, ex)
		return true
	})

	require.Equal(t, expected, extrinsics)
}
