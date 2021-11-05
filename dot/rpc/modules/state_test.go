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
	apimocks "github.com/ChainSafe/gossamer/dot/rpc/modules/mocks"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/stretchr/testify/mock"
	"net/http"
	"testing"
)

func TestStateModule_GetPairs(t *testing.T) {
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
			name: "GetStateRootFromBlock Error",
			fields: fields{nil, mockStorageAPIErr, nil},
			args: args{
				in0: nil,
				req: &StatePairRequest{
					Prefix: nil,
					Bhash:  &hash,
				},
				res: &res,
			},
			wantErr: true,
		},
		{
			name: "Nil Prefix OK",
			fields: fields{nil, mockStorageAPI, nil},
			args: args{
				in0: nil,
				req: &StatePairRequest{
					Prefix: nil,
					Bhash:  &hash,
				},
				res: &res,
			},
			wantErr: false,
		},
		{
			name: "Nil Prefix Err",
			fields: fields{nil, mockStorageAPIEntriesErr, nil},
			args: args{
				in0: nil,
				req: &StatePairRequest{
					Prefix: nil,
					Bhash:  &hash,
				},
				res: &res,
			},
			wantErr: true,
		},
		{
			name: "OK Case",
			fields: fields{nil, mockStorageAPI, nil},
			args: args{
				in0: nil,
				req: &StatePairRequest{
					Prefix: &str,
					Bhash:  &hash,
				},
				res: &res,
			},
			wantErr: false,
		},
		{
			name: "GetKeysWithPrefix Error",
			fields: fields{nil, mockStorageAPIGetKeysErr, nil},
			args: args{
				in0: nil,
				req: &StatePairRequest{
					Prefix: &str,
					Bhash:  &hash,
				},
				res: &res,
			},
			wantErr: true,
		},
		{
			name: "GetStorage Error",
			fields: fields{nil, mockStorageAPIGetStorageErr, nil},
			args: args{
				in0: nil,
				req: &StatePairRequest{
					Prefix: &str,
					Bhash:  &hash,
				},
				res: &res,
			},
			wantErr: true,
		},
		{
			name: "GetKeysWithPrefix Empty",
			fields: fields{nil, mockStorageAPIGetKeysEmpty, nil},
			args: args{
				in0: nil,
				req: &StatePairRequest{
					Prefix: &str,
					Bhash:  &hash,
				},
				res: &res,
			},
			wantErr: false,
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