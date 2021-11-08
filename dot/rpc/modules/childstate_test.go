// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package modules

import (
	"encoding/hex"
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
		Hash:   nil,
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
		Hash:   &currBlockHash,
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
	invalidHash := common.BytesToHash([]byte("invalid block hash"))

	tests := []struct {
		expect   uint64
		err      error
		hash     *common.Hash
		keyChild []byte
		entry    []byte
	}{
		{
			err:      nil,
			expect:   uint64(len([]byte(":child_first_value"))),
			hash:     nil,
			entry:    []byte(":child_first"),
			keyChild: []byte(":child_storage_key"),
		},
		{
			err:      nil,
			expect:   uint64(len([]byte(":child_second_value"))),
			hash:     &blockHash,
			entry:    []byte(":child_second"),
			keyChild: []byte(":child_storage_key"),
		},
		{
			err:      nil,
			expect:   0,
			hash:     nil,
			entry:    []byte(":not_found_so_size_0"),
			keyChild: []byte(":child_storage_key"),
		},
		{
			err:      fmt.Errorf("child trie does not exist at key %s%s", trie.ChildStorageKeyPrefix, []byte(":not_exist")),
			hash:     &blockHash,
			entry:    []byte(":child_second"),
			keyChild: []byte(":not_exist"),
		},
		{
			err:  chaindb.ErrKeyNotFound,
			hash: &invalidHash,
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

func TestGetStorageHash(t *testing.T) {
	mod, blockHash := setupChildStateStorage(t)
	invalidBlockHash := common.BytesToHash([]byte("invalid block hash"))

	tests := []struct {
		expect   string
		err      error
		hash     *common.Hash
		keyChild []byte
		entry    []byte
	}{
		{
			err:      nil,
			expect:   common.BytesToHash([]byte(":child_first_value")).String(),
			hash:     nil,
			entry:    []byte(":child_first"),
			keyChild: []byte(":child_storage_key"),
		},
		{
			err:      nil,
			expect:   common.BytesToHash([]byte(":child_second_value")).String(),
			hash:     &blockHash,
			entry:    []byte(":child_second"),
			keyChild: []byte(":child_storage_key"),
		},
		{
			err:      fmt.Errorf("child trie does not exist at key %s%s", trie.ChildStorageKeyPrefix, []byte(":not_exist")),
			hash:     &blockHash,
			entry:    []byte(":child_second"),
			keyChild: []byte(":not_exist"),
		},
		{
			err:  chaindb.ErrKeyNotFound,
			hash: &invalidBlockHash,
		},
	}

	for _, test := range tests {
		var req GetStorageHash
		var res string

		req.Hash = test.hash
		req.EntryKey = test.entry
		req.KeyChild = test.keyChild

		err := mod.GetStorageHash(nil, &req, &res)

		if test.err != nil {
			require.Error(t, err)
			require.Equal(t, err, test.err)
		} else {
			require.NoError(t, err)
		}

		if test.expect != "" {
			require.Equal(t, test.expect, res)
		}
	}
}

func TestGetChildStorage(t *testing.T) {
	mod, blockHash := setupChildStateStorage(t)
	randomHash, err := common.HexToHash(RandomHash)
	require.NoError(t, err)

	testCases := []struct {
		params   []string
		expected []byte
		errMsg   string
	}{
		{params: []string{":child_storage_key", ""}, expected: nil},
		{params: []string{":child_storage_key", ":child_first"}, expected: []byte(":child_first_value")},
		{params: []string{":child_storage_key", ":child_first", blockHash.String()}, expected: []byte(":child_first_value")},
		{params: []string{":child_storage_key", ":child_first", randomHash.String()}, errMsg: "Key not found"},
	}

	for _, test := range testCases {
		t.Run(fmt.Sprintf("%s", test.params), func(t *testing.T) {
			var res StateStorageResponse
			var req ChildStateStorageRequest

			if test.params[0] != "" {
				req.ChildStorageKey = []byte(test.params[0])
			}

			if test.params[1] != "" {
				req.Key = []byte(test.params[1])
			}

			if len(test.params) > 2 && test.params[2] != "" {
				req.Hash = &common.Hash{}
				*req.Hash, err = common.HexToHash(test.params[2])
				require.NoError(t, err)
			}

			err = mod.GetStorage(nil, &req, &res)
			// Handle error cases.
			if test.errMsg != "" {
				require.Error(t, err)
				require.Equal(t, err.Error(), test.errMsg)
				return
			}

			// Verify expected values.
			require.NoError(t, err)
			if test.expected != nil {
				// Convert human-readable result value to hex.
				expectedVal := "0x" + hex.EncodeToString(test.expected)
				require.Equal(t, StateStorageResponse(expectedVal), res)
			}
		})
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
		Body: types.Body{},
	}

	err = st.Block.AddBlock(b)
	require.NoError(t, err)

	hash, _ := st.Block.GetBlockHash(b.Header.Number)
	return NewChildStateModule(st.Storage, st.Block), hash
}
