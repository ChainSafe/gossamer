// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package modules

import (
	"errors"
	"fmt"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/stretchr/testify/assert"
	"net/http"
	"testing"

	apimocks "github.com/ChainSafe/gossamer/dot/rpc/modules/mocks"
)

func TestOffchainModule_LocalStorageGet(t *testing.T) {
	mockRuntimeStorageAPI := new(apimocks.RuntimeStorageAPI)
	mockRuntimeStorageAPI.On("GetPersistent", common.MustHexToBytes("0x11111111111111")).Return(nil, errors.New("GetPersistent error"))
	mockRuntimeStorageAPI.On("GetLocal", common.MustHexToBytes("0x11111111111111")).Return([]byte("some-value"), nil)
	offChainModule := NewOffchainModule(mockRuntimeStorageAPI)

	type fields struct {
		nodeStorage RuntimeStorageAPI
	}
	type args struct {
		in0 *http.Request
		req *OffchainLocalStorageGet
		res *StringResponse
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
		err     error
		exp     StringResponse
	}{
		{
			name: "GetPersistent error",
			fields: fields{
				offChainModule.nodeStorage,
			},
			args: args{
				req: &OffchainLocalStorageGet{
					Kind: offchainPersistent,
					Key:  "0x11111111111111",
				},
			},
			wantErr: true,
			err:     errors.New("GetPersistent error"),
		},
		{
			name: "Invalid Storage Kind",
			fields: fields{
				offChainModule.nodeStorage,
			},
			args: args{
				req: &OffchainLocalStorageGet{
					Kind: "invalid kind",
					Key:  "0x11111111111111",
				},
			},
			wantErr: true,
			err:     fmt.Errorf("storage kind not found: invalid kind"),
		},
		{
			name: "GetLocal OK",
			fields: fields{
				offChainModule.nodeStorage,
			},
			args: args{
				req: &OffchainLocalStorageGet{
					Kind: offchainLocal,
					Key:  "0x11111111111111",
				},
			},
			exp: StringResponse("0x736f6d652d76616c7565"),
		},
		{
			name: "Invalid key",
			fields: fields{
				offChainModule.nodeStorage,
			},
			args: args{
				req: &OffchainLocalStorageGet{
					Kind: offchainLocal,
					Key:  "0x1",
				},
			},
			wantErr: true,
			err:     errors.New("cannot decode an odd length string"),
		},
	}
	for _, tt := range tests {
		var res StringResponse
		tt.args.res = &res
		t.Run(tt.name, func(t *testing.T) {
			s := &OffchainModule{
				nodeStorage: tt.fields.nodeStorage,
			}
			var err error
			if err = s.LocalStorageGet(tt.args.in0, tt.args.req, tt.args.res); (err != nil) != tt.wantErr {
				t.Errorf("LocalStorageGet() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				assert.EqualError(t, err, tt.err.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.exp, *tt.args.res)
			}
		})
	}
}

func TestOffchainModule_LocalStorageSet(t *testing.T) {
	mockRuntimeStorageAPI := new(apimocks.RuntimeStorageAPI)
	mockRuntimeStorageAPI.On("SetLocal", common.MustHexToBytes("0x11111111111111"), common.MustHexToBytes("0x22222222222222")).Return(nil)
	mockRuntimeStorageAPI.On("SetPersistent", common.MustHexToBytes("0x11111111111111"), common.MustHexToBytes("0x22222222222222")).Return(errors.New("SetPersistent error"))

	type fields struct {
		nodeStorage RuntimeStorageAPI
	}
	type args struct {
		in0 *http.Request
		req *OffchainLocalStorageSet
		res *StringResponse
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
		err     error
	}{
		{
			name: "setLocal OK",
			fields: fields{
				mockRuntimeStorageAPI,
			},
			args: args{
				req: &OffchainLocalStorageSet{
					Kind:  offchainLocal,
					Key:   "0x11111111111111",
					Value: "0x22222222222222",
				},
			},
		},
		{
			name: "Invalid Key",
			fields: fields{
				mockRuntimeStorageAPI,
			},
			args: args{
				req: &OffchainLocalStorageSet{
					Kind:  offchainLocal,
					Key:   "0x1",
					Value: "0x22222222222222",
				},
			},
			wantErr: true,
			err:     errors.New("cannot decode an odd length string"),
		},
		{
			name: "Invalid Value",
			fields: fields{
				mockRuntimeStorageAPI,
			},
			args: args{
				req: &OffchainLocalStorageSet{
					Kind:  offchainLocal,
					Key:   "0x11111111111111",
					Value: "0x2",
				},
			},
			wantErr: true,
			err:     errors.New("cannot decode an odd length string"),
		},
		{
			name: "setPersistentError",
			fields: fields{
				mockRuntimeStorageAPI,
			},
			args: args{
				req: &OffchainLocalStorageSet{
					Kind:  offchainPersistent,
					Key:   "0x11111111111111",
					Value: "0x22222222222222",
				},
			},
			wantErr: true,
			err:     errors.New("SetPersistent error"),
		},
		{
			name: "Invalid Kind",
			fields: fields{
				mockRuntimeStorageAPI,
			},
			args: args{
				req: &OffchainLocalStorageSet{
					Kind:  "bad kind",
					Key:   "0x11111111111111",
					Value: "0x22222222222222",
				},
			},
			wantErr: true,
			err:     fmt.Errorf("storage kind not found: bad kind"),
		},
	}
	for _, tt := range tests {
		var res StringResponse
		tt.args.res = &res
		t.Run(tt.name, func(t *testing.T) {
			s := &OffchainModule{
				nodeStorage: tt.fields.nodeStorage,
			}
			var err error
			if err = s.LocalStorageSet(tt.args.in0, tt.args.req, tt.args.res); (err != nil) != tt.wantErr {
				t.Errorf("LocalStorageSet() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				assert.EqualError(t, err, tt.err.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
