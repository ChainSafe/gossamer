// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

//go:build integration

package modules

import (
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
	"testing"

	"github.com/ChainSafe/gossamer/dot/rpc/modules/mocks"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/trie"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

const (
	RandomHash     = "0x580d77a9136035a0bc3c3cd86286172f7f81291164c5914266073a30466fba21"
	ErrKeyNotFound = "pebble: not found"
)

func TestStateModule_GetRuntimeVersion(t *testing.T) {
	/* expected results based on responses from prior tests
	We can get this data from polkadot runtime release source code
	https://github.com/paritytech/polkadot/blob/v0.9.29/runtime/westend/src/lib.rs#L105-L117
	*/
	expected := StateRuntimeVersionResponse{
		SpecName:         "westend",
		ImplName:         "parity-westend",
		AuthoringVersion: 2,
		SpecVersion:      9290,
		ImplVersion:      0,
		Apis: []interface{}{
			[]interface{}{"0xdf6acb689907609b", uint32(4)},
			[]interface{}{"0x37e397fc7c91f5e4", uint32(1)},
			[]interface{}{"0x40fe3ad401f8959a", uint32(6)},
			[]interface{}{"0xd2bc9897eed08f15", uint32(3)},
			[]interface{}{"0xf78b278be53f454c", uint32(2)},
			[]interface{}{"0xaf2c0297a23e6d3d", uint32(2)},
			[]interface{}{"0x49eaaf1b548a0cb0", uint32(1)},
			[]interface{}{"0x91d5df18b0d2cf58", uint32(1)},
			[]interface{}{"0xed99c5acb25eedf5", uint32(3)},
			[]interface{}{"0xcbca25e39f142387", uint32(2)},
			[]interface{}{"0x687ad44ad37f03c2", uint32(1)},
			[]interface{}{"0xab3c0572291feb8b", uint32(1)},
			[]interface{}{"0xbc9d89904f5b923f", uint32(1)},
			[]interface{}{"0x37c8bb1350a9a2a8", uint32(1)},
			[]interface{}{"0xf3ff14d5ab527059", uint32(1)},
			[]interface{}{"0x17a6bc0d0062aeb3", uint32(1)},
		},
		TransactionVersion: 12,
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
			require.NoError(t, err)
			require.Equal(t, expected, res)
		})
	}

}

func TestStateModule_GetPairs(t *testing.T) {
	sm, hash, _ := setupStateModule(t)

	randomHash, err := common.HexToHash(RandomHash)
	require.NoError(t, err)

	hexEncode := func(s string) string {
		return "0x" + hex.EncodeToString([]byte(s))
	}

	testCases := []struct {
		params   []string
		expected []interface{}
		errMsg   string
	}{
		{params: []string{"0x00"}, expected: nil},
		{params: []string{""}, expected: []interface{}{
			[]string{hexEncode(":child_storage:default::child1"),
				"0x8f733acc98dff0e6527f97e2a87e4834cd8b2e601f56fb003084e9d43183d7ff"},
			[]string{hexEncode(":key1"), hexEncode("value1")},
			[]string{hexEncode(":key2"), hexEncode("value2")}}},
		{params: []string{hexEncode(":key1")}, expected: []interface{}{[]string{hexEncode(":key1"), hexEncode("value1")}}},
		{params: []string{"0x00", hash.String()}, expected: nil},
		{params: []string{"", hash.String()}, expected: []interface{}{
			[]string{hexEncode(":child_storage:default::child1"),
				"0x8f733acc98dff0e6527f97e2a87e4834cd8b2e601f56fb003084e9d43183d7ff"},
			[]string{hexEncode(":key1"), hexEncode("value1")},
			[]string{hexEncode(":key2"), hexEncode("value2")}}},
		{params: []string{hexEncode(":key1"), hash.String()},
			expected: []interface{}{[]string{hexEncode(":key1"), hexEncode("value1")}}},
		{params: []string{"", randomHash.String()}, errMsg: "pebble: not found"},
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
				var err error
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
				require.Equal(t, []string{expectedKV[0], expectedKV[1]}, kv)
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
		{params: []string{"", randomHash.String()}, errMsg: "pebble: not found"},
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

	hashOfNil := common.NewHash(common.MustHexToBytes("0x0e5751c026e543b2e8ab2eb06099daa1d1e5df47778f7787faab45cdf12fe3a8")) //nolint:lll
	hash1 := common.MustBlake2bHash([]byte("value1"))

	testCases := []struct {
		params   []string
		expected common.Hash
		errMsg   string
	}{
		{params: []string{""}, expected: hashOfNil},
		{params: []string{":key1"}, expected: hash1},
		{params: []string{":key1", hash.String()}, expected: hash1},
		{params: []string{"0x", randomHash.String()}, errMsg: "pebble: not found"},
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
			require.Equal(t, StateStorageHashResponse(test.expected.String()), res)
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
		{params: []string{"0x", randomHash.String()}, errMsg: "pebble: not found"},
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
	t.Run("When_starting_block_is_empty", func(t *testing.T) {
		module := new(StateModule)
		req := new(StateStorageQueryRangeRequest)

		var res []StorageChangeSetResponse
		err := module.QueryStorage(nil, req, &res)
		require.Error(t, err, "the start block hash cannot be an empty value")
	})

	t.Run("When_blockAPI_returns_error", func(t *testing.T) {
		mockError := errors.New("mock test error")
		ctrl := gomock.NewController(t)
		mockBlockAPI := NewMockBlockAPI(ctrl)
		mockBlockAPI.EXPECT().GetBlockByHash(common.Hash{1, 2}).Return(nil, mockError)
		module := new(StateModule)
		module.blockAPI = mockBlockAPI

		req := new(StateStorageQueryRangeRequest)
		req.StartBlock = common.NewHash([]byte{1, 2})

		var res []StorageChangeSetResponse
		err := module.QueryStorage(nil, req, &res)
		assert.ErrorIs(t, err, mockError)
	})

	t.Run("When_QueryStorage_returns_data", func(t *testing.T) {
		expectedChanges := [][2]*string{
			makeChange("0x90", stringToHex("value")),
			makeChange("0x80", stringToHex("another value")),
		}
		expected := []StorageChangeSetResponse{
			{
				Block:   &common.Hash{1, 2},
				Changes: expectedChanges,
			},
			{
				Block:   &common.Hash{3, 4},
				Changes: [][2]*string{},
			},
		}
		ctrl := gomock.NewController(t)
		mockBlockAPI := NewMockBlockAPI(ctrl)
		mockBlockAPI.EXPECT().GetBlockByHash(common.Hash{1, 2}).Return(&types.Block{
			Header: types.Header{
				Number: 3,
			},
		}, nil)
		mockBlockAPI.EXPECT().BestBlockHash().Return(common.Hash{3, 4})
		mockBlockAPI.EXPECT().GetBlockByHash(common.Hash{3, 4}).Return(&types.Block{
			Header: types.Header{
				Number: 4,
			},
		}, nil)
		mockBlockAPI.EXPECT().GetHashByNumber(uint(3)).Return(common.Hash{1, 2}, nil)
		mockBlockAPI.EXPECT().GetHashByNumber(uint(4)).Return(common.Hash{3, 4}, nil)

		mockStorageAPI := NewMockStorageAPI(ctrl)
		mockStorageAPI.EXPECT().GetStorageByBlockHash(&common.Hash{1, 2}, []byte{144}).Return([]byte(`value`), nil)
		mockStorageAPI.EXPECT().GetStorageByBlockHash(&common.Hash{1, 2}, []byte{128}).
			Return([]byte(`another value`), nil)
		mockStorageAPI.EXPECT().GetStorageByBlockHash(&common.Hash{3, 4}, []byte{144}).Return([]byte(`value`), nil)
		mockStorageAPI.EXPECT().GetStorageByBlockHash(&common.Hash{3, 4}, []byte{128}).
			Return([]byte(`another value`), nil)

		module := new(StateModule)
		module.blockAPI = mockBlockAPI
		module.storageAPI = mockStorageAPI

		req := new(StateStorageQueryRangeRequest)
		req.StartBlock = common.NewHash([]byte{1, 2})
		req.Keys = []string{"0x90", "0x80"}

		var res []StorageChangeSetResponse
		err := module.QueryStorage(nil, req, &res)
		assert.NoError(t, err)
		assert.Equal(t, expected, res)
	})
}

func TestStateModule_GetMetadata(t *testing.T) {
	t.Skip() // TODO: update expected_metadata (#1026)
	sm, hash, _ := setupStateModule(t)
	randomHash, err := common.HexToHash(RandomHash)
	require.NoError(t, err)

	expectedMetadata, err := os.ReadFile("./test_data/expected_metadata")
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
			}, expected: []string{
				"0x3a6368696c645f73746f726167653a64656661756c743a3a6368696c6431",
				"0x3a6b657931", "0x3a6b657932"}},
		{name: "allKeysTestBlockHash",
			params: StateStorageKeyRequest{
				Qty:   10,
				Block: stateRootHash,
			}, expected: []string{
				"0x3a6368696c645f73746f726167653a64656661756c743a3a6368696c6431",
				"0x3a6b657931", "0x3a6b657932"}},
		{name: "prefixMatchAll",
			params: StateStorageKeyRequest{
				Prefix: "0x3a6b6579",
				Qty:    10,
			}, expected: []string{
				"0x3a6b657931", "0x3a6b657932"}},
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
			}, expected: []string{"0x3a6368696c645f73746f726167653a64656661756c743a3a6368696c6431"}},
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
	ctrl := gomock.NewController(t)

	coreAPIMock := mocks.NewMockCoreAPI(ctrl)
	coreAPIMock.EXPECT().GetReadProofAt(gomock.Any(), gomock.Any()).
		Return(common.Hash{}, nil, errors.New("mocked error"))

	sm := new(StateModule)
	sm.coreAPI = coreAPIMock

	req := &StateGetReadProofRequest{
		Keys: []string{},
	}
	err := sm.GetReadProof(nil, req, nil)
	require.Error(t, err, "mocked error")
}

func TestGetReadProof_WhenReturnsProof(t *testing.T) {
	ctrl := gomock.NewController(t)

	expectedBlock := common.BytesToHash([]byte("random hash"))
	mockedProof := [][]byte{[]byte("proof-1"), []byte("proof-2")}

	coreAPIMock := mocks.NewMockCoreAPI(ctrl)
	coreAPIMock.
		EXPECT().GetReadProofAt(gomock.Any(), gomock.Any()).
		Return(expectedBlock, mockedProof, nil)

	sm := new(StateModule)
	sm.coreAPI = coreAPIMock

	req := &StateGetReadProofRequest{
		Keys: []string{},
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

	err = ts.Put([]byte(`:key2`), []byte(`value2`))
	require.NoError(t, err)

	err = ts.Put([]byte(`:key1`), []byte(`value1`))
	require.NoError(t, err)

	err = ts.SetChildStorage([]byte(`:child1`), []byte(`:key1`), []byte(`:childValue1`))
	require.NoError(t, err)

	sr1, err := ts.Root(trie.NoMaxInlineValueSize)
	require.NoError(t, err)
	err = chain.Storage.StoreTrie(ts, nil)
	require.NoError(t, err)

	digest := types.NewDigest()
	prd, err := types.NewBabeSecondaryPlainPreDigest(0, 1).ToPreRuntimeDigest()
	require.NoError(t, err)
	err = digest.Add(*prd)
	require.NoError(t, err)

	b := &types.Block{
		Header: types.Header{
			ParentHash: chain.Block.BestBlockHash(),
			Number:     3,
			StateRoot:  sr1,
			Digest:     digest,
		},
		Body: *types.NewBody([]types.Extrinsic{[]byte{}}),
	}

	err = chain.Block.AddBlock(b)
	require.NoError(t, err)

	rt, err := chain.Block.GetRuntime(b.Header.ParentHash)
	require.NoError(t, err)

	chain.Block.StoreRuntime(b.Header.Hash(), rt)

	hash, err := chain.Block.GetHashByNumber(3)
	require.NoError(t, err)

	core := newCoreService(t, chain)
	return NewStateModule(net, chain.Storage, core, nil), &hash, &sr1
}
