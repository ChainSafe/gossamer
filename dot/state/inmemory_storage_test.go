// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package state

import (
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/dot/telemetry"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/internal/database"
	"github.com/ChainSafe/gossamer/lib/common"
	inmemory_storate "github.com/ChainSafe/gossamer/lib/runtime/storage/inmemory"
	"github.com/ChainSafe/gossamer/pkg/trie"
	"github.com/ChainSafe/gossamer/pkg/trie/node"
	"go.uber.org/mock/gomock"

	"github.com/stretchr/testify/require"
)

func newTestStorageState(t *testing.T) *InmemoryStorageState {
	db := NewInMemoryDB(t)

	tries := newTriesEmpty()
	bs := newTestBlockState(t, tries)

	s, err := NewStorageState(db, bs, tries)
	require.NoError(t, err)
	return s
}

func TestStorage_StoreAndLoadTrie(t *testing.T) {
	storage := newTestStorageState(t)
	ts, err := storage.TrieState(&trie.EmptyHash)
	require.NoError(t, err)

	root, err := ts.Root()
	require.NoError(t, err)
	err = storage.StoreTrie(ts, nil)
	require.NoError(t, err)

	time.Sleep(time.Millisecond * 100)

	trie, err := storage.LoadFromDB(root)
	require.NoError(t, err)
	ts2 := inmemory_storate.NewTrieState(trie)
	newSnapshot := ts2.Snapshot()

	require.True(t, ts.Trie().Equal(newSnapshot))
}

func TestStorage_GetStorageByBlockHash(t *testing.T) {
	storage := newTestStorageState(t)
	ts, err := storage.TrieState(&trie.EmptyHash)
	require.NoError(t, err)

	key := []byte("testkey")
	value := []byte("testvalue")
	ts.Put(key, value)

	root, err := ts.Root()
	require.NoError(t, err)
	err = storage.StoreTrie(ts, nil)
	require.NoError(t, err)

	body, err := types.NewBodyFromBytes([]byte{})
	require.NoError(t, err)

	block := &types.Block{
		Header: types.Header{
			ParentHash: testGenesisHeader.Hash(),
			Number:     1,
			StateRoot:  root,
			Digest:     createPrimaryBABEDigest(t),
		},
		Body: *body,
	}
	err = storage.blockState.AddBlock(block)
	require.NoError(t, err)

	hash := block.Header.Hash()
	res, err := storage.GetStorageByBlockHash(&hash, key)
	require.NoError(t, err)
	require.Equal(t, value, res)
}

func TestStorage_TrieState(t *testing.T) {
	storage := newTestStorageState(t)
	ts, err := storage.TrieState(&trie.EmptyHash)
	require.NoError(t, err)
	ts.Put([]byte("noot"), []byte("washere"))

	root, err := ts.Root()
	require.NoError(t, err)
	err = storage.StoreTrie(ts, nil)
	require.NoError(t, err)

	time.Sleep(time.Millisecond * 100)

	// get trie from db
	storage.blockState.tries.delete(root)
	ts3, err := storage.TrieState(&root)
	require.NoError(t, err)
	require.Equal(t, ts.Trie().MustHash(), ts3.Trie().MustHash())
}

func TestStorage_LoadFromDB(t *testing.T) {
	storage := newTestStorageState(t)
	ts, err := storage.TrieState(&trie.EmptyHash)
	require.NoError(t, err)

	trieKV := []struct {
		key   []byte
		value []byte
	}{
		{value: []byte{}},
		{[]byte("key1"), []byte("value1")},
		{[]byte("key2"), []byte("value2")},
		{[]byte("xyzKey1"), []byte("xyzValue1")},
		{[]byte("long"), []byte("newvaluewithmorethan32byteslength")},
	}

	for _, kv := range trieKV {
		ts.Put(kv.key, kv.value)
	}

	root, err := ts.Root()
	require.NoError(t, err)

	// Write trie to disk.
	err = storage.StoreTrie(ts, nil)
	require.NoError(t, err)

	// Clear trie from cache and fetch data from disk.
	storage.blockState.tries.delete(root)

	data, err := storage.GetStorage(&root, trieKV[0].key)
	require.NoError(t, err)
	require.Equal(t, trieKV[0].value, data)

	storage.blockState.tries.delete(root)

	prefixKeys, err := storage.GetKeysWithPrefix(&root, []byte("ke"))
	require.NoError(t, err)
	require.Equal(t, 2, len(prefixKeys))

	storage.blockState.tries.delete(root)

	entries, err := storage.Entries(&root)
	require.NoError(t, err)
	require.Equal(t, 5, len(entries))
}

func TestStorage_StoreTrie_NotSyncing(t *testing.T) {
	storage := newTestStorageState(t)
	ts, err := storage.TrieState(&trie.EmptyHash)
	require.NoError(t, err)

	key := []byte("testkey")
	value := []byte("testvalue")
	ts.Put(key, value)

	err = storage.StoreTrie(ts, nil)
	require.NoError(t, err)
	require.Equal(t, 2, storage.blockState.tries.len())
}

func TestGetStorageChildAndGetStorageFromChild(t *testing.T) {
	// initialise database using data directory
	basepath := t.TempDir()
	db, err := database.LoadDatabase(basepath, false)
	require.NoError(t, err)

	_, genTrie, genHeader := newWestendDevGenesisWithTrieAndHeader(t)

	ctrl := gomock.NewController(t)
	telemetryMock := NewMockTelemetry(ctrl)
	telemetryMock.EXPECT().SendMessage(telemetry.NewNotifyFinalized(
		genHeader.Hash(),
		"0",
	))
	trieDB := NewMockDatabase(ctrl)
	trieDB.EXPECT().Get(gomock.Any()).Times(0)

	trieRoot := &node.Node{
		PartialKey:   []byte{1, 2},
		StorageValue: []byte{3, 4},
		Dirty:        true,
	}
	testChildTrie := trie.NewInMemoryTrie(trieRoot, trieDB)

	testChildTrie.Put([]byte("keyInsidechild"), []byte("voila"))

	err = genTrie.SetChild([]byte("keyToChild"), testChildTrie)
	require.NoError(t, err)

	tries := newTriesEmpty()

	blockState, err := NewBlockStateFromGenesis(db, tries, &genHeader, telemetryMock)
	require.NoError(t, err)

	storage, err := NewStorageState(db, blockState, tries)
	require.NoError(t, err)

	trieState := inmemory_storate.NewTrieState(genTrie)

	header := types.NewHeader(blockState.GenesisHash(), trieState.MustRoot(),
		common.Hash{}, 1, types.NewDigest())

	err = storage.StoreTrie(trieState, header)
	require.NoError(t, err)

	rootHash, err := genTrie.Hash()
	require.NoError(t, err)

	_, err = storage.GetStorageChild(&rootHash, []byte("keyToChild"))
	require.NoError(t, err)

	// Clear trie from cache and fetch data from disk.
	storage.blockState.tries.delete(rootHash)

	_, err = storage.GetStorageChild(&rootHash, []byte("keyToChild"))
	require.NoError(t, err)

	value, err := storage.GetStorageFromChild(&rootHash, []byte("keyToChild"), []byte("keyInsidechild"))
	require.NoError(t, err)

	require.Equal(t, []byte("voila"), value)
}
