// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package state

import (
	"math/big"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/dot/state/pruner"
	"github.com/ChainSafe/gossamer/dot/telemetry"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/internal/trie/node"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/genesis"
	runtime "github.com/ChainSafe/gossamer/lib/runtime/storage"
	"github.com/ChainSafe/gossamer/lib/trie"
	"github.com/ChainSafe/gossamer/lib/utils"
	"github.com/golang/mock/gomock"

	"github.com/stretchr/testify/require"
)

func newTestStorageState(t *testing.T, tries *Tries) *StorageState {
	db := NewInMemoryDB(t)

	bs := newTestBlockState(t, testGenesisHeader, tries)

	s, err := NewStorageState(db, bs, tries, pruner.Config{})
	require.NoError(t, err)
	return s
}

func TestStorage_StoreAndLoadTrie(t *testing.T) {
	ctrl := gomock.NewController(t)

	triesGauge := NewMockGauge(ctrl)
	triesGauge.EXPECT().Inc()
	tries := &Tries{
		rootToTrie: make(map[common.Hash]*trie.Trie),
		triesGauge: triesGauge,
	}

	storage := newTestStorageState(t, tries)
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
	ctrl := gomock.NewController(t)

	triesGauge := NewMockGauge(ctrl)
	triesGauge.EXPECT().Inc().Times(2)
	tries := &Tries{
		rootToTrie: make(map[common.Hash]*trie.Trie),
		triesGauge: triesGauge,
	}

	storage := newTestStorageState(t, tries)
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
	ctrl := gomock.NewController(t)

	triesGauge := NewMockGauge(ctrl)
	triesGauge.EXPECT().Inc().Times(3)
	triesGauge.EXPECT().Set(1.00).Times(1)
	tries := &Tries{
		rootToTrie: make(map[common.Hash]*trie.Trie),
		triesGauge: triesGauge,
	}

	storage := newTestStorageState(t, tries)
	ts, err := storage.TrieState(&trie.EmptyHash)
	require.NoError(t, err)
	ts.Set([]byte("noot"), []byte("washere"))

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
	ctrl := gomock.NewController(t)

	triesGauge := NewMockGauge(ctrl)
	triesGauge.EXPECT().Inc().Times(4)
	triesGauge.EXPECT().Set(1.00).Times(3)
	tries := &Tries{
		rootToTrie: make(map[common.Hash]*trie.Trie),
		triesGauge: triesGauge,
	}

	storage := newTestStorageState(t, tries)
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
	require.Equal(t, 3, len(entries))
}

func TestStorage_StoreTrie_NotSyncing(t *testing.T) {
	ctrl := gomock.NewController(t)

	triesGauge := NewMockGauge(ctrl)
	triesGauge.EXPECT().Inc().Times(2)
	tries := &Tries{
		rootToTrie: make(map[common.Hash]*trie.Trie),
		triesGauge: triesGauge,
	}

	storage := newTestStorageState(t, tries)
	ts, err := storage.TrieState(&trie.EmptyHash)
	require.NoError(t, err)

	key := []byte("testkey")
	value := []byte("testvalue")
	ts.Set(key, value)

	err = storage.StoreTrie(ts, nil)
	require.NoError(t, err)
	require.Equal(t, 2, storage.blockState.tries.len())
}

func TestGetStorageChildAndGetStorageFromChild(t *testing.T) {
	// initialise database using data directory
	basepath := t.TempDir()
	db, err := utils.SetupDatabase(basepath, false)
	require.NoError(t, err)

	_, genTrie, genHeader := genesis.NewTestGenesisWithTrieAndHeader(t)

	ctrl := gomock.NewController(t)
	telemetryMock := NewMockClient(ctrl)
	telemetryMock.EXPECT().SendMessage(telemetry.NewNotifyFinalized(
		genHeader.Hash(),
		"0",
	))

	key := []byte{1, 2}
	value := []byte{3, 4}
	const dirty = true
	const generation = 0
	testChildTrie := trie.NewTrie(node.NewLeaf(key, value, dirty, generation))

	testChildTrie.Put([]byte("keyInsidechild"), []byte("voila"))

	err = genTrie.PutChild([]byte("keyToChild"), testChildTrie)
	require.NoError(t, err)

	triesGauge := NewMockGauge(ctrl)
	triesGauge.EXPECT().Inc().Times(2)
	triesGauge.EXPECT().Set(0.00)

	tries := &Tries{
		rootToTrie: make(map[common.Hash]*trie.Trie),
		triesGauge: triesGauge,
	}

	blockState, err := NewBlockStateFromGenesis(db, tries, genHeader, telemetryMock)
	require.NoError(t, err)

	storage, err := NewStorageState(db, blockState, tries, pruner.Config{})
	require.NoError(t, err)

	trieState, err := runtime.NewTrieState(genTrie)
	require.NoError(t, err)

	header, err := types.NewHeader(blockState.GenesisHash(), trieState.MustRoot(),
		common.Hash{}, big.NewInt(1), types.NewDigest())
	require.NoError(t, err)

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

	value, err = storage.GetStorageFromChild(&rootHash, []byte("keyToChild"), []byte("keyInsidechild"))
	require.NoError(t, err)

	require.Equal(t, []byte("voila"), value)
}
