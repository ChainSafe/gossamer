// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package modules

import (
	"errors"
	"math/big"
	"net/http"
	"testing"

	"github.com/ChainSafe/gossamer/dot/rpc/modules/mocks"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	mocksruntime "github.com/ChainSafe/gossamer/lib/runtime/mocks"
	"github.com/ChainSafe/gossamer/pkg/scale"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPaymentModule_QueryInfo(t *testing.T) {
	testHash := common.NewHash([]byte{0x01, 0x02})
	u, err := scale.NewUint128(new(big.Int).SetBytes([]byte{1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6}))
	require.NoError(t, err)

	runtimeMock := new(mocksruntime.Instance)
	runtimeMock2 := new(mocksruntime.Instance)
	runtimeErrorMock := new(mocksruntime.Instance)

	blockAPIMock := new(mocks.BlockAPI)
	blockAPIMock2 := new(mocks.BlockAPI)
	blockErrorAPIMock1 := new(mocks.BlockAPI)
	blockErrorAPIMock2 := new(mocks.BlockAPI)

	blockAPIMock.On("BestBlockHash").Return(testHash, nil)
	blockAPIMock.On("GetRuntime", &testHash).Return(runtimeMock, nil)

	blockAPIMock2.On("GetRuntime", &testHash).Return(runtimeMock2, nil)

	blockErrorAPIMock1.On("GetRuntime", &testHash).Return(runtimeErrorMock, nil)

	blockErrorAPIMock2.On("GetRuntime", &testHash).Return(nil, errors.New("GetRuntime error"))

	runtimeMock.On("PaymentQueryInfo", common.MustHexToBytes("0x0000")).Return(nil, nil)
	runtimeMock2.On("PaymentQueryInfo", common.MustHexToBytes("0x0000")).Return(&types.TransactionPaymentQueryInfo{
		Weight:     uint64(21),
		Class:      21,
		PartialFee: u,
	}, nil)
	runtimeErrorMock.On("PaymentQueryInfo", common.MustHexToBytes("0x0000")).
		Return(nil, errors.New("PaymentQueryInfo error"))

	paymentModule := NewPaymentModule(blockAPIMock)
	type fields struct {
		blockAPI BlockAPI
	}
	type args struct {
		in0 *http.Request
		req *PaymentQueryInfoRequest
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		expErr error
		exp    PaymentQueryInfoResponse
	}{
		{
			name: "Nil Query Info",
			fields: fields{
				paymentModule.blockAPI,
			},
			args: args{
				req: &PaymentQueryInfoRequest{
					Ext:  "0x0000",
					Hash: &testHash,
				},
			},
			exp: PaymentQueryInfoResponse{},
		},
		{
			name: "Not Nil Query Info",
			fields: fields{
				blockAPIMock2,
			},
			args: args{
				req: &PaymentQueryInfoRequest{
					Ext:  "0x0000",
					Hash: &testHash,
				},
			},
			exp: PaymentQueryInfoResponse{
				Weight: uint64(21),
				Class:  21,
				PartialFee: scale.MustNewUint128(new(big.Int).SetBytes(
					[]byte{1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6}),
				).String(),
			},
		},
		{
			name: "Nil Hash",
			fields: fields{
				paymentModule.blockAPI,
			},
			args: args{
				req: &PaymentQueryInfoRequest{
					Ext: "0x0",
				},
			},
			expErr: errors.New("cannot decode an odd length string"),
		},
		{
			name: "Invalid Ext",
			fields: fields{
				paymentModule.blockAPI,
			},
			args: args{
				req: &PaymentQueryInfoRequest{
					Ext: "0x0000",
				},
			},
			exp: PaymentQueryInfoResponse{},
		},
		{
			name: "PaymentQueryInfo error",
			fields: fields{
				blockErrorAPIMock1,
			},
			args: args{
				req: &PaymentQueryInfoRequest{
					Ext:  "0x0000",
					Hash: &testHash,
				},
			},
			expErr: errors.New("PaymentQueryInfo error"),
		},
		{
			name: "GetRuntime error",
			fields: fields{
				blockErrorAPIMock2,
			},
			args: args{
				req: &PaymentQueryInfoRequest{
					Ext:  "0x0000",
					Hash: &testHash,
				},
			},
			expErr: errors.New("GetRuntime error"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &PaymentModule{
				blockAPI: tt.fields.blockAPI,
			}
			res := PaymentQueryInfoResponse{}
			err := p.QueryInfo(tt.args.in0, tt.args.req, &res)
			if tt.expErr != nil {
				assert.EqualError(t, err, tt.expErr.Error())
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.exp, res)
		})
	}
}
