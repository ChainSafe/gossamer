// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package core

import (
	"errors"

	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/dot/peerset"
	"github.com/ChainSafe/gossamer/dot/types"
	mocksruntime "github.com/ChainSafe/gossamer/lib/runtime/mocks"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/stretchr/testify/assert"
	"testing"

	"github.com/ChainSafe/gossamer/lib/transaction"
	"github.com/golang/mock/gomock"
)

var (
	errBestHeader = errors.New("best header error")
	errGetRuntime = errors.New("get runtime error")
)

func TestService_TransactionsCount(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockTxnStateEmpty := NewMockTransactionState(ctrl)
	mockTxnState := NewMockTransactionState(ctrl)

	txs := make([]*transaction.ValidTransaction, 2)

	mockTxnStateEmpty.EXPECT().PendingInPool().Return([]*transaction.ValidTransaction{})
	mockTxnState.EXPECT().PendingInPool().Return(txs)

	tests := []struct {
		name    string
		service *Service
		exp     int
	}{
		{
			name:    "empty",
			service: &Service{transactionState: mockTxnStateEmpty},
			exp:     0,
		},
		{
			name:    "not empty",
			service: &Service{transactionState: mockTxnState},
			exp:     2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := tt.service
			res := s.TransactionsCount()
			assert.Equal(t, tt.exp, res)
		})
	}
}

func TestServiceHandleTransactionMessage(t *testing.T) {
	testEmptyHeader := types.NewEmptyHeader()

	ctrl := gomock.NewController(t)

	// Netowrk
	mockNotSyncedNet := NewMockNetwork(ctrl)
	mockSyncedNet1 := NewMockNetwork(ctrl)
	mockSyncedNet2 := NewMockNetwork(ctrl)
	mockSyncedNetHappy := NewMockNetwork(ctrl)

	// BlockState
	mockBlockStateBestHeadErr := NewMockBlockState(ctrl)
	mockBlockStateRuntimeErr := NewMockBlockState(ctrl)
	mockBlockStateRuntimeOk := NewMockBlockState(ctrl)

	// Runtime
	runtimeMock := new(mocksruntime.Instance)

	mockNotSyncedNet.EXPECT().IsSynced().Return(false)

	mockSyncedNet1.EXPECT().IsSynced().Return(true)
	mockBlockStateBestHeadErr.EXPECT().BestBlockHeader().Return(nil, errBestHeader)

	mockSyncedNet2.EXPECT().IsSynced().Return(true)
	mockBlockStateRuntimeErr.EXPECT().BestBlockHeader().Return(testEmptyHeader, nil)
	mockBlockStateRuntimeErr.EXPECT().GetRuntime(gomock.Any()).Return(nil, errGetRuntime)

	mockSyncedNetHappy.EXPECT().IsSynced().Return(true)
	mockBlockStateRuntimeOk.EXPECT().BestBlockHeader().Return(testEmptyHeader, nil)
	mockBlockStateRuntimeOk.EXPECT().GetRuntime(gomock.Any()).Return(runtimeMock, nil)
	mockSyncedNetHappy.EXPECT().ReportPeer(peerset.ReputationChange{
		Value:  peerset.GoodTransactionValue,
		Reason: peerset.GoodTransactionReason,
	}, peer.ID("jimbo"))

	type args struct {
		peerID peer.ID
		msg    *network.TransactionMessage
	}
	tests := []struct {
		name    string
		service *Service
		args    args
		exp     bool
		expErr  error
	}{
		{
			name:    "not synced",
			service: &Service{net: mockNotSyncedNet},
			args:    args{},
			exp:     false,
		},
		{
			name: "best block header error",
			service: &Service{
				net:        mockSyncedNet1,
				blockState: mockBlockStateBestHeadErr,
			},
			args: args{
				msg: &network.TransactionMessage{Extrinsics: []types.Extrinsic{}},
			},
			expErr: errBestHeader,
		},
		{
			name: "get runtime error",
			service: &Service{
				net:        mockSyncedNet2,
				blockState: mockBlockStateRuntimeErr,
			},
			args: args{
				msg: &network.TransactionMessage{Extrinsics: []types.Extrinsic{}},
			},
			expErr: errGetRuntime,
		},
		{
			name: "happy path no loop",
			service: &Service{
				net:        mockSyncedNetHappy,
				blockState: mockBlockStateRuntimeOk,
			},
			args: args{
				peerID: peer.ID("jimbo"),
				msg:    &network.TransactionMessage{Extrinsics: []types.Extrinsic{}},
			},
			exp: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := tt.service
			res, err := s.HandleTransactionMessage(tt.args.peerID, tt.args.msg)
			if tt.expErr != nil {
				assert.EqualError(t, err, tt.expErr.Error())
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.exp, res)
		})
	}
}
