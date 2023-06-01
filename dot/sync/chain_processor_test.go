// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package sync

import (
	"context"
	"errors"
	"testing"

	"github.com/ChainSafe/gossamer/dot/telemetry"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/blocktree"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/runtime/storage"
	"github.com/ChainSafe/gossamer/lib/trie"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func Test_chainProcessor_handleBlock(t *testing.T) {
	t.Parallel()
	mockError := errors.New("test mock error")
	testHash := common.MustHexToHash("0x03170a2e7597b7b7e3d84c05391d139a62b157e78786d8c082f29dcf4c111314")
	testParentHash := common.MustHexToHash("0x7db9db5ed9967b80143100189ba69d9e4deab85ac3570e5df25686cabe32964a")

	tests := map[string]struct {
		chainProcessorBuilder func(ctrl *gomock.Controller) chainProcessor
		block                 *types.Block
		announce              bool
		wantErr               error
	}{
		"handle_getHeader_error": {
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
		"handle_trieState_error": {
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
		"handle_getRuntime_error": {
			chainProcessorBuilder: func(ctrl *gomock.Controller) (chainProcessor chainProcessor) {
				mockBlockState := NewMockBlockState(ctrl)
				mockBlockState.EXPECT().GetHeader(common.Hash{}).Return(&types.Header{
					StateRoot: testHash,
				}, nil)
				mockBlockState.EXPECT().GetRuntime(testParentHash).Return(nil, mockError)
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
		"handle_runtime_ExecuteBlock_error": {
			chainProcessorBuilder: func(ctrl *gomock.Controller) (chainProcessor chainProcessor) {
				trieState := storage.NewTrieState(nil)
				mockBlockState := NewMockBlockState(ctrl)
				mockBlockState.EXPECT().GetHeader(common.Hash{}).Return(&types.Header{
					StateRoot: testHash,
				}, nil)
				mockInstance := NewMockInstance(ctrl)
				mockInstance.EXPECT().SetContextStorage(trieState)
				mockInstance.EXPECT().ExecuteBlock(&types.Block{Body: types.Body{}}).Return(nil, mockError)
				mockBlockState.EXPECT().GetRuntime(testParentHash).Return(mockInstance, nil)
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
		"handle_block_import_error": {
			chainProcessorBuilder: func(ctrl *gomock.Controller) (chainProcessor chainProcessor) {
				trieState := storage.NewTrieState(nil)
				mockBlockState := NewMockBlockState(ctrl)
				mockBlockState.EXPECT().GetHeader(common.Hash{}).Return(&types.Header{
					StateRoot: testHash,
				}, nil)
				mockBlock := &types.Block{Body: types.Body{}}
				mockInstance := NewMockInstance(ctrl)
				mockInstance.EXPECT().SetContextStorage(trieState)
				mockInstance.EXPECT().ExecuteBlock(mockBlock).Return(nil, nil)
				mockBlockState.EXPECT().GetRuntime(testParentHash).Return(mockInstance, nil)
				chainProcessor.blockState = mockBlockState
				mockStorageState := NewMockStorageState(ctrl)
				mockStorageState.EXPECT().Lock()
				mockStorageState.EXPECT().TrieState(&testHash).Return(trieState, nil)
				mockStorageState.EXPECT().Unlock()
				chainProcessor.storageState = mockStorageState
				mockBlockImportHandler := NewMockBlockImportHandler(ctrl)
				mockBlockImportHandler.EXPECT().HandleBlockImport(mockBlock,
					trieState, false).Return(mockError)
				chainProcessor.blockImportHandler = mockBlockImportHandler
				return
			},
			block: &types.Block{
				Body: types.Body{},
			},
			wantErr: mockError,
		},
		"base_case": {
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

				mockInstance := NewMockInstance(ctrl)
				mockInstance.EXPECT().SetContextStorage(trieState)
				mockInstance.EXPECT().ExecuteBlock(mockBlock).Return(nil, nil)
				mockBlockState.EXPECT().GetRuntime(mockHeaderHash).Return(mockInstance, nil)
				chainProcessor.blockState = mockBlockState
				mockStorageState := NewMockStorageState(ctrl)
				mockStorageState.EXPECT().Lock()
				mockStorageState.EXPECT().Unlock()
				mockStorageState.EXPECT().TrieState(&trie.EmptyHash).Return(trieState, nil)
				chainProcessor.storageState = mockStorageState
				mockBlockImportHandler := NewMockBlockImportHandler(ctrl)
				mockBlockImportHandler.EXPECT().HandleBlockImport(mockBlock, trieState, false).Return(nil)
				chainProcessor.blockImportHandler = mockBlockImportHandler
				mockTelemetry := NewMockTelemetry(ctrl)
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
		"import_block_and_announce": {
			announce: true,
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

				mockInstance := NewMockInstance(ctrl)
				mockInstance.EXPECT().SetContextStorage(trieState)
				mockInstance.EXPECT().ExecuteBlock(mockBlock).Return(nil, nil)
				mockBlockState.EXPECT().GetRuntime(mockHeaderHash).Return(mockInstance, nil)
				chainProcessor.blockState = mockBlockState
				mockStorageState := NewMockStorageState(ctrl)
				mockStorageState.EXPECT().Lock()
				mockStorageState.EXPECT().Unlock()
				mockStorageState.EXPECT().TrieState(&trie.EmptyHash).Return(trieState, nil)
				chainProcessor.storageState = mockStorageState
				mockBlockImportHandler := NewMockBlockImportHandler(ctrl)
				mockBlockImportHandler.EXPECT().HandleBlockImport(mockBlock, trieState, true).Return(nil)
				chainProcessor.blockImportHandler = mockBlockImportHandler
				mockTelemetry := NewMockTelemetry(ctrl)
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

			err := s.handleBlock(tt.block, tt.announce)
			assert.ErrorIs(t, err, tt.wantErr)
		})
	}
	t.Run("panics_on_different_parent_state_root", func(t *testing.T) {
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
			_ = chainProcessor.handleBlock(bock, false)
		})
	})
}

func Test_chainProcessor_handleBody(t *testing.T) {
	t.Parallel()

	testExtrinsics := []types.Extrinsic{{1, 2, 3}, {7, 8, 9, 0}, {0xa, 0xb}}
	testBody := types.NewBody(testExtrinsics)

	t.Run("base_case", func(t *testing.T) {
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

	header := &types.Header{
		Number: 2,
	}
	headerHash := header.Hash()
	errTest := errors.New("test error")

	type args struct {
		header        *types.Header
		justification []byte
	}
	tests := map[string]struct {
		chainProcessorBuilder func(ctrl *gomock.Controller) chainProcessor
		args                  args
		sentinelError         error
		errorMessage          string
	}{
		"invalid_justification": {
			chainProcessorBuilder: func(ctrl *gomock.Controller) chainProcessor {
				mockFinalityGadget := NewMockFinalityGadget(ctrl)
				mockFinalityGadget.EXPECT().VerifyBlockJustification(headerHash,
					[]byte(`x`)).Return(errTest)
				return chainProcessor{
					finalityGadget: mockFinalityGadget,
				}
			},
			args: args{
				header:        header,
				justification: []byte(`x`),
			},
			sentinelError: errTest,
			errorMessage:  "verifying block number 2 justification: test error",
		},
		"set_justification_error": {
			chainProcessorBuilder: func(ctrl *gomock.Controller) chainProcessor {
				mockBlockState := NewMockBlockState(ctrl)
				mockBlockState.EXPECT().SetJustification(headerHash, []byte(`xx`)).Return(errTest)
				mockFinalityGadget := NewMockFinalityGadget(ctrl)
				mockFinalityGadget.EXPECT().VerifyBlockJustification(headerHash, []byte(`xx`)).Return(nil)
				return chainProcessor{
					blockState:     mockBlockState,
					finalityGadget: mockFinalityGadget,
				}
			},
			args: args{
				header:        header,
				justification: []byte(`xx`),
			},
			sentinelError: errTest,
			errorMessage:  "setting justification for block number 2: test error",
		},
		"base_case_set": {
			chainProcessorBuilder: func(ctrl *gomock.Controller) chainProcessor {
				mockBlockState := NewMockBlockState(ctrl)
				mockBlockState.EXPECT().SetJustification(headerHash, []byte(`1234`)).Return(nil)
				mockFinalityGadget := NewMockFinalityGadget(ctrl)
				mockFinalityGadget.EXPECT().VerifyBlockJustification(headerHash, []byte(`1234`)).Return(nil)
				return chainProcessor{
					blockState:     mockBlockState,
					finalityGadget: mockFinalityGadget,
				}
			},
			args: args{
				header:        header,
				justification: []byte(`1234`),
			},
		},
	}
	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)

			processor := tt.chainProcessorBuilder(ctrl)

			err := processor.handleJustification(tt.args.header, tt.args.justification)

			assert.ErrorIs(t, err, tt.sentinelError)
			if tt.sentinelError != nil {
				assert.EqualError(t, err, tt.errorMessage)
			}
		})
	}
}

func Test_chainProcessor_processBlockData(t *testing.T) {
	t.Parallel()

	mockError := errors.New("mock test error")

	tests := map[string]struct {
		chainProcessorBuilder func(ctrl *gomock.Controller) chainProcessor
		blockData             types.BlockData
		expectedError         error
	}{
		"handle_has_header_error": {
			chainProcessorBuilder: func(ctrl *gomock.Controller) chainProcessor {
				mockBlockState := NewMockBlockState(ctrl)
				mockBlockState.EXPECT().HasHeader(common.Hash{}).Return(false, mockError)

				return chainProcessor{
					blockState: mockBlockState,
				}
			},
			blockData:     types.BlockData{},
			expectedError: mockError,
		},
		"handle_has_block_body_error": {
			chainProcessorBuilder: func(ctrl *gomock.Controller) chainProcessor {
				mockBlockState := NewMockBlockState(ctrl)
				mockBlockState.EXPECT().HasHeader(common.Hash{}).Return(false, nil)
				mockBlockState.EXPECT().HasBlockBody(common.Hash{}).Return(false, mockError)
				return chainProcessor{
					blockState: mockBlockState,
				}
			},
			blockData:     types.BlockData{},
			expectedError: mockError,
		},
		"handle_getBlockByHash_error": {
			chainProcessorBuilder: func(ctrl *gomock.Controller) chainProcessor {
				mockBlockState := NewMockBlockState(ctrl)
				mockBlockState.EXPECT().HasHeader(common.Hash{}).Return(true, nil)
				mockBlockState.EXPECT().HasBlockBody(common.Hash{}).Return(true, nil)
				mockBlockState.EXPECT().GetBlockByHash(common.Hash{}).Return(nil, mockError)

				mockChainSync := NewMockChainSync(ctrl)
				mockChainSync.EXPECT().syncState().Return(bootstrap)
				return chainProcessor{
					blockState: mockBlockState,
					chainSync:  mockChainSync,
				}
			},
			blockData:     types.BlockData{},
			expectedError: mockError,
		},
		"handle_block_data_justification_!=_nil": {
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
					3}).Return(nil)
				mockStorageState := NewMockStorageState(ctrl)
				mockStorageState.EXPECT().TrieState(&common.Hash{}).Return(nil, nil)

				// given our current chain sync state is `tip`
				// the `HandleBlockImport` method should expect
				// true as the announce parameter
				mockChainSync := NewMockChainSync(ctrl)
				mockChainSync.EXPECT().syncState().Return(tip)

				mockBlockImportHandler := NewMockBlockImportHandler(ctrl)
				mockBlockImportHandler.EXPECT().HandleBlockImport(mockBlock,
					nil, true).Return(nil)

				return chainProcessor{
					chainSync:          mockChainSync,
					blockState:         mockBlockState,
					finalityGadget:     mockFinalityGadget,
					storageState:       mockStorageState,
					blockImportHandler: mockBlockImportHandler,
				}
			},
			blockData: types.BlockData{
				Justification: &[]byte{1, 2, 3},
			},
		},
		"handle_babe_verify_block_error": {
			chainProcessorBuilder: func(ctrl *gomock.Controller) chainProcessor {
				mockBlockState := NewMockBlockState(ctrl)
				mockBlockState.EXPECT().HasHeader(common.Hash{}).Return(false, nil)
				mockBlockState.EXPECT().HasBlockBody(common.Hash{}).Return(false, nil)
				mockBabeVerifier := NewMockBabeVerifier(ctrl)
				mockBabeVerifier.EXPECT().VerifyBlock(&types.Header{}).Return(mockError)

				mockChainSync := NewMockChainSync(ctrl)
				mockChainSync.EXPECT().syncState().Return(bootstrap)

				return chainProcessor{
					chainSync:    mockChainSync,
					blockState:   mockBlockState,
					babeVerifier: mockBabeVerifier,
				}
			},
			blockData: types.BlockData{
				Header: &types.Header{},
				Body:   &types.Body{},
			},
			expectedError: mockError,
		},
		"no_header_and_body_-_fail_to_handle_justification": {
			chainProcessorBuilder: func(ctrl *gomock.Controller) chainProcessor {
				blockState := NewMockBlockState(ctrl)
				blockState.EXPECT().HasHeader(common.Hash{1}).Return(false, nil)
				blockState.EXPECT().HasBlockBody(common.Hash{1}).Return(true, nil)

				finalityGadget := NewMockFinalityGadget(ctrl)
				expectedBlockDataHeader := &types.Header{Number: 2}
				expectedBlockDataHeaderHash := expectedBlockDataHeader.Hash()
				finalityGadget.EXPECT().
					VerifyBlockJustification(expectedBlockDataHeaderHash, []byte{1, 2, 3}).
					Return(mockError)

				mockChainSync := NewMockChainSync(ctrl)
				mockChainSync.EXPECT().syncState().Return(bootstrap)

				return chainProcessor{
					chainSync:      mockChainSync,
					blockState:     blockState,
					finalityGadget: finalityGadget,
				}
			},
			blockData: types.BlockData{
				Hash:          common.Hash{1},
				Header:        &types.Header{Number: 2},
				Justification: &[]byte{1, 2, 3},
			},
			expectedError: mockError,
		},
		"handle_compareAndSetBlockData_error": {
			chainProcessorBuilder: func(ctrl *gomock.Controller) chainProcessor {
				mockBlockState := NewMockBlockState(ctrl)
				mockBlockState.EXPECT().HasHeader(common.Hash{}).Return(false, nil)
				mockBlockState.EXPECT().HasBlockBody(common.Hash{}).Return(false, nil)
				mockBlockState.EXPECT().CompareAndSetBlockData(&types.BlockData{}).Return(mockError)

				mockChainSync := NewMockChainSync(ctrl)
				mockChainSync.EXPECT().syncState().Return(bootstrap)
				return chainProcessor{
					chainSync:  mockChainSync,
					blockState: mockBlockState,
				}
			},
			blockData:     types.BlockData{},
			expectedError: mockError,
		},
		"success_with_justification": {
			chainProcessorBuilder: func(ctrl *gomock.Controller) chainProcessor {
				stateRootHash := common.MustHexToHash("0x03170a2e7597b7b7e3d84c05391d139a62b157e78786d8c082f29dcf4c111314")
				runtimeHash := common.MustHexToHash("0x7db9db5ed9967b80143100189ba69d9e4deab85ac3570e5df25686cabe32964a")
				mockTrieState := storage.NewTrieState(nil)
				mockBlock := &types.Block{Header: types.Header{}, Body: types.Body{}}

				mockInstance := NewMockInstance(ctrl)
				mockInstance.EXPECT().SetContextStorage(mockTrieState)
				mockInstance.EXPECT().ExecuteBlock(mockBlock).Return(nil, nil)
				mockBlockState := NewMockBlockState(ctrl)
				mockBlockState.EXPECT().HasHeader(common.Hash{}).Return(false, nil)
				mockBlockState.EXPECT().HasBlockBody(common.Hash{}).Return(false, nil)
				mockBlockState.EXPECT().GetHeader(common.Hash{}).Return(&types.Header{
					Number:    0,
					StateRoot: stateRootHash,
				}, nil)
				mockBlockState.EXPECT().SetJustification(
					common.MustHexToHash("0xdcdd89927d8a348e00257e1ecc8617f45edb5118efff3ea2f9961b2ad9b7690a"), []byte{1, 2, 3})
				mockBlockState.EXPECT().CompareAndSetBlockData(gomock.AssignableToTypeOf(&types.BlockData{}))
				mockBlockState.EXPECT().GetRuntime(runtimeHash).Return(mockInstance, nil)
				mockBabeVerifier := NewMockBabeVerifier(ctrl)
				mockBabeVerifier.EXPECT().VerifyBlock(&types.Header{})
				mockStorageState := NewMockStorageState(ctrl)
				mockStorageState.EXPECT().Lock()
				mockStorageState.EXPECT().TrieState(&stateRootHash).Return(mockTrieState, nil)
				mockStorageState.EXPECT().Unlock()

				mockChainSync := NewMockChainSync(ctrl)
				mockChainSync.EXPECT().syncState().Return(bootstrap)

				mockBlockImportHandler := NewMockBlockImportHandler(ctrl)
				mockBlockImportHandler.EXPECT().HandleBlockImport(mockBlock, mockTrieState, false)

				mockTelemetry := NewMockTelemetry(ctrl)
				mockTelemetry.EXPECT().SendMessage(gomock.Any())
				mockFinalityGadget := NewMockFinalityGadget(ctrl)
				mockFinalityGadget.EXPECT().VerifyBlockJustification(
					common.MustHexToHash("0xdcdd89927d8a348e00257e1ecc8617f45edb5118efff3ea2f9961b2ad9b7690a"),
					[]byte{1, 2, 3}).Return(nil)
				return chainProcessor{
					chainSync:          mockChainSync,
					blockState:         mockBlockState,
					babeVerifier:       mockBabeVerifier,
					storageState:       mockStorageState,
					blockImportHandler: mockBlockImportHandler,
					telemetry:          mockTelemetry,
					finalityGadget:     mockFinalityGadget,
				}
			},
			blockData: types.BlockData{
				Header: &types.Header{
					Number: 0,
				},
				Body:          &types.Body{},
				Justification: &[]byte{1, 2, 3},
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

func Test_chainProcessor_processBlockDataWithStateHeaderAndBody(t *testing.T) {
	t.Parallel()

	errTest := errors.New("test error")

	testCases := map[string]struct {
		chainProcessorBuilder func(ctrl *gomock.Controller) chainProcessor
		blockData             types.BlockData
		announceImportedBlock bool
		sentinelError         error
		errorMessage          string
	}{
		"get_block_by_hash_error": {
			chainProcessorBuilder: func(ctrl *gomock.Controller) chainProcessor {
				blockState := NewMockBlockState(ctrl)
				blockState.EXPECT().GetBlockByHash(common.Hash{1}).
					Return(nil, errTest)
				return chainProcessor{
					blockState: blockState,
				}
			},
			blockData:     types.BlockData{Hash: common.Hash{1}},
			sentinelError: errTest,
			errorMessage:  "getting block by hash: test error",
		},
		"block_already_exists_in_blocktree": {
			chainProcessorBuilder: func(ctrl *gomock.Controller) chainProcessor {
				blockState := NewMockBlockState(ctrl)
				block := &types.Block{Header: types.Header{Number: 2}}
				blockState.EXPECT().GetBlockByHash(common.Hash{1}).Return(block, nil)
				blockState.EXPECT().AddBlockToBlockTree(block).Return(blocktree.ErrBlockExists)
				return chainProcessor{
					blockState: blockState,
				}
			},
			blockData: types.BlockData{Hash: common.Hash{1}},
		},
		"add_block_to_blocktree_error": {
			chainProcessorBuilder: func(ctrl *gomock.Controller) chainProcessor {
				blockState := NewMockBlockState(ctrl)
				block := &types.Block{Header: types.Header{Number: 2}}
				blockState.EXPECT().GetBlockByHash(common.Hash{1}).Return(block, nil)
				blockState.EXPECT().AddBlockToBlockTree(block).Return(errTest)
				return chainProcessor{
					blockState: blockState,
				}
			},
			blockData:     types.BlockData{Hash: common.Hash{1}},
			sentinelError: errTest,
			errorMessage:  "adding block to blocktree: test error",
		},
		"handle_justification_error": {
			chainProcessorBuilder: func(ctrl *gomock.Controller) chainProcessor {
				blockState := NewMockBlockState(ctrl)
				blockHeader := types.Header{Number: 2}
				blockHeaderHash := blockHeader.Hash()
				block := &types.Block{Header: blockHeader}
				blockState.EXPECT().GetBlockByHash(common.Hash{1}).Return(block, nil)
				blockState.EXPECT().AddBlockToBlockTree(block).Return(nil)

				finalityGadget := NewMockFinalityGadget(ctrl)
				finalityGadget.EXPECT().
					VerifyBlockJustification(blockHeaderHash, []byte{3}).
					Return(errTest)

				return chainProcessor{
					blockState:     blockState,
					finalityGadget: finalityGadget,
				}
			},
			blockData: types.BlockData{
				Hash:          common.Hash{1},
				Justification: &[]byte{3},
			},
			sentinelError: errTest,
			errorMessage:  "handling justification: verifying block number 2 justification: test error",
		},
		"trie_state_error": {
			chainProcessorBuilder: func(ctrl *gomock.Controller) chainProcessor {
				blockState := NewMockBlockState(ctrl)
				blockHeader := types.Header{StateRoot: common.Hash{2}}
				block := &types.Block{Header: blockHeader}
				blockState.EXPECT().GetBlockByHash(common.Hash{1}).Return(block, nil)
				blockState.EXPECT().AddBlockToBlockTree(block).Return(nil)

				storageState := NewMockStorageState(ctrl)
				storageState.EXPECT().TrieState(&common.Hash{2}).
					Return(nil, errTest)

				return chainProcessor{
					blockState:   blockState,
					storageState: storageState,
				}
			},
			blockData: types.BlockData{
				Hash: common.Hash{1},
			},
			sentinelError: errTest,
			errorMessage:  "loading trie state: test error",
		},
		"handle_block_import_error": {
			chainProcessorBuilder: func(ctrl *gomock.Controller) chainProcessor {
				blockState := NewMockBlockState(ctrl)
				blockHeader := types.Header{StateRoot: common.Hash{2}}
				block := &types.Block{Header: blockHeader}
				blockState.EXPECT().GetBlockByHash(common.Hash{1}).Return(block, nil)
				blockState.EXPECT().AddBlockToBlockTree(block).Return(nil)

				storageState := NewMockStorageState(ctrl)
				trieState := storage.NewTrieState(nil)
				storageState.EXPECT().TrieState(&common.Hash{2}).
					Return(trieState, nil)

				blockImportHandler := NewMockBlockImportHandler(ctrl)
				const announceImportedBlock = true
				blockImportHandler.EXPECT().HandleBlockImport(block, trieState, announceImportedBlock).
					Return(errTest)

				return chainProcessor{
					blockState:         blockState,
					storageState:       storageState,
					blockImportHandler: blockImportHandler,
				}
			},
			blockData: types.BlockData{
				Hash: common.Hash{1},
			},
			announceImportedBlock: true,
			sentinelError:         errTest,
			errorMessage:          "handling block import: test error",
		},
		"success": {
			chainProcessorBuilder: func(ctrl *gomock.Controller) chainProcessor {
				blockState := NewMockBlockState(ctrl)
				blockHeader := types.Header{StateRoot: common.Hash{2}}
				block := &types.Block{Header: blockHeader}
				blockState.EXPECT().GetBlockByHash(common.Hash{1}).Return(block, nil)
				blockState.EXPECT().AddBlockToBlockTree(block).Return(nil)

				storageState := NewMockStorageState(ctrl)
				trieState := storage.NewTrieState(nil)
				storageState.EXPECT().TrieState(&common.Hash{2}).
					Return(trieState, nil)

				blockImportHandler := NewMockBlockImportHandler(ctrl)
				const announceImportedBlock = true
				blockImportHandler.EXPECT().HandleBlockImport(block, trieState, announceImportedBlock).
					Return(nil)

				return chainProcessor{
					blockState:         blockState,
					storageState:       storageState,
					blockImportHandler: blockImportHandler,
				}
			},
			blockData: types.BlockData{
				Hash: common.Hash{1},
			},
			announceImportedBlock: true,
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)

			processor := testCase.chainProcessorBuilder(ctrl)

			err := processor.processBlockDataWithStateHeaderAndBody(
				testCase.blockData, testCase.announceImportedBlock)

			assert.ErrorIs(t, err, testCase.sentinelError)
			if testCase.sentinelError != nil {
				assert.EqualError(t, err, testCase.errorMessage)
			}
		})
	}
}

func Test_chainProcessor_processBlockDataWithHeaderAndBody(t *testing.T) {
	t.Parallel()

	errTest := errors.New("test error")

	testCases := map[string]struct {
		chainProcessorBuilder func(ctrl *gomock.Controller) chainProcessor
		blockData             types.BlockData
		announceImportedBlock bool
		sentinelError         error
		errorMessage          string
	}{
		"verify_block_error": {
			chainProcessorBuilder: func(ctrl *gomock.Controller) chainProcessor {
				babeVerifier := NewMockBabeVerifier(ctrl)
				babeVerifier.EXPECT().VerifyBlock(&types.Header{Number: 1}).
					Return(errTest)

				return chainProcessor{
					babeVerifier: babeVerifier,
				}
			},
			blockData: types.BlockData{
				Header: &types.Header{Number: 1},
			},
			sentinelError: errTest,
			errorMessage:  "babe verifying block: test error",
		},
		"handle_block_error": {
			chainProcessorBuilder: func(ctrl *gomock.Controller) chainProcessor {
				babeVerifier := NewMockBabeVerifier(ctrl)
				expectedHeader := &types.Header{ParentHash: common.Hash{1}}
				babeVerifier.EXPECT().VerifyBlock(expectedHeader).
					Return(nil)

				transactionState := NewMockTransactionState(ctrl)
				transactionState.EXPECT().RemoveExtrinsic(types.Extrinsic{2})

				blockState := NewMockBlockState(ctrl)
				blockState.EXPECT().GetHeader(common.Hash{1}).
					Return(nil, errTest)

				return chainProcessor{
					babeVerifier:     babeVerifier,
					transactionState: transactionState,
					blockState:       blockState,
				}
			},
			blockData: types.BlockData{
				Header: &types.Header{ParentHash: common.Hash{1}},
				Body:   &types.Body{{2}},
			},
			sentinelError: errFailedToGetParent,
			errorMessage:  "handling block: failed to get parent header: test error",
		},
		"success": {
			chainProcessorBuilder: func(ctrl *gomock.Controller) chainProcessor {
				babeVerifier := NewMockBabeVerifier(ctrl)
				expectedHeader := &types.Header{
					ParentHash: common.Hash{1},
					Number:     5,
				}
				babeVerifier.EXPECT().VerifyBlock(expectedHeader).
					Return(nil)

				transactionState := NewMockTransactionState(ctrl)
				transactionState.EXPECT().RemoveExtrinsic(types.Extrinsic{2})

				blockState := NewMockBlockState(ctrl)
				parentHeader := &types.Header{StateRoot: trie.EmptyHash}
				blockState.EXPECT().GetHeader(common.Hash{1}).
					Return(parentHeader, nil)

				storageState := NewMockStorageState(ctrl)
				lockCall := storageState.EXPECT().Lock()
				storageState.EXPECT().Unlock().After(lockCall)
				trieState := storage.NewTrieState(nil)
				storageState.EXPECT().TrieState(&trie.EmptyHash).
					Return(trieState, nil)

				parentHeaderHash := parentHeader.Hash()
				instance := NewMockInstance(ctrl)
				blockState.EXPECT().GetRuntime(parentHeaderHash).
					Return(instance, nil)

				instance.EXPECT().SetContextStorage(trieState)
				block := &types.Block{
					Header: *expectedHeader,
					Body:   types.Body{{2}},
				}
				instance.EXPECT().ExecuteBlock(block).Return(nil, nil)

				blockImportHandler := NewMockBlockImportHandler(ctrl)
				const announceImportedBlock = true
				blockImportHandler.EXPECT().HandleBlockImport(block, trieState, announceImportedBlock).
					Return(nil)

				telemetryClient := NewMockTelemetry(ctrl)
				headerHash := common.MustHexToHash("0x18d21d2901e4a4ac6a8c6431da2dfee1b8701f31a9e49283a082e6c744d4117c")
				message := telemetry.NewBlockImport(&headerHash, expectedHeader.Number, "NetworkInitialSync")
				telemetryClient.EXPECT().SendMessage(message)

				return chainProcessor{
					babeVerifier:       babeVerifier,
					transactionState:   transactionState,
					blockState:         blockState,
					storageState:       storageState,
					blockImportHandler: blockImportHandler,
					telemetry:          telemetryClient,
				}
			},
			blockData: types.BlockData{
				Header: &types.Header{
					ParentHash: common.Hash{1},
					Number:     5,
				},
				Body: &types.Body{{2}},
			},
			announceImportedBlock: true,
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)

			processor := testCase.chainProcessorBuilder(ctrl)

			err := processor.processBlockDataWithHeaderAndBody(
				testCase.blockData, testCase.announceImportedBlock)

			assert.ErrorIs(t, err, testCase.sentinelError)
			if testCase.sentinelError != nil {
				assert.EqualError(t, err, testCase.errorMessage)
			}
		})
	}
}

func Test_chainProcessor_processReadyBlocks(t *testing.T) {
	t.Parallel()
	mockError := errors.New("test mock error")
	tests := map[string]struct {
		chainSyncBuilder    func(ctrl *gomock.Controller) ChainSync
		blockStateBuilder   func(ctrl *gomock.Controller, done chan struct{}) BlockState
		blockData           *types.BlockData
		babeVerifierBuilder func(ctrl *gomock.Controller) BabeVerifier
		pendingBlockBuilder func(ctrl *gomock.Controller, done chan struct{}) DisjointBlockSet
		storageStateBuilder func(ctrl *gomock.Controller, done chan struct{}) StorageState
	}{
		"base_case": {
			chainSyncBuilder: func(ctrl *gomock.Controller) ChainSync {
				cs := NewMockChainSync(ctrl)
				cs.EXPECT().syncState().Return(bootstrap)
				return cs
			},
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
		"add_block": {
			chainSyncBuilder: func(ctrl *gomock.Controller) ChainSync {
				cs := NewMockChainSync(ctrl)
				cs.EXPECT().syncState().Return(bootstrap)
				return cs
			},
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
		"error_in_process_block": {
			chainSyncBuilder: func(ctrl *gomock.Controller) ChainSync {
				cs := NewMockChainSync(ctrl)
				cs.EXPECT().syncState().Return(bootstrap)
				return cs
			},
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
		"add_block_error": {
			chainSyncBuilder: func(ctrl *gomock.Controller) ChainSync {
				cs := NewMockChainSync(ctrl)
				cs.EXPECT().syncState().Return(bootstrap)
				return cs
			},
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
				chainSync:     tt.chainSyncBuilder(ctrl),
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
			name: "with_args",
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
			cpCfg := chainProcessorConfig{
				readyBlocks:        tt.args.readyBlocks,
				pendingBlocks:      tt.args.pendingBlocks,
				blockState:         tt.args.blockState,
				storageState:       tt.args.storageState,
				transactionState:   tt.args.transactionState,
				babeVerifier:       tt.args.babeVerifier,
				finalityGadget:     tt.args.finalityGadget,
				blockImportHandler: tt.args.blockImportHandler,
			}

			got := newChainProcessor(cpCfg)
			assert.NotNil(t, got.ctx)
			got.ctx = nil
			assert.NotNil(t, got.cancel)
			got.cancel = nil
			assert.Equal(t, tt.want, got)
		})
	}
}
