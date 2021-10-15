package modules

import (
	"errors"
	"testing"

	"github.com/ChainSafe/gossamer/dot/rpc/modules/mocks"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/ChainSafe/gossamer/lib/common"
	mocksruntime "github.com/ChainSafe/gossamer/lib/runtime/mocks"
)

func TestPaymentQueryInfo(t *testing.T) {
	state := newTestStateService(t)
	bestBlockHash := state.Block.BestBlockHash()

	t.Run("When there is no errors", func(t *testing.T) {
		mockedQueryInfo := &types.TransactionPaymentQueryInfo{
			Weight:     0,
			Class:      0,
			PartialFee: scale.MaxUint128,
		}

		expected := PaymentQueryInfoResponse{
			Weight:     0,
			Class:      0,
			PartialFee: scale.MaxUint128.String(),
		}

		runtimeMock := new(mocksruntime.MockInstance)
		runtimeMock.On("PaymentQueryInfo", mock.AnythingOfType("[]uint8")).Return(mockedQueryInfo, nil)

		blockAPIMock := new(mocks.MockBlockAPI)
		blockAPIMock.On("BestBlockHash").Return(bestBlockHash)

		blockAPIMock.On("GetRuntime", mock.AnythingOfType("*common.Hash")).Return(runtimeMock, nil)

		mod := &PaymentModule{
			blockAPI: blockAPIMock,
		}

		var req PaymentQueryInfoRequest
		req.Ext = "0x0001"
		req.Hash = nil

		var res PaymentQueryInfoResponse
		err := mod.QueryInfo(nil, &req, &res)

		require.NoError(t, err)
		require.Equal(t, expected, res)

		// should be called because req.Hash is nil
		blockAPIMock.AssertCalled(t, "BestBlockHash")
		blockAPIMock.AssertCalled(t, "GetRuntime", mock.AnythingOfType("*common.Hash"))
		runtimeMock.AssertCalled(t, "PaymentQueryInfo", mock.AnythingOfType("[]uint8"))
	})

	t.Run("When could not get runtime", func(t *testing.T) {
		blockAPIMock := new(mocks.MockBlockAPI)
		blockAPIMock.On("BestBlockHash").Return(bestBlockHash)

		blockAPIMock.On("GetRuntime", mock.AnythingOfType("*common.Hash")).
			Return(nil, errors.New("mocked problems"))

		mod := &PaymentModule{
			blockAPI: blockAPIMock,
		}

		var req PaymentQueryInfoRequest
		req.Ext = "0x0011"
		req.Hash = nil

		var res PaymentQueryInfoResponse
		err := mod.QueryInfo(nil, &req, &res)

		require.Error(t, err)
		require.Equal(t, res, PaymentQueryInfoResponse{})

		blockAPIMock.AssertCalled(t, "BestBlockHash")
		blockAPIMock.AssertCalled(t, "GetRuntime", mock.AnythingOfType("*common.Hash"))
	})

	t.Run("When PaymentQueryInfo returns error", func(t *testing.T) {
		runtimeMock := new(mocksruntime.MockInstance)
		runtimeMock.On("PaymentQueryInfo", mock.AnythingOfType("[]uint8")).Return(nil, errors.New("mocked error"))

		blockAPIMock := new(mocks.MockBlockAPI)
		blockAPIMock.On("GetRuntime", mock.AnythingOfType("*common.Hash")).Return(runtimeMock, nil)

		mod := &PaymentModule{
			blockAPI: blockAPIMock,
		}

		mockedHash := common.NewHash([]byte{0x01, 0x02})
		var req PaymentQueryInfoRequest
		req.Ext = "0x0000"
		req.Hash = &mockedHash

		var res PaymentQueryInfoResponse
		err := mod.QueryInfo(nil, &req, &res)

		require.Error(t, err)
		require.Equal(t, res, PaymentQueryInfoResponse{})

		// should be called because req.Hash is nil
		blockAPIMock.AssertNotCalled(t, "BestBlockHash")
		blockAPIMock.AssertCalled(t, "GetRuntime", mock.AnythingOfType("*common.Hash"))
		runtimeMock.AssertCalled(t, "PaymentQueryInfo", mock.AnythingOfType("[]uint8"))
	})

	t.Run("When PaymentQueryInfo returns a nil info", func(t *testing.T) {
		runtimeMock := new(mocksruntime.MockInstance)
		runtimeMock.On("PaymentQueryInfo", mock.AnythingOfType("[]uint8")).Return(nil, nil)

		blockAPIMock := new(mocks.MockBlockAPI)
		blockAPIMock.On("GetRuntime", mock.AnythingOfType("*common.Hash")).Return(runtimeMock, nil)

		mod := &PaymentModule{
			blockAPI: blockAPIMock,
		}

		mockedHash := common.NewHash([]byte{0x01, 0x02})
		var req PaymentQueryInfoRequest
		req.Ext = "0x0020"
		req.Hash = &mockedHash

		var res PaymentQueryInfoResponse
		err := mod.QueryInfo(nil, &req, &res)

		require.NoError(t, err)
		require.Equal(t, res, PaymentQueryInfoResponse{})

		// should be called because req.Hash is nil
		blockAPIMock.AssertNotCalled(t, "BestBlockHash")
		blockAPIMock.AssertCalled(t, "GetRuntime", mock.AnythingOfType("*common.Hash"))
		runtimeMock.AssertCalled(t, "PaymentQueryInfo", mock.AnythingOfType("[]uint8"))
	})
}
