// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package state

import (
	"testing"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/genesis"
	"github.com/ChainSafe/gossamer/lib/trie"

	"github.com/stretchr/testify/require"
)

func TestTrie_StoreAndLoadFromDB(t *testing.T) {
	db := newInMemoryDB(t)
	tt := trie.NewEmptyTrie()

	generator := newGenerator()
	const size = 500
	kv := generateKeyValues(t, generator, size)

	for keyString, value := range kv {
		key := []byte(keyString)
		tt.Put(key, value)
	}

	err := tt.WriteDirty(db)
	require.NoError(t, err)

	encroot, err := tt.Hash()
	require.NoError(t, err)

	expected := tt.MustHash()

	tt = trie.NewEmptyTrie()
	err = tt.Load(db, encroot)
	require.NoError(t, err)
	require.Equal(t, expected, tt.MustHash())
}

func TestStoreAndLoadGenesisData(t *testing.T) {
	db := newInMemoryDB(t)
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

func TestLoadStoreEpochLength(t *testing.T) {
	db := newInMemoryDB(t)
	base := NewBaseState(db)

	length := uint64(2222)
	err := base.storeEpochLength(length)
	require.NoError(t, err)

	ret, err := base.loadEpochLength()
	require.NoError(t, err)
	require.Equal(t, length, ret)
}

func TestLoadAndStoreSlotDuration(t *testing.T) {
	db := newInMemoryDB(t)
	base := NewBaseState(db)

	d := uint64(3000)
	err := base.storeSlotDuration(d)
	require.NoError(t, err)

	ret, err := base.loadSlotDuration()
	require.NoError(t, err)
	require.Equal(t, d, ret)
}
