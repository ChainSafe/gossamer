// Copyright 2020 ChainSafe Systems (ON) Corp.
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
	"encoding/hex"
	"errors"
	"fmt"
	"io/ioutil"
	"math/big"
	"sort"
	"strings"
	"testing"

	"github.com/ChainSafe/gossamer/dot/core"
	"github.com/ChainSafe/gossamer/dot/rpc/modules/mocks"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const (
	RandomHash     = "0x580d77a9136035a0bc3c3cd86286172f7f81291164c5914266073a30466fba21"
	ErrKeyNotFound = "Key not found"
)

func TestStateModule_GetRuntimeVersion(t *testing.T) {
	t.Skip() // failing in development
	// expected results based on responses from prior tests
	expected := StateRuntimeVersionResponse{
		SpecName:         "node",
		ImplName:         "substrate-node",
		AuthoringVersion: 10,
		SpecVersion:      264,
		ImplVersion:      0,
		Apis: []interface{}{
			[]interface{}{"0xdf6acb689907609b", uint32(3)},
			[]interface{}{"0x37e397fc7c91f5e4", uint32(1)},
			[]interface{}{"0x40fe3ad401f8959a", uint32(4)},
			[]interface{}{"0xd2bc9897eed08f15", uint32(2)},
			[]interface{}{"0xf78b278be53f454c", uint32(2)},
			[]interface{}{"0xed99c5acb25eedf5", uint32(2)},
			[]interface{}{"0xcbca25e39f142387", uint32(2)},
			[]interface{}{"0x687ad44ad37f03c2", uint32(1)},
			[]interface{}{"0xbc9d89904f5b923f", uint32(1)},
			[]interface{}{"0x68b66ba122c93fa7", uint32(1)},
			[]interface{}{"0x37c8bb1350a9a2a8", uint32(1)},
			[]interface{}{"0x91d5df18b0d2cf58", uint32(1)},
			[]interface{}{"0xab3c0572291feb8b", uint32(1)},
		},
		TransactionVersion: 2,
	}

	sm, hash, _ := setupStateModule(t)
	randomHash, err := common.HexToHash(RandomHash)
	require.NoError(t, err)

	testCases := []struct {
		params string
		errMsg string
	}{
		{params: ""},
		{params: hash.String()},
		{params: randomHash.String(), errMsg: ErrKeyNotFound},
	}

	for _, test := range testCases {
		t.Run(test.params, func(t *testing.T) {
			var res StateRuntimeVersionResponse
			var req StateRuntimeVersionRequest

			if test.params != "" {
				req.Bhash = &common.Hash{}
				*req.Bhash, err = common.HexToHash(test.params)
				require.NoError(t, err)
			}

			err := sm.GetRuntimeVersion(nil, &req, &res)
			// Handle error cases.
			if test.errMsg != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), test.errMsg)
				return
			}

			// Verify expected values.
			require.Nil(t, err)
			require.Equal(t, expected, res)
		})
	}

}

func TestStateModule_GetPairs(t *testing.T) {
	sm, hash, _ := setupStateModule(t)

	randomHash, err := common.HexToHash(RandomHash)
	require.NoError(t, err)

	testCases := []struct {
		params   []string
		expected []interface{}
		errMsg   string
	}{
		{params: []string{"0x00"}, expected: nil},
		{params: []string{""}, expected: []interface{}{[]string{":key1", "value1"}, []string{":key2", "value2"}}},
		{params: []string{":key1"}, expected: []interface{}{[]string{":key1", "value1"}}},
		{params: []string{"0x00", hash.String()}, expected: nil},
		{params: []string{"", hash.String()}, expected: []interface{}{[]string{":key1", "value1"}, []string{":key2", "value2"}}},
		{params: []string{":key1", hash.String()}, expected: []interface{}{[]string{":key1", "value1"}}},
		{params: []string{"", randomHash.String()}, errMsg: "Key not found"},
	}

	for _, test := range testCases {
		t.Run(fmt.Sprintf("%s", test.params), func(t *testing.T) {
			var req StatePairRequest
			var res StatePairResponse

			// Convert human-readable param value to hex.
			req.Prefix = &test.params[0]
			if test.params[0] != "" && !strings.HasPrefix(test.params[0], "0x") {
				*req.Prefix = "0x" + hex.EncodeToString([]byte(test.params[0]))
			}

			if len(test.params) > 1 && test.params[1] != "" {
				req.Bhash = &common.Hash{}
				*req.Bhash, err = common.HexToHash(test.params[1])
				require.NoError(t, err)
			}

			err := sm.GetPairs(nil, &req, &res)

			// Handle error cases.
			if test.errMsg != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), test.errMsg)
				return
			}

			// Verify expected values.
			require.NoError(t, err)
			sort.Slice(res, func(i, j int) bool {
				return res[i].([]string)[0] < res[j].([]string)[0]
			})

			require.Equal(t, len(test.expected), len(res))
			for idx, val := range test.expected {
				kv, _ := res[idx].([]string)
				require.Equal(t, len(kv), 2)

				// Convert human-readable result value to hex.
				expectedKV, _ := val.([]string)
				expectedKey := "0x" + hex.EncodeToString([]byte(expectedKV[0]))
				expectedVal := "0x" + hex.EncodeToString([]byte(expectedKV[1]))

				require.Equal(t, []string{expectedKey, expectedVal}, kv)
			}
		})
	}
}

func TestStateModule_GetStorage(t *testing.T) {
	sm, hash, _ := setupStateModule(t)
	randomHash, err := common.HexToHash(RandomHash)
	require.NoError(t, err)

	testCases := []struct {
		params   []string
		expected []byte
		errMsg   string
	}{
		{params: []string{""}, expected: nil},
		{params: []string{":key1"}, expected: []byte("value1")},
		{params: []string{":key1", hash.String()}, expected: []byte("value1")},
		{params: []string{"", randomHash.String()}, errMsg: "Key not found"},
	}

	for _, test := range testCases {
		t.Run(fmt.Sprintf("%s", test.params), func(t *testing.T) {
			var res StateStorageResponse
			var req StateStorageRequest

			if test.params[0] != "" {
				req.Key = "0x" + hex.EncodeToString([]byte(test.params[0]))
			}

			if len(test.params) > 1 && test.params[1] != "" {
				req.Bhash = &common.Hash{}
				*req.Bhash, err = common.HexToHash(test.params[1])
				require.NoError(t, err)
			}

			err = sm.GetStorage(nil, &req, &res)
			// Handle error cases.
			if test.errMsg != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), test.errMsg)
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

func TestStateModule_GetStorageHash(t *testing.T) {
	sm, hash, _ := setupStateModule(t)
	randomHash, err := common.HexToHash(RandomHash)
	require.NoError(t, err)

	testCases := []struct {
		params   []string
		expected []byte
		errMsg   string
	}{
		{params: []string{""}, expected: nil},
		{params: []string{":key1"}, expected: []byte("value1")},
		{params: []string{":key1", hash.String()}, expected: []byte("value1")},
		{params: []string{"0x", randomHash.String()}, errMsg: "Key not found"},
	}

	for _, test := range testCases {
		t.Run(fmt.Sprintf("%s", test.params), func(t *testing.T) {
			var res StateStorageHashResponse
			var req StateStorageHashRequest

			if test.params[0] != "" {
				req.Key = "0x" + hex.EncodeToString([]byte(test.params[0]))
			}

			if len(test.params) > 1 && test.params[1] != "" {
				req.Bhash = &common.Hash{}
				*req.Bhash, err = common.HexToHash(test.params[1])
				require.NoError(t, err)
			}

			err := sm.GetStorageHash(nil, &req, &res)
			// Handle error cases.
			if test.errMsg != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), test.errMsg)
				return
			}

			require.NoError(t, err)
			if test.expected == nil {
				require.Empty(t, res)
				return
			}

			// Convert human-readable result value to hex.
			expectedVal := common.BytesToHash(test.expected)
			require.Equal(t, StateStorageHashResponse(expectedVal.String()), res)
		})
	}
}

func TestStateModule_GetStorageSize(t *testing.T) {
	sm, hash, _ := setupStateModule(t)
	randomHash, err := common.HexToHash(RandomHash)
	require.NoError(t, err)

	testCases := []struct {
		params   []string
		expected StateStorageSizeResponse
		errMsg   string
	}{
		{params: []string{""}},
		{params: []string{":key1"}, expected: 6},
		{params: []string{":key1", hash.String()}, expected: 6},
		{params: []string{"0x", randomHash.String()}, errMsg: "Key not found"},
	}

	for _, test := range testCases {
		var res StateStorageSizeResponse
		var req StateStorageSizeRequest

		t.Run(fmt.Sprintf("%s", test.params), func(t *testing.T) {
			if test.params[0] != "" {
				req.Key = "0x" + hex.EncodeToString([]byte(test.params[0]))
			}

			if len(test.params) > 1 && test.params[1] != "" {
				req.Bhash = &common.Hash{}
				*req.Bhash, err = common.HexToHash(test.params[1])
				require.NoError(t, err)
			}

			err := sm.GetStorageSize(nil, &req, &res)
			// Handle error cases.
			if test.errMsg != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), test.errMsg)
				return
			}

			require.NoError(t, err)
			require.Equal(t, test.expected, res)
		})
	}
}

func TestStateModule_QueryStorage(t *testing.T) {
	t.Run("When starting block is empty", func(t *testing.T) {
		module := new(StateModule)
		req := new(StateStorageQueryRangeRequest)

		var res []StorageChangeSetResponse
		err := module.QueryStorage(nil, req, &res)
		require.Error(t, err, "the start block hash cannot be an empty value")
	})

	t.Run("When coreAPI QueryStorage returns error", func(t *testing.T) {
		coreapimock := new(mocks.MockCoreAPI)
		coreapimock.On("QueryStorage", mock.AnythingOfType("common.Hash"), mock.AnythingOfType("common.Hash")).Return(nil, errors.New("problem while querying"))

		module := new(StateModule)
		module.coreAPI = coreapimock

		req := new(StateStorageQueryRangeRequest)
		req.StartBlock = common.NewHash([]byte{1, 2})

		var res []StorageChangeSetResponse
		err := module.QueryStorage(nil, req, &res)
		require.Error(t, err)
		coreapimock.AssertCalled(t, "QueryStorage", mock.AnythingOfType("common.Hash"), mock.AnythingOfType("common.Hash"))
	})

	t.Run("When QueryStorage returns data", func(t *testing.T) {
		blockhash := common.NewHash([]byte{123})

		changes := map[common.Hash]core.QueryKeyValueChanges{
			blockhash: core.QueryKeyValueChanges(map[string]string{
				"0x80": "value",
				"0x90": "another value",
			}),
		}
		coreapimock := new(mocks.MockCoreAPI)
		coreapimock.On("QueryStorage", mock.AnythingOfType("common.Hash"), mock.AnythingOfType("common.Hash"), "0x90", "0x80").Return(changes, nil)

		module := new(StateModule)
		module.coreAPI = coreapimock

		req := new(StateStorageQueryRangeRequest)
		req.StartBlock = common.NewHash([]byte{1, 2})
		req.Keys = []string{"0x90", "0x80"}

		var res []StorageChangeSetResponse
		err := module.QueryStorage(nil, req, &res)
		require.NoError(t, err)
		require.Len(t, res, 1)

		coreapimock.AssertCalled(t, "QueryStorage", mock.AnythingOfType("common.Hash"), mock.AnythingOfType("common.Hash"), "0x90", "0x80")
	})
}

func TestStateModule_GetMetadata(t *testing.T) {
	t.Skip() // TODO: update expected_metadata (#1026)
	sm, hash, _ := setupStateModule(t)
	randomHash, err := common.HexToHash(RandomHash)
	require.NoError(t, err)

	expectedMetadata, err := ioutil.ReadFile("./test_data/expected_metadata")
	require.NoError(t, err)

	testCases := []struct {
		params string
		errMsg string
	}{
		{params: ""},
		{params: hash.String()},
		{params: randomHash.String(), errMsg: ErrKeyNotFound},
	}

	for _, test := range testCases {
		t.Run(test.params, func(t *testing.T) {
			var res StateMetadataResponse
			var req StateRuntimeMetadataQuery

			if test.params != "" {
				req.Bhash = &common.Hash{}
				*req.Bhash, err = common.HexToHash(test.params)
				require.NoError(t, err)
			}

			err := sm.GetMetadata(nil, &req, &res)
			if test.errMsg != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), test.errMsg)
				return
			}

			require.NoError(t, err)
			require.Equal(t, string(expectedMetadata), string(res))
		})
	}
}

func TestStateModule_GetKeysPaged(t *testing.T) {
	sm, _, stateRootHash := setupStateModule(t)

	testCases := []struct {
		name     string
		params   StateStorageKeyRequest
		expected []string
	}{
		{name: "allKeysNilBlockHash",
			params: StateStorageKeyRequest{
				Qty:   10,
				Block: nil,
			}, expected: []string{"0x3a6b657931", "0x3a6b657932"}},
		{name: "allKeysTestBlockHash",
			params: StateStorageKeyRequest{
				Qty:   10,
				Block: stateRootHash,
			}, expected: []string{"0x3a6b657931", "0x3a6b657932"}},
		{name: "prefixMatchAll",
			params: StateStorageKeyRequest{
				Prefix: "0x3a6b6579",
				Qty:    10,
			}, expected: []string{"0x3a6b657931", "0x3a6b657932"}},
		{name: "prefixMatchOne",
			params: StateStorageKeyRequest{
				Prefix: "0x3a6b657931",
				Qty:    10,
			}, expected: []string{"0x3a6b657931"}},
		{name: "prefixMatchNone",
			params: StateStorageKeyRequest{
				Prefix: "0x00",
				Qty:    10,
			}, expected: nil},
		{name: "qtyOne",
			params: StateStorageKeyRequest{
				Qty: 1,
			}, expected: []string{"0x3a6b657931"}},
		{name: "afterKey",
			params: StateStorageKeyRequest{
				Qty:      10,
				AfterKey: "0x3a6b657931",
			}, expected: []string{"0x3a6b657932"}},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			var res StateStorageKeysResponse

			err := sm.GetKeysPaged(nil, &test.params, &res)
			require.NoError(t, err)

			if test.expected == nil {
				require.Empty(t, res)
				return
			}

			require.Equal(t, StateStorageKeysResponse(test.expected), res)
		})
	}
}

func TestGetReadProof_WhenCoreAPIReturnsError(t *testing.T) {
	coreAPIMock := new(mocks.MockCoreAPI)
	coreAPIMock.
		On("GetReadProofAt", mock.AnythingOfType("common.Hash"), mock.AnythingOfType("[][]uint8")).
		Return(common.EmptyHash, nil, errors.New("mocked error"))

	sm := new(StateModule)
	sm.coreAPI = coreAPIMock

	req := &StateGetReadProofRequest{
		Keys: []string{},
		Hash: common.EmptyHash,
	}
	err := sm.GetReadProof(nil, req, nil)
	require.Error(t, err, "mocked error")
}

func TestGetReadProof_WhenReturnsProof(t *testing.T) {
	expectedBlock := common.BytesToHash([]byte("random hash"))
	mockedProof := [][]byte{[]byte("proof-1"), []byte("proof-2")}

	coreAPIMock := new(mocks.MockCoreAPI)
	coreAPIMock.
		On("GetReadProofAt", mock.AnythingOfType("common.Hash"), mock.AnythingOfType("[][]uint8")).
		Return(expectedBlock, mockedProof, nil)

	sm := new(StateModule)
	sm.coreAPI = coreAPIMock

	req := &StateGetReadProofRequest{
		Keys: []string{},
		Hash: common.EmptyHash,
	}

	res := new(StateGetReadProofResponse)
	err := sm.GetReadProof(nil, req, res)
	require.NoError(t, err)
	require.Equal(t, res.At, expectedBlock)

	expectedProof := []string{
		common.BytesToHex([]byte("proof-1")),
		common.BytesToHex([]byte("proof-2")),
	}

	require.Equal(t, res.Proof, expectedProof)
}

func setupStateModule(t *testing.T) (*StateModule, *common.Hash, *common.Hash) {
	// setup service
	net := newNetworkService(t)
	chain := newTestStateService(t)
	// init storage with test data
	ts, err := chain.Storage.TrieState(nil)
	require.NoError(t, err)

	ts.Set([]byte(`:key2`), []byte(`value2`))
	ts.Set([]byte(`:key1`), []byte(`value1`))
	ts.SetChildStorage([]byte(`:child1`), []byte(`:key1`), []byte(`:childValue1`))

	sr1, err := ts.Root()
	require.NoError(t, err)
	err = chain.Storage.StoreTrie(ts, nil)
	require.NoError(t, err)

	b := &types.Block{
		Header: types.Header{
			ParentHash: chain.Block.BestBlockHash(),
			Number:     big.NewInt(2),
			StateRoot:  sr1,
		},
		Body: *types.NewBody([]types.Extrinsic{[]byte{}}),
	}

	err = chain.Block.AddBlock(b)
	require.NoError(t, err)

	rt, err := chain.Block.GetRuntime(&b.Header.ParentHash)
	require.NoError(t, err)

	chain.Block.StoreRuntime(b.Header.Hash(), rt)

	hash, _ := chain.Block.GetBlockHash(big.NewInt(2))
	core := newCoreService(t, chain)
	return NewStateModule(net, chain.Storage, core), &hash, &sr1
}
