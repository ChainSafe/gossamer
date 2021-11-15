package modules

import (
	"errors"
	"math/big"
	"net/http"
	"testing"

	apimocks "github.com/ChainSafe/gossamer/dot/rpc/modules/mocks"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	mocksruntime "github.com/ChainSafe/gossamer/lib/runtime/mocks"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/stretchr/testify/require"
)

func TestPaymentModule_QueryInfo(t *testing.T) {
	testHash := common.NewHash([]byte{0x01, 0x02})
	u, err := scale.NewUint128(new(big.Int).SetBytes([]byte{1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6}))
	require.NoError(t, err)

	runtimeMock := new(mocksruntime.Instance)
	runtimeMock2 := new(mocksruntime.Instance)
	runtimeErrorMock := new(mocksruntime.Instance)

	blockAPIMock := new(apimocks.BlockAPI)
	blockAPIMock2 := new(apimocks.BlockAPI)
	blockErrorAPIMock1 := new(apimocks.BlockAPI)
	blockErrorAPIMock2 := new(apimocks.BlockAPI)

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
	runtimeErrorMock.On("PaymentQueryInfo", common.MustHexToBytes("0x0000")).Return(nil, errors.New("PaymentQueryInfo error"))

	paymentModule := NewPaymentModule(blockAPIMock)
	var res PaymentQueryInfoResponse
	type fields struct {
		blockAPI BlockAPI
	}
	type args struct {
		in0 *http.Request
		req *PaymentQueryInfoRequest
		res *PaymentQueryInfoResponse
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
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
				res: &res,
			},
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
				res: &res,
			},
		},
		{
			name: "Nil Hash",
			fields: fields{
				paymentModule.blockAPI,
			},
			args: args{
				req: &PaymentQueryInfoRequest{
					Ext:  "0x0",
				},
				res: &res,
			},
			wantErr: true,
		},
		{
			name: "Invalid Ext",
			fields: fields{
				paymentModule.blockAPI,
			},
			args: args{
				req: &PaymentQueryInfoRequest{
					Ext:  "0x0000",
				},
				res: &res,
			},
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
				res: &res,
			},
			wantErr: true,
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
				res: &res,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &PaymentModule{
				blockAPI: tt.fields.blockAPI,
			}
			if err := p.QueryInfo(tt.args.in0, tt.args.req, tt.args.res); (err != nil) != tt.wantErr {
				t.Errorf("QueryInfo() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
