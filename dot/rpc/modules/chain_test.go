// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package modules

import (
	"errors"
	"github.com/ChainSafe/gossamer/lib/common"
	"net/http"
	"testing"

	apimocks "github.com/ChainSafe/gossamer/dot/rpc/modules/mocks"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/stretchr/testify/mock"
)

func TestChainModule_GetBlock(t *testing.T) {
	mockedHash := common.NewHash([]byte{0x01, 0x02})
	emptyBlock := types.NewEmptyBlock()

	mockBlockAPI := new(apimocks.BlockAPI)
	mockBlockAPI.On("GetBlockByHash", mock.AnythingOfType("common.Hash")).Return(&emptyBlock, nil)
	mockBlockAPI.On("BestBlockHash").Return(mockedHash, nil)

	mockBlockAPIGetHashErr := new(apimocks.BlockAPI)
	mockBlockAPIGetHashErr.On("GetBlockByHash", mock.AnythingOfType("common.Hash")).Return(nil, errors.New("GetJustification error"))

	bodyBlock := types.NewEmptyBlock()
	bodyBlock.Body = types.BytesArrayToExtrinsics([][]byte{{1}})
	mockBlockAPIWithBody := new(apimocks.BlockAPI)
	mockBlockAPIWithBody.On("GetBlockByHash", mock.AnythingOfType("common.Hash")).Return(&bodyBlock, nil)


	chainModule := NewChainModule(mockBlockAPI)
	res := ChainBlockResponse{}
	type fields struct {
		blockAPI BlockAPI
	}
	type args struct {
		r   *http.Request
		req *ChainHashRequest
		res *ChainBlockResponse
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "GetBlock OK",
			fields: fields{
				chainModule.blockAPI,
			},
			args: args{
				r: nil,
				req: &ChainHashRequest{},
				res: &res,
			},
			wantErr: false,
		},
		{
			name: "GetBlockByHash Err",
			fields: fields{
				mockBlockAPIGetHashErr,
			},
			args: args{
				r: nil,
				req: &ChainHashRequest{&mockedHash},
				res: &res,
			},
			wantErr: true,
		},
		{
			name: "GetBlock with body OK",
			fields: fields{
				mockBlockAPIWithBody,
			},
			args: args{
				r: nil,
				req: &ChainHashRequest{&mockedHash},
				res: &res,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cm := &ChainModule{
				blockAPI: tt.fields.blockAPI,
			}
			if err := cm.GetBlock(tt.args.r, tt.args.req, tt.args.res); (err != nil) != tt.wantErr {
				t.Errorf("GetBlock() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestChainModule_GetBlockHash(t *testing.T) {
	mockedHash := common.NewHash([]byte{0x01, 0x02})
	i := make([]interface{}, 1)
	i[0] = "a"

	mockBlockAPI := new(apimocks.BlockAPI)
	mockBlockAPI.On("BestBlockHash").Return(mockedHash, nil)
	mockBlockAPI.On("GetBlockHash", mock.AnythingOfType("*big.Int")).Return(mockedHash, nil)

	var res ChainHashResponse
	type fields struct {
		blockAPI BlockAPI
	}
	type args struct {
		r   *http.Request
		req *ChainBlockNumberRequest
		res *ChainHashResponse
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "GetBlockHash nil req OK",
			fields: fields{
				mockBlockAPI,
			},
			args: args{
				r: nil,
				req: &ChainBlockNumberRequest{},
				res: &res,
			},
			wantErr: false,
		},
		{
			name: "GetBlockHash string req OK",
			fields: fields{
				mockBlockAPI,
			},
			args: args{
				r: nil,
				req: &ChainBlockNumberRequest{"21"},
				res: &res,
			},
			wantErr: false,
		},
		{
			name: "GetBlockHash float req OK",
			fields: fields{
				mockBlockAPI,
			},
			args: args{
				r: nil,
				req: &ChainBlockNumberRequest{float64(21)},
				res: &res,
			},
			wantErr: false,
		},
		{
			name: "GetBlockHash unknown request number",
			fields: fields{
				mockBlockAPI,
			},
			args: args{
				r: nil,
				req: &ChainBlockNumberRequest{uintptr(1)},
				res: &res,
			},
			wantErr: true,
		},
		{
			name: "GetBlockHash string slice req err",
			fields: fields{
				mockBlockAPI,
			},
			args: args{
				r: nil,
				req: &ChainBlockNumberRequest{i},
				res: &res,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cm := &ChainModule{
				blockAPI: tt.fields.blockAPI,
			}
			if err := cm.GetBlockHash(tt.args.r, tt.args.req, tt.args.res); (err != nil) != tt.wantErr {
				t.Errorf("GetBlockHash() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}