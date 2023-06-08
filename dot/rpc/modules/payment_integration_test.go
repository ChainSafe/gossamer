// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

//go:build integration

package modules

import (
	"errors"
	"testing"

	"github.com/ChainSafe/gossamer/dot/rpc/modules/mocks"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	"github.com/ChainSafe/gossamer/lib/common"
	mocksruntime "github.com/ChainSafe/gossamer/lib/runtime/mocks"
)

func TestPaymentQueryInfo(t *testing.T) {
	state := newTestStateService(t)
	bestBlockHash := state.Block.BestBlockHash()

	t.Run("When_there_is_no_errors", func(t *testing.T) {
		ctrl := gomock.NewController(t)

		mockedQueryInfo := &types.RuntimeDispatchInfo{
			Weight:     0,
			Class:      0,
			PartialFee: scale.MaxUint128,
		}

		expected := PaymentQueryInfoResponse{
			Weight:     0,
			Class:      0,
			PartialFee: scale.MaxUint128.String(),
		}

		runtimeMock := mocksruntime.NewMockInstance(ctrl)
		runtimeMock.EXPECT().PaymentQueryInfo(gomock.Any()).Return(mockedQueryInfo, nil)

		blockAPIMock := mocks.NewMockBlockAPI(ctrl)
		blockAPIMock.EXPECT().BestBlockHash().Return(bestBlockHash)

		blockAPIMock.EXPECT().GetRuntime(bestBlockHash).Return(runtimeMock, nil)

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
	})

	t.Run("When_could_not_get_runtime", func(t *testing.T) {
		ctrl := gomock.NewController(t)

		blockAPIMock := mocks.NewMockBlockAPI(ctrl)
		blockAPIMock.EXPECT().BestBlockHash().Return(bestBlockHash)

		blockAPIMock.EXPECT().GetRuntime(bestBlockHash).
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
	})

	t.Run("When_PaymentQueryInfo_returns_error", func(t *testing.T) {
		ctrl := gomock.NewController(t)

		runtimeMock := mocksruntime.NewMockInstance(ctrl)
		runtimeMock.EXPECT().PaymentQueryInfo(gomock.Any()).Return(nil, errors.New("mocked error"))

		blockAPIMock := mocks.NewMockBlockAPI(ctrl)
		blockAPIMock.EXPECT().GetRuntime(common.Hash{1, 2}).Return(runtimeMock, nil)

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
	})

	t.Run("When_PaymentQueryInfo_returns_a_nil_info", func(t *testing.T) {
		ctrl := gomock.NewController(t)

		runtimeMock := mocksruntime.NewMockInstance(ctrl)
		runtimeMock.EXPECT().PaymentQueryInfo(gomock.Any()).Return(nil, nil)

		blockAPIMock := mocks.NewMockBlockAPI(ctrl)
		blockAPIMock.EXPECT().GetRuntime(common.Hash{1, 2}).Return(runtimeMock, nil)

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
	})
}
