// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package core

import (
	"errors"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/runtime"
	"github.com/ChainSafe/gossamer/lib/runtime/storage"

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
	errTrieState  = errors.New("trie state error")
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
	mockSyncedNet3 := NewMockNetwork(ctrl)
	mockSyncedNet4 := NewMockNetwork(ctrl)

	// BlockState
	mockBlockStateBestHeadErr := NewMockBlockState(ctrl)
	mockBlockStateRuntimeErr := NewMockBlockState(ctrl)
	mockBlockStateRuntimeOk1 := NewMockBlockState(ctrl)
	mockBlockStateRuntimeOk2 := NewMockBlockState(ctrl)
	mockBlockStateRuntimeOk3 := NewMockBlockState(ctrl)

	// Storage
	mockStorageStateTrieStateErr := NewMockStorageState(ctrl)
	mockStorageStateTrieState := NewMockStorageState(ctrl)

	// Runtime
	runtimeMock := new(mocksruntime.Instance)
	runtimeMock2 := new(mocksruntime.Instance)

	mockNotSyncedNet.EXPECT().IsSynced().Return(false)

	mockSyncedNet1.EXPECT().IsSynced().Return(true)
	mockBlockStateBestHeadErr.EXPECT().BestBlockHeader().Return(nil, errBestHeader)

	mockSyncedNet2.EXPECT().IsSynced().Return(true)
	mockBlockStateRuntimeErr.EXPECT().BestBlockHeader().Return(testEmptyHeader, nil)
	mockBlockStateRuntimeErr.EXPECT().GetRuntime(gomock.Any()).Return(nil, errGetRuntime)

	mockSyncedNetHappy.EXPECT().IsSynced().Return(true)
	mockBlockStateRuntimeOk1.EXPECT().BestBlockHeader().Return(testEmptyHeader, nil)
	mockBlockStateRuntimeOk1.EXPECT().GetRuntime(gomock.Any()).Return(runtimeMock, nil)
	mockSyncedNetHappy.EXPECT().ReportPeer(peerset.ReputationChange{
		Value:  peerset.GoodTransactionValue,
		Reason: peerset.GoodTransactionReason,
	}, peer.ID("jimbo"))

	//Enter loop
	mockSyncedNet3.EXPECT().IsSynced().Return(true)
	mockBlockStateRuntimeOk2.EXPECT().BestBlockHeader().Return(testEmptyHeader, nil)
	mockBlockStateRuntimeOk2.EXPECT().GetRuntime(gomock.Any()).Return(runtimeMock, nil)
	mockStorageStateTrieStateErr.EXPECT().Lock()
	mockStorageStateTrieStateErr.EXPECT().Unlock()
	mockStorageStateTrieStateErr.EXPECT().TrieState(&common.Hash{}).Return(nil, errTrieState)

	// invalid txn report peer runtime.ErrInvalidTransaction
	test := []types.Extrinsic{{1, 2, 3}}
	mockSyncedNet4.EXPECT().IsSynced().Return(true)
	mockBlockStateRuntimeOk3.EXPECT().BestBlockHeader().Return(testEmptyHeader, nil)
	mockBlockStateRuntimeOk3.EXPECT().GetRuntime(gomock.Any()).Return(runtimeMock2, nil)
	mockStorageStateTrieState.EXPECT().Lock()
	mockStorageStateTrieState.EXPECT().Unlock()
	mockStorageStateTrieState.EXPECT().TrieState(&common.Hash{}).Return(&storage.TrieState{}, nil)
	runtimeMock2.On("SetContextStorage", &storage.TrieState{})
	runtimeMock2.On("ValidateTransaction", types.Extrinsic(append([]byte{byte(types.TxnExternal)}, test[0]...))).Return(nil, runtime.ErrInvalidTransaction)
	mockSyncedNet4.EXPECT().ReportPeer(peerset.ReputationChange{
		Value:  peerset.BadTransactionValue,
		Reason: peerset.BadTransactionReason,
	}, peer.ID("jimbo"))
	mockSyncedNet4.EXPECT().ReportPeer(peerset.ReputationChange{
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
				blockState: mockBlockStateRuntimeOk1,
			},
			args: args{
				peerID: peer.ID("jimbo"),
				msg:    &network.TransactionMessage{Extrinsics: []types.Extrinsic{}},
			},
			exp: false,
		},
		{
			name: "trie state error",
			service: &Service{
				net:          mockSyncedNet3,
				blockState:   mockBlockStateRuntimeOk2,
				storageState: mockStorageStateTrieStateErr,
			},
			args: args{
				peerID: peer.ID("jimbo"),
				msg: &network.TransactionMessage{
					Extrinsics: []types.Extrinsic{{1, 2, 3}, {7, 8, 9, 0}, {0xa, 0xb}},
				},
			},
			expErr: errTrieState,
		},
		{
			name: "runtime.ErrInvalidTransaction",
			service: &Service{
				net:          mockSyncedNet4,
				blockState:   mockBlockStateRuntimeOk3,
				storageState: mockStorageStateTrieState,
			},
			args: args{
				peerID: peer.ID("jimbo"),
				msg: &network.TransactionMessage{
					Extrinsics: []types.Extrinsic{{1, 2, 3}},
				},
			},
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
