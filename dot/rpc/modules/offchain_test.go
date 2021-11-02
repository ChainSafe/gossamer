package modules

import (
	"errors"
	"github.com/stretchr/testify/mock"
	"net/http"
	"testing"

	apimocks "github.com/ChainSafe/gossamer/dot/rpc/modules/mocks"
)

func TestOffchainModule_LocalStorageGet(t *testing.T) {
	mockRuntimeStorageAPI := new(apimocks.RuntimeStorageAPI)
	mockRuntimeStorageAPI.On("GetPersistent", mock.AnythingOfType("[]uint8")).Return(nil, errors.New("GetPersistent error"))
	mockRuntimeStorageAPI.On("GetLocal", mock.AnythingOfType("[]uint8")).Return([]byte("some-value"), nil)

	offChainModule := NewOffchainModule(mockRuntimeStorageAPI)

	var res StringResponse
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
	}{
		{
			name: "GetPersistent error",
			fields: fields{
				offChainModule.nodeStorage,
			},
			args: args{
				in0: nil,
				req: &OffchainLocalStorageGet{
					Kind: offchainPersistent,
					Key:  "0x11111111111111",
				},
				res: &res,
			},
			wantErr: true,
		},
		{
			name: "Invalid Storage Kind",
			fields: fields{
				offChainModule.nodeStorage,
			},
			args: args{
				in0: nil,
				req: &OffchainLocalStorageGet{
					Kind: "invalid kind",
					Key:  "0x11111111111111",
				},
				res: &res,
			},
			wantErr: true,
		},
		{
			name: "GetLocal OK",
			fields: fields{
				offChainModule.nodeStorage,
			},
			args: args{
				in0: nil,
				req: &OffchainLocalStorageGet{
					Kind: offchainLocal,
					Key:  "0x11111111111111",
				},
				res: &res,
			},
			wantErr: false,
		},
		{
			name: "Invalid key",
			fields: fields{
				offChainModule.nodeStorage,
			},
			args: args{
				in0: nil,
				req: &OffchainLocalStorageGet{
					Kind: offchainLocal,
					Key:  "0x1",
				},
				res: &res,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &OffchainModule{
				nodeStorage: tt.fields.nodeStorage,
			}
			if err := s.LocalStorageGet(tt.args.in0, tt.args.req, tt.args.res); (err != nil) != tt.wantErr {
				t.Errorf("LocalStorageGet() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestOffchainModule_LocalStorageSet(t *testing.T) {
	mockRuntimeStorageAPI := new(apimocks.RuntimeStorageAPI)
	mockRuntimeStorageAPI.On("SetLocal", mock.AnythingOfType("[]uint8"), mock.AnythingOfType("[]uint8")).Return( nil)
	mockRuntimeStorageAPI.On("SetPersistent", mock.AnythingOfType("[]uint8"), mock.AnythingOfType("[]uint8")).Return( errors.New("SetPersistent error"))


	var res StringResponse
	type fields struct {
		nodeStorage RuntimeStorageAPI
	}
	type args struct {
		in0 *http.Request
		req *OffchainLocalStorageSet
		in2 *StringResponse
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "setLocal OK",
			fields: fields{
				mockRuntimeStorageAPI,
			},
			args: args{
				in0: nil,
				req: &OffchainLocalStorageSet{
					Kind:  offchainLocal,
					Key:   "0x11111111111111",
					Value: "0x22222222222222",
				},
				in2: &res,
			},
			wantErr: false,
		},
		{
			name: "Invalid Key",
			fields: fields{
				mockRuntimeStorageAPI,
			},
			args: args{
				in0: nil,
				req: &OffchainLocalStorageSet{
					Kind:  offchainLocal,
					Key:   "0x1",
					Value: "0x22222222222222",
				},
				in2: &res,
			},
			wantErr: true,
		},
		{
			name: "Invalid Value",
			fields: fields{
				mockRuntimeStorageAPI,
			},
			args: args{
				in0: nil,
				req: &OffchainLocalStorageSet{
					Kind:  offchainLocal,
					Key:   "0x11111111111111",
					Value: "0x2",
				},
				in2: &res,
			},
			wantErr: true,
		},
		{
			name: "setPersistentError",
			fields: fields{
				mockRuntimeStorageAPI,
			},
			args: args{
				in0: nil,
				req: &OffchainLocalStorageSet{
					Kind:  offchainPersistent,
					Key:   "0x11111111111111",
					Value: "0x22222222222222",
				},
				in2: &res,
			},
			wantErr: true,
		},
		{
			name: "Invalid Kind",
			fields: fields{
				mockRuntimeStorageAPI,
			},
			args: args{
				in0: nil,
				req: &OffchainLocalStorageSet{
					Kind:  "bad kind",
					Key:   "0x11111111111111",
					Value: "0x22222222222222",
				},
				in2: &res,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &OffchainModule{
				nodeStorage: tt.fields.nodeStorage,
			}
			if err := s.LocalStorageSet(tt.args.in0, tt.args.req, tt.args.in2); (err != nil) != tt.wantErr {
				t.Errorf("LocalStorageSet() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}