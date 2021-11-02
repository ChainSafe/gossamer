package modules

import (
	"errors"
	"github.com/ChainSafe/gossamer/lib/genesis"
	"github.com/stretchr/testify/mock"
	"net/http"
	"testing"

	apimocks "github.com/ChainSafe/gossamer/dot/rpc/modules/mocks"
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
				in0: nil,
				req: &GenSyncSpecRequest{
					Raw: true,
				},
				res: &res,
			},
			wantErr: false,
		},
		{
			name: "GenSyncSpec Err",
			fields: fields{
				mockSyncStateAPIErr,
			},
			args: args{
				in0: nil,
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