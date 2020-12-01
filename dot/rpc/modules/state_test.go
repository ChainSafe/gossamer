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
	"fmt"
	"io/ioutil"
	"math/big"
	"sort"
	"strings"
	"testing"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/stretchr/testify/require"
)

const (
	RandomHash     = "0x580d77a9136035a0bc3c3cd86286172f7f81291164c5914266073a30466fba21"
	ErrKeyNotFound = "Key not found"
)

func TestStateModule_GetRuntimeVersion(t *testing.T) {
	// expected results based on responses from prior tests
	expected := StateRuntimeVersionResponse{
		SpecName:         "node",
		ImplName:         "substrate-node",
		AuthoringVersion: 10,
		SpecVersion:      193,
		ImplVersion:      193,
		Apis: []interface{}{[]interface{}{"0xdf6acb689907609b", int32(2)},
			[]interface{}{"0x37e397fc7c91f5e4", int32(1)},
			[]interface{}{"0x40fe3ad401f8959a", int32(3)},
			[]interface{}{"0xd2bc9897eed08f15", int32(1)},
			[]interface{}{"0xf78b278be53f454c", int32(1)},
			[]interface{}{"0xed99c5acb25eedf5", int32(2)},
			[]interface{}{"0xcbca25e39f142387", int32(1)},
			[]interface{}{"0xbc9d89904f5b923f", int32(1)},
			[]interface{}{"0x68b66ba122c93fa7", int32(1)},
			[]interface{}{"0x37c8bb1350a9a2a8", int32(1)},
			[]interface{}{"0xab3c0572291feb8b", int32(1)},
		},
	}
	sm, hash := setupStateModule(t)

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

	res := StateRuntimeVersionResponse{}
	for _, test := range testCases {
		t.Run(test.params, func(t *testing.T) {
			err := sm.GetRuntimeVersion(nil, &test.params, &res)

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
	sm, hash := setupStateModule(t)

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
			// Convert human-readable param value to hex.
			if test.params[0] != "" && !strings.HasPrefix(test.params[0], "0x") {
				test.params[0] = "0x" + hex.EncodeToString([]byte(test.params[0]))
			}

			var res []interface{}
			err := sm.GetPairs(nil, &test.params, &res)

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
	sm, hash := setupStateModule(t)
	var res interface{}

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
			if test.params[0] != "" {
				test.params[0] = "0x" + hex.EncodeToString([]byte(test.params[0]))
			}

			err = sm.GetStorage(nil, &test.params, &res)

			// Handle error cases.
			if test.errMsg != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), test.errMsg)
				return
			}

			// Verify expected values.
			require.NoError(t, err)
			if test.expected != nil {
				require.NoError(t, err)

				// Convert human-readable result value to hex.
				expectedVal := "0x" + hex.EncodeToString(test.expected)
				require.Equal(t, expectedVal, res)
			}
		})
	}
}

func TestStateModule_GetStorageHash(t *testing.T) {
	sm, hash := setupStateModule(t)
	var res interface{}

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
			if test.params[0] != "" {
				test.params[0] = "0x" + hex.EncodeToString([]byte(test.params[0]))
			}

			err := sm.GetStorageHash(nil, &test.params, &res)
			// Handle error cases.
			if test.errMsg != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), test.errMsg)
				return
			}

			require.NoError(t, err)
			if test.expected == nil {
				require.Nil(t, res)
				return
			}

			// Convert human-readable result value to hex.
			expectedVal := common.BytesToHash(test.expected)
			require.Equal(t, expectedVal.String(), res)
		})
	}
}

func TestStateModule_GetStorageSize(t *testing.T) {
	sm, hash := setupStateModule(t)
	var res interface{}

	randomHash, err := common.HexToHash(RandomHash)
	require.NoError(t, err)

	testCases := []struct {
		params   []string
		expected interface{}
		errMsg   string
	}{
		{params: []string{""}},
		{params: []string{":key1"}, expected: 6},
		{params: []string{":key1", hash.String()}, expected: 6},
		{params: []string{"0x", randomHash.String()}, errMsg: "Key not found"},
	}

	for _, test := range testCases {
		t.Run(fmt.Sprintf("%s", test.params), func(t *testing.T) {
			if test.params[0] != "" {
				test.params[0] = "0x" + hex.EncodeToString([]byte(test.params[0]))
			}

			err := sm.GetStorageSize(nil, &test.params, &res)
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

func TestStateModule_GetMetadata(t *testing.T) {
	sm, hash := setupStateModule(t)
	var res string

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
			err := sm.GetMetadata(nil, &test.params, &res)
			if test.errMsg != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), test.errMsg)
				return
			}

			require.NoError(t, err)
			require.Equal(t, string(expectedMetadata), res)
		})
	}
}

func setupStateModule(t *testing.T) (*StateModule, *common.Hash) {
	// setup service
	net := newNetworkService(t)
	chain := newTestStateService(t)
	// init storage with test data
	ts, err := chain.Storage.TrieState(nil)
	require.NoError(t, err)

	err = ts.Set([]byte(`:key1`), []byte(`value1`))
	require.NoError(t, err)
	err = ts.Set([]byte(`:key2`), []byte(`value2`))
	require.NoError(t, err)

	sr1, err := ts.Root()
	require.NoError(t, err)
	err = chain.Storage.StoreTrie(sr1, ts)
	require.NoError(t, err)

	err = chain.Block.AddBlock(&types.Block{
		Header: &types.Header{
			ParentHash: chain.Block.BestBlockHash(),
			Number:     big.NewInt(2),
			StateRoot:  sr1,
		},
		Body: types.NewBody([]byte{}),
	})
	require.NoError(t, err)

	hash, _ := chain.Block.GetBlockHash(big.NewInt(2))
	core := newCoreService(t, chain)
	return NewStateModule(net, chain.Storage, core), hash
}
