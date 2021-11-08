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
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestStateModuleGetPairs(t *testing.T) {
	hash := common.MustHexToHash("0x3aa96b0149b6ca3688878bdbd19464448624136398e3ce45b9e755d3ab61355a")
	m := make(map[string][]byte)
	m["a"] = []byte{21, 22}
	m["b"] = []byte{23, 24}

	mockStorageAPI := new(apimocks.StorageAPI)
	mockStorageAPI.On("GetStateRootFromBlock", mock.AnythingOfType("*common.Hash")).Return(&hash, nil)
	mockStorageAPI.On("Entries", mock.AnythingOfType("*common.Hash")).Return(m, nil)
	mockStorageAPI.On("GetKeysWithPrefix", mock.AnythingOfType("*common.Hash"), mock.AnythingOfType("[]uint8")).Return([][]byte{{1}, {2}}, nil)
	mockStorageAPI.On("GetStorage", mock.AnythingOfType("*common.Hash"), mock.AnythingOfType("[]uint8")).Return([]byte{21}, nil)

	mockStorageAPIGetKeysEmpty := new(apimocks.StorageAPI)
	mockStorageAPIGetKeysEmpty.On("GetStateRootFromBlock", mock.AnythingOfType("*common.Hash")).Return(&hash, nil)
	mockStorageAPIGetKeysEmpty.On("Entries", mock.AnythingOfType("*common.Hash")).Return(m, nil)
	mockStorageAPIGetKeysEmpty.On("GetKeysWithPrefix", mock.AnythingOfType("*common.Hash"), mock.AnythingOfType("[]uint8")).Return([][]byte{}, nil)
	mockStorageAPIGetKeysEmpty.On("GetStorage", mock.AnythingOfType("*common.Hash"), mock.AnythingOfType("[]uint8")).Return([]byte{21}, nil)

	mockStorageAPIGetKeysErr := new(apimocks.StorageAPI)
	mockStorageAPIGetKeysErr.On("GetStateRootFromBlock", mock.AnythingOfType("*common.Hash")).Return(&hash, nil)
	mockStorageAPIGetKeysErr.On("Entries", mock.AnythingOfType("*common.Hash")).Return(m, nil)
	mockStorageAPIGetKeysErr.On("GetKeysWithPrefix", mock.AnythingOfType("*common.Hash"), mock.AnythingOfType("[]uint8")).Return(nil, errors.New("GetKeysWithPrefix Err"))

	mockStorageAPIEntriesErr := new(apimocks.StorageAPI)
	mockStorageAPIEntriesErr.On("GetStateRootFromBlock", mock.AnythingOfType("*common.Hash")).Return(&hash, nil)
	mockStorageAPIEntriesErr.On("Entries", mock.AnythingOfType("*common.Hash")).Return(nil, errors.New("entries Err"))

	mockStorageAPIErr := new(apimocks.StorageAPI)
	mockStorageAPIErr.On("GetStateRootFromBlock", mock.AnythingOfType("*common.Hash")).Return(nil, errors.New("GetStateRootFromBlock Err"))

	mockStorageAPIGetStorageErr := new(apimocks.StorageAPI)
	mockStorageAPIGetStorageErr.On("GetStateRootFromBlock", mock.AnythingOfType("*common.Hash")).Return(&hash, nil)
	mockStorageAPIGetStorageErr.On("Entries", mock.AnythingOfType("*common.Hash")).Return(m, nil)
	mockStorageAPIGetStorageErr.On("GetKeysWithPrefix", mock.AnythingOfType("*common.Hash"), mock.AnythingOfType("[]uint8")).Return([][]byte{{1}, {2}}, nil)
	mockStorageAPIGetStorageErr.On("GetStorage", mock.AnythingOfType("*common.Hash"), mock.AnythingOfType("[]uint8")).Return(nil, errors.New("GetStorage Err"))

	str := "jimbo"
	var res StatePairResponse
	type fields struct {
		networkAPI NetworkAPI
		storageAPI StorageAPI
		coreAPI    CoreAPI
	}
	type args struct {
		in0 *http.Request
		req *StatePairRequest
		res *StatePairResponse
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name:   "GetStateRootFromBlock Error",
			fields: fields{nil, mockStorageAPIErr, nil},
			args: args{
				req: &StatePairRequest{
					Bhash:  &hash,
				},
				res: &res,
			},
			wantErr: true,
		},
		{
			name:   "Nil Prefix OK",
			fields: fields{nil, mockStorageAPI, nil},
			args: args{
				req: &StatePairRequest{
					Bhash:  &hash,
				},
				res: &res,
			},
		},
		{
			name:   "Nil Prefix Err",
			fields: fields{nil, mockStorageAPIEntriesErr, nil},
			args: args{
				req: &StatePairRequest{
					Bhash:  &hash,
				},
				res: &res,
			},
			wantErr: true,
		},
		{
			name:   "OK Case",
			fields: fields{nil, mockStorageAPI, nil},
			args: args{
				req: &StatePairRequest{
					Prefix: &str,
					Bhash:  &hash,
				},
				res: &res,
			},
		},
		{
			name:   "GetKeysWithPrefix Error",
			fields: fields{nil, mockStorageAPIGetKeysErr, nil},
			args: args{
				req: &StatePairRequest{
					Prefix: &str,
					Bhash:  &hash,
				},
				res: &res,
			},
			wantErr: true,
		},
		{
			name:   "GetStorage Error",
			fields: fields{nil, mockStorageAPIGetStorageErr, nil},
			args: args{
				req: &StatePairRequest{
					Prefix: &str,
					Bhash:  &hash,
				},
				res: &res,
			},
			wantErr: true,
		},
		{
			name:   "GetKeysWithPrefix Empty",
			fields: fields{nil, mockStorageAPIGetKeysEmpty, nil},
			args: args{
				req: &StatePairRequest{
					Prefix: &str,
					Bhash:  &hash,
				},
				res: &res,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sm := &StateModule{
				networkAPI: tt.fields.networkAPI,
				storageAPI: tt.fields.storageAPI,
				coreAPI:    tt.fields.coreAPI,
			}
			if err := sm.GetPairs(tt.args.in0, tt.args.req, tt.args.res); (err != nil) != tt.wantErr {
				t.Errorf("GetPairs() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestStateModuleGetKeysPaged(t *testing.T) {
	mockStorageAPI := new(apimocks.StorageAPI)
	mockStorageAPI.On("GetKeysWithPrefix", mock.AnythingOfType("*common.Hash"), mock.AnythingOfType("[]uint8")).Return([][]byte{{1}, {2}}, nil)

	mockStorageAPI2 := new(apimocks.StorageAPI)
	mockStorageAPI2.On("GetKeysWithPrefix", mock.AnythingOfType("*common.Hash"), mock.AnythingOfType("[]uint8")).Return([][]byte{{1, 1, 1}, {1, 1, 1}}, nil)

	mockStorageAPIErr := new(apimocks.StorageAPI)
	mockStorageAPIErr.On("GetKeysWithPrefix", mock.AnythingOfType("*common.Hash"), mock.AnythingOfType("[]uint8")).Return(nil, errors.New("GetKeysWithPrefix Err"))

	var res StateStorageKeysResponse
	type fields struct {
		networkAPI NetworkAPI
		storageAPI StorageAPI
		coreAPI    CoreAPI
	}
	type args struct {
		in0 *http.Request
		req *StateStorageKeyRequest
		res *StateStorageKeysResponse
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name:   "OK",
			fields: fields{nil, mockStorageAPI, nil},
			args: args{
				req: &StateStorageKeyRequest{
					AfterKey: "0x01",
				},
				res: &res,
			},
		},
		{
			name:   "ResCount break",
			fields: fields{nil, mockStorageAPI2, nil},
			args: args{
				req: &StateStorageKeyRequest{
					Qty:      1,
					AfterKey: "0x01",
				},
				res: &res,
			},
		},
		{
			name:   "GetKeysWithPrefix Error",
			fields: fields{nil, mockStorageAPIErr, nil},
			args: args{
				req: &StateStorageKeyRequest{
					AfterKey: "0x01",
				},
				res: &res,
			},
			wantErr: true,
		},
		{
			name:   "Request Prefix Error",
			fields: fields{nil, mockStorageAPI, nil},
			args: args{
				req: &StateStorageKeyRequest{
					Prefix:   "a",
					AfterKey: "0x01",
				},
				res: &res,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sm := &StateModule{
				networkAPI: tt.fields.networkAPI,
				storageAPI: tt.fields.storageAPI,
				coreAPI:    tt.fields.coreAPI,
			}
			if err := sm.GetKeysPaged(tt.args.in0, tt.args.req, tt.args.res); (err != nil) != tt.wantErr {
				t.Errorf("GetKeysPaged() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// Implement Tests once function is implemented
func TestCall(t *testing.T) {
	mockNetworkAPI := new(apimocks.NetworkAPI)
	mockStorageAPI := new(apimocks.StorageAPI)
	sm := NewStateModule(mockNetworkAPI, mockStorageAPI, nil)

	err := sm.Call(nil, nil, nil)
	require.NoError(t, err)
}

func TestStateModuleGetMetadata(t *testing.T) {
	hash := common.MustHexToHash("0x3aa96b0149b6ca3688878bdbd19464448624136398e3ce45b9e755d3ab61355a")

	mockCoreAPI := new(apimocks.CoreAPI)
	mockCoreAPI.On("GetMetadata", mock.AnythingOfType("*common.Hash")).Return(common.MustHexToBytes(testdata.TestData), nil)

	mockCoreAPIErr := new(apimocks.CoreAPI)
	mockCoreAPIErr.On("GetMetadata", mock.AnythingOfType("*common.Hash")).Return(nil, errors.New("GetMetadata Error"))

	mockStateModule := NewStateModule(nil, nil, mockCoreAPIErr)
	var res StateMetadataResponse
	type fields struct {
		networkAPI NetworkAPI
		storageAPI StorageAPI
		coreAPI    CoreAPI
	}
	type args struct {
		in0 *http.Request
		req *StateRuntimeMetadataQuery
		res *StateMetadataResponse
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name:   "OK Case",
			fields: fields{nil, nil, mockCoreAPI},
			args: args{
				req: &StateRuntimeMetadataQuery{Bhash: &hash},
				res: &res,
			},
		},
		{
			name:   "GetMetadata Error",
			fields: fields{nil, nil, mockStateModule.coreAPI},
			args: args{
				req: &StateRuntimeMetadataQuery{Bhash: &hash},
				res: &res,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sm := &StateModule{
				networkAPI: tt.fields.networkAPI,
				storageAPI: tt.fields.storageAPI,
				coreAPI:    tt.fields.coreAPI,
			}
			if err := sm.GetMetadata(tt.args.in0, tt.args.req, tt.args.res); (err != nil) != tt.wantErr {
				t.Errorf("GetMetadata() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestStateModuleGetReadProof(t *testing.T) {
	hash := common.MustHexToHash("0x3aa96b0149b6ca3688878bdbd19464448624136398e3ce45b9e755d3ab61355a")

	mockCoreAPI := new(apimocks.CoreAPI)
	mockCoreAPI.On("GetReadProofAt", mock.AnythingOfType("common.Hash"), mock.AnythingOfType("[][]uint8")).Return(hash, [][]byte{{1, 1, 1}, {1, 1, 1}}, nil)

	mockCoreAPIErr := new(apimocks.CoreAPI)
	mockCoreAPIErr.On("GetReadProofAt", mock.AnythingOfType("common.Hash"), mock.AnythingOfType("[][]uint8")).Return(nil, nil, errors.New("GetReadProofAt Error"))

	var res StateGetReadProofResponse
	type fields struct {
		networkAPI NetworkAPI
		storageAPI StorageAPI
		coreAPI    CoreAPI
	}
	type args struct {
		in0 *http.Request
		req *StateGetReadProofRequest
		res *StateGetReadProofResponse
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name:   "OK Case",
			fields: fields{nil, nil, mockCoreAPI},
			args: args{
				req: &StateGetReadProofRequest{
					Keys: []string{"0x1111", "0x2222"},
					Hash: hash,
				},
				res: &res,
			},
		},
		{
			name:   "GetReadProofAt Error",
			fields: fields{nil, nil, mockCoreAPIErr},
			args: args{
				req: &StateGetReadProofRequest{
					Keys: []string{"0x1111", "0x2222"},
					Hash: hash,
				},
				res: &res,
			},
			wantErr: true,
		},
		{
			name:   "InvalidKeys Error",
			fields: fields{nil, nil, mockCoreAPIErr},
			args: args{
				req: &StateGetReadProofRequest{
					Keys: []string{"jimbo", "test"},
					Hash: hash,
				},
				res: &res,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sm := &StateModule{
				networkAPI: tt.fields.networkAPI,
				storageAPI: tt.fields.storageAPI,
				coreAPI:    tt.fields.coreAPI,
			}
			if err := sm.GetReadProof(tt.args.in0, tt.args.req, tt.args.res); (err != nil) != tt.wantErr {
				t.Errorf("GetReadProof() error = %v, wantErr %v", err, tt.wantErr)
			}
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
	mockCoreAPI.On("GetRuntimeVersion", mock.AnythingOfType("*common.Hash")).Return(version, nil)

	mockCoreAPIErr := new(apimocks.CoreAPI)
	mockCoreAPIErr.On("GetRuntimeVersion", mock.AnythingOfType("*common.Hash")).Return(nil, errors.New("GetRuntimeVersion Error"))

	var res StateRuntimeVersionResponse
	type fields struct {
		networkAPI NetworkAPI
		storageAPI StorageAPI
		coreAPI    CoreAPI
	}
	type args struct {
		in0 *http.Request
		req *StateRuntimeVersionRequest
		res *StateRuntimeVersionResponse
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name:   "OK Case",
			fields: fields{nil, nil, mockCoreAPI},
			args: args{
				req: &StateRuntimeVersionRequest{&hash},
				res: &res,
			},
		},
		{
			name:   "GetRuntimeVersion Error",
			fields: fields{nil, nil, mockCoreAPIErr},
			args: args{
				req: &StateRuntimeVersionRequest{&hash},
				res: &res,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sm := &StateModule{
				networkAPI: tt.fields.networkAPI,
				storageAPI: tt.fields.storageAPI,
				coreAPI:    tt.fields.coreAPI,
			}
			if err := sm.GetRuntimeVersion(tt.args.in0, tt.args.req, tt.args.res); (err != nil) != tt.wantErr {
				t.Errorf("GetRuntimeVersion() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestStateModuleGetStorage(t *testing.T) {
	hash := common.MustHexToHash("0x3aa96b0149b6ca3688878bdbd19464448624136398e3ce45b9e755d3ab61355a")

	mockStorageAPI := new(apimocks.StorageAPI)
	mockStorageAPI.On("GetStorageByBlockHash", mock.AnythingOfType("*common.Hash"), mock.AnythingOfType("[]uint8")).Return([]byte{21}, nil)
	mockStorageAPI.On("GetStorage", mock.AnythingOfType("*common.Hash"), mock.AnythingOfType("[]uint8")).Return([]byte{21}, nil)

	mockStorageAPIErr := new(apimocks.StorageAPI)
	mockStorageAPIErr.On("GetStorageByBlockHash", mock.AnythingOfType("*common.Hash"), mock.AnythingOfType("[]uint8")).Return(nil, errors.New("GetStorageByBlockHash Error"))
	mockStorageAPIErr.On("GetStorage", mock.AnythingOfType("*common.Hash"), mock.AnythingOfType("[]uint8")).Return(nil, errors.New("GetStorage Error"))

	var res StateStorageResponse
	type fields struct {
		networkAPI NetworkAPI
		storageAPI StorageAPI
		coreAPI    CoreAPI
	}
	type args struct {
		in0 *http.Request
		req *StateStorageRequest
		res *StateStorageResponse
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name:   "bHash Not Nil OK",
			fields: fields{nil, mockStorageAPI, nil},
			args: args{
				req: &StateStorageRequest{
					Key:   "0x3aa96b0149b6ca3688878bdbd19464448624136398e3ce45b9e755d3ab61355a",
					Bhash: &hash,
				},
				res: &res,
			},
		},
		{
			name:   "bHash Nil OK",
			fields: fields{nil, mockStorageAPI, nil},
			args: args{
				req: &StateStorageRequest{
					Key:   "0x3aa96b0149b6ca3688878bdbd19464448624136398e3ce45b9e755d3ab61355a",
				},
				res: &res,
			},
		},
		{
			name:   "bHash Not Nil Err",
			fields: fields{nil, mockStorageAPIErr, nil},
			args: args{
				req: &StateStorageRequest{
					Key:   "0x3aa96b0149b6ca3688878bdbd19464448624136398e3ce45b9e755d3ab61355a",
					Bhash: &hash,
				},
				res: &res,
			},
			wantErr: true,
		},
		{
			name:   "bHash Nil Err",
			fields: fields{nil, mockStorageAPIErr, nil},
			args: args{
				req: &StateStorageRequest{
					Key:   "0x3aa96b0149b6ca3688878bdbd19464448624136398e3ce45b9e755d3ab61355a",
				},
				res: &res,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sm := &StateModule{
				networkAPI: tt.fields.networkAPI,
				storageAPI: tt.fields.storageAPI,
				coreAPI:    tt.fields.coreAPI,
			}
			if err := sm.GetStorage(tt.args.in0, tt.args.req, tt.args.res); (err != nil) != tt.wantErr {
				t.Errorf("GetStorage() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestStateModuleGetStorageHash(t *testing.T) {
	hash := common.MustHexToHash("0x3aa96b0149b6ca3688878bdbd19464448624136398e3ce45b9e755d3ab61355a")

	mockStorageAPI := new(apimocks.StorageAPI)
	mockStorageAPI.On("GetStorageByBlockHash", mock.AnythingOfType("*common.Hash"), mock.AnythingOfType("[]uint8")).Return([]byte{21}, nil)
	mockStorageAPI.On("GetStorage", mock.AnythingOfType("*common.Hash"), mock.AnythingOfType("[]uint8")).Return([]byte{21}, nil)

	mockStorageAPIErr := new(apimocks.StorageAPI)
	mockStorageAPIErr.On("GetStorageByBlockHash", mock.AnythingOfType("*common.Hash"), mock.AnythingOfType("[]uint8")).Return(nil, errors.New("GetStorageByBlockHash Error"))
	mockStorageAPIErr.On("GetStorage", mock.AnythingOfType("*common.Hash"), mock.AnythingOfType("[]uint8")).Return(nil, errors.New("GetStorage Error"))

	var res StateStorageHashResponse
	type fields struct {
		networkAPI NetworkAPI
		storageAPI StorageAPI
		coreAPI    CoreAPI
	}
	type args struct {
		in0 *http.Request
		req *StateStorageHashRequest
		res *StateStorageHashResponse
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name:   "bHash Not Nil OK",
			fields: fields{nil, mockStorageAPI, nil},
			args: args{
				req: &StateStorageHashRequest{
					Key:   "0x3aa96b0149b6ca3688878bdbd19464448624136398e3ce45b9e755d3ab61355a",
					Bhash: &hash,
				},
				res: &res,
			},
		},
		{
			name:   "bHash Nil OK",
			fields: fields{nil, mockStorageAPI, nil},
			args: args{
				req: &StateStorageHashRequest{
					Key:   "0x3aa96b0149b6ca3688878bdbd19464448624136398e3ce45b9e755d3ab61355a",
				},
				res: &res,
			},
		},
		{
			name:   "bHash Not Nil Err",
			fields: fields{nil, mockStorageAPIErr, nil},
			args: args{
				req: &StateStorageHashRequest{
					Key:   "0x3aa96b0149b6ca3688878bdbd19464448624136398e3ce45b9e755d3ab61355a",
					Bhash: &hash,
				},
				res: &res,
			},
			wantErr: true,
		},
		{
			name:   "bHash Nil Err",
			fields: fields{nil, mockStorageAPIErr, nil},
			args: args{
				req: &StateStorageHashRequest{
					Key:   "0x3aa96b0149b6ca3688878bdbd19464448624136398e3ce45b9e755d3ab61355a",
				},
				res: &res,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sm := &StateModule{
				networkAPI: tt.fields.networkAPI,
				storageAPI: tt.fields.storageAPI,
				coreAPI:    tt.fields.coreAPI,
			}
			if err := sm.GetStorageHash(tt.args.in0, tt.args.req, tt.args.res); (err != nil) != tt.wantErr {
				t.Errorf("GetStorageHash() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestStateModuleGetStorageSize(t *testing.T) {
	hash := common.MustHexToHash("0x3aa96b0149b6ca3688878bdbd19464448624136398e3ce45b9e755d3ab61355a")

	mockStorageAPI := new(apimocks.StorageAPI)
	mockStorageAPI.On("GetStorageByBlockHash", mock.AnythingOfType("*common.Hash"), mock.AnythingOfType("[]uint8")).Return([]byte{21}, nil)
	mockStorageAPI.On("GetStorage", mock.AnythingOfType("*common.Hash"), mock.AnythingOfType("[]uint8")).Return([]byte{21}, nil)

	mockStorageAPIErr := new(apimocks.StorageAPI)
	mockStorageAPIErr.On("GetStorageByBlockHash", mock.AnythingOfType("*common.Hash"), mock.AnythingOfType("[]uint8")).Return(nil, errors.New("GetStorageByBlockHash Error"))
	mockStorageAPIErr.On("GetStorage", mock.AnythingOfType("*common.Hash"), mock.AnythingOfType("[]uint8")).Return(nil, errors.New("GetStorage Error"))

	var res StateStorageSizeResponse
	type fields struct {
		networkAPI NetworkAPI
		storageAPI StorageAPI
		coreAPI    CoreAPI
	}
	type args struct {
		in0 *http.Request
		req *StateStorageSizeRequest
		res *StateStorageSizeResponse
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name:   "bHash Not Nil OK",
			fields: fields{nil, mockStorageAPI, nil},
			args: args{
				req: &StateStorageSizeRequest{
					Key:   "0x3aa96b0149b6ca3688878bdbd19464448624136398e3ce45b9e755d3ab61355a",
					Bhash: &hash,
				},
				res: &res,
			},
		},
		{
			name:   "bHash Nil OK",
			fields: fields{nil, mockStorageAPI, nil},
			args: args{
				req: &StateStorageSizeRequest{
					Key:   "0x3aa96b0149b6ca3688878bdbd19464448624136398e3ce45b9e755d3ab61355a",
				},
				res: &res,
			},
		},
		{
			name:   "bHash Not Nil Err",
			fields: fields{nil, mockStorageAPIErr, nil},
			args: args{
				req: &StateStorageSizeRequest{
					Key:   "0x3aa96b0149b6ca3688878bdbd19464448624136398e3ce45b9e755d3ab61355a",
					Bhash: &hash,
				},
				res: &res,
			},
			wantErr: true,
		},
		{
			name:   "bHash Nil Err",
			fields: fields{nil, mockStorageAPIErr, nil},
			args: args{
				req: &StateStorageSizeRequest{
					Key:   "0x3aa96b0149b6ca3688878bdbd19464448624136398e3ce45b9e755d3ab61355a",
				},
				res: &res,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sm := &StateModule{
				networkAPI: tt.fields.networkAPI,
				storageAPI: tt.fields.storageAPI,
				coreAPI:    tt.fields.coreAPI,
			}
			if err := sm.GetStorageSize(tt.args.in0, tt.args.req, tt.args.res); (err != nil) != tt.wantErr {
				t.Errorf("GetStorageSize() error = %v, wantErr %v", err, tt.wantErr)
			}
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
	mockCoreAPI.On("QueryStorage", mock.AnythingOfType("common.Hash"), mock.AnythingOfType("common.Hash"), mock.AnythingOfType("string")).Return(m, nil)

	mockCoreAPIErr := new(apimocks.CoreAPI)
	mockCoreAPIErr.On("QueryStorage", mock.AnythingOfType("common.Hash"), mock.AnythingOfType("common.Hash"), mock.AnythingOfType("string")).Return(nil, errors.New("QueryStorage Error"))

	var res []StorageChangeSetResponse
	type fields struct {
		networkAPI NetworkAPI
		storageAPI StorageAPI
		coreAPI    CoreAPI
	}
	type args struct {
		in0 *http.Request
		req *StateStorageQueryRangeRequest
		res *[]StorageChangeSetResponse
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
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
				res: &res,
			},
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
				res: &res,
			},
			wantErr: true,
		},
		{
			name:   "Empty Start Block Error",
			fields: fields{nil, nil, mockCoreAPI},
			args: args{
				req: &StateStorageQueryRangeRequest{
					Keys:       []string{"jimbo"},
					EndBlock:   hash2,
				},
				res: &res,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sm := &StateModule{
				networkAPI: tt.fields.networkAPI,
				storageAPI: tt.fields.storageAPI,
				coreAPI:    tt.fields.coreAPI,
			}
			if err := sm.QueryStorage(tt.args.in0, tt.args.req, tt.args.res); (err != nil) != tt.wantErr {
				t.Errorf("QueryStorage() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
