// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package modules

import (
	"errors"
	apimocks "github.com/ChainSafe/gossamer/dot/rpc/modules/mocks"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/stretchr/testify/mock"
	"net/http"
	"testing"
)

func TestGrandpaModule_ProveFinality(t *testing.T) {
	mockedHash := common.NewHash([]byte{0x01, 0x02})
	mockedHashSlice := []common.Hash{mockedHash, mockedHash, mockedHash}

	mockBlockFinalityAPI := new(apimocks.BlockFinalityAPI)
	mockBlockAPI := new(apimocks.BlockAPI)
	mockBlockAPI.On("SubChain", mock.AnythingOfType("common.Hash"), mock.AnythingOfType("common.Hash")).Return(mockedHashSlice, nil)
	mockBlockAPI.On("HasJustification", mock.AnythingOfType("common.Hash")).Return(true, nil)
	mockBlockAPI.On("GetJustification", mock.AnythingOfType("common.Hash")).Return([]byte("test"), nil)

	mockBlockAPIHasJustErr := new(apimocks.BlockAPI)
	mockBlockAPIHasJustErr.On("SubChain", mock.AnythingOfType("common.Hash"), mock.AnythingOfType("common.Hash")).Return(mockedHashSlice, nil)
	mockBlockAPIHasJustErr.On("HasJustification", mock.AnythingOfType("common.Hash")).Return(false, nil)

	mockBlockAPIGetJustErr := new(apimocks.BlockAPI)
	mockBlockAPIGetJustErr.On("SubChain", mock.AnythingOfType("common.Hash"), mock.AnythingOfType("common.Hash")).Return(mockedHashSlice, nil)
	mockBlockAPIGetJustErr.On("HasJustification", mock.AnythingOfType("common.Hash")).Return(true, nil)
	mockBlockAPIGetJustErr.On("GetJustification", mock.AnythingOfType("common.Hash")).Return(nil, errors.New("GetJustification error"))

	mockBlockAPISubChainErr := new(apimocks.BlockAPI)
	mockBlockAPISubChainErr.On("SubChain", mock.AnythingOfType("common.Hash"), mock.AnythingOfType("common.Hash")).Return(nil, errors.New("SubChain error"))

	grandpaModule := NewGrandpaModule(mockBlockAPISubChainErr, mockBlockFinalityAPI)
	var res ProveFinalityResponse
	type fields struct {
		blockAPI         BlockAPI
		blockFinalityAPI BlockFinalityAPI
	}
	type args struct {
		r   *http.Request
		req *ProveFinalityRequest
		res *ProveFinalityResponse
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "SubChain Err",
			fields: fields{
				grandpaModule.blockAPI,
				grandpaModule.blockFinalityAPI,
			},
			args: args{
				r: nil,
				req: &ProveFinalityRequest{
					blockHashStart: mockedHash,
					blockHashEnd:  mockedHash,
					authorityID: uint64(21),
				},
				res: &res,
			},
			wantErr: true,
		},
		{
			name: "OK Case",
			fields: fields{
				mockBlockAPI,
				mockBlockFinalityAPI,
			},
			args: args{
				r: nil,
				req: &ProveFinalityRequest{
					blockHashStart: mockedHash,
					blockHashEnd:  mockedHash,
					authorityID: uint64(21),
				},
				res: &res,
			},
			wantErr: false,
		},
		{
			name: "HasJustification Error",
			fields: fields{
				mockBlockAPIHasJustErr,
				mockBlockFinalityAPI,
			},
			args: args{
				r: nil,
				req: &ProveFinalityRequest{
					blockHashStart: mockedHash,
					blockHashEnd:  mockedHash,
					authorityID: uint64(21),
				},
				res: &res,
			},
			wantErr: false,
		},
		{
			name: "GetJustification Error",
			fields: fields{
				mockBlockAPIGetJustErr,
				mockBlockFinalityAPI,
			},
			args: args{
				r: nil,
				req: &ProveFinalityRequest{
					blockHashStart: mockedHash,
					blockHashEnd:  mockedHash,
					authorityID: uint64(21),
				},
				res: &res,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gm := &GrandpaModule{
				blockAPI:         tt.fields.blockAPI,
				blockFinalityAPI: tt.fields.blockFinalityAPI,
			}
			if err := gm.ProveFinality(tt.args.r, tt.args.req, tt.args.res); (err != nil) != tt.wantErr {
				t.Errorf("ProveFinality() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}