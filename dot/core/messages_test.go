// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package core

import (
	"errors"
	"testing"

	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/dot/peerset"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/runtime"
	"github.com/ChainSafe/gossamer/lib/runtime/storage"
	"github.com/ChainSafe/gossamer/lib/transaction"
	"github.com/ChainSafe/gossamer/lib/trie"

	"github.com/golang/mock/gomock"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/assert"
)

var errDummyErr = errors.New("dummy error for testing")

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

func Test_Service_HandleTransactionMessage(t *testing.T) {
	t.Parallel()

	errTest := errors.New("test error")

	someHeader := &types.Header{
		Number:    2,
		StateRoot: common.Hash{2},
	}
	someHeaderHash := someHeader.Hash()

	testCases := map[string]struct {
		serviceBuilder        func(ctrl *gomock.Controller) *Service
		peerID                peer.ID
		message               *network.TransactionMessage
		propagateTransactions bool
		errSentinel           error
		errMessage            string
		expectedMessage       *network.TransactionMessage
	}{
		"not_synced": {
			serviceBuilder: func(ctrl *gomock.Controller) *Service {
				network := NewMockNetwork(ctrl)
				network.EXPECT().IsSynced().Return(false)
				return &Service{
					net: network,
				}
			},
		},
		"best_block_header_error": {
			serviceBuilder: func(ctrl *gomock.Controller) *Service {
				network := NewMockNetwork(ctrl)
				network.EXPECT().IsSynced().Return(true)

				blockState := NewMockBlockState(ctrl)
				blockState.EXPECT().BestBlockHeader().
					Return(nil, errTest)

				return &Service{
					net:        network,
					blockState: blockState,
				}
			},
			message:         &network.TransactionMessage{},
			errSentinel:     errTest,
			errMessage:      "getting best block header: test error",
			expectedMessage: &network.TransactionMessage{},
		},
		"get_runtime_error": {
			serviceBuilder: func(ctrl *gomock.Controller) *Service {
				network := NewMockNetwork(ctrl)
				network.EXPECT().IsSynced().Return(true)

				blockState := NewMockBlockState(ctrl)
				blockState.EXPECT().BestBlockHeader().
					Return(someHeader, nil)
				blockState.EXPECT().GetRuntime(someHeaderHash).
					Return(nil, errTest)

				return &Service{
					net:        network,
					blockState: blockState,
				}
			},
			message:         &network.TransactionMessage{},
			errSentinel:     errTest,
			errMessage:      "getting runtime from block state: test error",
			expectedMessage: &network.TransactionMessage{},
		},
		"zero_transaction": {
			serviceBuilder: func(ctrl *gomock.Controller) *Service {
				network := NewMockNetwork(ctrl)
				network.EXPECT().IsSynced().Return(true)

				blockState := NewMockBlockState(ctrl)
				blockState.EXPECT().BestBlockHeader().
					Return(someHeader, nil)
				blockState.EXPECT().GetRuntime(someHeaderHash).
					Return(nil, nil)

				network.EXPECT().ReportPeer(peerset.ReputationChange{
					Value:  peerset.GoodTransactionValue,
					Reason: peerset.GoodTransactionReason,
				}, peer.ID("a"))

				return &Service{
					net:        network,
					blockState: blockState,
				}
			},
			peerID:          peer.ID("a"),
			message:         &network.TransactionMessage{},
			expectedMessage: &network.TransactionMessage{},
		},
		"valid_transaction_to_propagate": {
			serviceBuilder: func(ctrl *gomock.Controller) *Service {
				network := NewMockNetwork(ctrl)
				network.EXPECT().IsSynced().Return(true)

				blockState := NewMockBlockState(ctrl)
				blockState.EXPECT().BestBlockHeader().
					Return(someHeader, nil)
				runtimeInstance := NewMockRuntimeInstance(ctrl)
				blockState.EXPECT().GetRuntime(someHeaderHash).
					Return(runtimeInstance, nil)

				storageState := NewMockStorageState(ctrl)
				storageState.EXPECT().Lock()
				trieState := storage.NewTrieState(trie.NewEmptyTrie())
				storageState.EXPECT().TrieState(&common.Hash{2}).
					Return(trieState, nil)
				storageState.EXPECT().Unlock()

				version := runtime.Version{
					APIItems: []runtime.APIItem{{
						Name: common.MustBlake2b8([]byte("TaggedTransactionQueue")),
						Ver:  2,
					}},
				}
				runtimeInstance.EXPECT().Version().Return(version)
				runtimeInstance.EXPECT().SetContextStorage(trieState.Snapshot())
				runtimeInstance.EXPECT().SetContextStorage(nil)
				validity := &transaction.Validity{Propagate: true}
				externalExtrinsic := []byte{2, 1, 2, 3}
				runtimeInstance.EXPECT().ValidateTransaction(externalExtrinsic).
					Return(validity, nil)

				transactionState := NewMockTransactionState(ctrl)
				validTransaction := &transaction.ValidTransaction{
					Extrinsic: types.Extrinsic{1, 2, 3},
					Validity:  validity,
				}
				transactionState.EXPECT().AddToPool(validTransaction).
					Return(common.Hash{3})

				network.EXPECT().ReportPeer(peerset.ReputationChange{
					Value:  peerset.GoodTransactionValue,
					Reason: peerset.GoodTransactionReason,
				}, peer.ID("a"))

				return &Service{
					net:              network,
					blockState:       blockState,
					storageState:     storageState,
					transactionState: transactionState,
				}
			},
			peerID: peer.ID("a"),
			message: &network.TransactionMessage{
				Extrinsics: []types.Extrinsic{{1, 2, 3}},
			},
			propagateTransactions: true,
			expectedMessage: &network.TransactionMessage{
				Extrinsics: []types.Extrinsic{{1, 2, 3}},
			},
		},
		"valid_transaction_to_not_propagate": {
			serviceBuilder: func(ctrl *gomock.Controller) *Service {
				network := NewMockNetwork(ctrl)
				network.EXPECT().IsSynced().Return(true)

				blockState := NewMockBlockState(ctrl)
				blockState.EXPECT().BestBlockHeader().
					Return(someHeader, nil)
				runtimeInstance := NewMockRuntimeInstance(ctrl)
				blockState.EXPECT().GetRuntime(someHeaderHash).
					Return(runtimeInstance, nil)

				storageState := NewMockStorageState(ctrl)
				storageState.EXPECT().Lock()
				trieState := storage.NewTrieState(trie.NewEmptyTrie())
				storageState.EXPECT().TrieState(&common.Hash{2}).
					Return(trieState, nil)
				storageState.EXPECT().Unlock()

				version := runtime.Version{
					APIItems: []runtime.APIItem{{
						Name: common.MustBlake2b8([]byte("TaggedTransactionQueue")),
						Ver:  2,
					}},
				}
				runtimeInstance.EXPECT().Version().Return(version)
				runtimeInstance.EXPECT().SetContextStorage(trieState.Snapshot())
				runtimeInstance.EXPECT().SetContextStorage(nil)
				validity := &transaction.Validity{}
				externalExtrinsic := []byte{2, 1, 2, 3}
				runtimeInstance.EXPECT().ValidateTransaction(externalExtrinsic).
					Return(validity, nil)

				transactionState := NewMockTransactionState(ctrl)
				validTransaction := &transaction.ValidTransaction{
					Extrinsic: types.Extrinsic{1, 2, 3},
					Validity:  validity,
				}
				transactionState.EXPECT().AddToPool(validTransaction).
					Return(common.Hash{3})

				network.EXPECT().ReportPeer(peerset.ReputationChange{
					Value:  peerset.GoodTransactionValue,
					Reason: peerset.GoodTransactionReason,
				}, peer.ID("a"))

				return &Service{
					net:              network,
					blockState:       blockState,
					storageState:     storageState,
					transactionState: transactionState,
				}
			},
			peerID: peer.ID("a"),
			message: &network.TransactionMessage{
				Extrinsics: []types.Extrinsic{{1, 2, 3}},
			},
			expectedMessage: &network.TransactionMessage{},
		},
		"invalid_transaction": {
			serviceBuilder: func(ctrl *gomock.Controller) *Service {
				network := NewMockNetwork(ctrl)
				network.EXPECT().IsSynced().Return(true)

				blockState := NewMockBlockState(ctrl)
				blockState.EXPECT().BestBlockHeader().
					Return(someHeader, nil)
				runtimeInstance := NewMockRuntimeInstance(ctrl)
				blockState.EXPECT().GetRuntime(someHeaderHash).
					Return(runtimeInstance, nil)

				storageState := NewMockStorageState(ctrl)
				storageState.EXPECT().Lock()
				trieState := storage.NewTrieState(trie.NewEmptyTrie())
				storageState.EXPECT().TrieState(&common.Hash{2}).
					Return(trieState, nil)
				storageState.EXPECT().Unlock()

				version := runtime.Version{
					APIItems: []runtime.APIItem{{
						Name: common.MustBlake2b8([]byte("TaggedTransactionQueue")),
						Ver:  2,
					}},
				}
				runtimeInstance.EXPECT().Version().Return(version)
				runtimeInstance.EXPECT().SetContextStorage(trieState.Snapshot())
				externalExtrinsic := []byte{2, 1, 2, 3}
				runtimeInstance.EXPECT().ValidateTransaction(externalExtrinsic).
					Return(nil, runtime.InvalidTransaction{})

				network.EXPECT().ReportPeer(peerset.ReputationChange{
					Value:  peerset.BadTransactionValue,
					Reason: peerset.BadTransactionReason,
				}, peer.ID("a"))

				return &Service{
					net:          network,
					blockState:   blockState,
					storageState: storageState,
				}
			},
			peerID: peer.ID("a"),
			message: &network.TransactionMessage{
				Extrinsics: []types.Extrinsic{{1, 2, 3}},
			},
			expectedMessage: &network.TransactionMessage{},
		},
		"unknown_transaction": {
			serviceBuilder: func(ctrl *gomock.Controller) *Service {
				network := NewMockNetwork(ctrl)
				network.EXPECT().IsSynced().Return(true)

				blockState := NewMockBlockState(ctrl)
				blockState.EXPECT().BestBlockHeader().
					Return(someHeader, nil)
				runtimeInstance := NewMockRuntimeInstance(ctrl)
				blockState.EXPECT().GetRuntime(someHeaderHash).
					Return(runtimeInstance, nil)

				storageState := NewMockStorageState(ctrl)
				storageState.EXPECT().Lock()
				trieState := storage.NewTrieState(trie.NewEmptyTrie())
				storageState.EXPECT().TrieState(&common.Hash{2}).
					Return(trieState, nil)
				storageState.EXPECT().Unlock()

				version := runtime.Version{
					APIItems: []runtime.APIItem{{
						Name: common.MustBlake2b8([]byte("TaggedTransactionQueue")),
						Ver:  2,
					}},
				}
				runtimeInstance.EXPECT().Version().Return(version)
				runtimeInstance.EXPECT().SetContextStorage(trieState.Snapshot())
				externalExtrinsic := []byte{2, 1, 2, 3}
				runtimeInstance.EXPECT().ValidateTransaction(externalExtrinsic).
					Return(nil, runtime.UnknownTransaction{})

				return &Service{
					net:          network,
					blockState:   blockState,
					storageState: storageState,
				}
			},
			peerID: peer.ID("a"),
			message: &network.TransactionMessage{
				Extrinsics: []types.Extrinsic{{1, 2, 3}},
			},
			expectedMessage: &network.TransactionMessage{},
		},
		"validate_transaction_other_error": {
			serviceBuilder: func(ctrl *gomock.Controller) *Service {
				network := NewMockNetwork(ctrl)
				network.EXPECT().IsSynced().Return(true)

				blockState := NewMockBlockState(ctrl)
				blockState.EXPECT().BestBlockHeader().
					Return(someHeader, nil)
				runtimeInstance := NewMockRuntimeInstance(ctrl)
				blockState.EXPECT().GetRuntime(someHeaderHash).
					Return(runtimeInstance, nil)

				storageState := NewMockStorageState(ctrl)
				storageState.EXPECT().Lock()
				storageState.EXPECT().TrieState(&common.Hash{2}).
					Return(nil, errTest)
				storageState.EXPECT().Unlock()

				return &Service{
					net:          network,
					blockState:   blockState,
					storageState: storageState,
				}
			},
			peerID: peer.ID("a"),
			message: &network.TransactionMessage{
				Extrinsics: []types.Extrinsic{{1, 2, 3}},
			},
			errSentinel: errTest,
			errMessage: "validating transaction from peerID 2g: " +
				"getting trie state from storage: test error",
			expectedMessage: &network.TransactionMessage{},
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)

			service := testCase.serviceBuilder(ctrl)

			propagateTransactions, err := service.HandleTransactionMessage(testCase.peerID, testCase.message)

			assert.Equal(t, testCase.propagateTransactions, propagateTransactions)
			assert.ErrorIs(t, err, testCase.errSentinel)
			if testCase.errSentinel != nil {
				assert.EqualError(t, err, testCase.errMessage)
			}
		})
	}
}
