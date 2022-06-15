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

	"github.com/ChainSafe/chaindb"
	"github.com/ChainSafe/gossamer/dot/rpc/modules/mocks"
	testdata "github.com/ChainSafe/gossamer/dot/rpc/modules/test_data"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/runtime"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestStateModuleGetPairs(t *testing.T) {
	str := "0x01"
	hash := common.MustHexToHash("0x3aa96b0149b6ca3688878bdbd19464448624136398e3ce45b9e755d3ab61355a")
	m := make(map[string][]byte)
	m["a"] = []byte{21, 22}
	m["b"] = []byte{23, 24}

	mockStorageAPI := new(mocks.StorageAPI)
	mockStorageAPI.On("GetStateRootFromBlock", &hash).Return(&hash, nil)
	mockStorageAPI.On("Entries", &hash).Return(m, nil)
	mockStorageAPI.On("GetKeysWithPrefix", &hash, common.MustHexToBytes(str)).Return([][]byte{{1}, {1}}, nil)
	mockStorageAPI.On("GetStorage", &hash, []byte{1}).Return([]byte{21}, nil)

	mockStorageAPINil := new(mocks.StorageAPI)
	mockStorageAPINil.On("GetStateRootFromBlock", &hash).Return(&hash, nil)
	mockStorageAPINil.On("Entries", &hash).Return(m, nil)

	mockStorageAPIGetKeysEmpty := new(mocks.StorageAPI)
	mockStorageAPIGetKeysEmpty.On("GetStateRootFromBlock", &hash).Return(&hash, nil)
	mockStorageAPIGetKeysEmpty.On("Entries", &hash).Return(m, nil)
	mockStorageAPIGetKeysEmpty.On("GetKeysWithPrefix", &hash, common.MustHexToBytes(str)).Return([][]byte{}, nil)

	mockStorageAPIGetKeysErr := new(mocks.StorageAPI)
	mockStorageAPIGetKeysErr.On("GetStateRootFromBlock", &hash).Return(&hash, nil)
	mockStorageAPIGetKeysErr.On("Entries", &hash).Return(m, nil)
	mockStorageAPIGetKeysErr.On("GetKeysWithPrefix", &hash, common.MustHexToBytes(str)).
		Return(nil, errors.New("GetKeysWithPrefix Err"))

	mockStorageAPIEntriesErr := new(mocks.StorageAPI)
	mockStorageAPIEntriesErr.On("GetStateRootFromBlock", &hash).Return(&hash, nil)
	mockStorageAPIEntriesErr.On("Entries", &hash).Return(nil, errors.New("entries Err"))

	mockStorageAPIErr := new(mocks.StorageAPI)
	mockStorageAPIErr.On("GetStateRootFromBlock", &hash).Return(nil, errors.New("GetStateRootFromBlock Err"))

	mockStorageAPIGetStorageErr := new(mocks.StorageAPI)
	mockStorageAPIGetStorageErr.On("GetStateRootFromBlock", &hash).Return(&hash, nil)
	mockStorageAPIGetStorageErr.On("Entries", &hash).Return(m, nil)
	mockStorageAPIGetStorageErr.On("GetKeysWithPrefix", &hash, common.MustHexToBytes(str)).Return([][]byte{{2}, {2}}, nil)
	mockStorageAPIGetStorageErr.On("GetStorage", &hash, []byte{2}).Return(nil, errors.New("GetStorage Err"))

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
	mockStorageAPI := new(mocks.StorageAPI)
	mockStorageAPI.On("GetKeysWithPrefix", (*common.Hash)(nil), common.MustHexToBytes("0x")).
		Return([][]byte{{1}, {2}}, nil)

	mockStorageAPI2 := new(mocks.StorageAPI)
	mockStorageAPI2.On("GetKeysWithPrefix", (*common.Hash)(nil), common.MustHexToBytes("0x")).
		Return([][]byte{{1, 1, 1}, {1, 1, 1}}, nil)

	mockStorageAPIErr := new(mocks.StorageAPI)
	mockStorageAPIErr.On("GetKeysWithPrefix", (*common.Hash)(nil), common.MustHexToBytes("0x")).
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

// Implement Tests once function is implemented
func TestCall(t *testing.T) {
	mockNetworkAPI := new(mocks.NetworkAPI)
	mockStorageAPI := new(mocks.StorageAPI)
	sm := NewStateModule(mockNetworkAPI, mockStorageAPI, nil, nil)

	err := sm.Call(nil, nil, nil)
	assert.NoError(t, err)
}

func TestStateModuleGetMetadata(t *testing.T) {
	hash := common.MustHexToHash("0x3aa96b0149b6ca3688878bdbd19464448624136398e3ce45b9e755d3ab61355a")

	mockCoreAPI := new(mocks.CoreAPI)
	mockCoreAPI.On("GetMetadata", &hash).Return(common.MustHexToBytes(testdata.NewTestMetadata()), nil)

	mockCoreAPIErr := new(mocks.CoreAPI)
	mockCoreAPIErr.On("GetMetadata", &hash).Return(nil, errors.New("GetMetadata Error"))

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
	hash := common.MustHexToHash("0x3aa96b0149b6ca3688878bdbd19464448624136398e3ce45b9e755d3ab61355a")
	keys := []string{"0x1111", "0x2222"}
	expKeys := make([][]byte, len(keys))
	for i, hexKey := range keys {
		bKey, err := common.HexToBytes(hexKey)
		assert.NoError(t, err)

		expKeys[i] = bKey
	}

	mockCoreAPI := new(mocks.CoreAPI)
	mockCoreAPI.On("GetReadProofAt", hash, expKeys).Return(hash, [][]byte{{1, 1, 1}, {1, 1, 1}}, nil)

	mockCoreAPIErr := new(mocks.CoreAPI)
	mockCoreAPIErr.On("GetReadProofAt", hash, expKeys).Return(nil, nil, errors.New("GetReadProofAt Error"))

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
	hash := common.MustHexToHash("0x3aa96b0149b6ca3688878bdbd19464448624136398e3ce45b9e755d3ab61355a")
	testAPIItem := runtime.APIItem{
		Name: [8]byte{1, 2, 3, 4, 5, 6, 7, 8},
		Ver:  99,
	}
	version := runtime.NewVersionData(
		[]byte("polkadot"),
		[]byte("parity-polkadot"),
		0,
		25,
		0,
		[]runtime.APIItem{testAPIItem},
		5,
	)

	mockCoreAPI := new(mocks.CoreAPI)
	mockCoreAPI.On("GetRuntimeVersion", &hash).Return(version, nil)

	mockCoreAPIErr := new(mocks.CoreAPI)
	mockCoreAPIErr.On("GetRuntimeVersion", &hash).Return(nil, errors.New("GetRuntimeVersion Error"))

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
	hash := common.MustHexToHash("0x3aa96b0149b6ca3688878bdbd19464448624136398e3ce45b9e755d3ab61355a")
	reqBytes := common.MustHexToBytes("0x3aa96b0149b6ca3688878bdbd19464448624136398e3ce45b9e755d3ab61355a")

	mockStorageAPI := new(mocks.StorageAPI)
	mockStorageAPI.On("GetStorageByBlockHash", &hash, reqBytes).Return([]byte{21}, nil)
	mockStorageAPI.On("GetStorage", (*common.Hash)(nil), reqBytes).Return([]byte{21}, nil)

	mockStorageAPIErr := new(mocks.StorageAPI)
	mockStorageAPIErr.On("GetStorageByBlockHash", &hash, reqBytes).Return(nil, errors.New("GetStorageByBlockHash Error"))
	mockStorageAPIErr.On("GetStorage", (*common.Hash)(nil), reqBytes).Return(nil, errors.New("GetStorage Error"))

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
	hash := common.MustHexToHash("0x3aa96b0149b6ca3688878bdbd19464448624136398e3ce45b9e755d3ab61355a")
	reqBytes := common.MustHexToBytes("0x3aa96b0149b6ca3688878bdbd19464448624136398e3ce45b9e755d3ab61355a")

	mockStorageAPI := new(mocks.StorageAPI)
	mockStorageAPI.On("GetStorageByBlockHash", &hash, reqBytes).Return([]byte{21}, nil)
	mockStorageAPI.On("GetStorage", (*common.Hash)(nil), reqBytes).Return([]byte{21}, nil)

	mockStorageAPIErr := new(mocks.StorageAPI)
	mockStorageAPIErr.On("GetStorageByBlockHash", &hash, reqBytes).Return(nil, errors.New("GetStorageByBlockHash Error"))
	mockStorageAPIErr.On("GetStorage", (*common.Hash)(nil), reqBytes).Return(nil, errors.New("GetStorage Error"))

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
	hash := common.MustHexToHash("0x3aa96b0149b6ca3688878bdbd19464448624136398e3ce45b9e755d3ab61355a")
	reqBytes := common.MustHexToBytes("0x3aa96b0149b6ca3688878bdbd19464448624136398e3ce45b9e755d3ab61355a")

	mockStorageAPI := new(mocks.StorageAPI)
	mockStorageAPI.On("GetStorageByBlockHash", &hash, reqBytes).Return([]byte{21}, nil)
	mockStorageAPI.On("GetStorage", (*common.Hash)(nil), reqBytes).Return([]byte{21}, nil)

	mockStorageAPIErr := new(mocks.StorageAPI)
	mockStorageAPIErr.On("GetStorageByBlockHash", &hash, reqBytes).Return(nil, errors.New("GetStorageByBlockHash Error"))
	mockStorageAPIErr.On("GetStorage", (*common.Hash)(nil), reqBytes).Return(nil, errors.New("GetStorage Error"))

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

	type fields struct {
		storageAPIBuilder func(ctrl *gomock.Controller) StorageAPI
		blockAPIBuilder   func(ctrl *gomock.Controller) BlockAPI
	}
	type args struct {
		in0 *http.Request
		req *StateStorageQueryRangeRequest
	}
	tests := map[string]struct {
		fields fields
		args   args
		expErr error
		exp    []StorageChangeSetResponse
	}{
		"missing start block error": {
			fields: fields{func(ctrl *gomock.Controller) StorageAPI {
				return NewMockStorageAPI(ctrl)
			},
				func(ctrl *gomock.Controller) BlockAPI {
					return NewMockBlockAPI(ctrl)
				}},
			args: args{
				req: &StateStorageQueryRangeRequest{
					Keys: []string{"0x010203"},
				},
			},
			exp:    []StorageChangeSetResponse{},
			expErr: ErrStartBlockValueEmpty,
		},
		"missing start block not found error": {
			fields: fields{func(ctrl *gomock.Controller) StorageAPI {
				return NewMockStorageAPI(ctrl)
			},
				func(ctrl *gomock.Controller) BlockAPI {
					mockBlockAPI := NewMockBlockAPI(ctrl)
					mockBlockAPI.EXPECT().GetBlockByHash(common.Hash{1}).Return(nil, chaindb.ErrKeyNotFound)
					return mockBlockAPI
				}},
			args: args{
				req: &StateStorageQueryRangeRequest{
					Keys:       []string{"0x010203"},
					StartBlock: common.Hash{1},
				},
			},
			exp:    []StorageChangeSetResponse{},
			expErr: chaindb.ErrKeyNotFound,
		},
		"start block ok": {
			fields: fields{func(ctrl *gomock.Controller) StorageAPI {
				mockStorageAPI := NewMockStorageAPI(ctrl)
				mockStorageAPI.EXPECT().GetStorageByBlockHash(common.Hash{2}, []byte{1, 2, 4})
				return NewMockStorageAPI(ctrl)
			},
				func(ctrl *gomock.Controller) BlockAPI {
					mockBlockAPI := NewMockBlockAPI(ctrl)
					mockBlockAPI.EXPECT().GetBlockByHash(common.Hash{2}).Return(&types.Block{
						Header: types.Header{
							Number: 1,
						},
					}, nil)
					mockBlockAPI.EXPECT().BestBlockHash().Return(common.Hash{4})
					mockBlockAPI.EXPECT().GetBlockByHash(common.Hash{4}).Return(&types.Block{
						Header: types.Header{Number: 3},
					}, nil)
					mockBlockAPI.EXPECT().GetHashByNumber(uint(1)).Return(common.Hash{2}, nil)
					return mockBlockAPI
				}},
			args: args{
				req: &StateStorageQueryRangeRequest{
					Keys:       []string{"0x010204"},
					StartBlock: common.Hash{2},
				},
			},
			exp:    []StorageChangeSetResponse{},
			expErr: chaindb.ErrKeyNotFound,
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
			if tt.expErr != nil {
				assert.EqualError(t, err, tt.expErr.Error())
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.exp, res)
		})
	}
}
