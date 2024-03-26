// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package core

import (
	"bytes"
	"errors"
	"testing"

	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/dot/peerset"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/runtime"
	"github.com/ChainSafe/gossamer/lib/runtime/storage"
	"github.com/ChainSafe/gossamer/lib/transaction"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

var errDummyErr = errors.New("dummy error for testing")

type mockReportPeer struct {
	change peerset.ReputationChange
	id     peer.ID
}

type mockNetwork struct {
	IsSynced   bool
	ReportPeer *mockReportPeer
}

type mockBestHeader struct {
	header *types.Header
	err    error
}

type mockGetRuntime struct {
	runtime runtime.Instance
	err     error
}

type mockBlockState struct {
	bestHeader         *mockBestHeader
	getRuntime         *mockGetRuntime
	callsBestBlockHash bool
}

type mockStorageState struct {
	input     *common.Hash
	trieState *storage.TrieState
	err       error
}

type mockTxnState struct {
	input *transaction.ValidTransaction
	hash  common.Hash
}

type mockSetContextStorage struct {
	trieState *storage.TrieState
}

type mockValidateTxn struct {
	input    types.Extrinsic
	validity *transaction.Validity
	err      error
}

type mockRuntime struct {
	runtime           *MockInstance
	setContextStorage *mockSetContextStorage
	validateTxn       *mockValidateTxn
}

func TestService_TransactionsCount(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockTxnStateEmpty := NewMockTransactionState(ctrl)
	mockTxnState := NewMockTransactionState(ctrl)

	txs := []*transaction.ValidTransaction{nil, nil}

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
	testExtrinsic := []types.Extrinsic{{1, 2, 3}}

	ctrl := gomock.NewController(t)
	runtimeMock := NewMockInstance(ctrl)
	runtimeMock2 := NewMockInstance(ctrl)
	runtimeMock3 := NewMockInstance(ctrl)

	invalidTransaction := runtime.NewInvalidTransaction()
	err := invalidTransaction.SetValue(runtime.Future{})
	require.NoError(t, err)

	type args struct {
		peerID peer.ID
		msg    *network.TransactionMessage
	}
	tests := []struct {
		name             string
		service          *Service
		args             args
		exp              bool
		expErr           error
		expErrMsg        string
		ctrl             *gomock.Controller
		mockNetwork      *mockNetwork
		mockBlockState   *mockBlockState
		mockStorageState *mockStorageState
		mockTxnState     *mockTxnState
		mockRuntime      *mockRuntime
	}{
		{
			name: "not_synced",
			mockNetwork: &mockNetwork{
				IsSynced: false,
			},
		},
		{
			name: "best_block_header_error",
			mockNetwork: &mockNetwork{
				IsSynced: true,
			},
			mockBlockState: &mockBlockState{
				bestHeader: &mockBestHeader{
					err: errDummyErr,
				},
			},
			args: args{
				msg: &network.TransactionMessage{Extrinsics: []types.Extrinsic{}},
			},
			expErr:    errDummyErr,
			expErrMsg: errDummyErr.Error(),
		},
		{
			name: "get_runtime_error",
			mockNetwork: &mockNetwork{
				IsSynced: true,
			},
			mockBlockState: &mockBlockState{
				bestHeader: &mockBestHeader{
					header: testEmptyHeader,
				},
				getRuntime: &mockGetRuntime{
					err: errDummyErr,
				},
			},
			args: args{
				msg: &network.TransactionMessage{Extrinsics: []types.Extrinsic{}},
			},
			expErr:    errDummyErr,
			expErrMsg: errDummyErr.Error(),
		},
		{
			name: "happy_path_no_loop",
			mockNetwork: &mockNetwork{
				IsSynced: true,
				ReportPeer: &mockReportPeer{
					change: peerset.ReputationChange{
						Value:  peerset.GoodTransactionValue,
						Reason: peerset.GoodTransactionReason,
					},
				},
			},
			mockBlockState: &mockBlockState{
				bestHeader: &mockBestHeader{
					header: testEmptyHeader,
				},
				getRuntime: &mockGetRuntime{
					runtime: runtimeMock,
				},
			},
			args: args{
				peerID: peer.ID("jimbo"),
				msg:    &network.TransactionMessage{Extrinsics: []types.Extrinsic{}},
			},
		},
		{
			name: "trie_state_error",
			mockNetwork: &mockNetwork{
				IsSynced: true,
			},
			mockBlockState: &mockBlockState{
				bestHeader: &mockBestHeader{
					header: testEmptyHeader,
				},
				getRuntime: &mockGetRuntime{
					runtime: runtimeMock,
				},
			},
			mockStorageState: &mockStorageState{
				input: &common.Hash{},
				err:   errDummyErr,
			},
			args: args{
				peerID: peer.ID("jimbo"),
				msg: &network.TransactionMessage{
					Extrinsics: []types.Extrinsic{{1, 2, 3}, {7, 8, 9, 0}, {0xa, 0xb}},
				},
			},
			expErr: errDummyErr,
			expErrMsg: "validating transaction from peerID D1KeRhQ: cannot get trie state from storage" +
				" for root 0x0000000000000000000000000000000000000000000000000000000000000000: dummy error for testing",
		},
		{
			name: "runtime.ErrInvalidTransaction",
			mockNetwork: &mockNetwork{
				IsSynced: true,
				ReportPeer: &mockReportPeer{
					change: peerset.ReputationChange{
						Value:  peerset.BadTransactionValue,
						Reason: peerset.BadTransactionReason,
					},
					id: peer.ID("jimbo"),
				},
			},
			mockBlockState: &mockBlockState{
				bestHeader: &mockBestHeader{
					header: testEmptyHeader,
				},
				getRuntime: &mockGetRuntime{
					runtime: runtimeMock2,
				},
				callsBestBlockHash: true,
			},
			mockStorageState: &mockStorageState{
				input:     &common.Hash{},
				trieState: &storage.TrieState{},
			},
			mockRuntime: &mockRuntime{
				runtime:           runtimeMock2,
				setContextStorage: &mockSetContextStorage{trieState: &storage.TrieState{}},
				validateTxn: &mockValidateTxn{
					input: types.Extrinsic(bytes.Join([][]byte{
						{byte(types.TxnExternal)},
						testExtrinsic[0],
						testEmptyHeader.StateRoot.ToBytes(),
					}, nil)),
					err: invalidTransaction,
				},
			},
			args: args{
				peerID: peer.ID("jimbo"),
				msg: &network.TransactionMessage{
					Extrinsics: []types.Extrinsic{{1, 2, 3}},
				},
			},
		},
		{
			name: "validTransaction",
			mockNetwork: &mockNetwork{
				IsSynced: true,
				ReportPeer: &mockReportPeer{
					change: peerset.ReputationChange{
						Value:  peerset.GoodTransactionValue,
						Reason: peerset.GoodTransactionReason,
					},
					id: peer.ID("jimbo"),
				},
			},
			mockBlockState: &mockBlockState{
				bestHeader: &mockBestHeader{
					header: testEmptyHeader,
				},
				getRuntime: &mockGetRuntime{
					runtime: runtimeMock3,
				},
				callsBestBlockHash: true,
			},
			mockStorageState: &mockStorageState{
				input:     &common.Hash{},
				trieState: &storage.TrieState{},
			},
			mockTxnState: &mockTxnState{
				input: transaction.NewValidTransaction(
					types.Extrinsic{1, 2, 3},
					&transaction.Validity{
						Propagate: true,
					}),
				hash: common.Hash{},
			},
			mockRuntime: &mockRuntime{
				runtime:           runtimeMock3,
				setContextStorage: &mockSetContextStorage{trieState: &storage.TrieState{}},
				validateTxn: &mockValidateTxn{
					input: types.Extrinsic(bytes.Join([][]byte{
						{byte(types.TxnExternal)},
						testExtrinsic[0],
						testEmptyHeader.StateRoot.ToBytes(),
					}, nil)),
					validity: &transaction.Validity{Propagate: true},
				},
			},
			args: args{
				peerID: peer.ID("jimbo"),
				msg: &network.TransactionMessage{
					Extrinsics: []types.Extrinsic{{1, 2, 3}},
				},
			},
			exp: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Service{}
			ctrl := gomock.NewController(t)
			if tt.mockNetwork != nil {
				mockNet := NewMockNetwork(ctrl)
				mockNet.EXPECT().IsSynced().Return(tt.mockNetwork.IsSynced)
				if tt.mockNetwork.ReportPeer != nil {
					mockNet.EXPECT().ReportPeer(tt.mockNetwork.ReportPeer.change, tt.args.peerID)
				}
				s.net = mockNet
			}
			if tt.mockBlockState != nil {
				blockState := NewMockBlockState(ctrl)
				blockState.EXPECT().BestBlockHeader().Return(
					tt.mockBlockState.bestHeader.header,
					tt.mockBlockState.bestHeader.err)

				if tt.mockBlockState.getRuntime != nil {
					blockState.EXPECT().GetRuntime(gomock.Any()).Return(
						tt.mockBlockState.getRuntime.runtime,
						tt.mockBlockState.getRuntime.err)
				}
				if tt.mockBlockState.callsBestBlockHash {
					blockState.EXPECT().BestBlockHash().Return(common.Hash{})
				}
				s.blockState = blockState
			}
			if tt.mockStorageState != nil {
				storageState := NewMockStorageState(ctrl)
				storageState.EXPECT().Lock()
				storageState.EXPECT().Unlock()
				storageState.EXPECT().TrieState(tt.mockStorageState.input).Return(
					tt.mockStorageState.trieState,
					tt.mockStorageState.err)
				s.storageState = storageState
			}
			if tt.mockTxnState != nil {
				txnState := NewMockTransactionState(ctrl)
				txnState.EXPECT().AddToPool(tt.mockTxnState.input).Return(tt.mockTxnState.hash)
				s.transactionState = txnState
			}
			if tt.mockRuntime != nil {
				rt := tt.mockRuntime.runtime
				rt.EXPECT().SetContextStorage(tt.mockRuntime.setContextStorage.trieState)
				rt.EXPECT().ValidateTransaction(tt.mockRuntime.validateTxn.input).
					Return(tt.mockRuntime.validateTxn.validity, tt.mockRuntime.validateTxn.err)
				rt.EXPECT().Version().Return(runtime.Version{
					SpecName:         []byte("polkadot"),
					ImplName:         []byte("parity-polkadot"),
					AuthoringVersion: authoringVersion,
					SpecVersion:      specVersion,
					ImplVersion:      implVersion,
					APIItems: []runtime.APIItem{{
						Name: common.MustBlake2b8([]byte("TaggedTransactionQueue")),
						Ver:  3,
					}},
					TransactionVersion: transactionVersion,
					StateVersion:       stateVersion,
				}, nil)
			}

			res, err := s.HandleTransactionMessage(tt.args.peerID, tt.args.msg)
			assert.ErrorIs(t, err, tt.expErr)
			if tt.expErr != nil {
				assert.EqualError(t, err, tt.expErrMsg)
			}
			assert.Equal(t, tt.exp, res)
		})
	}
}
