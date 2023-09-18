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
	"github.com/golang/mock/gomock"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPaymentModule_QueryInfo(t *testing.T) {
	ctrl := gomock.NewController(t)

	testHash := common.NewHash([]byte{0x01, 0x02})
	u, err := scale.NewUint128(new(big.Int).SetBytes([]byte{1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6}))
	require.NoError(t, err)

	runtimeMock := mocksruntime.NewMockInstance(ctrl)
	runtimeMock2 := mocksruntime.NewMockInstance(ctrl)
	runtimeErrorMock := mocksruntime.NewMockInstance(ctrl)

	blockAPIMock := mocks.NewMockBlockAPI(ctrl)
	blockAPIMock2 := mocks.NewMockBlockAPI(ctrl)
	blockErrorAPIMock1 := mocks.NewMockBlockAPI(ctrl)
	blockErrorAPIMock2 := mocks.NewMockBlockAPI(ctrl)

	blockAPIMock.EXPECT().BestBlockHash().Return(testHash).Times(2)
	blockAPIMock.EXPECT().GetRuntime(testHash).Return(runtimeMock, nil).Times(3)

	blockAPIMock2.EXPECT().GetRuntime(testHash).Return(runtimeMock2, nil)

	blockErrorAPIMock1.EXPECT().GetRuntime(testHash).Return(runtimeErrorMock, nil)

	blockErrorAPIMock2.EXPECT().GetRuntime(testHash).Return(nil, errors.New("GetRuntime error"))

	runtimeMock.EXPECT().PaymentQueryInfo(common.MustHexToBytes("0x0000")).Return(nil, nil).Times(2)
	runtimeMock2.EXPECT().PaymentQueryInfo(common.MustHexToBytes("0x0000")).Return(&types.RuntimeDispatchInfo{
		Weight:     uint64(21),
		Class:      21,
		PartialFee: u,
	}, nil)
	runtimeErrorMock.EXPECT().PaymentQueryInfo(common.MustHexToBytes("0x0000")).
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
			name: "Nil_Query_Info",
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
			name: "Not_Nil_Query_Info",
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
			name: "Nil_Hash",
			fields: fields{
				paymentModule.blockAPI,
			},
			args: args{
				req: &PaymentQueryInfoRequest{
					Ext: "0x0",
				},
			},
			expErr: errors.New("encoding/hex: odd length hex string: 0x0"),
		},
		{
			name: "Invalid_Ext",
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
			name: "PaymentQueryInfo_error",
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
			name: "GetRuntime_error",
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
