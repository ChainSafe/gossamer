package state

import (
	"bytes"
	"testing"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/genesis"
	"github.com/ChainSafe/gossamer/lib/trie"

	"github.com/stretchr/testify/require"
)

func TestTrie_StoreAndLoadFromDB(t *testing.T) {
	db := NewInMemoryDB(t)
	tt := trie.NewEmptyTrie()

	rt := trie.GenerateRandomTests(t, 1000)
	for _, test := range rt {
		tt.Put(test.Key(), test.Value())

		val := tt.Get(test.Key())
		if !bytes.Equal(val, test.Value()) {
			t.Errorf("Fail to get key %x with value %x: got %x", test.Key(), test.Value(), val)
		}
	}

	err := tt.Store(db)
	require.NoError(t, err)

	encroot, err := tt.Hash()
	require.NoError(t, err)

	expected := tt.MustHash()

	tt = trie.NewEmptyTrie()
	err = tt.Load(db, encroot)
	require.NoError(t, err)
	require.Equal(t, expected, tt.MustHash())
}

type test struct {
	key   []byte
	value []byte
}

func TestStoreAndLoadLatestStorageHash(t *testing.T) {
	db := NewInMemoryDB(t)
	base := NewBaseState(db)
	tt := trie.NewEmptyTrie()

	tests := []test{
		{key: []byte{0x01, 0x35}, value: []byte("pen")},
		{key: []byte{0x01, 0x35, 0x79}, value: []byte("penguin")},
		{key: []byte{0x01, 0x35, 0x7}, value: []byte("g")},
		{key: []byte{0xf2}, value: []byte("feather")},
		{key: []byte{0xf2, 0x3}, value: []byte("f")},
		{key: []byte{0x09, 0xd3}, value: []byte("noot")},
		{key: []byte{0x07}, value: []byte("ramen")},
		{key: []byte{0}, value: nil},
	}

	for _, test := range tests {
		tt.Put(test.key, test.value)
	}

	expected, err := tt.Hash()
	require.NoError(t, err)

	err = base.StoreLatestStorageHash(expected)
	require.NoError(t, err)

	hash, err := base.LoadLatestStorageHash()
	require.NoError(t, err)
	require.Equal(t, expected, hash)
}

func TestStoreAndLoadGenesisData(t *testing.T) {
	db := NewInMemoryDB(t)
	base := NewBaseState(db)

	bootnodes := common.StringArrayToBytes([]string{
		"/ip4/127.0.0.1/tcp/7001/p2p/12D3KooWHHzSeKaY8xuZVzkLbKFfvNgPPeKhFBGrMbNzbm5akpqu",
		"/ip4/127.0.0.1/tcp/7001/p2p/12D3KooWHHzSeKaY8xuZVzkLbKFfvNgPPeKhFBGrMbNzbm5akpqu",
	})

	expected := &genesis.Data{
		Name:       "gossamer",
		ID:         "gossamer",
		Bootnodes:  bootnodes,
		ProtocolID: "/gossamer/test/0",
	}

	err := base.StoreGenesisData(expected)
	require.NoError(t, err)

	gen, err := base.LoadGenesisData()
	require.NoError(t, err)
	require.Equal(t, expected, gen)
}

func TestStoreAndLoadBestBlockHash(t *testing.T) {
	db := NewInMemoryDB(t)
	base := NewBaseState(db)

	hash, _ := common.HexToHash("0x3f5a19b9e9507e05276216f3877bb289e47885f8184010c65d0e41580d3663cc")

	err := base.StoreBestBlockHash(hash)
	require.NoError(t, err)

	res, err := base.LoadBestBlockHash()
	require.NoError(t, err)
	require.Equal(t, hash, res)
}

func TestLoadStoreEpochLength(t *testing.T) {
	db := NewInMemoryDB(t)
	base := NewBaseState(db)

	length := uint64(2222)
	err := base.storeEpochLength(length)
	require.NoError(t, err)

	ret, err := base.loadEpochLength()
	require.NoError(t, err)
	require.Equal(t, length, ret)
}

func TestLoadAndStoreSlotDuration(t *testing.T) {
	db := NewInMemoryDB(t)
	base := NewBaseState(db)

	d := uint64(3000)
	err := base.storeSlotDuration(d)
	require.NoError(t, err)

	ret, err := base.loadSlotDuration()
	require.NoError(t, err)
	require.Equal(t, d, ret)
}
