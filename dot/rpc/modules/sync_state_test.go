package modules

import (
	"errors"
	"net/http"
	"testing"

	apimocks "github.com/ChainSafe/gossamer/dot/rpc/modules/mocks"
	"github.com/ChainSafe/gossamer/lib/genesis"
	"github.com/stretchr/testify/mock"
)

func TestSyncStateModule_GenSyncSpec(t *testing.T) {
	g := new(genesis.Genesis)
	mockSyncStateAPI := new(apimocks.SyncStateAPI)
	mockSyncStateAPI.On("GenSyncSpec", mock.AnythingOfType("bool")).Return(g, nil)

	mockSyncStateAPIErr := new(apimocks.SyncStateAPI)
	mockSyncStateAPIErr.On("GenSyncSpec", mock.AnythingOfType("bool")).Return(nil, errors.New("GenSyncSpec error"))

	syncStateModule := NewSyncStateModule(mockSyncStateAPI)
	var res genesis.Genesis
	type fields struct {
		syncStateAPI SyncStateAPI
	}
	type args struct {
		in0 *http.Request
		req *GenSyncSpecRequest
		res *genesis.Genesis
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "GenSyncSpec OK",
			fields: fields{
				syncStateModule.syncStateAPI,
			},
			args: args{
				req: &GenSyncSpecRequest{
					Raw: true,
				},
				res: &res,
			},
		},
		{
			name: "GenSyncSpec Err",
			fields: fields{
				mockSyncStateAPIErr,
			},
			args: args{
				req: &GenSyncSpecRequest{
					Raw: true,
				},
				res: &res,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ss := &SyncStateModule{
				syncStateAPI: tt.fields.syncStateAPI,
			}
			if err := ss.GenSyncSpec(tt.args.in0, tt.args.req, tt.args.res); (err != nil) != tt.wantErr {
				t.Errorf("GenSyncSpec() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNewStateSync(t *testing.T) {
	g := &genesis.Genesis{}
	raw := make(map[string][]byte)
	mockStorageAPI := new(apimocks.StorageAPI)
	mockStorageAPI.On("Entries", mock.AnythingOfType("*common.Hash")).Return(raw, nil)

	mockStorageAPIErr := new(apimocks.StorageAPI)
	mockStorageAPIErr.On("Entries", mock.AnythingOfType("*common.Hash")).Return(nil, errors.New("entries error"))

	type args struct {
		gData      *genesis.Data
		storageAPI StorageAPI
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "OK Case",
			args: args{
				gData:      g.GenesisData(),
				storageAPI: mockStorageAPI,
			},
		},
		{
			name: "Err Case",
			args: args{
				gData:      g.GenesisData(),
				storageAPI: mockStorageAPIErr,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewStateSync(tt.args.gData, tt.args.storageAPI)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewStateSync() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func Test_syncState_GenSyncSpec(t *testing.T) {
	g := &genesis.Genesis{}
	type fields struct {
		chainSpecification *genesis.Genesis
	}
	type args struct {
		raw bool
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name:   "GenSyncSpec False",
			fields: fields{g},
		},
		{
			name:   "GenSyncSpec True",
			fields: fields{g},
			args: args{
				raw: true,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := syncState{
				chainSpecification: tt.fields.chainSpecification,
			}
			_, err := s.GenSyncSpec(tt.args.raw)
			if (err != nil) != tt.wantErr {
				t.Errorf("GenSyncSpec() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}
