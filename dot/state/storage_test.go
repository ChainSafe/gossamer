package state

import (
	"math/big"
	"testing"

	"github.com/ChainSafe/gossamer/dot/types"
	runtime "github.com/ChainSafe/gossamer/lib/runtime/storage"
	"github.com/ChainSafe/gossamer/lib/trie"

	database "github.com/ChainSafe/chaindb"
	"github.com/stretchr/testify/require"
)

func newTestStorageState(t *testing.T) *StorageState {
	db := database.NewMemDatabase()
	bs := newTestBlockState(t, testGenesisHeader)

	s, err := NewStorageState(db, bs, trie.NewEmptyTrie())
	require.NoError(t, err)
	return s
}

func TestStorage_StoreAndLoadTrie(t *testing.T) {
	storage := newTestStorageState(t)
	ts, err := storage.TrieState(&trie.EmptyHash)
	require.NoError(t, err)

	root, err := ts.Root()
	require.NoError(t, err)
	err = storage.StoreTrie(root, ts)
	require.NoError(t, err)

	trie, err := storage.LoadFromDB(root)
	require.NoError(t, err)
	ts2, err := runtime.NewTrieState(storage.baseDB, trie)
	require.NoError(t, err)
	require.Equal(t, ts.Trie(), ts2.Trie())
}

func TestStorage_GetStorageByBlockHash(t *testing.T) {
	storage := newTestStorageState(t)
	ts, err := storage.TrieState(&trie.EmptyHash)
	require.NoError(t, err)

	key := []byte("testkey")
	value := []byte("testvalue")
	err = ts.Set(key, value)
	require.NoError(t, err)

	root, err := ts.Root()
	require.NoError(t, err)
	err = storage.StoreTrie(root, ts)
	require.NoError(t, err)

	block := &types.Block{
		Header: &types.Header{
			ParentHash: testGenesisHeader.Hash(),
			Number:     big.NewInt(1),
			StateRoot:  root,
		},
		Body: types.NewBody([]byte{}),
	}
	err = storage.blockState.AddBlock(block)
	require.NoError(t, err)

	res, err := storage.GetStorageByBlockHash(block.Header.Hash(), key)
	require.NoError(t, err)
	require.Equal(t, value, res)
}
