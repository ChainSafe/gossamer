package state

import (
	"bytes"
	"reflect"
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

	err := StoreTrie(db, tt)
	require.NoError(t, err)

	encroot, err := tt.Hash()
	require.NoError(t, err)

	expected := tt.MustHash()

	tt = trie.NewEmptyTrie()
	err = LoadTrie(db, tt, encroot)
	require.NoError(t, err)
	require.Equal(t, expected, tt.MustHash())
}

type test struct {
	key   []byte
	value []byte
}

func TestStoreAndLoadLatestStorageHash(t *testing.T) {
	db := NewInMemoryDB(t)
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
	if err != nil {
		t.Fatal(err)
	}

	err = StoreLatestStorageHash(db, expected)
	if err != nil {
		t.Fatal(err)
	}

	hash, err := LoadLatestStorageHash(db)
	if err != nil {
		t.Fatal(err)
	}

	if hash != expected {
		t.Fatalf("Fail: got %x expected %x", hash, expected)
	}
}

func TestStoreAndLoadGenesisData(t *testing.T) {
	db := NewInMemoryDB(t)

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

	err := StoreGenesisData(db, expected)
	if err != nil {
		t.Fatal(err)
	}

	gen, err := LoadGenesisData(db)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(gen, expected) {
		t.Fatalf("Fail: got %v expected %v", gen, expected)
	}
}

func TestStoreAndLoadBestBlockHash(t *testing.T) {
	db := NewInMemoryDB(t)
	hash, _ := common.HexToHash("0x3f5a19b9e9507e05276216f3877bb289e47885f8184010c65d0e41580d3663cc")

	err := StoreBestBlockHash(db, hash)
	if err != nil {
		t.Fatal(err)
	}

	res, err := LoadBestBlockHash(db)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(res, hash) {
		t.Fatalf("Fail: got %x expected %x", res, hash)
	}
}
