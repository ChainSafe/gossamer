// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package state

import (
	"testing"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/genesis"
	inmemory_trie "github.com/ChainSafe/gossamer/pkg/trie/inmemory"

	"github.com/stretchr/testify/require"
)

func TestTrie_StoreAndLoadFromDB(t *testing.T) {
	db := NewInMemoryDB(t)
	tt := inmemory_trie.NewEmptyTrie()

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

	tt = inmemory_trie.NewEmptyTrie()
	err = tt.Load(db, encroot)
	require.NoError(t, err)
	require.Equal(t, expected, tt.MustHash())
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
