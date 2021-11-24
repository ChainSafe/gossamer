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

	"github.com/ChainSafe/gossamer/dot/core"
	apimocks "github.com/ChainSafe/gossamer/dot/rpc/modules/mocks"
	testdata "github.com/ChainSafe/gossamer/dot/rpc/modules/test_data"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/runtime"
	"github.com/ChainSafe/gossamer/pkg/scale"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStateModuleGetPairs(t *testing.T) {
	str := "0x01"
	hash := common.MustHexToHash("0x3aa96b0149b6ca3688878bdbd19464448624136398e3ce45b9e755d3ab61355a")
	m := make(map[string][]byte)
	m["a"] = []byte{21, 22}
	m["b"] = []byte{23, 24}

	mockStorageAPI := new(apimocks.StorageAPI)
	mockStorageAPI.On("GetStateRootFromBlock", &hash).Return(&hash, nil)
	mockStorageAPI.On("Entries", &hash).Return(m, nil)
	mockStorageAPI.On("GetKeysWithPrefix", &hash, common.MustHexToBytes(str)).Return([][]byte{{1}, {1}}, nil)
	mockStorageAPI.On("GetStorage", &hash, []byte{1}).Return([]byte{21}, nil)

	mockStorageAPINil := new(apimocks.StorageAPI)
	mockStorageAPINil.On("GetStateRootFromBlock", &hash).Return(&hash, nil)
	mockStorageAPINil.On("Entries", &hash).Return(m, nil)

	mockStorageAPIGetKeysEmpty := new(apimocks.StorageAPI)
	mockStorageAPIGetKeysEmpty.On("GetStateRootFromBlock", &hash).Return(&hash, nil)
	mockStorageAPIGetKeysEmpty.On("Entries", &hash).Return(m, nil)
	mockStorageAPIGetKeysEmpty.On("GetKeysWithPrefix", &hash, common.MustHexToBytes(str)).Return([][]byte{}, nil)

	mockStorageAPIGetKeysErr := new(apimocks.StorageAPI)
	mockStorageAPIGetKeysErr.On("GetStateRootFromBlock", &hash).Return(&hash, nil)
	mockStorageAPIGetKeysErr.On("Entries", &hash).Return(m, nil)
	mockStorageAPIGetKeysErr.On("GetKeysWithPrefix", &hash, common.MustHexToBytes(str)).Return(nil, errors.New("GetKeysWithPrefix Err"))

	mockStorageAPIEntriesErr := new(apimocks.StorageAPI)
	mockStorageAPIEntriesErr.On("GetStateRootFromBlock", &hash).Return(&hash, nil)
	mockStorageAPIEntriesErr.On("Entries", &hash).Return(nil, errors.New("entries Err"))

	mockStorageAPIErr := new(apimocks.StorageAPI)
	mockStorageAPIErr.On("GetStateRootFromBlock", &hash).Return(nil, errors.New("GetStateRootFromBlock Err"))

	mockStorageAPIGetStorageErr := new(apimocks.StorageAPI)
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
		res StatePairResponse
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
		err     error
		exp     StatePairResponse
	}{
		{
			name:   "GetStateRootFromBlock Error",
			fields: fields{nil, mockStorageAPIErr, nil},
			args: args{
				req: &StatePairRequest{
					Bhash: &hash,
				},
			},
			wantErr: true,
			err:     errors.New("GetStateRootFromBlock Err"),
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
			exp: []interface{}{},
			wantErr: true,
			err:     errors.New("entries Err"),
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
			wantErr: true,
			err:     errors.New("GetKeysWithPrefix Err"),
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
			exp: StatePairResponse{interface{}(nil), interface{}(nil)},
			wantErr: true,
			err:     errors.New("GetStorage Err"),
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
			tt.args.res = StatePairResponse{}
			sm := &StateModule{
				networkAPI: tt.fields.networkAPI,
				storageAPI: tt.fields.storageAPI,
				coreAPI:    tt.fields.coreAPI,
			}
			err := sm.GetPairs(tt.args.in0, tt.args.req, &tt.args.res)
			if tt.wantErr {
				require.Error(t, err)
				assert.EqualError(t, err, tt.err.Error())
			} else {
				assert.NoError(t, err)
			}
			assert.ElementsMatch(t, tt.exp, tt.args.res)
		})
	}
}

func TestStateModuleGetKeysPaged(t *testing.T) {
	mockStorageAPI := new(apimocks.StorageAPI)
	mockStorageAPI.On("GetKeysWithPrefix", (*common.Hash)(nil), common.MustHexToBytes("0x")).Return([][]byte{{1}, {2}}, nil)

	mockStorageAPI2 := new(apimocks.StorageAPI)
	mockStorageAPI2.On("GetKeysWithPrefix", (*common.Hash)(nil), common.MustHexToBytes("0x")).Return([][]byte{{1, 1, 1}, {1, 1, 1}}, nil)

	mockStorageAPIErr := new(apimocks.StorageAPI)
	mockStorageAPIErr.On("GetKeysWithPrefix", (*common.Hash)(nil), common.MustHexToBytes("0x")).Return(nil, errors.New("GetKeysWithPrefix Err"))

	type fields struct {
		networkAPI NetworkAPI
		storageAPI StorageAPI
		coreAPI    CoreAPI
	}
	type args struct {
		in0 *http.Request
		req *StateStorageKeyRequest
		res StateStorageKeysResponse
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
		err     error
		exp     StateStorageKeysResponse
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
			wantErr: true,
			err:     errors.New("GetKeysWithPrefix Err"),
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
			wantErr: true,
			err:     errors.New("invalid string"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.args.res = StateStorageKeysResponse(nil)
			sm := &StateModule{
				networkAPI: tt.fields.networkAPI,
				storageAPI: tt.fields.storageAPI,
				coreAPI:    tt.fields.coreAPI,
			}
			err := sm.GetKeysPaged(tt.args.in0, tt.args.req, &tt.args.res)
			if tt.wantErr {
				require.Error(t, err)
				assert.EqualError(t, err, tt.err.Error())
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.exp, tt.args.res)
		})
	}
}

// Implement Tests once function is implemented
func TestCall(t *testing.T) {
	mockNetworkAPI := new(apimocks.NetworkAPI)
	mockStorageAPI := new(apimocks.StorageAPI)
	sm := NewStateModule(mockNetworkAPI, mockStorageAPI, nil)

	err := sm.Call(nil, nil, nil)
	assert.NoError(t, err)
}

func TestStateModuleGetMetadata(t *testing.T) {
	hash := common.MustHexToHash("0x3aa96b0149b6ca3688878bdbd19464448624136398e3ce45b9e755d3ab61355a")

	mockCoreAPI := new(apimocks.CoreAPI)
	mockCoreAPI.On("GetMetadata", &hash).Return(common.MustHexToBytes(testdata.GetTestData()), nil)

	mockCoreAPIErr := new(apimocks.CoreAPI)
	mockCoreAPIErr.On("GetMetadata", &hash).Return(nil, errors.New("GetMetadata Error"))

	mockStateModule := NewStateModule(nil, nil, mockCoreAPIErr)

	var expRes []byte
	err := scale.Unmarshal(common.MustHexToBytes(testdata.GetTestData()), &expRes)
	assert.NoError(t, err)
	type fields struct {
		networkAPI NetworkAPI
		storageAPI StorageAPI
		coreAPI    CoreAPI
	}
	type args struct {
		in0 *http.Request
		req *StateRuntimeMetadataQuery
		res StateMetadataResponse
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
		err     error
		exp     StateMetadataResponse
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
			wantErr: true,
			err:     errors.New("GetMetadata Error"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.args.res = StateMetadataResponse("")
			sm := &StateModule{
				networkAPI: tt.fields.networkAPI,
				storageAPI: tt.fields.storageAPI,
				coreAPI:    tt.fields.coreAPI,
			}
			err := sm.GetMetadata(tt.args.in0, tt.args.req, &tt.args.res)
			if tt.wantErr {
				require.Error(t, err)
				assert.EqualError(t, err, tt.err.Error())
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.exp, tt.args.res)
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

	mockCoreAPI := new(apimocks.CoreAPI)
	mockCoreAPI.On("GetReadProofAt", hash, expKeys).Return(hash, [][]byte{{1, 1, 1}, {1, 1, 1}}, nil)

	mockCoreAPIErr := new(apimocks.CoreAPI)
	mockCoreAPIErr.On("GetReadProofAt", hash, expKeys).Return(nil, nil, errors.New("GetReadProofAt Error"))

	type fields struct {
		networkAPI NetworkAPI
		storageAPI StorageAPI
		coreAPI    CoreAPI
	}
	type args struct {
		in0 *http.Request
		req *StateGetReadProofRequest
		res StateGetReadProofResponse
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
		err     error
		exp     StateGetReadProofResponse
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
				At:    common.Hash{0x3a, 0xa9, 0x6b, 0x1, 0x49, 0xb6, 0xca, 0x36, 0x88, 0x87, 0x8b, 0xdb, 0xd1, 0x94, 0x64, 0x44, 0x86, 0x24, 0x13, 0x63, 0x98, 0xe3, 0xce, 0x45, 0xb9, 0xe7, 0x55, 0xd3, 0xab, 0x61, 0x35, 0x5a},
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
			wantErr: true,
			err:     errors.New("GetReadProofAt Error"),
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
			wantErr: true,
			err:     errors.New("GetReadProofAt Error"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.args.res = StateGetReadProofResponse{}
			sm := &StateModule{
				networkAPI: tt.fields.networkAPI,
				storageAPI: tt.fields.storageAPI,
				coreAPI:    tt.fields.coreAPI,
			}
			err := sm.GetReadProof(tt.args.in0, tt.args.req, &tt.args.res)
			if tt.wantErr {
				require.Error(t, err)
				assert.EqualError(t, err, tt.err.Error())
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.exp, tt.args.res)
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

	mockCoreAPI := new(apimocks.CoreAPI)
	mockCoreAPI.On("GetRuntimeVersion", &hash).Return(version, nil)

	mockCoreAPIErr := new(apimocks.CoreAPI)
	mockCoreAPIErr.On("GetRuntimeVersion", &hash).Return(nil, errors.New("GetRuntimeVersion Error"))

	type fields struct {
		networkAPI NetworkAPI
		storageAPI StorageAPI
		coreAPI    CoreAPI
	}
	type args struct {
		in0 *http.Request
		req *StateRuntimeVersionRequest
		res StateRuntimeVersionResponse
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
		err     error
		exp     StateRuntimeVersionResponse
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
			wantErr: true,
			err:     errors.New("GetRuntimeVersion Error"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.args.res = StateRuntimeVersionResponse{}
			sm := &StateModule{
				networkAPI: tt.fields.networkAPI,
				storageAPI: tt.fields.storageAPI,
				coreAPI:    tt.fields.coreAPI,
			}
			err := sm.GetRuntimeVersion(tt.args.in0, tt.args.req, &tt.args.res)
			if tt.wantErr {
				require.Error(t, err)
				assert.EqualError(t, err, tt.err.Error())
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.exp, tt.args.res)
		})
	}
}

func TestStateModuleGetStorage(t *testing.T) {
	hash := common.MustHexToHash("0x3aa96b0149b6ca3688878bdbd19464448624136398e3ce45b9e755d3ab61355a")
	reqBytes := common.MustHexToBytes("0x3aa96b0149b6ca3688878bdbd19464448624136398e3ce45b9e755d3ab61355a")

	mockStorageAPI := new(apimocks.StorageAPI)
	mockStorageAPI.On("GetStorageByBlockHash", &hash, reqBytes).Return([]byte{21}, nil)
	mockStorageAPI.On("GetStorage", (*common.Hash)(nil), reqBytes).Return([]byte{21}, nil)

	mockStorageAPIErr := new(apimocks.StorageAPI)
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
		res StateStorageResponse
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
		err     error
		exp     StateStorageResponse
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
			wantErr: true,
			err:     errors.New("GetStorageByBlockHash Error"),
		},
		{
			name:   "bHash Nil Err",
			fields: fields{nil, mockStorageAPIErr, nil},
			args: args{
				req: &StateStorageRequest{
					Key: "0x3aa96b0149b6ca3688878bdbd19464448624136398e3ce45b9e755d3ab61355a",
				},
			},
			wantErr: true,
			err:     errors.New("GetStorage Error"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.args.res = StateStorageResponse("")
			sm := &StateModule{
				networkAPI: tt.fields.networkAPI,
				storageAPI: tt.fields.storageAPI,
				coreAPI:    tt.fields.coreAPI,
			}
			err := sm.GetStorage(tt.args.in0, tt.args.req, &tt.args.res)
			if tt.wantErr {
				require.Error(t, err)
				assert.EqualError(t, err, tt.err.Error())
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.exp, tt.args.res)
		})
	}
}

func TestStateModuleGetStorageHash(t *testing.T) {
	hash := common.MustHexToHash("0x3aa96b0149b6ca3688878bdbd19464448624136398e3ce45b9e755d3ab61355a")
	reqBytes := common.MustHexToBytes("0x3aa96b0149b6ca3688878bdbd19464448624136398e3ce45b9e755d3ab61355a")

	mockStorageAPI := new(apimocks.StorageAPI)
	mockStorageAPI.On("GetStorageByBlockHash", &hash, reqBytes).Return([]byte{21}, nil)
	mockStorageAPI.On("GetStorage", (*common.Hash)(nil), reqBytes).Return([]byte{21}, nil)

	mockStorageAPIErr := new(apimocks.StorageAPI)
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
		res StateStorageHashResponse
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
		err     error
		exp     StateStorageHashResponse
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
			exp: StateStorageHashResponse("0x0000000000000000000000000000000000000000000000000000000000000015"),
		},
		{
			name:   "bHash Nil OK",
			fields: fields{nil, mockStorageAPI, nil},
			args: args{
				req: &StateStorageHashRequest{
					Key: "0x3aa96b0149b6ca3688878bdbd19464448624136398e3ce45b9e755d3ab61355a",
				},
			},
			exp: StateStorageHashResponse("0x0000000000000000000000000000000000000000000000000000000000000015"),
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
			wantErr: true,
			err:     errors.New("GetStorageByBlockHash Error"),
		},
		{
			name:   "bHash Nil Err",
			fields: fields{nil, mockStorageAPIErr, nil},
			args: args{
				req: &StateStorageHashRequest{
					Key: "0x3aa96b0149b6ca3688878bdbd19464448624136398e3ce45b9e755d3ab61355a",
				},
			},
			wantErr: true,
			err:     errors.New("GetStorage Error"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.args.res = StateStorageHashResponse("")
			sm := &StateModule{
				networkAPI: tt.fields.networkAPI,
				storageAPI: tt.fields.storageAPI,
				coreAPI:    tt.fields.coreAPI,
			}
			err := sm.GetStorageHash(tt.args.in0, tt.args.req, &tt.args.res)
			if tt.wantErr {
				require.Error(t, err)
				assert.EqualError(t, err, tt.err.Error())
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.exp, tt.args.res)
		})
	}
}

func TestStateModuleGetStorageSize(t *testing.T) {
	hash := common.MustHexToHash("0x3aa96b0149b6ca3688878bdbd19464448624136398e3ce45b9e755d3ab61355a")
	reqBytes := common.MustHexToBytes("0x3aa96b0149b6ca3688878bdbd19464448624136398e3ce45b9e755d3ab61355a")

	mockStorageAPI := new(apimocks.StorageAPI)
	mockStorageAPI.On("GetStorageByBlockHash", &hash, reqBytes).Return([]byte{21}, nil)
	mockStorageAPI.On("GetStorage", (*common.Hash)(nil), reqBytes).Return([]byte{21}, nil)

	mockStorageAPIErr := new(apimocks.StorageAPI)
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
		res StateStorageSizeResponse
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
		err     error
		exp     StateStorageSizeResponse
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
			wantErr: true,
			err:     errors.New("GetStorageByBlockHash Error"),
		},
		{
			name:   "bHash Nil Err",
			fields: fields{nil, mockStorageAPIErr, nil},
			args: args{
				req: &StateStorageSizeRequest{
					Key: "0x3aa96b0149b6ca3688878bdbd19464448624136398e3ce45b9e755d3ab61355a",
				},
			},
			wantErr: true,
			err:     errors.New("GetStorage Error"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.args.res = StateStorageSizeResponse(0)
			sm := &StateModule{
				networkAPI: tt.fields.networkAPI,
				storageAPI: tt.fields.storageAPI,
				coreAPI:    tt.fields.coreAPI,
			}
			err := sm.GetStorageSize(tt.args.in0, tt.args.req, &tt.args.res)
			if tt.wantErr {
				require.Error(t, err)
				assert.EqualError(t, err, tt.err.Error())
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.exp, tt.args.res)
		})
	}
}

func TestStateModuleQueryStorage(t *testing.T) {
	qkvc1 := core.QueryKeyValueChanges{}
	qkvc1["p1"] = "jimmy"
	qkvc2 := core.QueryKeyValueChanges{}
	qkvc2["p2"] = "jimbo"

	hash1 := common.MustHexToHash("0x3aa96b0149b6ca3688878bdbd19464448624136398e3ce45b9e755d3ab61355a")
	hash2 := common.MustHexToHash("0x3aa96b0149b6ca3688878bdbd19464448624136398e3ce45b9e755d3ab61355a")

	m := map[common.Hash]core.QueryKeyValueChanges{}
	m[hash1] = qkvc1
	m[hash2] = qkvc2

	mockCoreAPI := new(apimocks.CoreAPI)
	mockCoreAPI.On("QueryStorage", hash1, hash2, "jimbo").Return(m, nil)

	mockCoreAPIErr := new(apimocks.CoreAPI)
	mockCoreAPIErr.On("QueryStorage", hash1, hash2, "jimbo").Return(nil, errors.New("QueryStorage Error"))

	type fields struct {
		networkAPI NetworkAPI
		storageAPI StorageAPI
		coreAPI    CoreAPI
	}
	type args struct {
		in0 *http.Request
		req *StateStorageQueryRangeRequest
		res []StorageChangeSetResponse
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
		err     error
		exp     []StorageChangeSetResponse
	}{
		{
			name:   "OK Case",
			fields: fields{nil, nil, mockCoreAPI},
			args: args{
				req: &StateStorageQueryRangeRequest{
					Keys:       []string{"jimbo"},
					StartBlock: hash1,
					EndBlock:   hash2,
				},
			},
			exp: []StorageChangeSetResponse{{Block: &hash1, Changes: [][]string{{"p2", "jimbo"}}}},
		},
		{
			name:   "QueryStorage Error",
			fields: fields{nil, nil, mockCoreAPIErr},
			args: args{
				req: &StateStorageQueryRangeRequest{
					Keys:       []string{"jimbo"},
					StartBlock: hash1,
					EndBlock:   hash2,
				},
			},
			exp: []StorageChangeSetResponse{},
			wantErr: true,
			err:     errors.New("QueryStorage Error"),
		},
		{
			name:   "Empty Start Block Error",
			fields: fields{nil, nil, mockCoreAPI},
			args: args{
				req: &StateStorageQueryRangeRequest{
					Keys:     []string{"jimbo"},
					EndBlock: hash2,
				},
			},
			exp: []StorageChangeSetResponse{},
			wantErr: true,
			err:     errors.New("the start block hash cannot be an empty value"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.args.res = []StorageChangeSetResponse{}
			sm := &StateModule{
				networkAPI: tt.fields.networkAPI,
				storageAPI: tt.fields.storageAPI,
				coreAPI:    tt.fields.coreAPI,
			}
			err := sm.QueryStorage(tt.args.in0, tt.args.req, &tt.args.res)
			if tt.wantErr {
				require.Error(t, err)
				assert.EqualError(t, err, tt.err.Error())
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.exp, tt.args.res)
		})
	}
}
