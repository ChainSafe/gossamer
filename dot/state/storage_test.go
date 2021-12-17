// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package state

import (
	"fmt"
	"math/big"
	"sync"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/dot/state/pruner"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/genesis"
	runtime "github.com/ChainSafe/gossamer/lib/runtime/storage"
	"github.com/ChainSafe/gossamer/lib/trie"
	"github.com/ChainSafe/gossamer/lib/utils"

	"github.com/stretchr/testify/require"
)

func newTestStorageState(t *testing.T) *StorageState {
	db := NewInMemoryDB(t)
	bs := newTestBlockState(t, testGenesisHeader)

	s, err := NewStorageState(db, bs, trie.NewEmptyTrie(), pruner.Config{})
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
	ts2, err := runtime.NewTrieState(trie)
	require.NoError(t, err)
	new := ts2.Snapshot()
	require.Equal(t, ts.Trie(), new)
}

func TestStorage_GetStorageByBlockHash(t *testing.T) {
	storage := newTestStorageState(t)
	ts, err := storage.TrieState(&trie.EmptyHash)
	require.NoError(t, err)

	key := []byte("testkey")
	value := []byte("testvalue")
	ts.Set(key, value)

	root, err := ts.Root()
	require.NoError(t, err)
	err = storage.StoreTrie(ts, nil)
	require.NoError(t, err)

	body, err := types.NewBodyFromBytes([]byte{})
	require.NoError(t, err)

	block := &types.Block{
		Header: types.Header{
			ParentHash: testGenesisHeader.Hash(),
			Number:     big.NewInt(1),
			StateRoot:  root,
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
	ts.Set([]byte("noot"), []byte("washere"))

	root, err := ts.Root()
	require.NoError(t, err)
	err = storage.StoreTrie(ts, nil)
	require.NoError(t, err)

	time.Sleep(time.Millisecond * 100)

	// get trie from db
	storage.tries.Delete(root)
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
	}{{},
		{[]byte("key1"), []byte("value1")},
		{[]byte("key2"), []byte("value2")},
		{[]byte("xyzKey1"), []byte("xyzValue1")},
	}

	for _, kv := range trieKV {
		ts.Set(kv.key, kv.value)
	}

	root, err := ts.Root()
	require.NoError(t, err)

	// Write trie to disk.
	err = storage.StoreTrie(ts, nil)
	require.NoError(t, err)

	// Clear trie from cache and fetch data from disk.
	storage.tries.Delete(root)

	data, err := storage.GetStorage(&root, trieKV[0].key)
	require.NoError(t, err)
	require.Equal(t, trieKV[0].value, data)

	storage.tries.Delete(root)

	prefixKeys, err := storage.GetKeysWithPrefix(&root, []byte("ke"))
	require.NoError(t, err)
	require.Equal(t, 2, len(prefixKeys))

	storage.tries.Delete(root)

	entries, err := storage.Entries(&root)
	require.NoError(t, err)
	require.Equal(t, 3, len(entries))
}

func syncMapLen(m *sync.Map) int {
	l := 0
	m.Range(func(_, _ interface{}) bool {
		l++
		return true
	})
	return l
}

func TestStorage_StoreTrie_Syncing(t *testing.T) {
	storage := newTestStorageState(t)
	ts, err := storage.TrieState(&trie.EmptyHash)
	require.NoError(t, err)

	key := []byte("testkey")
	value := []byte("testvalue")
	ts.Set(key, value)

	storage.SetSyncing(true)
	err = storage.StoreTrie(ts, nil)
	require.NoError(t, err)
	require.Equal(t, 1, syncMapLen(storage.tries))
}

func TestStorage_StoreTrie_NotSyncing(t *testing.T) {
	storage := newTestStorageState(t)
	ts, err := storage.TrieState(&trie.EmptyHash)
	require.NoError(t, err)

	key := []byte("testkey")
	value := []byte("testvalue")
	ts.Set(key, value)

	storage.SetSyncing(false)
	err = storage.StoreTrie(ts, nil)
	require.NoError(t, err)
	require.Equal(t, 2, syncMapLen(storage.tries))
}

func TestGetStorageChildAndGetStorageFromChild(t *testing.T) {
	// initialise database using data directory
	basepath := t.TempDir()
	db, err := utils.SetupDatabase(basepath, false)
	require.NoError(t, err)

	_, genTrie, genHeader := genesis.NewTestGenesisWithTrieAndHeader(t)

	blockState, err := NewBlockStateFromGenesis(db, genHeader)
	require.NoError(t, err)

	testChildTrie := trie.NewEmptyTrie()
	testChildTrie.Put([]byte("keyInsidechild"), []byte("voila"))

	err = genTrie.PutChild([]byte("keyToChild"), testChildTrie)
	require.NoError(t, err)

	storage, err := NewStorageState(db, blockState, genTrie, pruner.Config{})
	require.NoError(t, err)

	ts, err := runtime.NewTrieState(genTrie)
	require.NoError(t, err)

	header, err := types.NewHeader(blockState.GenesisHash(), ts.MustRoot(),
		common.Hash{}, big.NewInt(1), types.NewDigest())
	require.NoError(t, err)

	err = storage.StoreTrie(ts, header)
	require.NoError(t, err)

	root, err := genTrie.Hash()
	require.NoError(t, err)

	_, err = storage.GetStorageChild(&root, []byte("keyToChild"))
	require.NoError(t, err)

	// Clear trie from cache and fetch data from disk.
	storage.tries.Delete(root)
	time.Sleep(time.Millisecond * 100)

	// Should these databases really be different?
	fmt.Println(storage.db == db)

	_, err = storage.GetStorageChild(&root, []byte("keyToChild"))
	require.NoError(t, err)

	value, err := storage.GetStorageFromChild(&root, []byte("keyToChild"), []byte("keyInsidechild"))
	require.NoError(t, err)

	require.Equal(t, []byte("voila"), value)
}
