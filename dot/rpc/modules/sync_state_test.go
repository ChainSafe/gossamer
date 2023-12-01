// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package modules

import (
	"errors"
	"net/http"
	"testing"

	"github.com/ChainSafe/gossamer/dot/rpc/modules/mocks"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/genesis"
	"go.uber.org/mock/gomock"

	"github.com/stretchr/testify/assert"
)

func TestSyncStateModule_GenSyncSpec(t *testing.T) {
	ctrl := gomock.NewController(t)

	g := new(genesis.Genesis)
	mockSyncStateAPI := mocks.NewMockSyncStateAPI(ctrl)
	mockSyncStateAPI.EXPECT().GenSyncSpec(true).Return(g, nil)

	mockSyncStateAPIErr := mocks.NewMockSyncStateAPI(ctrl)
	mockSyncStateAPIErr.EXPECT().GenSyncSpec(true).Return(nil, errors.New("GenSyncSpec error"))

	syncStateModule := NewSyncStateModule(mockSyncStateAPI)
	type fields struct {
		syncStateAPI SyncStateAPI
	}
	type args struct {
		in0 *http.Request
		req *GenSyncSpecRequest
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		expErr error
		exp    genesis.Genesis
	}{
		{
			name: "GenSyncSpec_OK",
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
			name: "GenSyncSpec_Err",
			fields: fields{
				mockSyncStateAPIErr,
			},
			args: args{
				req: &GenSyncSpecRequest{
					Raw: true,
				},
			},
			expErr: errors.New("GenSyncSpec error"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ss := &SyncStateModule{
				syncStateAPI: tt.fields.syncStateAPI,
			}
			res := genesis.Genesis{}
			err := ss.GenSyncSpec(tt.args.in0, tt.args.req, &res)
			if tt.expErr != nil {
				assert.EqualError(t, err, tt.expErr.Error())
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.exp, res)
		})
	}
}

func TestNewStateSync(t *testing.T) {
	ctrl := gomock.NewController(t)

	g1 := &genesis.Genesis{}
	g2 := &genesis.Genesis{}
	raw := make(map[string][]byte)
	mockStorageAPI := mocks.NewMockStorageAPI(ctrl)
	mockStorageAPI.EXPECT().Entries((*common.Hash)(nil)).Return(raw, nil)

	mockStorageAPIErr := mocks.NewMockStorageAPI(ctrl)
	mockStorageAPIErr.EXPECT().Entries((*common.Hash)(nil)).Return(nil, errors.New("entries error"))

	type args struct {
		gData      *genesis.Data
		storageAPI StorageAPI
	}
	tests := []struct {
		name   string
		args   args
		expErr error
		exp    SyncStateAPI
	}{
		{
			name: "OK_Case",
			args: args{
				gData:      g1.GenesisData(),
				storageAPI: mockStorageAPI,
			},
			exp: syncState{chainSpecification: &genesis.Genesis{
				Name:       "",
				ID:         "",
				Bootnodes:  []string{},
				ProtocolID: "",
				Genesis: genesis.Fields{
					Raw:     map[string]map[string]string{},
					Runtime: new(genesis.Runtime),
				},
			},
			},
		},
		{
			name: "Err_Case",
			args: args{
				gData:      g2.GenesisData(),
				storageAPI: mockStorageAPIErr,
			},
			expErr: errors.New("entries error"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := NewStateSync(tt.args.gData, tt.args.storageAPI)
			if tt.expErr != nil {
				assert.EqualError(t, err, tt.expErr.Error())
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.exp, res)
		})
	}
}

func Test_syncState_GenSyncSpec(t *testing.T) {
	type fields struct {
		chainSpecification genesis.Genesis
	}
	type args struct {
		raw bool
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		expErr error
		exp    genesis.Genesis
	}{
		{
			name:   "GenSyncSpec False",
			fields: fields{genesis.Genesis{}},
			exp:    genesis.Genesis{},
		},
		{
			name:   "GenSyncSpec True",
			fields: fields{genesis.Genesis{}},
			args: args{
				raw: true,
			},
			exp: genesis.Genesis{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := syncState{
				chainSpecification: &tt.fields.chainSpecification,
			}
			res, err := s.GenSyncSpec(tt.args.raw)
			if tt.expErr != nil {
				assert.EqualError(t, err, tt.expErr.Error())
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.exp, *res)
		})
	}
}
