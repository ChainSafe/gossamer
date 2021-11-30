// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package modules

import (
	"errors"
	"math/big"
	"net/http"
	"testing"

	apimocks "github.com/ChainSafe/gossamer/dot/rpc/modules/mocks"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChainModule_GetBlock(t *testing.T) {
	testHash := common.NewHash([]byte{0x01, 0x02})
	inputHash := common.MustHexToHash("0x0102000000000000000000000000000000000000000000000000000000000000")
	emptyBlock := types.NewEmptyBlock()

	mockBlockAPI := new(apimocks.BlockAPI)
	mockBlockAPI.On("GetBlockByHash", inputHash).Return(&emptyBlock, nil)
	mockBlockAPI.On("BestBlockHash").Return(testHash, nil)

	mockBlockAPIGetHashErr := new(apimocks.BlockAPI)
	mockBlockAPIGetHashErr.On("GetBlockByHash", inputHash).Return(nil, errors.New("GetJustification error"))

	bodyBlock := types.NewEmptyBlock()
	bodyBlock.Body = types.BytesArrayToExtrinsics([][]byte{{1}})
	mockBlockAPIWithBody := new(apimocks.BlockAPI)
	mockBlockAPIWithBody.On("GetBlockByHash", inputHash).Return(&bodyBlock, nil)

	chainModule := NewChainModule(mockBlockAPI)
	type fields struct {
		blockAPI BlockAPI
	}
	type args struct {
		r   *http.Request
		req *ChainHashRequest
		res ChainBlockResponse
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		exp     ChainBlockResponse
		wantErr bool
		expErr  error
	}{
		{
			name: "GetBlock OK",
			fields: fields{
				chainModule.blockAPI,
			},
			args: args{
				req: &ChainHashRequest{},
			},
			exp: ChainBlockResponse{Block: ChainBlock{
				Header: ChainBlockHeaderResponse{
					ParentHash:     "0x0000000000000000000000000000000000000000000000000000000000000000",
					Number:         "0x00",
					StateRoot:      "0x0000000000000000000000000000000000000000000000000000000000000000",
					ExtrinsicsRoot: "0x0000000000000000000000000000000000000000000000000000000000000000",
					Digest:         ChainBlockHeaderDigest{},
				},
				Body: nil,
			}},
		},
		{
			name: "GetBlockByHash Err",
			fields: fields{
				mockBlockAPIGetHashErr,
			},
			args: args{
				req: &ChainHashRequest{&testHash},
			},
			wantErr: true,
			expErr:  errors.New("GetJustification error"),
		},
		{
			name: "GetBlock with body OK",
			fields: fields{
				mockBlockAPIWithBody,
			},
			args: args{
				req: &ChainHashRequest{&testHash},
			},
			exp: ChainBlockResponse{Block: ChainBlock{
				Header: ChainBlockHeaderResponse{
					ParentHash:     "0x0000000000000000000000000000000000000000000000000000000000000000",
					Number:         "0x00",
					StateRoot:      "0x0000000000000000000000000000000000000000000000000000000000000000",
					ExtrinsicsRoot: "0x0000000000000000000000000000000000000000000000000000000000000000",
					Digest:         ChainBlockHeaderDigest{},
				},
				Body: []string{"0x0401"},
			}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.args.res = ChainBlockResponse{}
			cm := &ChainModule{
				blockAPI: tt.fields.blockAPI,
			}
			err := cm.GetBlock(tt.args.r, tt.args.req, &tt.args.res)
			if tt.wantErr {
				require.Error(t, err)
				assert.EqualError(t, err, tt.expErr.Error())
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.exp, tt.args.res)
		})
	}
}

func TestChainModule_GetBlockHash(t *testing.T) {
	testHash := common.NewHash([]byte{0x01, 0x02})
	i := []interface{}{"a"}

	mockBlockAPI := new(apimocks.BlockAPI)
	mockBlockAPI.On("BestBlockHash").Return(testHash, nil)
	mockBlockAPI.On("GetBlockHash", new(big.Int).SetInt64(int64(21))).Return(testHash, nil)

	mockBlockAPIErr := new(apimocks.BlockAPI)
	mockBlockAPIErr.On("BestBlockHash").Return(testHash, nil)
	mockBlockAPIErr.On("GetBlockHash", new(big.Int).SetInt64(int64(21))).Return(nil, errors.New("GetBlockHash Error"))

	expRes := ChainHashResponse(testHash.String())
	type fields struct {
		blockAPI BlockAPI
	}
	type args struct {
		r   *http.Request
		req *ChainBlockNumberRequest
		res ChainHashResponse
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
		expErr  error
		exp     ChainHashResponse
	}{
		{
			name: "GetBlockHash nil req OK",
			fields: fields{
				mockBlockAPI,
			},
			args: args{
				req: &ChainBlockNumberRequest{},
			},
			exp: expRes,
		},
		{
			name: "GetBlockHash string req OK",
			fields: fields{
				mockBlockAPI,
			},
			args: args{
				req: &ChainBlockNumberRequest{"21"},
			},
			exp: expRes,
		},
		{
			name: "GetBlockHash float req OK",
			fields: fields{
				mockBlockAPI,
			},
			args: args{
				req: &ChainBlockNumberRequest{float64(21)},
			},
			exp: expRes,
		},
		{
			name: "GetBlockHash unknown request number",
			fields: fields{
				mockBlockAPI,
			},
			args: args{
				req: &ChainBlockNumberRequest{uintptr(1)},
			},
			exp:     []string(nil),
			wantErr: true,
			expErr:  errors.New("unknown request number type: uintptr"),
		},
		{
			name: "GetBlockHash string slice req err",
			fields: fields{
				mockBlockAPI,
			},
			args: args{
				req: &ChainBlockNumberRequest{i},
			},
			exp:     []string(nil),
			wantErr: true,
			expErr:  errors.New("error setting number from string"),
		},
		{
			name: "GetBlockHash string req Err",
			fields: fields{
				mockBlockAPIErr,
			},
			args: args{
				req: &ChainBlockNumberRequest{"21"},
			},
			exp:     []string(nil),
			wantErr: true,
			expErr:  errors.New("GetBlockHash Error"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.args.res = ChainHashResponse(nil)
			cm := &ChainModule{
				blockAPI: tt.fields.blockAPI,
			}
			err := cm.GetBlockHash(tt.args.r, tt.args.req, &tt.args.res)
			if tt.wantErr {
				require.Error(t, err)
				assert.EqualError(t, err, tt.expErr.Error())
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.exp, tt.args.res)
		})
	}
}

func TestChainModule_GetFinalizedHead(t *testing.T) {
	testHash := common.NewHash([]byte{0x01, 0x02})
	mockBlockAPI := new(apimocks.BlockAPI)
	mockBlockAPI.On("GetHighestFinalisedHash").Return(testHash, nil)

	mockBlockAPIErr := new(apimocks.BlockAPI)
	mockBlockAPIErr.On("GetHighestFinalisedHash").Return(nil, errors.New("GetHighestFinalisedHash Error"))

	expRes := ChainHashResponse(common.BytesToHex(testHash[:]))
	type fields struct {
		blockAPI BlockAPI
	}
	type args struct {
		r   *http.Request
		req *EmptyRequest
		res ChainHashResponse
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
		expErr  error
		exp     ChainHashResponse
	}{
		{
			name: "happy path",
			fields: fields{
				mockBlockAPI,
			},
			args: args{
				req: &EmptyRequest{},
			},
			exp: expRes,
		},
		{
			name: "error case",
			fields: fields{
				mockBlockAPIErr,
			},
			args: args{
				req: &EmptyRequest{},
			},
			wantErr: true,
			expErr:  errors.New("GetHighestFinalisedHash Error"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.args.res = ChainHashResponse(nil)
			cm := &ChainModule{
				blockAPI: tt.fields.blockAPI,
			}
			err := cm.GetFinalizedHead(tt.args.r, tt.args.req, &tt.args.res)
			if tt.wantErr {
				require.Error(t, err)
				assert.EqualError(t, err, tt.expErr.Error())
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.exp, tt.args.res)
		})
	}
}

func TestChainModule_GetFinalizedHeadByRound(t *testing.T) {
	testHash := common.NewHash([]byte{0x01, 0x02})
	mockBlockAPI := new(apimocks.BlockAPI)
	mockBlockAPI.On("GetFinalisedHash", uint64(21), uint64(21)).Return(testHash, nil)

	mockBlockAPIErr := new(apimocks.BlockAPI)
	mockBlockAPIErr.On("GetFinalisedHash", uint64(21), uint64(21)).Return(nil, errors.New("GetFinalisedHash Error"))

	expRes := ChainHashResponse(common.BytesToHex(testHash[:]))
	type fields struct {
		blockAPI BlockAPI
	}
	type args struct {
		r   *http.Request
		req *ChainFinalizedHeadRequest
		res ChainHashResponse
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
		expErr  error
		exp     ChainHashResponse
	}{
		{
			name: "GetFinalisedHash OK",
			fields: fields{
				mockBlockAPI,
			},
			args: args{
				req: &ChainFinalizedHeadRequest{
					Round: uint64(21),
					SetID: uint64(21),
				},
			},
			exp: expRes,
		},
		{
			name: "GetFinalisedHash ERR",
			fields: fields{
				mockBlockAPIErr,
			},
			args: args{
				req: &ChainFinalizedHeadRequest{
					Round: uint64(21),
					SetID: uint64(21),
				},
			},
			wantErr: true,
			expErr:  errors.New("GetFinalisedHash Error"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.args.res = ChainHashResponse(nil)
			cm := &ChainModule{
				blockAPI: tt.fields.blockAPI,
			}
			err := cm.GetFinalizedHeadByRound(tt.args.r, tt.args.req, &tt.args.res)
			if tt.wantErr {
				require.Error(t, err)
				assert.EqualError(t, err, tt.expErr.Error())
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.exp, tt.args.res)
		})
	}
}

func TestChainModule_GetHeader(t *testing.T) {
	emptyHeader := types.NewEmptyHeader()
	testHash := common.NewHash([]byte{0x01, 0x02})
	inputHash, err := common.HexToHash("0x0102000000000000000000000000000000000000000000000000000000000000")
	require.NoError(t, err)

	mockBlockAPI := new(apimocks.BlockAPI)
	mockBlockAPI.On("GetHeader", inputHash).Return(emptyHeader, nil)

	mockBlockAPIErr := new(apimocks.BlockAPI)
	mockBlockAPIErr.On("GetHeader", inputHash).Return(nil, errors.New("GetFinalisedHash Error"))

	expRes, err := HeaderToJSON(*emptyHeader)
	require.NoError(t, err)

	type fields struct {
		blockAPI BlockAPI
	}
	type args struct {
		r   *http.Request
		req *ChainHashRequest
		res ChainBlockHeaderResponse
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
		expErr  error
		exp     ChainBlockHeaderResponse
	}{
		{
			name: "GetHeader OK",
			fields: fields{
				mockBlockAPI,
			},
			args: args{
				req: &ChainHashRequest{&testHash},
			},
			exp: expRes,
		},
		{
			name: "GetHeader ERR",
			fields: fields{
				mockBlockAPIErr,
			},
			args: args{
				req: &ChainHashRequest{&testHash},
			},
			wantErr: true,
			expErr:  errors.New("GetFinalisedHash Error"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.args.res = ChainBlockHeaderResponse{}
			cm := &ChainModule{
				blockAPI: tt.fields.blockAPI,
			}
			err := cm.GetHeader(tt.args.r, tt.args.req, &tt.args.res)
			if tt.wantErr {
				require.Error(t, err)
				assert.EqualError(t, err, tt.expErr.Error())
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.exp, tt.args.res)
		})
	}
}

func TestChainModule_ErrSubscriptionTransport(t *testing.T) {
	req := &EmptyRequest{}
	res := &ChainBlockHeaderResponse{}
	cm := NewChainModule(new(apimocks.BlockAPI))

	err := cm.SubscribeFinalizedHeads(nil, req, res)
	require.Equal(t, err, ErrSubscriptionTransport)

	err = cm.SubscribeNewHead(nil, req, res)
	require.Equal(t, err, ErrSubscriptionTransport)

	err = cm.SubscribeNewHeads(nil, req, res)
	require.Equal(t, err, ErrSubscriptionTransport)
}

func TestHeaderToJSON(t *testing.T) {
	emptyHeader := types.NewEmptyHeader()
	vdts := types.NewDigest()
	err := vdts.Add(
		types.PreRuntimeDigest{
			ConsensusEngineID: types.BabeEngineID,
			Data:              common.MustHexToBytes("0x0201000000ef55a50f00000000"),
		},
		types.ConsensusDigest{
			ConsensusEngineID: types.BabeEngineID,
			Data:              common.MustHexToBytes("0x0118ca239392960473fe1bc65f94ee27d890a49c1b200c006ff5dcc525330ecc16770100000000000000b46f01874ce7abbb5220e8fd89bede0adad14c73039d91e28e881823433e723f0100000000000000d684d9176d6eb69887540c9a89fa6097adea82fc4b0ff26d1062b488f352e179010000000000000068195a71bdde49117a616424bdc60a1733e96acb1da5aeab5d268cf2a572e94101000000000000001a0575ef4ae24bdfd31f4cb5bd61239ae67c12d4e64ae51ac756044aa6ad8200010000000000000018168f2aad0081a25728961ee00627cfe35e39833c805016632bf7c14da5800901000000000000000000000000000000000000000000000000000000000000000000000000000000"),
		},
		types.SealDigest{
			ConsensusEngineID: types.BabeEngineID,
			Data:              common.MustHexToBytes("0x4625284883e564bc1e4063f5ea2b49846cdddaa3761d04f543b698c1c3ee935c40d25b869247c36c6b8a8cbbd7bb2768f560ab7c276df3c62df357a7e3b1ec8d"),
		},
	)
	require.NoError(t, err)

	header, err := types.NewHeader(common.Hash{}, common.Hash{}, common.Hash{}, big.NewInt(21), vdts)
	require.NoError(t, err)

	expRes, err := HeaderToJSON(*header)
	require.NoError(t, err)
	expResEmpty, err := HeaderToJSON(*emptyHeader)
	require.NoError(t, err)
	type args struct {
		header types.Header
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
		exp     ChainBlockHeaderResponse
	}{
		{
			name: "HeaderToJSON Empty Header",
			args: args{
				header: *emptyHeader,
			},
			exp: expResEmpty,
		},
		{
			name: "HeaderToJSON NonEmpty Header",
			args: args{
				header: *header,
			},
			exp: expRes,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := HeaderToJSON(tt.args.header)
			assert.Equal(t, tt.wantErr, err != nil)
			if err == nil {
				assert.Equal(t, tt.exp, res)
			} else {
				assert.Error(t, err)
			}
		})
	}
}
