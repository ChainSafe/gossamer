// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package sync

import (
	"context"
	"errors"
	mocksruntime "github.com/ChainSafe/gossamer/lib/runtime/mocks"
	"testing"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/blocktree"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/runtime/storage"
	"github.com/ChainSafe/gossamer/lib/trie"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

//go:generate mockgen -destination=mock_instance_test.go -package=$GOPACKAGE github.com/ChainSafe/gossamer/lib/runtime Instance

func Test_chainProcessor_handleBlock(t *testing.T) {
	t.Parallel()
	mockError := errors.New("test mock error")
	testHash := common.MustHexToHash("0x03170a2e7597b7b7e3d84c05391d139a62b157e78786d8c082f29dcf4c111314")
	testParentHash := common.MustHexToHash("0x7db9db5ed9967b80143100189ba69d9e4deab85ac3570e5df25686cabe32964a")

	tests := map[string]struct {
		chainProcessorBuilder func(ctrl *gomock.Controller) chainProcessor
		block                 *types.Block
		wantErr               error
	}{
		"handle getHeader error": {
			chainProcessorBuilder: func(ctrl *gomock.Controller) (chainProcessor chainProcessor) {
				mockBlockState := NewMockBlockState(ctrl)
				mockBlockState.EXPECT().GetHeader(common.Hash{}).Return(nil, mockError)
				chainProcessor.blockState = mockBlockState
				return
			},
			block: &types.Block{
				Body: types.Body{},
			},
			wantErr: errFailedToGetParent,
		},
		"handle trieState error": {
			chainProcessorBuilder: func(ctrl *gomock.Controller) (chainProcessor chainProcessor) {
				mockBlockState := NewMockBlockState(ctrl)
				mockBlockState.EXPECT().GetHeader(common.Hash{}).Return(&types.Header{}, nil)
				chainProcessor.blockState = mockBlockState
				mockStorageState := NewMockStorageState(ctrl)
				mockStorageState.EXPECT().Lock()
				mockStorageState.EXPECT().TrieState(&common.Hash{}).Return(nil, mockError)
				mockStorageState.EXPECT().Unlock()
				chainProcessor.storageState = mockStorageState
				return
			},
			block: &types.Block{
				Body: types.Body{},
			},
			wantErr: mockError,
		},
		"handle getRuntime error": {
			chainProcessorBuilder: func(ctrl *gomock.Controller) (chainProcessor chainProcessor) {
				mockBlockState := NewMockBlockState(ctrl)
				mockBlockState.EXPECT().GetHeader(common.Hash{}).Return(&types.Header{
					StateRoot: testHash,
				}, nil)
				mockBlockState.EXPECT().GetRuntime(&testParentHash).Return(nil, mockError)
				chainProcessor.blockState = mockBlockState
				trieState := storage.NewTrieState(nil)
				mockStorageState := NewMockStorageState(ctrl)
				mockStorageState.EXPECT().Lock()
				mockStorageState.EXPECT().TrieState(&testHash).Return(trieState, nil)
				mockStorageState.EXPECT().Unlock()
				chainProcessor.storageState = mockStorageState
				return
			},
			block: &types.Block{
				Body: types.Body{},
			},
			wantErr: mockError,
		},
		"handle runtime ExecuteBlock error": {
			chainProcessorBuilder: func(ctrl *gomock.Controller) (chainProcessor chainProcessor) {
				trieState := storage.NewTrieState(nil)
				mockBlockState := NewMockBlockState(ctrl)
				mockBlockState.EXPECT().GetHeader(common.Hash{}).Return(&types.Header{
					StateRoot: testHash,
				}, nil)
				mockInstance := mocksruntime.NewInstance(t)
				mockInstance.On("SetContextStorage", trieState)
				mockInstance.On("ExecuteBlock", &types.Block{Body: types.Body{}}).
					Return(nil, mockError)
				mockBlockState.EXPECT().GetRuntime(&testParentHash).Return(mockInstance, nil)
				chainProcessor.blockState = mockBlockState
				mockStorageState := NewMockStorageState(ctrl)
				mockStorageState.EXPECT().Lock()
				mockStorageState.EXPECT().TrieState(&testHash).Return(trieState, nil)
				mockStorageState.EXPECT().Unlock()
				chainProcessor.storageState = mockStorageState
				return
			},
			block: &types.Block{
				Body: types.Body{},
			},
			wantErr: mockError,
		},
		"handle block import error": {
			chainProcessorBuilder: func(ctrl *gomock.Controller) (chainProcessor chainProcessor) {
				trieState := storage.NewTrieState(nil)
				mockBlockState := NewMockBlockState(ctrl)
				mockBlockState.EXPECT().GetHeader(common.Hash{}).Return(&types.Header{
					StateRoot: testHash,
				}, nil)
				mockBlock := &types.Block{Body: types.Body{}}
				mockInstance := mocksruntime.NewInstance(t)
				mockInstance.On("SetContextStorage", trieState)
				mockInstance.On("ExecuteBlock", mockBlock).Return(nil, nil)
				mockBlockState.EXPECT().GetRuntime(&testParentHash).Return(mockInstance, nil)
				chainProcessor.blockState = mockBlockState
				mockStorageState := NewMockStorageState(ctrl)
				mockStorageState.EXPECT().Lock()
				mockStorageState.EXPECT().TrieState(&testHash).Return(trieState, nil)
				mockStorageState.EXPECT().Unlock()
				chainProcessor.storageState = mockStorageState
				mockBlockImportHandler := NewMockBlockImportHandler(ctrl)
				mockBlockImportHandler.EXPECT().HandleBlockImport(mockBlock,
					trieState).Return(mockError)
				chainProcessor.blockImportHandler = mockBlockImportHandler
				return
			},
			block: &types.Block{
				Body: types.Body{},
			},
			wantErr: mockError,
		},
		"base case": {
			chainProcessorBuilder: func(ctrl *gomock.Controller) (chainProcessor chainProcessor) {
				mockBlock := &types.Block{
					Body: types.Body{}, // empty slice of extrinsics
				}
				trieState := storage.NewTrieState(nil)
				mockBlockState := NewMockBlockState(ctrl)
				mockHeader := &types.Header{
					Number:    0,
					StateRoot: trie.EmptyHash,
				}
				mockHeaderHash := mockHeader.Hash()
				mockBlockState.EXPECT().GetHeader(common.Hash{}).Return(mockHeader, nil)

				mockInstance := mocksruntime.NewInstance(t)
				mockInstance.On("SetContextStorage", trieState)
				mockInstance.On("ExecuteBlock", mockBlock).Return(nil, nil)
				mockBlockState.EXPECT().GetRuntime(&mockHeaderHash).Return(mockInstance, nil)
				chainProcessor.blockState = mockBlockState
				mockStorageState := NewMockStorageState(ctrl)
				mockStorageState.EXPECT().Lock()
				mockStorageState.EXPECT().Unlock()
				mockStorageState.EXPECT().TrieState(&trie.EmptyHash).Return(trieState, nil)
				chainProcessor.storageState = mockStorageState
				mockBlockImportHandler := NewMockBlockImportHandler(ctrl)
				mockBlockImportHandler.EXPECT().HandleBlockImport(mockBlock, trieState).Return(nil)
				chainProcessor.blockImportHandler = mockBlockImportHandler
				mockTelemetry := NewMockClient(ctrl)
				mockTelemetry.EXPECT().SendMessage(gomock.Any())
				chainProcessor.telemetry = mockTelemetry
				return
			},
			block: &types.Block{
				Header: types.Header{
					Number: 0,
				},
				Body: types.Body{},
			},
		},
	}
	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			s := tt.chainProcessorBuilder(ctrl)

			err := s.handleBlock(tt.block)
			assert.ErrorIs(t, err, tt.wantErr)
		})
	}
	t.Run("panics on different parent state root", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		bock := &types.Block{
			Header: types.Header{
				ParentHash: common.Hash{1},
			},
		}
		blockState := NewMockBlockState(ctrl)
		blockState.EXPECT().GetHeader(common.Hash{1}).
			Return(&types.Header{StateRoot: common.Hash{2}}, nil)
		trieState := storage.NewTrieState(nil)
		storageState := NewMockStorageState(ctrl)
		lockCall := storageState.EXPECT().Lock()
		trieStateCall := storageState.EXPECT().TrieState(&common.Hash{2}).
			Return(trieState, nil).After(lockCall)
		storageState.EXPECT().Unlock().After(trieStateCall)
		chainProcessor := &chainProcessor{
			blockState:   blockState,
			storageState: storageState,
		}
		const expectedPanicValue = "parent state root does not match snapshot state root"
		assert.PanicsWithValue(t, expectedPanicValue, func() {
			_ = chainProcessor.handleBlock(bock)
		})
	})
}

func Test_chainProcessor_handleBody(t *testing.T) {
	t.Parallel()

	testExtrinsics := []types.Extrinsic{{1, 2, 3}, {7, 8, 9, 0}, {0xa, 0xb}}
	testBody := types.NewBody(testExtrinsics)

	t.Run("base case", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)

		mockTransactionState := NewMockTransactionState(ctrl)
		mockTransactionState.EXPECT().RemoveExtrinsic(testExtrinsics[0])
		mockTransactionState.EXPECT().RemoveExtrinsic(testExtrinsics[1])
		mockTransactionState.EXPECT().RemoveExtrinsic(testExtrinsics[2])
		processor := chainProcessor{
			transactionState: mockTransactionState,
		}
		processor.handleBody(testBody)
	})
}

func Test_chainProcessor_handleJustification(t *testing.T) {
	t.Parallel()

	expectedHash := common.MustHexToHash("0xdcdd89927d8a348e00257e1ecc8617f45edb5118efff3ea2f9961b2ad9b7690a")

	type args struct {
		header        *types.Header
		justification []byte
	}
	tests := map[string]struct {
		chainProcessorBuilder func(ctrl *gomock.Controller) chainProcessor
		args                  args
	}{
		"nil justification and header": {
			chainProcessorBuilder: func(ctrl *gomock.Controller) chainProcessor {
				return chainProcessor{}
			},
		},
		"invalid justification": {
			chainProcessorBuilder: func(ctrl *gomock.Controller) chainProcessor {
				mockFinalityGadget := NewMockFinalityGadget(ctrl)
				mockFinalityGadget.EXPECT().VerifyBlockJustification(expectedHash,
					[]byte(`x`)).Return(nil, errors.New("error"))
				return chainProcessor{
					finalityGadget: mockFinalityGadget,
				}
			},
			args: args{
				header: &types.Header{
					Number: 0,
				},
				justification: []byte(`x`),
			},
		},
		"set justification error": {
			chainProcessorBuilder: func(ctrl *gomock.Controller) chainProcessor {
				mockBlockState := NewMockBlockState(ctrl)
				mockBlockState.EXPECT().SetJustification(expectedHash, []byte(`xx`)).Return(errors.New("fake error"))
				mockFinalityGadget := NewMockFinalityGadget(ctrl)
				mockFinalityGadget.EXPECT().VerifyBlockJustification(expectedHash, []byte(`xx`)).Return([]byte(`xx`), nil)
				return chainProcessor{
					blockState:     mockBlockState,
					finalityGadget: mockFinalityGadget,
				}
			},
			args: args{
				header: &types.Header{
					Number: 0,
				},
				justification: []byte(`xx`),
			},
		},
		"base case set": {
			chainProcessorBuilder: func(ctrl *gomock.Controller) chainProcessor {
				mockBlockState := NewMockBlockState(ctrl)
				mockBlockState.EXPECT().SetJustification(expectedHash, []byte(`1234`)).Return(nil)
				mockFinalityGadget := NewMockFinalityGadget(ctrl)
				mockFinalityGadget.EXPECT().VerifyBlockJustification(expectedHash, []byte(`1234`)).Return([]byte(`1234`), nil)
				return chainProcessor{
					blockState:     mockBlockState,
					finalityGadget: mockFinalityGadget,
				}
			},
			args: args{
				header: &types.Header{
					Number: 0,
				},
				justification: []byte(`1234`),
			},
		},
	}
	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			s := tt.chainProcessorBuilder(ctrl)
			s.handleJustification(tt.args.header, tt.args.justification)
		})
	}
}

func Test_chainProcessor_processBlockData(t *testing.T) {
	t.Parallel()

	mockError := errors.New("mock test error")
	justification := []byte{0, 1, 2}

	tests := map[string]struct {
		chainProcessorBuilder func(ctrl *gomock.Controller) chainProcessor
		blockData             *types.BlockData
		expectedError         error
	}{
		"nil block data": {
			chainProcessorBuilder: func(ctrl *gomock.Controller) chainProcessor {
				return chainProcessor{}
			},
			blockData:     nil,
			expectedError: ErrNilBlockData,
		},
		"handle has header error": {
			chainProcessorBuilder: func(ctrl *gomock.Controller) chainProcessor {
				mockBlockState := NewMockBlockState(ctrl)
				mockBlockState.EXPECT().HasHeader(common.Hash{}).Return(false, mockError)
				return chainProcessor{
					blockState: mockBlockState,
				}
			},
			blockData:     &types.BlockData{},
			expectedError: mockError,
		},
		"handle has block body error": {
			chainProcessorBuilder: func(ctrl *gomock.Controller) chainProcessor {
				mockBlockState := NewMockBlockState(ctrl)
				mockBlockState.EXPECT().HasHeader(common.Hash{}).Return(false, nil)
				mockBlockState.EXPECT().HasBlockBody(common.Hash{}).Return(false, mockError)
				return chainProcessor{
					blockState: mockBlockState,
				}
			},
			blockData:     &types.BlockData{},
			expectedError: mockError,
		},
		"handle getBlockByHash error": {
			chainProcessorBuilder: func(ctrl *gomock.Controller) chainProcessor {
				mockBlockState := NewMockBlockState(ctrl)
				mockBlockState.EXPECT().HasHeader(common.Hash{}).Return(true, nil)
				mockBlockState.EXPECT().HasBlockBody(common.Hash{}).Return(true, nil)
				mockBlockState.EXPECT().GetBlockByHash(common.Hash{}).Return(nil, mockError)
				return chainProcessor{
					blockState: mockBlockState,
				}
			},
			blockData:     &types.BlockData{},
			expectedError: mockError,
		},
		"handle AddBlockToBlockTree error": {
			chainProcessorBuilder: func(ctrl *gomock.Controller) chainProcessor {
				mockBlock := &types.Block{
					Header: types.Header{
						Number: uint(1),
					},
				}
				mockBlockState := NewMockBlockState(ctrl)
				mockBlockState.EXPECT().HasHeader(common.Hash{}).Return(true, nil)
				mockBlockState.EXPECT().HasBlockBody(common.Hash{}).Return(true, nil)
				mockBlockState.EXPECT().GetBlockByHash(common.Hash{}).Return(mockBlock, nil)
				mockBlockState.EXPECT().AddBlockToBlockTree(&types.Block{
					Header: types.Header{Number: 1}}).Return(blocktree.ErrBlockExists)
				mockFinalityGadget := NewMockFinalityGadget(ctrl)
				mockStorageState := NewMockStorageState(ctrl)
				mockBlockImportHandler := NewMockBlockImportHandler(ctrl)
				return chainProcessor{
					blockState:         mockBlockState,
					finalityGadget:     mockFinalityGadget,
					storageState:       mockStorageState,
					blockImportHandler: mockBlockImportHandler,
				}
			},
			blockData: &types.BlockData{
				Justification: &[]byte{1, 2, 3},
			},
		},
		"handle block data justification != nil": {
			chainProcessorBuilder: func(ctrl *gomock.Controller) chainProcessor {
				mockBlock := &types.Block{
					Header: types.Header{
						Number: uint(1),
					},
				}
				mockBlockState := NewMockBlockState(ctrl)
				mockBlockState.EXPECT().HasHeader(common.Hash{}).Return(true, nil)
				mockBlockState.EXPECT().HasBlockBody(common.Hash{}).Return(true, nil)
				mockBlockState.EXPECT().GetBlockByHash(common.Hash{}).Return(mockBlock, nil)
				mockBlockState.EXPECT().AddBlockToBlockTree(&types.Block{
					Header: types.Header{Number: 1}}).Return(nil)
				mockBlockState.EXPECT().SetJustification(common.MustHexToHash(
					"0x6443a0b46e0412e626363028115a9f2cf963eeed526b8b33e5316f08b50d0dc3"), []byte{1, 2, 3})
				mockFinalityGadget := NewMockFinalityGadget(ctrl)
				mockFinalityGadget.EXPECT().VerifyBlockJustification(common.MustHexToHash(
					"0x6443a0b46e0412e626363028115a9f2cf963eeed526b8b33e5316f08b50d0dc3"), []byte{1, 2,
					3}).Return([]byte{1, 2, 3}, nil)
				mockStorageState := NewMockStorageState(ctrl)
				mockStorageState.EXPECT().TrieState(&common.Hash{}).Return(nil, nil)
				mockBlockImportHandler := NewMockBlockImportHandler(ctrl)
				mockBlockImportHandler.EXPECT().HandleBlockImport(mockBlock,
					nil).Return(nil)
				return chainProcessor{
					blockState:         mockBlockState,
					finalityGadget:     mockFinalityGadget,
					storageState:       mockStorageState,
					blockImportHandler: mockBlockImportHandler,
				}
			},
			blockData: &types.BlockData{
				Justification: &[]byte{1, 2, 3},
			},
		},
		"handle trie state error": {
			chainProcessorBuilder: func(ctrl *gomock.Controller) chainProcessor {
				mockBlock := &types.Block{
					Header: types.Header{
						Number: uint(1),
					},
				}
				mockBlockState := NewMockBlockState(ctrl)
				mockBlockState.EXPECT().HasHeader(common.Hash{}).Return(true, nil)
				mockBlockState.EXPECT().HasBlockBody(common.Hash{}).Return(true, nil)
				mockBlockState.EXPECT().GetBlockByHash(common.Hash{}).Return(mockBlock, nil)
				mockBlockState.EXPECT().AddBlockToBlockTree(&types.Block{
					Header: types.Header{Number: 1}}).Return(nil)
				mockBlockState.EXPECT().SetJustification(common.MustHexToHash(
					"0x6443a0b46e0412e626363028115a9f2cf963eeed526b8b33e5316f08b50d0dc3"), []byte{1, 2, 3})
				mockFinalityGadget := NewMockFinalityGadget(ctrl)
				mockFinalityGadget.EXPECT().VerifyBlockJustification(common.MustHexToHash(
					"0x6443a0b46e0412e626363028115a9f2cf963eeed526b8b33e5316f08b50d0dc3"), []byte{1, 2,
					3}).Return([]byte{1, 2, 3}, nil)
				mockStorageState := NewMockStorageState(ctrl)
				mockStorageState.EXPECT().TrieState(&common.Hash{}).Return(nil, mockError)
				return chainProcessor{
					blockState:     mockBlockState,
					finalityGadget: mockFinalityGadget,
					storageState:   mockStorageState,
				}
			},
			blockData: &types.BlockData{
				Justification: &[]byte{1, 2, 3},
			},
			expectedError: mockError,
		},
		"handle block import handler error": {
			chainProcessorBuilder: func(ctrl *gomock.Controller) chainProcessor {
				mockBlock := &types.Block{
					Header: types.Header{
						Number: uint(1),
					},
				}
				mockBlockState := NewMockBlockState(ctrl)
				mockBlockState.EXPECT().HasHeader(common.Hash{}).Return(true, nil)
				mockBlockState.EXPECT().HasBlockBody(common.Hash{}).Return(true, nil)
				mockBlockState.EXPECT().GetBlockByHash(common.Hash{}).Return(mockBlock, nil)
				mockBlockState.EXPECT().AddBlockToBlockTree(&types.Block{
					Header: types.Header{Number: 1}}).Return(nil)
				mockBlockState.EXPECT().SetJustification(common.MustHexToHash(
					"0x6443a0b46e0412e626363028115a9f2cf963eeed526b8b33e5316f08b50d0dc3"), []byte{1, 2, 3})
				mockFinalityGadget := NewMockFinalityGadget(ctrl)
				mockFinalityGadget.EXPECT().VerifyBlockJustification(common.MustHexToHash(
					"0x6443a0b46e0412e626363028115a9f2cf963eeed526b8b33e5316f08b50d0dc3"), []byte{1, 2,
					3}).Return([]byte{1, 2, 3}, nil)
				mockStorageState := NewMockStorageState(ctrl)
				mockStorageState.EXPECT().TrieState(&common.Hash{}).Return(nil, nil)
				mockBlockImportHandler := NewMockBlockImportHandler(ctrl)
				mockBlockImportHandler.EXPECT().HandleBlockImport(mockBlock,
					nil).Return(mockError)
				return chainProcessor{
					blockState:         mockBlockState,
					finalityGadget:     mockFinalityGadget,
					storageState:       mockStorageState,
					blockImportHandler: mockBlockImportHandler,
				}
			},
			blockData: &types.BlockData{
				Justification: &[]byte{1, 2, 3},
			},
		},
		"has header body false": {
			chainProcessorBuilder: func(ctrl *gomock.Controller) chainProcessor {
				mockBlockState := NewMockBlockState(ctrl)
				mockBlockState.EXPECT().HasHeader(common.Hash{}).Return(false, nil)
				mockBlockState.EXPECT().HasBlockBody(common.Hash{}).Return(false, nil)
				mockBlockState.EXPECT().CompareAndSetBlockData(&types.BlockData{}).Return(nil)
				return chainProcessor{
					blockState: mockBlockState,
				}
			},
			blockData: &types.BlockData{},
		},
		"handle babe verify block error": {
			chainProcessorBuilder: func(ctrl *gomock.Controller) chainProcessor {
				mockBlockState := NewMockBlockState(ctrl)
				mockBlockState.EXPECT().HasHeader(common.Hash{}).Return(false, nil)
				mockBlockState.EXPECT().HasBlockBody(common.Hash{}).Return(false, nil)
				mockBabeVerifier := NewMockBabeVerifier(ctrl)
				mockBabeVerifier.EXPECT().VerifyBlock(&types.Header{}).Return(mockError)
				return chainProcessor{
					blockState:   mockBlockState,
					babeVerifier: mockBabeVerifier,
				}
			},
			blockData: &types.BlockData{
				Header: &types.Header{},
				Body:   &types.Body{},
			},
			expectedError: mockError,
		},
		"handle error with handleBlock": {
			chainProcessorBuilder: func(ctrl *gomock.Controller) chainProcessor {
				mockBlockState := NewMockBlockState(ctrl)
				mockBlockState.EXPECT().HasHeader(common.Hash{}).Return(false, nil)
				mockBlockState.EXPECT().HasBlockBody(common.Hash{}).Return(false, nil)
				mockBlockState.EXPECT().GetHeader(common.Hash{}).Return(nil, mockError)
				mockBabeVerifier := NewMockBabeVerifier(ctrl)
				mockBabeVerifier.EXPECT().VerifyBlock(&types.Header{}).Return(nil)
				return chainProcessor{
					blockState:   mockBlockState,
					babeVerifier: mockBabeVerifier,
				}
			},
			blockData: &types.BlockData{
				Header: &types.Header{},
				Body:   &types.Body{},
			},
			expectedError: errFailedToGetParent,
		},
		"error adding block": {
			chainProcessorBuilder: func(ctrl *gomock.Controller) chainProcessor {
				mockBlockState := NewMockBlockState(ctrl)
				mockBlockState.EXPECT().HasHeader(common.Hash{1, 2, 3}).Return(true, nil)
				mockBlockState.EXPECT().HasBlockBody(common.Hash{1, 2, 3}).Return(true, nil)
				mockBlockState.EXPECT().GetBlockByHash(common.Hash{1, 2, 3}).Return(&types.Block{
					Header: types.Header{
						Number: uint(1),
					},
				}, nil)
				mockBlockState.EXPECT().AddBlockToBlockTree(&types.Block{
					Header: types.Header{Number: 1}}).Return(mockError)
				return chainProcessor{
					blockState: mockBlockState,
				}
			},
			blockData: &types.BlockData{
				Hash: common.Hash{1, 2, 3},
			},
			expectedError: mockError,
		},
		"handle block import": {
			chainProcessorBuilder: func(ctrl *gomock.Controller) chainProcessor {
				mockTrieState := storage.NewTrieState(nil)
				mockBlockState := NewMockBlockState(ctrl)
				mockBlockState.EXPECT().HasHeader(common.Hash{1, 2, 3}).Return(true, nil)
				mockBlockState.EXPECT().HasBlockBody(common.Hash{1, 2, 3}).Return(true, nil)
				mockBlockState.EXPECT().GetBlockByHash(common.Hash{1, 2, 3}).Return(&types.Block{
					Header: types.Header{
						Number: uint(1),
					},
				}, nil)
				mockBlockState.EXPECT().AddBlockToBlockTree(&types.Block{Header: types.Header{Number: 1}}).Return(nil)
				mockStorageState := NewMockStorageState(ctrl)
				mockStorageState.EXPECT().TrieState(&common.Hash{}).Return(mockTrieState, nil)
				mockBlockImportHandler := NewMockBlockImportHandler(ctrl)
				mockBlockImportHandler.EXPECT().HandleBlockImport(&types.Block{Header: types.Header{Number: 1}}, mockTrieState)
				return chainProcessor{
					blockState:         mockBlockState,
					storageState:       mockStorageState,
					blockImportHandler: mockBlockImportHandler,
				}
			},
			blockData: &types.BlockData{
				Hash: common.Hash{1, 2, 3},
			},
		},
		"handle compareAndSetBlockData error": {
			chainProcessorBuilder: func(ctrl *gomock.Controller) chainProcessor {
				mockBlockState := NewMockBlockState(ctrl)
				mockBlockState.EXPECT().HasHeader(common.Hash{}).Return(false, nil)
				mockBlockState.EXPECT().HasBlockBody(common.Hash{}).Return(false, nil)
				mockBlockState.EXPECT().CompareAndSetBlockData(&types.BlockData{}).Return(mockError)
				return chainProcessor{
					blockState: mockBlockState,
				}
			},
			blockData:     &types.BlockData{},
			expectedError: mockError,
		},
		"handle header": {
			chainProcessorBuilder: func(ctrl *gomock.Controller) chainProcessor {
				stateRootHash := common.MustHexToHash("0x03170a2e7597b7b7e3d84c05391d139a62b157e78786d8c082f29dcf4c111314")
				mockTrieState := storage.NewTrieState(nil)
				runtimeHash := common.MustHexToHash("0x7db9db5ed9967b80143100189ba69d9e4deab85ac3570e5df25686cabe32964a")
				mockBlock := &types.Block{Header: types.Header{}, Body: types.Body{}}

				mockInstance := mocksruntime.NewInstance(t)
				mockInstance.On("SetContextStorage", mockTrieState)
				mockInstance.On("ExecuteBlock", mockBlock).Return(nil, nil)
				mockBlockState := NewMockBlockState(ctrl)
				mockBlockState.EXPECT().HasHeader(common.Hash{}).Return(false, nil)
				mockBlockState.EXPECT().HasBlockBody(common.Hash{}).Return(false, nil)
				mockBlockState.EXPECT().GetHeader(common.Hash{}).Return(&types.Header{
					Number:    0,
					StateRoot: stateRootHash,
				}, nil)
				mockBlockState.EXPECT().CompareAndSetBlockData(&types.BlockData{Header: &types.Header{}, Body: &types.Body{}})
				mockBlockState.EXPECT().GetRuntime(&runtimeHash).Return(mockInstance, nil)
				mockBabeVerifier := NewMockBabeVerifier(ctrl)
				mockBabeVerifier.EXPECT().VerifyBlock(&types.Header{})
				mockStorageState := NewMockStorageState(ctrl)
				mockStorageState.EXPECT().Lock()
				mockStorageState.EXPECT().TrieState(&stateRootHash).Return(mockTrieState, nil)
				mockStorageState.EXPECT().Unlock()
				mockBlockImportHandler := NewMockBlockImportHandler(ctrl)
				mockBlockImportHandler.EXPECT().HandleBlockImport(mockBlock, mockTrieState)
				mockTelemetry := NewMockClient(ctrl)
				mockTelemetry.EXPECT().SendMessage(gomock.Any()).AnyTimes()
				return chainProcessor{
					blockState:         mockBlockState,
					babeVerifier:       mockBabeVerifier,
					storageState:       mockStorageState,
					blockImportHandler: mockBlockImportHandler,
					telemetry:          mockTelemetry,
				}
			},
			blockData: &types.BlockData{
				Header: &types.Header{
					Number: 0,
				},
				Body: &types.Body{},
			},
		},
		"handle justification": {
			chainProcessorBuilder: func(ctrl *gomock.Controller) chainProcessor {
				stateRootHash := common.MustHexToHash("0x03170a2e7597b7b7e3d84c05391d139a62b157e78786d8c082f29dcf4c111314")
				runtimeHash := common.MustHexToHash("0x7db9db5ed9967b80143100189ba69d9e4deab85ac3570e5df25686cabe32964a")
				mockTrieState := storage.NewTrieState(nil)
				mockBlock := &types.Block{Header: types.Header{}, Body: types.Body{}}

				mockInstance := mocksruntime.NewInstance(t)
				mockInstance.On("SetContextStorage", mockTrieState)
				mockInstance.On("ExecuteBlock", mockBlock).Return(nil, nil)
				mockBlockState := NewMockBlockState(ctrl)
				mockBlockState.EXPECT().HasHeader(common.Hash{}).Return(false, nil)
				mockBlockState.EXPECT().HasBlockBody(common.Hash{}).Return(false, nil)
				mockBlockState.EXPECT().GetHeader(common.Hash{}).Return(&types.Header{
					Number:    0,
					StateRoot: stateRootHash,
				}, nil)
				mockBlockState.EXPECT().SetJustification(
					common.MustHexToHash("0xdcdd89927d8a348e00257e1ecc8617f45edb5118efff3ea2f9961b2ad9b7690a"), justification)
				mockBlockState.EXPECT().CompareAndSetBlockData(gomock.AssignableToTypeOf(&types.BlockData{}))
				mockBlockState.EXPECT().GetRuntime(&runtimeHash).Return(mockInstance, nil)
				mockBabeVerifier := NewMockBabeVerifier(ctrl)
				mockBabeVerifier.EXPECT().VerifyBlock(&types.Header{})
				mockStorageState := NewMockStorageState(ctrl)
				mockStorageState.EXPECT().Lock()
				mockStorageState.EXPECT().TrieState(&stateRootHash).Return(mockTrieState, nil)
				mockStorageState.EXPECT().Unlock()
				mockBlockImportHandler := NewMockBlockImportHandler(ctrl)
				mockBlockImportHandler.EXPECT().HandleBlockImport(mockBlock, mockTrieState)
				mockTelemetry := NewMockClient(ctrl)
				mockTelemetry.EXPECT().SendMessage(gomock.Any()).AnyTimes()
				mockFinalityGadget := NewMockFinalityGadget(ctrl)
				mockFinalityGadget.EXPECT().VerifyBlockJustification(
					common.MustHexToHash("0xdcdd89927d8a348e00257e1ecc8617f45edb5118efff3ea2f9961b2ad9b7690a"),
					justification).Return(justification, nil)
				return chainProcessor{
					blockState:         mockBlockState,
					babeVerifier:       mockBabeVerifier,
					storageState:       mockStorageState,
					blockImportHandler: mockBlockImportHandler,
					telemetry:          mockTelemetry,
					finalityGadget:     mockFinalityGadget,
				}
			},
			blockData: &types.BlockData{
				Header: &types.Header{
					Number: 0,
				},
				Body:          &types.Body{},
				Justification: &justification,
			},
		},
	}

	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			processor := tt.chainProcessorBuilder(ctrl)
			err := processor.processBlockData(tt.blockData)
			assert.ErrorIs(t, err, tt.expectedError)
		})
	}
}

func Test_chainProcessor_processReadyBlocks(t *testing.T) {
	t.Parallel()
	mockError := errors.New("test mock error")
	tests := map[string]struct {
		blockStateBuilder   func(ctrl *gomock.Controller, done chan struct{}) BlockState
		blockData           *types.BlockData
		babeVerifierBuilder func(ctrl *gomock.Controller) BabeVerifier
		pendingBlockBuilder func(ctrl *gomock.Controller, done chan struct{}) DisjointBlockSet
		storageStateBuilder func(ctrl *gomock.Controller, done chan struct{}) StorageState
	}{
		"base case": {
			blockStateBuilder: func(ctrl *gomock.Controller, done chan struct{}) BlockState {
				mockBlockState := NewMockBlockState(ctrl)
				mockBlockState.EXPECT().HasHeader(common.Hash{}).Return(false, nil)
				mockBlockState.EXPECT().HasBlockBody(common.Hash{}).Return(false, nil)
				mockBlockState.EXPECT().CompareAndSetBlockData(&types.BlockData{}).DoAndReturn(func(*types.
				BlockData) error {
					close(done)
					return nil
				})
				return mockBlockState
			},
			blockData: &types.BlockData{
				Hash: common.Hash{},
			},
			babeVerifierBuilder: func(ctrl *gomock.Controller) BabeVerifier {
				return nil
			},
			pendingBlockBuilder: func(ctrl *gomock.Controller, done chan struct{}) DisjointBlockSet {
				return nil
			},
			storageStateBuilder: func(ctrl *gomock.Controller, done chan struct{}) StorageState {
				return nil
			},
		},
		"add block": {
			blockStateBuilder: func(ctrl *gomock.Controller, done chan struct{}) BlockState {
				mockBlockState := NewMockBlockState(ctrl)
				mockBlockState.EXPECT().HasHeader(common.Hash{}).Return(false, nil)
				mockBlockState.EXPECT().HasBlockBody(common.Hash{}).Return(false, nil)
				mockBlockState.EXPECT().GetHeader(common.Hash{}).Return(nil, mockError)
				return mockBlockState
			},
			blockData: &types.BlockData{
				Hash:   common.Hash{},
				Header: &types.Header{},
				Body:   &types.Body{},
			},
			babeVerifierBuilder: func(ctrl *gomock.Controller) BabeVerifier {
				mockBabeVerifier := NewMockBabeVerifier(ctrl)
				mockBabeVerifier.EXPECT().VerifyBlock(&types.Header{}).Return(nil)
				return mockBabeVerifier
			},
			pendingBlockBuilder: func(ctrl *gomock.Controller, done chan struct{}) DisjointBlockSet {
				mockDisjointBlockSet := NewMockDisjointBlockSet(ctrl)
				mockDisjointBlockSet.EXPECT().addBlock(&types.Block{
					Header: types.Header{},
					Body:   types.Body{},
				}).DoAndReturn(func(block *types.Block) error {
					close(done)
					return nil
				})
				return mockDisjointBlockSet
			},
			storageStateBuilder: func(ctrl *gomock.Controller, done chan struct{}) StorageState {
				return nil
			},
		},
		"error in process block": {
			blockStateBuilder: func(ctrl *gomock.Controller, done chan struct{}) BlockState {
				mockBlockState := NewMockBlockState(ctrl)
				mockBlockState.EXPECT().HasHeader(common.Hash{}).Return(false, nil)
				mockBlockState.EXPECT().HasBlockBody(common.Hash{}).Return(false, nil)
				mockBlockState.EXPECT().GetHeader(common.Hash{}).Return(&types.Header{}, nil)
				return mockBlockState
			},
			blockData: &types.BlockData{
				Hash:   common.Hash{},
				Header: &types.Header{},
				Body:   &types.Body{},
			},
			babeVerifierBuilder: func(ctrl *gomock.Controller) BabeVerifier {
				mockBabeVerifier := NewMockBabeVerifier(ctrl)
				mockBabeVerifier.EXPECT().VerifyBlock(&types.Header{}).Return(nil)
				return mockBabeVerifier
			},
			pendingBlockBuilder: func(ctrl *gomock.Controller, done chan struct{}) DisjointBlockSet {
				return nil
			},
			storageStateBuilder: func(ctrl *gomock.Controller, done chan struct{}) StorageState {
				mockStorageState := NewMockStorageState(ctrl)
				mockStorageState.EXPECT().Lock()
				mockStorageState.EXPECT().Unlock()
				mockStorageState.EXPECT().TrieState(&common.Hash{}).DoAndReturn(func(hash *common.Hash) (*storage.
				TrieState, error) {
					close(done)
					return nil, mockError
				})
				return mockStorageState
			},
		},
		"add block error": {
			blockStateBuilder: func(ctrl *gomock.Controller, done chan struct{}) BlockState {
				mockBlockState := NewMockBlockState(ctrl)
				mockBlockState.EXPECT().HasHeader(common.Hash{}).Return(false, nil)
				mockBlockState.EXPECT().HasBlockBody(common.Hash{}).Return(false, nil)
				mockBlockState.EXPECT().GetHeader(common.Hash{}).Return(nil, mockError)
				return mockBlockState
			},
			blockData: &types.BlockData{
				Hash:   common.Hash{},
				Header: &types.Header{},
				Body:   &types.Body{},
			},
			babeVerifierBuilder: func(ctrl *gomock.Controller) BabeVerifier {
				mockBabeVerifier := NewMockBabeVerifier(ctrl)
				mockBabeVerifier.EXPECT().VerifyBlock(&types.Header{}).Return(nil)
				return mockBabeVerifier
			},
			pendingBlockBuilder: func(ctrl *gomock.Controller, done chan struct{}) DisjointBlockSet {
				mockDisjointBlockSet := NewMockDisjointBlockSet(ctrl)
				mockDisjointBlockSet.EXPECT().addBlock(&types.Block{
					Header: types.Header{},
					Body:   types.Body{},
				}).DoAndReturn(func(block *types.Block) error {
					close(done)
					return mockError
				})
				return mockDisjointBlockSet
			},
			storageStateBuilder: func(ctrl *gomock.Controller, done chan struct{}) StorageState {
				return nil
			},
		},
	}
	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			ctx, cancel := context.WithCancel(context.Background())
			readyBlock := newBlockQueue(5)
			done := make(chan struct{})

			s := &chainProcessor{
				ctx:           ctx,
				cancel:        cancel,
				readyBlocks:   readyBlock,
				blockState:    tt.blockStateBuilder(ctrl, done),
				babeVerifier:  tt.babeVerifierBuilder(ctrl),
				pendingBlocks: tt.pendingBlockBuilder(ctrl, done),
				storageState:  tt.storageStateBuilder(ctrl, done),
			}

			go s.processReadyBlocks()

			readyBlock.push(tt.blockData)
			<-done
			s.cancel()
		})
	}
}

func Test_newChainProcessor(t *testing.T) {
	t.Parallel()

	mockReadyBlock := newBlockQueue(5)
	mockDisjointBlockSet := NewMockDisjointBlockSet(nil)
	mockBlockState := NewMockBlockState(nil)
	mockStorageState := NewMockStorageState(nil)
	mockTransactionState := NewMockTransactionState(nil)
	mockBabeVerifier := NewMockBabeVerifier(nil)
	mockFinalityGadget := NewMockFinalityGadget(nil)
	mockBlockImportHandler := NewMockBlockImportHandler(nil)

	type args struct {
		readyBlocks        *blockQueue
		pendingBlocks      DisjointBlockSet
		blockState         BlockState
		storageState       StorageState
		transactionState   TransactionState
		babeVerifier       BabeVerifier
		finalityGadget     FinalityGadget
		blockImportHandler BlockImportHandler
	}
	tests := []struct {
		name string
		args args
		want *chainProcessor
	}{
		{
			name: "with args",
			args: args{
				readyBlocks:        mockReadyBlock,
				pendingBlocks:      mockDisjointBlockSet,
				blockState:         mockBlockState,
				storageState:       mockStorageState,
				transactionState:   mockTransactionState,
				babeVerifier:       mockBabeVerifier,
				finalityGadget:     mockFinalityGadget,
				blockImportHandler: mockBlockImportHandler,
			},
			want: &chainProcessor{
				readyBlocks:        mockReadyBlock,
				pendingBlocks:      mockDisjointBlockSet,
				blockState:         mockBlockState,
				storageState:       mockStorageState,
				transactionState:   mockTransactionState,
				babeVerifier:       mockBabeVerifier,
				finalityGadget:     mockFinalityGadget,
				blockImportHandler: mockBlockImportHandler,
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := newChainProcessor(tt.args.readyBlocks, tt.args.pendingBlocks, tt.args.blockState,
				tt.args.storageState, tt.args.transactionState, tt.args.babeVerifier, tt.args.finalityGadget,
				tt.args.blockImportHandler, nil)
			assert.NotNil(t, got.ctx)
			got.ctx = nil
			assert.NotNil(t, got.cancel)
			got.cancel = nil
			assert.Equal(t, tt.want, got)
		})
	}
}
