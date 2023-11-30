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
	"errors"
	"net/http"
	"testing"

	wazero_runtime "github.com/ChainSafe/gossamer/lib/runtime/wazero"
	"github.com/ChainSafe/gossamer/lib/trie"

	"github.com/ChainSafe/gossamer/dot/rpc/modules/mocks"
	testdata "github.com/ChainSafe/gossamer/dot/rpc/modules/test_data"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/blocktree"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/runtime"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStateModuleGetPairs(t *testing.T) {
	ctrl := gomock.NewController(t)

	str := "0x01"
	hash := common.MustHexToHash("0x3aa96b0149b6ca3688878bdbd19464448624136398e3ce45b9e755d3ab61355a")
	m := make(map[string][]byte)
	m["a"] = []byte{21, 22}
	m["b"] = []byte{23, 24}

	mockStorageAPI := mocks.NewMockStorageAPI(ctrl)
	mockStorageAPI.EXPECT().GetStateRootFromBlock(&hash).Return(&hash, nil)
	mockStorageAPI.EXPECT().GetKeysWithPrefix(&hash, common.MustHexToBytes(str)).Return([][]byte{{1}, {1}}, nil)
	mockStorageAPI.EXPECT().GetStorage(&hash, []byte{1}).Return([]byte{21}, nil).Times(2)

	mockStorageAPINil := mocks.NewMockStorageAPI(ctrl)
	mockStorageAPINil.EXPECT().GetStateRootFromBlock(&hash).Return(&hash, nil)
	mockStorageAPINil.EXPECT().Entries(&hash).Return(m, nil)

	mockStorageAPIGetKeysEmpty := mocks.NewMockStorageAPI(ctrl)
	mockStorageAPIGetKeysEmpty.EXPECT().GetStateRootFromBlock(&hash).Return(&hash, nil)
	mockStorageAPIGetKeysEmpty.EXPECT().GetKeysWithPrefix(&hash, common.MustHexToBytes(str)).Return([][]byte{}, nil)

	mockStorageAPIGetKeysErr := mocks.NewMockStorageAPI(ctrl)
	mockStorageAPIGetKeysErr.EXPECT().GetStateRootFromBlock(&hash).Return(&hash, nil)
	mockStorageAPIGetKeysErr.EXPECT().GetKeysWithPrefix(&hash, common.MustHexToBytes(str)).
		Return(nil, errors.New("GetKeysWithPrefix Err"))

	mockStorageAPIEntriesErr := mocks.NewMockStorageAPI(ctrl)
	mockStorageAPIEntriesErr.EXPECT().GetStateRootFromBlock(&hash).Return(&hash, nil)
	mockStorageAPIEntriesErr.EXPECT().Entries(&hash).Return(nil, errors.New("entries Err"))

	mockStorageAPIErr := mocks.NewMockStorageAPI(ctrl)
	mockStorageAPIErr.EXPECT().GetStateRootFromBlock(&hash).Return(nil, errors.New("GetStateRootFromBlock Err"))

	mockStorageAPIGetStorageErr := mocks.NewMockStorageAPI(ctrl)
	mockStorageAPIGetStorageErr.EXPECT().GetStateRootFromBlock(&hash).Return(&hash, nil)
	mockStorageAPIGetStorageErr.EXPECT().GetKeysWithPrefix(&hash, common.MustHexToBytes(str)).
		Return([][]byte{{2}, {2}}, nil)
	mockStorageAPIGetStorageErr.EXPECT().GetStorage(&hash, []byte{2}).Return(nil, errors.New("GetStorage Err"))

	var expRes StatePairResponse
	for k, v := range m {
		expRes = append(expRes, []string{common.BytesToHex([]byte(k)), common.BytesToHex(v)})
	}
	type fields struct {
		networkAPI NetworkAPI
		storageAPI StorageAPI
		coreAPI    CoreAPI
	}
	type args struct {
		in0 *http.Request
		req *StatePairRequest
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		expErr error
		exp    StatePairResponse
	}{
		{
			name:   "GetStateRootFromBlock Error",
			fields: fields{nil, mockStorageAPIErr, nil},
			args: args{
				req: &StatePairRequest{
					Bhash: &hash,
				},
			},
			expErr: errors.New("GetStateRootFromBlock Err"),
		},
		{
			name:   "Nil Prefix OK",
			fields: fields{nil, mockStorageAPINil, nil},
			args: args{
				req: &StatePairRequest{
					Bhash: &hash,
				},
			},
			exp: expRes,
		},
		{
			name:   "Nil Prefix Err",
			fields: fields{nil, mockStorageAPIEntriesErr, nil},
			args: args{
				req: &StatePairRequest{
					Bhash: &hash,
				},
			},
			exp:    []interface{}{},
			expErr: errors.New("entries Err"),
		},
		{
			name:   "OK Case",
			fields: fields{nil, mockStorageAPI, nil},
			args: args{
				req: &StatePairRequest{
					Prefix: &str,
					Bhash:  &hash,
				},
			},
			exp: StatePairResponse{[]string{"0x01", "0x15"}, []string{"0x01", "0x15"}},
		},
		{
			name:   "GetKeysWithPrefix Error",
			fields: fields{nil, mockStorageAPIGetKeysErr, nil},
			args: args{
				req: &StatePairRequest{
					Prefix: &str,
					Bhash:  &hash,
				},
			},
			expErr: errors.New("GetKeysWithPrefix Err"),
		},
		{
			name:   "GetStorage Error",
			fields: fields{nil, mockStorageAPIGetStorageErr, nil},
			args: args{
				req: &StatePairRequest{
					Prefix: &str,
					Bhash:  &hash,
				},
			},
			exp:    StatePairResponse{interface{}(nil), interface{}(nil)},
			expErr: errors.New("GetStorage Err"),
		},
		{
			name:   "GetKeysWithPrefix Empty",
			fields: fields{nil, mockStorageAPIGetKeysEmpty, nil},
			args: args{
				req: &StatePairRequest{
					Prefix: &str,
					Bhash:  &hash,
				},
			},
			exp: StatePairResponse{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sm := &StateModule{
				networkAPI: tt.fields.networkAPI,
				storageAPI: tt.fields.storageAPI,
				coreAPI:    tt.fields.coreAPI,
			}
			res := StatePairResponse{}
			err := sm.GetPairs(tt.args.in0, tt.args.req, &res)
			if tt.expErr != nil {
				assert.EqualError(t, err, tt.expErr.Error())
			} else {
				assert.NoError(t, err)
			}
			assert.ElementsMatch(t, tt.exp, res)
		})
	}
}

func TestStateModuleGetKeysPaged(t *testing.T) {
	ctrl := gomock.NewController(t)

	mockStorageAPI := mocks.NewMockStorageAPI(ctrl)
	mockStorageAPI.EXPECT().GetKeysWithPrefix((*common.Hash)(nil), common.MustHexToBytes("0x")).
		Return([][]byte{{1}, {2}}, nil)

	mockStorageAPI2 := mocks.NewMockStorageAPI(ctrl)
	mockStorageAPI2.EXPECT().GetKeysWithPrefix((*common.Hash)(nil), common.MustHexToBytes("0x")).
		Return([][]byte{{1, 1, 1}, {1, 1, 1}}, nil)

	mockStorageAPIErr := mocks.NewMockStorageAPI(ctrl)
	mockStorageAPIErr.EXPECT().GetKeysWithPrefix((*common.Hash)(nil), common.MustHexToBytes("0x")).
		Return(nil, errors.New("GetKeysWithPrefix Err"))

	type fields struct {
		networkAPI NetworkAPI
		storageAPI StorageAPI
		coreAPI    CoreAPI
	}
	type args struct {
		in0 *http.Request
		req *StateStorageKeyRequest
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		expErr error
		exp    StateStorageKeysResponse
	}{
		{
			name:   "OK",
			fields: fields{nil, mockStorageAPI, nil},
			args: args{
				req: &StateStorageKeyRequest{
					AfterKey: "0x01",
				},
			},
			exp: StateStorageKeysResponse(nil),
		},
		{
			name:   "ResCount break",
			fields: fields{nil, mockStorageAPI2, nil},
			args: args{
				req: &StateStorageKeyRequest{
					Qty:      1,
					AfterKey: "0x01",
				},
			},
			exp: StateStorageKeysResponse{"0x010101"},
		},
		{
			name:   "GetKeysWithPrefix Error",
			fields: fields{nil, mockStorageAPIErr, nil},
			args: args{
				req: &StateStorageKeyRequest{
					AfterKey: "0x01",
				},
			},
			expErr: errors.New("cannot get keys with prefix : GetKeysWithPrefix Err"),
		},
		{
			name:   "Request Prefix Error",
			fields: fields{nil, mockStorageAPI, nil},
			args: args{
				req: &StateStorageKeyRequest{
					Prefix:   "a",
					AfterKey: "0x01",
				},
			},
			expErr: errors.New("could not byteify non 0x prefixed string: a"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := StateStorageKeysResponse(nil)
			sm := &StateModule{
				networkAPI: tt.fields.networkAPI,
				storageAPI: tt.fields.storageAPI,
				coreAPI:    tt.fields.coreAPI,
			}
			err := sm.GetKeysPaged(tt.args.in0, tt.args.req, &res)
			if tt.expErr != nil {
				assert.EqualError(t, err, tt.expErr.Error())
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.exp, res)
		})
	}
}

// TestCall tests the state_call.
// TODO: Improve runtime tests
// https://github.com/ChainSafe/gossamer/issues/3234
func TestCall(t *testing.T) {
	ctrl := gomock.NewController(t)
	testHash := common.NewHash([]byte{0x01, 0x02})
	rt := wazero_runtime.NewTestInstance(t, runtime.WESTEND_RUNTIME_v0929)

	mockNetworkAPI := mocks.NewMockNetworkAPI(ctrl)
	mockStorageAPI := mocks.NewMockStorageAPI(ctrl)
	mockBlockAPI := mocks.NewMockBlockAPI(ctrl)
	mockBlockAPI.EXPECT().BestBlockHash().Return(testHash)
	mockBlockAPI.EXPECT().GetRuntime(testHash).Return(rt, nil)

	sm := NewStateModule(mockNetworkAPI, mockStorageAPI, nil, mockBlockAPI)

	req := &StateCallRequest{
		Method: "Core_version",
		Params: "0x",
	}
	var res StateCallResponse
	err := sm.Call(nil, req, &res)
	assert.NoError(t, err)
	assert.NotEmpty(t, res)
}

func TestStateTrie(t *testing.T) {
	expecificBlockHash := common.Hash([32]byte{6, 6, 6, 6, 6, 6})
	var expectedEncodedSlice []string
	entries := []trie.Entry{
		{Key: []byte("entry-1"), Value: []byte{0, 1, 2, 3}},
		{Key: []byte("entry-2"), Value: []byte{3, 4, 5, 6}},
	}

	for _, entry := range entries {
		expectedEncodedSlice = append(expectedEncodedSlice, common.BytesToHex(scale.MustMarshal(entry)))
	}

	testcases := map[string]struct {
		request        StateTrieAtRequest
		newStateModule func(t *testing.T) *StateModule
		expected       StateTrieResponse
	}{
		"blockhash_parameter_nil": {
			request:  StateTrieAtRequest{At: nil},
			expected: expectedEncodedSlice,
			newStateModule: func(t *testing.T) *StateModule {
				ctrl := gomock.NewController(t)

				bestBlockHash := common.Hash([32]byte{1, 0, 1, 0, 1})
				blockAPIMock := NewMockBlockAPI(ctrl)
				blockAPIMock.EXPECT().BestBlockHash().Return(bestBlockHash)

				fakeStateRoot := common.Hash([32]byte{5, 5, 5, 5, 5})
				fakeBlockHeader := types.NewHeader(common.EmptyHash, fakeStateRoot,
					common.EmptyHash, 1, scale.VaryingDataTypeSlice{})

				blockAPIMock.EXPECT().GetHeader(bestBlockHash).Return(fakeBlockHeader, nil)

				fakeEntries := map[string][]byte{
					"entry-1": {0, 1, 2, 3},
					"entry-2": {3, 4, 5, 6},
				}
				storageAPIMock := NewMockStorageAPI(ctrl)
				storageAPIMock.EXPECT().Entries(&fakeStateRoot).
					Return(fakeEntries, nil)

				sm := NewStateModule(nil, storageAPIMock, nil, blockAPIMock)
				return sm
			},
		},
		"blockhash_parameter_not_nil": {
			request:  StateTrieAtRequest{At: &expecificBlockHash},
			expected: expectedEncodedSlice,
			newStateModule: func(t *testing.T) *StateModule {
				ctrl := gomock.NewController(t)
				blockAPIMock := NewMockBlockAPI(ctrl)

				bestBlockHash := common.Hash([32]byte{1, 0, 1, 0, 1})
				blockAPIMock.EXPECT().BestBlockHash().Return(bestBlockHash)

				fakeStateRoot := common.Hash([32]byte{5, 5, 5, 5, 5})
				fakeBlockHeader := types.NewHeader(common.EmptyHash, fakeStateRoot,
					common.EmptyHash, 1, scale.VaryingDataTypeSlice{})

				blockAPIMock.EXPECT().GetHeader(expecificBlockHash).
					Return(fakeBlockHeader, nil)

				fakeEntries := map[string][]byte{
					"entry-1": {0, 1, 2, 3},
					"entry-2": {3, 4, 5, 6},
				}
				storageAPIMock := NewMockStorageAPI(ctrl)
				storageAPIMock.EXPECT().Entries(&fakeStateRoot).
					Return(fakeEntries, nil)

				sm := NewStateModule(nil, storageAPIMock, nil, blockAPIMock)
				return sm
			},
		},
	}

	for tname, tt := range testcases {
		tt := tt

		t.Run(tname, func(t *testing.T) {
			sm := tt.newStateModule(t)

			var res StateTrieResponse
			err := sm.Trie(nil, &tt.request, &res)
			require.NoError(t, err)
			require.Equal(t, tt.expected, res)
		})
	}
}

func TestStateModuleGetMetadata(t *testing.T) {
	ctrl := gomock.NewController(t)

	hash := common.MustHexToHash("0x3aa96b0149b6ca3688878bdbd19464448624136398e3ce45b9e755d3ab61355a")

	mockCoreAPI := mocks.NewMockCoreAPI(ctrl)
	mockCoreAPI.EXPECT().GetMetadata(&hash).Return(common.MustHexToBytes(testdata.NewTestMetadata()), nil)

	mockCoreAPIErr := mocks.NewMockCoreAPI(ctrl)
	mockCoreAPIErr.EXPECT().GetMetadata(&hash).Return(nil, errors.New("GetMetadata Error"))

	mockStateModule := NewStateModule(nil, nil, mockCoreAPIErr, nil)

	var expRes []byte
	err := scale.Unmarshal(common.MustHexToBytes(testdata.NewTestMetadata()), &expRes)
	assert.NoError(t, err)
	type fields struct {
		networkAPI NetworkAPI
		storageAPI StorageAPI
		coreAPI    CoreAPI
	}
	type args struct {
		in0 *http.Request
		req *StateRuntimeMetadataQuery
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		expErr error
		exp    StateMetadataResponse
	}{
		{
			name:   "OK Case",
			fields: fields{nil, nil, mockCoreAPI},
			args: args{
				req: &StateRuntimeMetadataQuery{Bhash: &hash},
			},
			exp: StateMetadataResponse(common.BytesToHex(expRes)),
		},
		{
			name:   "GetMetadata Error",
			fields: fields{nil, nil, mockStateModule.coreAPI},
			args: args{
				req: &StateRuntimeMetadataQuery{Bhash: &hash},
			},
			expErr: errors.New("GetMetadata Error"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sm := &StateModule{
				networkAPI: tt.fields.networkAPI,
				storageAPI: tt.fields.storageAPI,
				coreAPI:    tt.fields.coreAPI,
			}
			res := StateMetadataResponse("")
			err := sm.GetMetadata(tt.args.in0, tt.args.req, &res)
			if tt.expErr != nil {
				assert.EqualError(t, err, tt.expErr.Error())
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.exp, res)
		})
	}
}

func TestStateModuleGetReadProof(t *testing.T) {
	ctrl := gomock.NewController(t)

	hash := common.MustHexToHash("0x3aa96b0149b6ca3688878bdbd19464448624136398e3ce45b9e755d3ab61355a")
	keys := []string{"0x1111", "0x2222"}
	expKeys := make([][]byte, len(keys))
	for i, hexKey := range keys {
		bKey, err := common.HexToBytes(hexKey)
		assert.NoError(t, err)

		expKeys[i] = bKey
	}

	mockCoreAPI := mocks.NewMockCoreAPI(ctrl)
	mockCoreAPI.EXPECT().GetReadProofAt(hash, expKeys).Return(hash, [][]byte{{1, 1, 1}, {1, 1, 1}}, nil)

	mockCoreAPIErr := mocks.NewMockCoreAPI(ctrl)
	mockCoreAPIErr.EXPECT().GetReadProofAt(hash, expKeys).
		Return(common.Hash{}, nil, errors.New("GetReadProofAt Error")).Times(2)

	type fields struct {
		networkAPI NetworkAPI
		storageAPI StorageAPI
		coreAPI    CoreAPI
	}
	type args struct {
		in0 *http.Request
		req *StateGetReadProofRequest
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		expErr error
		exp    StateGetReadProofResponse
	}{
		{
			name:   "OK Case",
			fields: fields{nil, nil, mockCoreAPI},
			args: args{
				req: &StateGetReadProofRequest{
					Keys: keys,
					Hash: hash,
				},
			},
			exp: StateGetReadProofResponse{
				At: common.Hash{
					0x3a, 0xa9, 0x6b, 0x1, 0x49, 0xb6, 0xca, 0x36, 0x88, 0x87, 0x8b, 0xdb, 0xd1, 0x94, 0x64, 0x44,
					0x86, 0x24, 0x13, 0x63, 0x98, 0xe3, 0xce, 0x45, 0xb9, 0xe7, 0x55, 0xd3, 0xab, 0x61, 0x35, 0x5a},
				Proof: []string{"0x010101", "0x010101"},
			},
		},
		{
			name:   "GetReadProofAt Error",
			fields: fields{nil, nil, mockCoreAPIErr},
			args: args{
				req: &StateGetReadProofRequest{
					Keys: keys,
					Hash: hash,
				},
			},
			expErr: errors.New("GetReadProofAt Error"),
		},
		{
			name:   "InvalidKeys Error",
			fields: fields{nil, nil, mockCoreAPIErr},
			args: args{
				req: &StateGetReadProofRequest{
					Keys: keys,
					Hash: hash,
				},
			},
			expErr: errors.New("GetReadProofAt Error"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sm := &StateModule{
				networkAPI: tt.fields.networkAPI,
				storageAPI: tt.fields.storageAPI,
				coreAPI:    tt.fields.coreAPI,
			}
			res := StateGetReadProofResponse{}
			err := sm.GetReadProof(tt.args.in0, tt.args.req, &res)
			if tt.expErr != nil {
				assert.EqualError(t, err, tt.expErr.Error())
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.exp, res)
		})
	}
}

func TestStateModuleGetRuntimeVersion(t *testing.T) {
	ctrl := gomock.NewController(t)

	hash := common.MustHexToHash("0x3aa96b0149b6ca3688878bdbd19464448624136398e3ce45b9e755d3ab61355a")
	version := runtime.Version{
		SpecName:         []byte("polkadot"),
		ImplName:         []byte("parity-polkadot"),
		AuthoringVersion: 0,
		SpecVersion:      25,
		ImplVersion:      0,
		APIItems: []runtime.APIItem{{
			Name: [8]byte{1, 2, 3, 4, 5, 6, 7, 8},
			Ver:  99,
		}},
		TransactionVersion: 5,
	}

	mockCoreAPI := mocks.NewMockCoreAPI(ctrl)
	mockCoreAPI.EXPECT().GetRuntimeVersion(&hash).Return(version, nil)

	mockCoreAPIErr := mocks.NewMockCoreAPI(ctrl)
	mockCoreAPIErr.EXPECT().GetRuntimeVersion(&hash).
		Return(runtime.Version{}, errors.New("GetRuntimeVersion Error"))

	type fields struct {
		networkAPI NetworkAPI
		storageAPI StorageAPI
		coreAPI    CoreAPI
	}
	type args struct {
		in0 *http.Request
		req *StateRuntimeVersionRequest
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		expErr error
		exp    StateRuntimeVersionResponse
	}{
		{
			name:   "OK Case",
			fields: fields{nil, nil, mockCoreAPI},
			args: args{
				req: &StateRuntimeVersionRequest{&hash},
			},
			exp: StateRuntimeVersionResponse{
				SpecName:           "polkadot",
				ImplName:           "parity-polkadot",
				AuthoringVersion:   0x0,
				SpecVersion:        0x19,
				ImplVersion:        0x0,
				TransactionVersion: 0x5,
				Apis:               []interface{}{[]interface{}{"0x0102030405060708", uint32(99)}}},
		},
		{
			name:   "GetRuntimeVersion Error",
			fields: fields{nil, nil, mockCoreAPIErr},
			args: args{
				req: &StateRuntimeVersionRequest{&hash},
			},
			expErr: errors.New("GetRuntimeVersion Error"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sm := &StateModule{
				networkAPI: tt.fields.networkAPI,
				storageAPI: tt.fields.storageAPI,
				coreAPI:    tt.fields.coreAPI,
			}
			res := StateRuntimeVersionResponse{}
			err := sm.GetRuntimeVersion(tt.args.in0, tt.args.req, &res)
			if tt.expErr != nil {
				assert.EqualError(t, err, tt.expErr.Error())
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.exp, res)
		})
	}
}

func TestStateModuleGetStorage(t *testing.T) {
	ctrl := gomock.NewController(t)

	hash := common.MustHexToHash("0x3aa96b0149b6ca3688878bdbd19464448624136398e3ce45b9e755d3ab61355a")
	reqBytes := common.MustHexToBytes("0x3aa96b0149b6ca3688878bdbd19464448624136398e3ce45b9e755d3ab61355a")

	mockStorageAPI := mocks.NewMockStorageAPI(ctrl)
	mockStorageAPI.EXPECT().GetStorageByBlockHash(&hash, reqBytes).Return([]byte{21}, nil)
	mockStorageAPI.EXPECT().GetStorage((*common.Hash)(nil), reqBytes).Return([]byte{21}, nil)

	mockStorageAPIErr := mocks.NewMockStorageAPI(ctrl)
	mockStorageAPIErr.EXPECT().GetStorageByBlockHash(&hash, reqBytes).
		Return(nil, errors.New("GetStorageByBlockHash Error"))
	mockStorageAPIErr.EXPECT().GetStorage((*common.Hash)(nil), reqBytes).
		Return(nil, errors.New("GetStorage Error"))

	type fields struct {
		networkAPI NetworkAPI
		storageAPI StorageAPI
		coreAPI    CoreAPI
	}
	type args struct {
		in0 *http.Request
		req *StateStorageRequest
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		expErr error
		exp    StateStorageResponse
	}{
		{
			name:   "bHash Not Nil OK",
			fields: fields{nil, mockStorageAPI, nil},
			args: args{
				req: &StateStorageRequest{
					Key:   "0x3aa96b0149b6ca3688878bdbd19464448624136398e3ce45b9e755d3ab61355a",
					Bhash: &hash,
				},
			},
			exp: StateStorageResponse("0x15"),
		},
		{
			name:   "bHash Nil OK",
			fields: fields{nil, mockStorageAPI, nil},
			args: args{
				req: &StateStorageRequest{
					Key: "0x3aa96b0149b6ca3688878bdbd19464448624136398e3ce45b9e755d3ab61355a",
				},
			},
			exp: StateStorageResponse("0x15"),
		},
		{
			name:   "bHash Not Nil Err",
			fields: fields{nil, mockStorageAPIErr, nil},
			args: args{
				req: &StateStorageRequest{
					Key:   "0x3aa96b0149b6ca3688878bdbd19464448624136398e3ce45b9e755d3ab61355a",
					Bhash: &hash,
				},
			},
			expErr: errors.New("GetStorageByBlockHash Error"),
		},
		{
			name:   "bHash Nil Err",
			fields: fields{nil, mockStorageAPIErr, nil},
			args: args{
				req: &StateStorageRequest{
					Key: "0x3aa96b0149b6ca3688878bdbd19464448624136398e3ce45b9e755d3ab61355a",
				},
			},
			expErr: errors.New("GetStorage Error"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sm := &StateModule{
				networkAPI: tt.fields.networkAPI,
				storageAPI: tt.fields.storageAPI,
				coreAPI:    tt.fields.coreAPI,
			}
			res := StateStorageResponse("")
			err := sm.GetStorage(tt.args.in0, tt.args.req, &res)
			if tt.expErr != nil {
				assert.EqualError(t, err, tt.expErr.Error())
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.exp, res)
		})
	}
}

func TestStateModuleGetStorageHash(t *testing.T) {
	ctrl := gomock.NewController(t)

	hash := common.MustHexToHash("0x3aa96b0149b6ca3688878bdbd19464448624136398e3ce45b9e755d3ab61355a")
	reqBytes := common.MustHexToBytes("0x3aa96b0149b6ca3688878bdbd19464448624136398e3ce45b9e755d3ab61355a")

	mockStorageAPI := mocks.NewMockStorageAPI(ctrl)
	mockStorageAPI.EXPECT().GetStorageByBlockHash(&hash, reqBytes).Return([]byte{21}, nil)
	mockStorageAPI.EXPECT().GetStorage((*common.Hash)(nil), reqBytes).
		Return([]byte{21}, nil)

	mockStorageAPIErr := mocks.NewMockStorageAPI(ctrl)
	mockStorageAPIErr.EXPECT().GetStorageByBlockHash(&hash, reqBytes).
		Return(nil, errors.New("GetStorageByBlockHash Error"))
	mockStorageAPIErr.EXPECT().GetStorage((*common.Hash)(nil), reqBytes).
		Return(nil, errors.New("GetStorage Error"))

	type fields struct {
		networkAPI NetworkAPI
		storageAPI StorageAPI
		coreAPI    CoreAPI
	}
	type args struct {
		in0 *http.Request
		req *StateStorageHashRequest
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		expErr error
		exp    StateStorageHashResponse
	}{
		{
			name:   "bHash Not Nil OK",
			fields: fields{nil, mockStorageAPI, nil},
			args: args{
				req: &StateStorageHashRequest{
					Key:   "0x3aa96b0149b6ca3688878bdbd19464448624136398e3ce45b9e755d3ab61355a",
					Bhash: &hash,
				},
			},
			exp: StateStorageHashResponse("0x8c272b95141731e2069ed10ad288146965eb76f0a566885323195f4cd7d58f3b"),
		},
		{
			name:   "bHash Nil OK",
			fields: fields{nil, mockStorageAPI, nil},
			args: args{
				req: &StateStorageHashRequest{
					Key: "0x3aa96b0149b6ca3688878bdbd19464448624136398e3ce45b9e755d3ab61355a",
				},
			},
			exp: StateStorageHashResponse("0x8c272b95141731e2069ed10ad288146965eb76f0a566885323195f4cd7d58f3b"),
		},
		{
			name:   "bHash Not Nil Err",
			fields: fields{nil, mockStorageAPIErr, nil},
			args: args{
				req: &StateStorageHashRequest{
					Key:   "0x3aa96b0149b6ca3688878bdbd19464448624136398e3ce45b9e755d3ab61355a",
					Bhash: &hash,
				},
			},
			expErr: errors.New("GetStorageByBlockHash Error"),
		},
		{
			name:   "bHash Nil Err",
			fields: fields{nil, mockStorageAPIErr, nil},
			args: args{
				req: &StateStorageHashRequest{
					Key: "0x3aa96b0149b6ca3688878bdbd19464448624136398e3ce45b9e755d3ab61355a",
				},
			},
			expErr: errors.New("GetStorage Error"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sm := &StateModule{
				networkAPI: tt.fields.networkAPI,
				storageAPI: tt.fields.storageAPI,
				coreAPI:    tt.fields.coreAPI,
			}
			res := StateStorageHashResponse("")
			err := sm.GetStorageHash(tt.args.in0, tt.args.req, &res)
			if tt.expErr != nil {
				assert.EqualError(t, err, tt.expErr.Error())
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.exp, res)
		})
	}
}

func TestStateModuleGetStorageSize(t *testing.T) {
	ctrl := gomock.NewController(t)

	hash := common.MustHexToHash("0x3aa96b0149b6ca3688878bdbd19464448624136398e3ce45b9e755d3ab61355a")
	reqBytes := common.MustHexToBytes("0x3aa96b0149b6ca3688878bdbd19464448624136398e3ce45b9e755d3ab61355a")

	mockStorageAPI := mocks.NewMockStorageAPI(ctrl)
	mockStorageAPI.EXPECT().GetStorageByBlockHash(&hash, reqBytes).Return([]byte{21}, nil)
	mockStorageAPI.EXPECT().GetStorage((*common.Hash)(nil), reqBytes).Return([]byte{21}, nil)

	mockStorageAPIErr := mocks.NewMockStorageAPI(ctrl)
	mockStorageAPIErr.EXPECT().GetStorageByBlockHash(&hash, reqBytes).
		Return(nil, errors.New("GetStorageByBlockHash Error"))
	mockStorageAPIErr.EXPECT().GetStorage((*common.Hash)(nil), reqBytes).
		Return(nil, errors.New("GetStorage Error"))

	type fields struct {
		networkAPI NetworkAPI
		storageAPI StorageAPI
		coreAPI    CoreAPI
	}
	type args struct {
		in0 *http.Request
		req *StateStorageSizeRequest
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		expErr error
		exp    StateStorageSizeResponse
	}{
		{
			name:   "bHash Not Nil OK",
			fields: fields{nil, mockStorageAPI, nil},
			args: args{
				req: &StateStorageSizeRequest{
					Key:   "0x3aa96b0149b6ca3688878bdbd19464448624136398e3ce45b9e755d3ab61355a",
					Bhash: &hash,
				},
			},
			exp: StateStorageSizeResponse(1),
		},
		{
			name:   "bHash Nil OK",
			fields: fields{nil, mockStorageAPI, nil},
			args: args{
				req: &StateStorageSizeRequest{
					Key: "0x3aa96b0149b6ca3688878bdbd19464448624136398e3ce45b9e755d3ab61355a",
				},
			},
			exp: StateStorageSizeResponse(1),
		},
		{
			name:   "bHash Not Nil Err",
			fields: fields{nil, mockStorageAPIErr, nil},
			args: args{
				req: &StateStorageSizeRequest{
					Key:   "0x3aa96b0149b6ca3688878bdbd19464448624136398e3ce45b9e755d3ab61355a",
					Bhash: &hash,
				},
			},
			expErr: errors.New("GetStorageByBlockHash Error"),
		},
		{
			name:   "bHash Nil Err",
			fields: fields{nil, mockStorageAPIErr, nil},
			args: args{
				req: &StateStorageSizeRequest{
					Key: "0x3aa96b0149b6ca3688878bdbd19464448624136398e3ce45b9e755d3ab61355a",
				},
			},
			expErr: errors.New("GetStorage Error"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sm := &StateModule{
				networkAPI: tt.fields.networkAPI,
				storageAPI: tt.fields.storageAPI,
				coreAPI:    tt.fields.coreAPI,
			}
			res := StateStorageSizeResponse(0)
			err := sm.GetStorageSize(tt.args.in0, tt.args.req, &res)
			if tt.expErr != nil {
				assert.EqualError(t, err, tt.expErr.Error())
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.exp, res)
		})
	}
}

func TestStateModuleQueryStorage(t *testing.T) {
	t.Parallel()
	errTest := errors.New("test error")

	type fields struct {
		storageAPIBuilder func(ctrl *gomock.Controller) StorageAPI
		blockAPIBuilder   func(ctrl *gomock.Controller) BlockAPI
	}
	type args struct {
		in0 *http.Request
		req *StateStorageQueryRangeRequest
	}
	tests := map[string]struct {
		fields    fields
		args      args
		errRegexp string
		exp       []StorageChangeSetResponse
	}{
		"missing_start_block_error": {
			fields: fields{
				storageAPIBuilder: func(ctrl *gomock.Controller) StorageAPI {
					return NewMockStorageAPI(ctrl)
				},
				blockAPIBuilder: func(ctrl *gomock.Controller) BlockAPI {
					return NewMockBlockAPI(ctrl)
				}},
			args: args{
				req: &StateStorageQueryRangeRequest{
					Keys: []string{"0x010203"},
				},
			},
			exp:       []StorageChangeSetResponse{},
			errRegexp: "the start block hash cannot be an empty value",
		},
		"missing_start_block_not_found_error": {
			fields: fields{
				storageAPIBuilder: func(ctrl *gomock.Controller) StorageAPI {
					return NewMockStorageAPI(ctrl)
				},
				blockAPIBuilder: func(ctrl *gomock.Controller) BlockAPI {
					mockBlockAPI := NewMockBlockAPI(ctrl)
					mockBlockAPI.EXPECT().GetBlockByHash(common.Hash{1}).Return(nil, errTest)
					return mockBlockAPI
				}},
			args: args{
				req: &StateStorageQueryRangeRequest{
					Keys:       []string{"0x010203"},
					StartBlock: common.Hash{1},
				},
			},
			exp:       []StorageChangeSetResponse{},
			errRegexp: "test error",
		},
		"start_block/no_end_block/multi_keys/key_0_changes,_key_1_unchanged": {
			fields: fields{
				storageAPIBuilder: func(ctrl *gomock.Controller) StorageAPI {
					mockStorageAPI := NewMockStorageAPI(ctrl)
					mockStorageAPI.EXPECT().GetStorageByBlockHash(&common.Hash{2}, []byte{1, 2, 4}).
						Return([]byte{1, 1, 1}, nil)
					mockStorageAPI.EXPECT().GetStorageByBlockHash(&common.Hash{2}, []byte{9, 9, 9}).
						Return([]byte{9, 9, 9, 9}, nil)
					mockStorageAPI.EXPECT().GetStorageByBlockHash(&common.Hash{3}, []byte{1, 2, 4}).
						Return([]byte{2, 2, 2}, nil)
					mockStorageAPI.EXPECT().GetStorageByBlockHash(&common.Hash{3}, []byte{9, 9, 9}).
						Return([]byte{9, 9, 9, 9}, nil)
					mockStorageAPI.EXPECT().GetStorageByBlockHash(&common.Hash{4}, []byte{1, 2, 4}).
						Return([]byte{3, 3, 3}, nil)
					mockStorageAPI.EXPECT().GetStorageByBlockHash(&common.Hash{4}, []byte{9, 9, 9}).
						Return([]byte{9, 9, 9, 9}, nil)
					return mockStorageAPI
				},
				blockAPIBuilder: func(ctrl *gomock.Controller) BlockAPI {
					mockBlockAPI := NewMockBlockAPI(ctrl)
					mockBlockAPI.EXPECT().GetBlockByHash(common.Hash{2}).
						Return(&types.Block{Header: types.Header{Number: 1}}, nil)
					mockBlockAPI.EXPECT().BestBlockHash().Return(common.Hash{4})
					mockBlockAPI.EXPECT().GetBlockByHash(common.Hash{4}).
						Return(&types.Block{Header: types.Header{Number: 3}}, nil)
					mockBlockAPI.EXPECT().GetHashByNumber(uint(1)).Return(common.Hash{2}, nil)
					mockBlockAPI.EXPECT().GetHashByNumber(uint(2)).Return(common.Hash{3}, nil)
					mockBlockAPI.EXPECT().GetHashByNumber(uint(3)).Return(common.Hash{4}, nil)
					return mockBlockAPI
				}},
			args: args{
				req: &StateStorageQueryRangeRequest{
					Keys:       []string{"0x010204", "0x090909"},
					StartBlock: common.Hash{2},
				},
			},
			exp: []StorageChangeSetResponse{
				{
					Block: &common.Hash{2},
					Changes: [][2]*string{
						makeChange("0x010204", "0x010101"),
						makeChange("0x090909", "0x09090909"),
					},
				},
				{
					Block: &common.Hash{3},
					Changes: [][2]*string{
						makeChange("0x010204", "0x020202"),
					},
				},
				{
					Block: &common.Hash{4},
					Changes: [][2]*string{
						makeChange("0x010204", "0x030303"),
					},
				},
			},
		},
		"start_block/no_end_block/value_changes": {
			fields: fields{
				storageAPIBuilder: func(ctrl *gomock.Controller) StorageAPI {
					mockStorageAPI := NewMockStorageAPI(ctrl)
					mockStorageAPI.EXPECT().GetStorageByBlockHash(&common.Hash{1}, []byte{1, 2, 4}).
						Return([]byte{1, 1, 1}, nil)
					mockStorageAPI.EXPECT().GetStorageByBlockHash(&common.Hash{2}, []byte{1, 2, 4}).
						Return([]byte(nil), nil)
					mockStorageAPI.EXPECT().GetStorageByBlockHash(&common.Hash{3}, []byte{1, 2, 4}).
						Return([]byte{}, nil)
					mockStorageAPI.EXPECT().GetStorageByBlockHash(&common.Hash{4}, []byte{1, 2, 4}).
						Return([]byte{3, 3, 3}, nil)
					return mockStorageAPI
				},
				blockAPIBuilder: func(ctrl *gomock.Controller) BlockAPI {
					mockBlockAPI := NewMockBlockAPI(ctrl)
					mockBlockAPI.EXPECT().GetBlockByHash(common.Hash{1}).
						Return(&types.Block{Header: types.Header{Number: 0}}, nil)
					mockBlockAPI.EXPECT().BestBlockHash().Return(common.Hash{4})
					mockBlockAPI.EXPECT().GetBlockByHash(common.Hash{4}).
						Return(&types.Block{Header: types.Header{Number: 3}}, nil)
					mockBlockAPI.EXPECT().GetHashByNumber(uint(0)).Return(common.Hash{1}, nil)
					mockBlockAPI.EXPECT().GetHashByNumber(uint(1)).Return(common.Hash{2}, nil)
					mockBlockAPI.EXPECT().GetHashByNumber(uint(2)).Return(common.Hash{3}, nil)
					mockBlockAPI.EXPECT().GetHashByNumber(uint(3)).Return(common.Hash{4}, nil)
					return mockBlockAPI
				}},
			args: args{
				req: &StateStorageQueryRangeRequest{
					Keys:       []string{"0x010204"},
					StartBlock: common.Hash{1},
				},
			},
			exp: []StorageChangeSetResponse{
				{
					Block: &common.Hash{1},
					Changes: [][2]*string{
						makeChange("0x010204", "0x010101"),
					},
				},
				{
					Block: &common.Hash{2},
					Changes: [][2]*string{
						{stringPtr("0x010204"), nil},
					},
				},
				{
					Block: &common.Hash{3},
					Changes: [][2]*string{
						makeChange("0x010204", "0x"),
					},
				},
				{
					Block: &common.Hash{4},
					Changes: [][2]*string{
						makeChange("0x010204", "0x030303"),
					},
				},
			},
		},
		"start_block,_end_block,_ok": {
			fields: fields{
				storageAPIBuilder: func(ctrl *gomock.Controller) StorageAPI {
					mockStorageAPI := NewMockStorageAPI(ctrl)
					mockStorageAPI.EXPECT().GetStorageByBlockHash(&common.Hash{2}, []byte{1, 2, 4}).
						Return([]byte{1, 1, 1}, nil)
					mockStorageAPI.EXPECT().GetStorageByBlockHash(&common.Hash{3}, []byte{1, 2, 4}).
						Return([]byte{1, 1, 1}, nil)
					return mockStorageAPI
				},
				blockAPIBuilder: func(ctrl *gomock.Controller) BlockAPI {
					mockBlockAPI := NewMockBlockAPI(ctrl)
					mockBlockAPI.EXPECT().GetBlockByHash(common.Hash{2}).
						Return(&types.Block{Header: types.Header{Number: 1}}, nil)
					mockBlockAPI.EXPECT().GetBlockByHash(common.Hash{3}).
						Return(&types.Block{Header: types.Header{Number: 2}}, nil)
					mockBlockAPI.EXPECT().GetHashByNumber(uint(1)).Return(common.Hash{2}, nil)
					mockBlockAPI.EXPECT().GetHashByNumber(uint(2)).Return(common.Hash{3}, nil)
					return mockBlockAPI
				}},
			args: args{
				req: &StateStorageQueryRangeRequest{
					Keys:       []string{"0x010204"},
					StartBlock: common.Hash{2},
					EndBlock:   common.Hash{3},
				},
			},
			exp: []StorageChangeSetResponse{
				{
					Block: &common.Hash{2},
					Changes: [][2]*string{
						makeChange("0x010204", "0x010101"),
					},
				},
				{
					Block:   &common.Hash{3},
					Changes: [][2]*string{},
				},
			},
		},
		"start_block/end_block/error_end_hash": {
			fields: fields{
				storageAPIBuilder: func(ctrl *gomock.Controller) StorageAPI {
					mockStorageAPI := NewMockStorageAPI(ctrl)
					return mockStorageAPI
				},
				blockAPIBuilder: func(ctrl *gomock.Controller) BlockAPI {
					mockBlockAPI := NewMockBlockAPI(ctrl)
					mockBlockAPI.EXPECT().GetBlockByHash(common.Hash{2}).
						Return(&types.Block{Header: types.Header{Number: 1}}, nil)
					mockBlockAPI.EXPECT().GetBlockByHash(common.Hash{99}).Return(nil, errTest)
					return mockBlockAPI
				}},
			args: args{
				req: &StateStorageQueryRangeRequest{
					Keys:       []string{"0x010204"},
					StartBlock: common.Hash{2},
					EndBlock:   common.Hash{99},
				},
			},
			exp:       []StorageChangeSetResponse{},
			errRegexp: "getting block by hash: test error",
		},
		"start_block/end_block/error_get_hash_by_number": {
			fields: fields{
				storageAPIBuilder: func(ctrl *gomock.Controller) StorageAPI {
					mockStorageAPI := NewMockStorageAPI(ctrl)
					return mockStorageAPI
				},
				blockAPIBuilder: func(ctrl *gomock.Controller) BlockAPI {
					mockBlockAPI := NewMockBlockAPI(ctrl)
					mockBlockAPI.EXPECT().GetBlockByHash(common.Hash{2}).
						Return(&types.Block{Header: types.Header{Number: 1}}, nil)
					mockBlockAPI.EXPECT().GetBlockByHash(common.Hash{3}).
						Return(&types.Block{Header: types.Header{Number: 2}}, nil)
					mockBlockAPI.EXPECT().GetHashByNumber(uint(1)).Return(common.Hash{}, blocktree.ErrNumLowerThanRoot)
					return mockBlockAPI
				}},
			args: args{
				req: &StateStorageQueryRangeRequest{
					Keys:       []string{"0x010204"},
					StartBlock: common.Hash{2},
					EndBlock:   common.Hash{3},
				},
			},
			exp:       []StorageChangeSetResponse{},
			errRegexp: "cannot get hash by number: cannot find node with number lower than root node",
		},
		"start_block/end_block/error_get_storage_by_block_hash": {
			fields: fields{func(ctrl *gomock.Controller) StorageAPI {
				mockStorageAPI := NewMockStorageAPI(ctrl)
				mockStorageAPI.EXPECT().GetStorageByBlockHash(&common.Hash{2}, []byte{1, 2, 4}).Return(nil, errTest)
				return mockStorageAPI
			},
				func(ctrl *gomock.Controller) BlockAPI {
					mockBlockAPI := NewMockBlockAPI(ctrl)
					mockBlockAPI.EXPECT().GetBlockByHash(common.Hash{2}).Return(&types.Block{
						Header: types.Header{
							Number: 1,
						},
					}, nil)
					mockBlockAPI.EXPECT().GetBlockByHash(common.Hash{3}).Return(&types.Block{
						Header: types.Header{Number: 2},
					}, nil)
					mockBlockAPI.EXPECT().GetHashByNumber(uint(1)).Return(common.Hash{2}, nil)
					return mockBlockAPI
				}},
			args: args{
				req: &StateStorageQueryRangeRequest{
					Keys:       []string{"0x010204"},
					StartBlock: common.Hash{2},
					EndBlock:   common.Hash{3},
				},
			},
			exp:       []StorageChangeSetResponse{},
			errRegexp: "getting value by block hash: test error",
		},
	}
	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			sm := &StateModule{
				storageAPI: tt.fields.storageAPIBuilder(ctrl),
				blockAPI:   tt.fields.blockAPIBuilder(ctrl),
			}
			res := []StorageChangeSetResponse{}
			err := sm.QueryStorage(tt.args.in0, tt.args.req, &res)
			if tt.errRegexp != "" {
				assert.Regexp(t, tt.errRegexp, err.Error())
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.exp, res)
		})
	}
}
func TestStateModuleQueryStorageAt(t *testing.T) {
	t.Parallel()
	errTest := errors.New("test error")

	type fields struct {
		storageAPIBuilder func(ctrl *gomock.Controller) *MockStorageAPI
		blockAPIBuilder   func(ctrl *gomock.Controller) *MockBlockAPI
	}

	tests := map[string]struct {
		fields           fields
		request          *StateStorageQueryAtRequest
		expectedError    error
		expectedResponse []StorageChangeSetResponse
	}{
		"missing_start_block": {
			fields: fields{
				storageAPIBuilder: func(ctrl *gomock.Controller) *MockStorageAPI {
					mockStorageAPI := NewMockStorageAPI(ctrl)
					mockStorageAPI.EXPECT().GetStorageByBlockHash(&common.Hash{2}, []byte{1, 2, 3}).
						Return([]byte{1, 1, 1}, nil)
					return mockStorageAPI
				},
				blockAPIBuilder: func(ctrl *gomock.Controller) *MockBlockAPI {
					mockBlockAPI := NewMockBlockAPI(ctrl)
					mockBlockAPI.EXPECT().BestBlockHash().Return(common.Hash{2})
					return mockBlockAPI
				}},
			request: &StateStorageQueryAtRequest{
				Keys: []string{"0x010203"},
			},
			expectedResponse: []StorageChangeSetResponse{
				{
					Block: &common.Hash{2},
					Changes: [][2]*string{
						makeChange("0x010203", "0x010101"),
					},
				},
			},
		},
		"start_block_not_found_error": {
			fields: fields{
				storageAPIBuilder: func(ctrl *gomock.Controller) *MockStorageAPI {
					mockStorageAPI := NewMockStorageAPI(ctrl)
					mockStorageAPI.EXPECT().GetStorageByBlockHash(&common.Hash{1}, []byte{1, 2, 3}).Return(nil, errTest)
					return mockStorageAPI
				},
				blockAPIBuilder: func(ctrl *gomock.Controller) *MockBlockAPI {
					return NewMockBlockAPI(ctrl)
				}},
			request: &StateStorageQueryAtRequest{
				Keys: []string{"0x010203"},
				At:   common.Hash{1},
			},
			expectedResponse: []StorageChangeSetResponse{},
			expectedError:    errors.New("getting value by block hash: test error"),
		},
		"start_block/multi_keys": {
			fields: fields{
				storageAPIBuilder: func(ctrl *gomock.Controller) *MockStorageAPI {
					mockStorageAPI := NewMockStorageAPI(ctrl)
					mockStorageAPI.EXPECT().GetStorageByBlockHash(&common.Hash{2}, []byte{8, 8, 8}).
						Return([]byte{8, 8, 8, 8}, nil)
					mockStorageAPI.EXPECT().GetStorageByBlockHash(&common.Hash{2}, []byte{1, 2, 4}).
						Return([]byte{}, nil)
					mockStorageAPI.EXPECT().GetStorageByBlockHash(&common.Hash{2}, []byte{9, 9, 9}).
						Return([]byte(nil), nil)
					return mockStorageAPI
				},
				blockAPIBuilder: func(ctrl *gomock.Controller) *MockBlockAPI {
					return NewMockBlockAPI(ctrl)
				}},
			request: &StateStorageQueryAtRequest{
				Keys: []string{"0x080808", "0x010204", "0x090909"},
				At:   common.Hash{2},
			},
			expectedResponse: []StorageChangeSetResponse{
				{
					Block: &common.Hash{2},
					Changes: [][2]*string{
						makeChange("0x080808", "0x08080808"),
						makeChange("0x010204", "0x"),
						{stringPtr("0x090909"), nil},
					},
				},
			},
		},
		"missing_start_block/multi_keys": {
			fields: fields{
				storageAPIBuilder: func(ctrl *gomock.Controller) *MockStorageAPI {
					mockStorageAPI := NewMockStorageAPI(ctrl)
					mockStorageAPI.EXPECT().GetStorageByBlockHash(&common.Hash{2}, []byte{1, 2, 4}).
						Return([]byte{1, 1, 1}, nil)
					mockStorageAPI.EXPECT().GetStorageByBlockHash(&common.Hash{2}, []byte{9, 9, 9}).
						Return([]byte{9, 9, 9, 9}, nil)
					return mockStorageAPI
				},
				blockAPIBuilder: func(ctrl *gomock.Controller) *MockBlockAPI {
					mockBlockAPI := NewMockBlockAPI(ctrl)
					mockBlockAPI.EXPECT().BestBlockHash().Return(common.Hash{2})
					return mockBlockAPI
				}},
			request: &StateStorageQueryAtRequest{
				Keys: []string{"0x010204", "0x090909"},
			},
			expectedResponse: []StorageChangeSetResponse{
				{
					Block: &common.Hash{2},
					Changes: [][2]*string{
						makeChange("0x010204", "0x010101"),
						makeChange("0x090909", "0x09090909"),
					},
				},
			},
		},
	}
	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			sm := &StateModule{
				storageAPI: tt.fields.storageAPIBuilder(ctrl),
				blockAPI:   tt.fields.blockAPIBuilder(ctrl),
			}
			response := []StorageChangeSetResponse{}
			err := sm.QueryStorageAt(nil, tt.request, &response)
			if tt.expectedError != nil {
				assert.EqualError(t, err, tt.expectedError.Error())
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.expectedResponse, response)
		})
	}
}
