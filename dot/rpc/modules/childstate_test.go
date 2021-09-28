// Copyright 2019 ChainSafe Systems (ON) Corp.
// This file is part of gossamer.
//
// The gossamer library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The gossamer library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the gossamer library. If not, see <http://www.gnu.org/licenses/>.

package modules

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/ChainSafe/chaindb"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/trie"
	"github.com/stretchr/testify/require"
)

func TestChildStateGetKeys(t *testing.T) {
	childStateModule, currBlockHash := setupChildStateStorage(t)

	req := &GetKeysRequest{
		Key:    []byte(":child_storage_key"),
		Prefix: []byte{},
		Hash:   common.EmptyHash,
	}

	res := make([]string, 0)
	err := childStateModule.GetKeys(nil, req, &res)
	require.NoError(t, err)
	require.Len(t, res, 3)

	for _, r := range res {
		b, dErr := common.HexToBytes(r)
		require.NoError(t, dErr)
		require.Contains(t, []string{
			":child_first", ":child_second", ":another_child",
		}, string(b))
	}

	req = &GetKeysRequest{
		Key:    []byte(":child_storage_key"),
		Prefix: []byte(":child_"),
		Hash:   currBlockHash,
	}

	err = childStateModule.GetKeys(nil, req, &res)
	require.NoError(t, err)
	require.Len(t, res, 2)

	for _, r := range res {
		b, err := common.HexToBytes(r)
		require.NoError(t, err)
		require.Contains(t, []string{
			":child_first", ":child_second",
		}, string(b))
	}
}

func TestChildStateGetStorageSize(t *testing.T) {
	mod, blockHash := setupChildStateStorage(t)

	tests := []struct {
		expect   uint64
		err      error
		hash     common.Hash
		keyChild []byte
		entry    []byte
	}{
		{
			err:      nil,
			expect:   uint64(len([]byte(":child_first_value"))),
			hash:     common.EmptyHash,
			entry:    []byte(":child_first"),
			keyChild: []byte(":child_storage_key"),
		},
		{
			err:      nil,
			expect:   uint64(len([]byte(":child_second_value"))),
			hash:     blockHash,
			entry:    []byte(":child_second"),
			keyChild: []byte(":child_storage_key"),
		},
		{
			err:      nil,
			expect:   0,
			hash:     common.EmptyHash,
			entry:    []byte(":not_found_so_size_0"),
			keyChild: []byte(":child_storage_key"),
		},
		{
			err:      fmt.Errorf("child trie does not exist at key %s%s", trie.ChildStorageKeyPrefix, []byte(":not_exist")),
			hash:     blockHash,
			entry:    []byte(":child_second"),
			keyChild: []byte(":not_exist"),
		},
		{
			err:  chaindb.ErrKeyNotFound,
			hash: common.BytesToHash([]byte("invalid block hash")),
		},
	}

	for _, test := range tests {
		var req GetChildStorageRequest
		var res uint64

		req.Hash = test.hash
		req.EntryKey = test.entry
		req.KeyChild = test.keyChild

		err := mod.GetStorageSize(nil, &req, &res)

		if test.err != nil {
			require.Error(t, err)
			require.Equal(t, err, test.err)
		} else {
			require.NoError(t, err)
		}

		require.Equal(t, test.expect, res)
	}
}

func setupChildStateStorage(t *testing.T) (*ChildStateModule, common.Hash) {
	t.Helper()

	st := newTestStateService(t)

	tr, err := st.Storage.TrieState(nil)
	require.NoError(t, err)

	tr.Set([]byte(":first_key"), []byte(":value1"))
	tr.Set([]byte(":second_key"), []byte(":second_value"))

	childTr := trie.NewEmptyTrie()
	childTr.Put([]byte(":child_first"), []byte(":child_first_value"))
	childTr.Put([]byte(":child_second"), []byte(":child_second_value"))
	childTr.Put([]byte(":another_child"), []byte("value"))

	err = tr.SetChild([]byte(":child_storage_key"), childTr)
	require.NoError(t, err)

	stateRoot, err := tr.Root()
	require.NoError(t, err)

	bb, err := st.Block.BestBlock()
	require.NoError(t, err)

	err = st.Storage.StoreTrie(tr, nil)
	require.NoError(t, err)

	b := &types.Block{
		Header: types.Header{
			ParentHash: bb.Header.Hash(),
			Number:     big.NewInt(0).Add(big.NewInt(1), bb.Header.Number),
			StateRoot:  stateRoot,
		},
		Body: []byte{},
	}

	err = st.Block.AddBlock(b)
	require.NoError(t, err)

	hash, _ := st.Block.GetBlockHash(b.Header.Number)
	return NewChildStateModule(st.Storage, st.Block), hash
}
