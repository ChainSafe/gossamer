package modules

import (
	"errors"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/stretchr/testify/assert"
	"net/http"
	"testing"

	apimocks "github.com/ChainSafe/gossamer/dot/rpc/modules/mocks"
	"github.com/ChainSafe/gossamer/lib/genesis"
)

func TestSyncStateModule_GenSyncSpec(t *testing.T) {
	g := new(genesis.Genesis)
	mockSyncStateAPI := new(apimocks.SyncStateAPI)
	mockSyncStateAPI.On("GenSyncSpec", true).Return(g, nil)

	mockSyncStateAPIErr := new(apimocks.SyncStateAPI)
	mockSyncStateAPIErr.On("GenSyncSpec", true).Return(nil, errors.New("GenSyncSpec error"))

	syncStateModule := NewSyncStateModule(mockSyncStateAPI)
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
		err     error
		exp     genesis.Genesis
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
			},
			exp: genesis.Genesis{},
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
			},
			wantErr: true,
			err:     errors.New("GenSyncSpec error"),
		},
	}
	for _, tt := range tests {
		var res genesis.Genesis
		tt.args.res = &res
		t.Run(tt.name, func(t *testing.T) {
			ss := &SyncStateModule{
				syncStateAPI: tt.fields.syncStateAPI,
			}
			var err error
			if err = ss.GenSyncSpec(tt.args.in0, tt.args.req, tt.args.res); (err != nil) != tt.wantErr {
				t.Errorf("GenSyncSpec() error = %v, wantErr %v", err, tt.wantErr)
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

func TestNewStateSync(t *testing.T) {
	g := &genesis.Genesis{}
	raw := make(map[string][]byte)
	mockStorageAPI := new(apimocks.StorageAPI)
	mockStorageAPI.On("Entries", (*common.Hash)(nil)).Return(raw, nil)

	mockStorageAPIErr := new(apimocks.StorageAPI)
	mockStorageAPIErr.On("Entries", (*common.Hash)(nil)).Return(nil, errors.New("entries error"))

	type args struct {
		gData      *genesis.Data
		storageAPI StorageAPI
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
		err     error
		exp     SyncStateAPI
	}{
		{
			name: "OK Case",
			args: args{
				gData:      g.GenesisData(),
				storageAPI: mockStorageAPI,
			},
			exp: syncState{chainSpecification: &genesis.Genesis{
				Name:       "",
				ID:         "",
				Bootnodes:  []string{},
				ProtocolID: "",
				Genesis: genesis.Fields{
					Raw:     map[string]map[string]string{},
					Runtime: map[string]map[string]interface{}{},
				},
			},
			},
		},
		{
			name: "Err Case",
			args: args{
				gData:      g.GenesisData(),
				storageAPI: mockStorageAPIErr,
			},
			wantErr: true,
			err:     errors.New("entries error"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := NewStateSync(tt.args.gData, tt.args.storageAPI)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewStateSync() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				assert.EqualError(t, err, tt.err.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.exp, res)
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
		err     error
		exp     *genesis.Genesis
	}{
		{
			name:   "GenSyncSpec False",
			fields: fields{g},
			exp:    &genesis.Genesis{},
		},
		{
			name:   "GenSyncSpec True",
			fields: fields{g},
			args: args{
				raw: true,
			},
			exp: &genesis.Genesis{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := syncState{
				chainSpecification: tt.fields.chainSpecification,
			}
			res, err := s.GenSyncSpec(tt.args.raw)
			if (err != nil) != tt.wantErr {
				t.Errorf("GenSyncSpec() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				assert.EqualError(t, err, tt.err.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, *tt.exp, *res)
			}
		})
	}
}
