// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package sync

import (
	"context"
	"errors"
	"fmt"
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

//go:generate mockgen -destination=mock_instance_test.go -package=$GOPACKAGE github.com/ChainSafe/gossamer/lib/runtime Instance

func Test_chainProcessor_handleBlock(t *testing.T) {
	t.Parallel()
	mockError := errors.New("test mock error")

	tests := map[string]struct {
		chainProcessorBuilder func(ctrl *gomock.Controller) chainProcessor
		block                 *types.Block
		wantErr               error
		errMessage            string
	}{
		"block state GetHeader error": {
			chainProcessorBuilder: func(ctrl *gomock.Controller) chainProcessor {
				blockState := NewMockBlockState(ctrl)
				blockState.EXPECT().GetHeader(common.Hash{1}).
					Return(nil, mockError)
				return chainProcessor{
					blockState: blockState,
				}
			},
			block: &types.Block{
				Header: types.Header{ParentHash: common.Hash{1}},
			},
			wantErr:    errFailedToGetParent,
			errMessage: "failed to get parent header: test mock error",
		},
		"block state GetRuntime error": {
			chainProcessorBuilder: func(ctrl *gomock.Controller) chainProcessor {
				blockState := NewMockBlockState(ctrl)
				parentHeader := &types.Header{Number: 1}
				blockState.EXPECT().GetHeader(common.Hash{1}).
					Return(parentHeader, nil)

				parentHash := parentHeader.Hash()
				blockState.EXPECT().GetRuntime(&parentHash).
					Return(nil, mockError)

				return chainProcessor{
					blockState: blockState,
				}
			},
			block: &types.Block{
				Header: types.Header{ParentHash: common.Hash{1}},
			},
			wantErr:    mockError,
			errMessage: "getting runtime for parent hash: test mock error",
		},
		"storage state TrieState error": {
			chainProcessorBuilder: func(ctrl *gomock.Controller) chainProcessor {
				blockState := NewMockBlockState(ctrl)
				parentHeader := &types.Header{Number: 1, StateRoot: common.Hash{2}}
				blockState.EXPECT().GetHeader(common.Hash{1}).
					Return(parentHeader, nil)

				parentHash := parentHeader.Hash()
				runtimeInstance := NewMockInstance(ctrl)
				blockState.EXPECT().GetRuntime(&parentHash).
					Return(runtimeInstance, nil)

				runtimeInstance.EXPECT().StateVersion().Return(trie.V0)

				storageState := NewMockStorageState(ctrl)
				lockCall := storageState.EXPECT().Lock()
				storageState.EXPECT().Unlock().After(lockCall)
				storageState.EXPECT().TrieState(&common.Hash{2}, trie.V0).
					Return(nil, mockError)

				return chainProcessor{
					blockState:   blockState,
					storageState: storageState,
				}
			},
			block: &types.Block{
				Header: types.Header{ParentHash: common.Hash{1}},
			},
			wantErr:    mockError,
			errMessage: "trie state: test mock error",
		},
		"runtime ExecuteBlock error": {
			chainProcessorBuilder: func(ctrl *gomock.Controller) chainProcessor {
				blockState := NewMockBlockState(ctrl)
				parentHeader := &types.Header{Number: 1, StateRoot: trie.EmptyHash}
				blockState.EXPECT().GetHeader(common.Hash{1}).
					Return(parentHeader, nil)

				parentHash := parentHeader.Hash()
				runtimeInstance := NewMockInstance(ctrl)
				blockState.EXPECT().GetRuntime(&parentHash).
					Return(runtimeInstance, nil)

				runtimeInstance.EXPECT().StateVersion().Return(trie.V0)

				storageState := NewMockStorageState(ctrl)
				lockCall := storageState.EXPECT().Lock()
				storageState.EXPECT().Unlock().After(lockCall)

				trieState := storage.NewTrieState(nil)
				storageState.EXPECT().TrieState(&trie.EmptyHash, trie.V0).
					Return(trieState, nil)

				runtimeInstance.EXPECT().SetContextStorage(trieState)

				expectedBlockArgument := &types.Block{
					Header: types.Header{
						Number:     1,
						ParentHash: common.Hash{1},
					},
				}
				runtimeInstance.EXPECT().ExecuteBlock(expectedBlockArgument).
					Return(nil, mockError)

				return chainProcessor{
					blockState:   blockState,
					storageState: storageState,
				}
			},
			block: &types.Block{
				Header: types.Header{
					Number:     1,
					ParentHash: common.Hash{1},
				},
			},
			wantErr:    mockError,
			errMessage: "failed to execute block 1: test mock error",
		},
		"handle block import error": {
			chainProcessorBuilder: func(ctrl *gomock.Controller) chainProcessor {
				blockState := NewMockBlockState(ctrl)
				parentHeader := &types.Header{Number: 1, StateRoot: trie.EmptyHash}
				blockState.EXPECT().GetHeader(common.Hash{1}).
					Return(parentHeader, nil)

				parentHash := parentHeader.Hash()
				runtimeInstance := NewMockInstance(ctrl)
				blockState.EXPECT().GetRuntime(&parentHash).
					Return(runtimeInstance, nil)

				runtimeInstance.EXPECT().StateVersion().Return(trie.V0)

				storageState := NewMockStorageState(ctrl)
				lockCall := storageState.EXPECT().Lock()
				storageState.EXPECT().Unlock().After(lockCall)

				trieState := storage.NewTrieState(nil)
				storageState.EXPECT().TrieState(&trie.EmptyHash, trie.V0).
					Return(trieState, nil)

				runtimeInstance.EXPECT().SetContextStorage(trieState)

				expectedBlockArgument := &types.Block{
					Header: types.Header{ParentHash: common.Hash{1}},
				}
				runtimeInstance.EXPECT().ExecuteBlock(expectedBlockArgument).
					Return(nil, nil)

				blockImportHandler := NewMockBlockImportHandler(ctrl)
				blockImportHandler.EXPECT().HandleBlockImport(expectedBlockArgument, trieState).
					Return(mockError)

				return chainProcessor{
					blockState:         blockState,
					storageState:       storageState,
					blockImportHandler: blockImportHandler,
				}
			},
			block: &types.Block{
				Header: types.Header{ParentHash: common.Hash{1}},
			},
			wantErr:    mockError,
			errMessage: "handling block import: test mock error",
		},
		"success": {
			chainProcessorBuilder: func(ctrl *gomock.Controller) chainProcessor {
				blockState := NewMockBlockState(ctrl)
				parentHeader := &types.Header{Number: 1, StateRoot: trie.EmptyHash}
				blockState.EXPECT().GetHeader(common.Hash{1}).
					Return(parentHeader, nil)

				parentHash := parentHeader.Hash()
				runtimeInstance := NewMockInstance(ctrl)
				blockState.EXPECT().GetRuntime(&parentHash).
					Return(runtimeInstance, nil)

				runtimeInstance.EXPECT().StateVersion().Return(trie.V0)

				storageState := NewMockStorageState(ctrl)
				lockCall := storageState.EXPECT().Lock()
				storageState.EXPECT().Unlock().After(lockCall)

				trieState := storage.NewTrieState(nil)
				storageState.EXPECT().TrieState(&trie.EmptyHash, trie.V0).
					Return(trieState, nil)

				runtimeInstance.EXPECT().SetContextStorage(trieState)

				expectedBlockArgument := &types.Block{
					Header: types.Header{
						ParentHash: common.Hash{1},
						Number:     3,
					},
				}
				runtimeInstance.EXPECT().ExecuteBlock(expectedBlockArgument).
					Return(nil, nil)

				blockImportHandler := NewMockBlockImportHandler(ctrl)
				blockImportHandler.EXPECT().HandleBlockImport(expectedBlockArgument, trieState).
					Return(nil)

				telemetryClient := NewMockClient(ctrl)
				blockHeader := types.Header{
					ParentHash: common.Hash{1},
					Number:     3,
				}
				blockHash := blockHeader.Hash()
				telemetryBlockImport := &telemetry.BlockImport{
					BestHash: &blockHash,
					Height:   3,
					Origin:   "NetworkInitialSync",
				}
				telemetryClient.EXPECT().SendMessage(telemetryBlockImport)

				return chainProcessor{
					blockState:         blockState,
					storageState:       storageState,
					blockImportHandler: blockImportHandler,
					telemetry:          telemetryClient,
				}
			},
			block: &types.Block{
				Header: types.Header{
					ParentHash: common.Hash{1},
					Number:     3,
				},
			},
		},
	}
	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			processor := tt.chainProcessorBuilder(ctrl)

			err := processor.handleBlock(tt.block)
			assert.ErrorIs(t, err, tt.wantErr)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.errMessage)
			}
		})
	}

	t.Run("panics on different parent state root", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)

		blockState := NewMockBlockState(ctrl)
		parentHeader := &types.Header{Number: 1, StateRoot: common.Hash{2}}
		blockState.EXPECT().GetHeader(common.Hash{1}).
			Return(parentHeader, nil)

		parentHash := parentHeader.Hash()
		runtimeInstance := NewMockInstance(ctrl)
		blockState.EXPECT().GetRuntime(&parentHash).
			Return(runtimeInstance, nil)

		runtimeInstance.EXPECT().StateVersion().Return(trie.V0)

		storageState := NewMockStorageState(ctrl)
		lockCall := storageState.EXPECT().Lock()
		storageState.EXPECT().Unlock().After(lockCall)
		trieState := storage.NewTrieState(nil)
		storageState.EXPECT().TrieState(&common.Hash{2}, trie.V0).
			Return(trieState, nil)

		processor := chainProcessor{
			blockState:   blockState,
			storageState: storageState,
		}

		block := &types.Block{
			Header: types.Header{ParentHash: common.Hash{1}},
		}

		const expectedPanicValue = "parent state root does not match snapshot state root"
		assert.PanicsWithValue(t, expectedPanicValue, func() {
			_ = processor.handleBlock(block)
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
		"nil justification and header": {
			chainProcessorBuilder: func(ctrl *gomock.Controller) chainProcessor {
				return chainProcessor{}
			},
		},
		"invalid justification": {
			chainProcessorBuilder: func(ctrl *gomock.Controller) chainProcessor {
				mockFinalityGadget := NewMockFinalityGadget(ctrl)
				mockFinalityGadget.EXPECT().VerifyBlockJustification(headerHash,
					[]byte(`x`)).Return(nil, errTest)
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
		"set justification error": {
			chainProcessorBuilder: func(ctrl *gomock.Controller) chainProcessor {
				mockBlockState := NewMockBlockState(ctrl)
				mockBlockState.EXPECT().SetJustification(headerHash, []byte(`xx`)).Return(errTest)
				mockFinalityGadget := NewMockFinalityGadget(ctrl)
				mockFinalityGadget.EXPECT().VerifyBlockJustification(headerHash, []byte(`xx`)).Return([]byte(`xx`), nil)
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
		"base case set": {
			chainProcessorBuilder: func(ctrl *gomock.Controller) chainProcessor {
				mockBlockState := NewMockBlockState(ctrl)
				mockBlockState.EXPECT().SetJustification(headerHash, []byte(`1234`)).Return(nil)
				mockFinalityGadget := NewMockFinalityGadget(ctrl)
				mockFinalityGadget.EXPECT().VerifyBlockJustification(headerHash, []byte(`1234`)).Return([]byte(`1234`), nil)
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
		blockData             *types.BlockData
		errSentinel           error
		errString             string
	}{
		"nil block data": {
			chainProcessorBuilder: func(_ *gomock.Controller) chainProcessor {
				return chainProcessor{}
			},
			errSentinel: ErrNilBlockData,
			errString:   "got nil BlockData",
		},
		"has header error": {
			chainProcessorBuilder: func(ctrl *gomock.Controller) chainProcessor {
				mockBlockState := NewMockBlockState(ctrl)
				mockBlockState.EXPECT().HasHeader(common.Hash{1}).Return(false, mockError)
				return chainProcessor{
					blockState: mockBlockState,
				}
			},
			blockData: &types.BlockData{
				Hash: common.Hash{1},
			},
			errSentinel: mockError,
			errString:   "checking header in block state: mock test error",
		},
		"has block body error": {
			chainProcessorBuilder: func(ctrl *gomock.Controller) chainProcessor {
				mockBlockState := NewMockBlockState(ctrl)
				mockBlockState.EXPECT().HasHeader(common.Hash{1}).Return(false, nil)
				mockBlockState.EXPECT().HasBlockBody(common.Hash{1}).Return(false, mockError)
				return chainProcessor{
					blockState: mockBlockState,
				}
			},
			blockData: &types.BlockData{
				Hash: common.Hash{1},
			},
			errSentinel: mockError,
			errString:   "checking block body in block state: mock test error",
		},
		"has header and body - GetBlockByHash error": {
			chainProcessorBuilder: func(ctrl *gomock.Controller) chainProcessor {
				mockBlockState := NewMockBlockState(ctrl)
				mockBlockState.EXPECT().HasHeader(common.Hash{1}).Return(true, nil)
				mockBlockState.EXPECT().HasBlockBody(common.Hash{1}).Return(true, nil)
				mockBlockState.EXPECT().GetBlockByHash(common.Hash{1}).Return(nil, mockError)
				return chainProcessor{
					blockState: mockBlockState,
				}
			},
			blockData: &types.BlockData{
				Hash: common.Hash{1},
			},
			errSentinel: mockError,
			errString:   "getting block by hash from block state: mock test error",
		},
		"has header and body - block exists in blocktree": {
			chainProcessorBuilder: func(ctrl *gomock.Controller) chainProcessor {
				mockBlock := &types.Block{
					Header: types.Header{
						ParentHash: common.Hash{2},
					},
				}
				mockBlockState := NewMockBlockState(ctrl)
				mockBlockState.EXPECT().HasHeader(common.Hash{1}).Return(true, nil)
				mockBlockState.EXPECT().HasBlockBody(common.Hash{1}).Return(true, nil)
				mockBlockState.EXPECT().GetBlockByHash(common.Hash{1}).
					Return(mockBlock, nil)
				mockBlockState.EXPECT().AddBlockToBlockTree(mockBlock).
					Return(fmt.Errorf("%w", blocktree.ErrBlockExists))
				return chainProcessor{
					blockState: mockBlockState,
				}
			},
			blockData: &types.BlockData{
				Hash: common.Hash{1},
			},
		},
		"has header and body - fail to add block to block tree": {
			chainProcessorBuilder: func(ctrl *gomock.Controller) chainProcessor {
				header := types.Header{
					Number: uint(1),
				}
				block := &types.Block{
					Header: header,
				}

				blockState := NewMockBlockState(ctrl)
				blockState.EXPECT().HasHeader(common.Hash{1}).Return(true, nil)
				blockState.EXPECT().HasBlockBody(common.Hash{1}).Return(true, nil)
				blockState.EXPECT().GetBlockByHash(common.Hash{1}).
					Return(block, nil)
				blockState.EXPECT().AddBlockToBlockTree(block).Return(mockError)

				return chainProcessor{
					blockState: blockState,
				}
			},
			blockData: &types.BlockData{
				Hash: common.Hash{1},
			},
			errSentinel: mockError,
			errString:   "adding block to block tree: mock test error",
		},
		"has header and body - fail to handle justification": {
			chainProcessorBuilder: func(ctrl *gomock.Controller) chainProcessor {
				header := types.Header{
					Number: uint(1),
				}
				headerHash := header.Hash()
				block := &types.Block{
					Header: header,
				}

				blockState := NewMockBlockState(ctrl)
				blockState.EXPECT().HasHeader(common.Hash{1}).Return(true, nil)
				blockState.EXPECT().HasBlockBody(common.Hash{1}).Return(true, nil)
				blockState.EXPECT().GetBlockByHash(common.Hash{1}).
					Return(block, nil)
				blockState.EXPECT().AddBlockToBlockTree(block).Return(nil)

				finalityGadget := NewMockFinalityGadget(ctrl)
				finalityGadget.EXPECT().
					VerifyBlockJustification(headerHash, []byte{1, 2, 3}).
					Return(nil, mockError)

				return chainProcessor{
					blockState:     blockState,
					finalityGadget: finalityGadget,
				}
			},
			blockData: &types.BlockData{
				Hash:          common.Hash{1},
				Justification: &[]byte{1, 2, 3},
			},
			errSentinel: mockError,
			errString: "handling justification: " +
				"verifying block number 1 justification: mock test error",
		},
		"has header and body - get runtime error": {
			chainProcessorBuilder: func(ctrl *gomock.Controller) chainProcessor {
				expectedJustification := []byte{1, 2, 3}
				blockHeader := types.Header{
					ParentHash: common.Hash{2},
					StateRoot:  common.Hash{3},
				}
				block := &types.Block{
					Header: blockHeader,
				}
				mockBlockState := NewMockBlockState(ctrl)
				mockBlockState.EXPECT().HasHeader(common.Hash{1}).Return(true, nil)
				mockBlockState.EXPECT().HasBlockBody(common.Hash{1}).Return(true, nil)
				mockBlockState.EXPECT().GetBlockByHash(common.Hash{1}).Return(block, nil)
				mockBlockState.EXPECT().AddBlockToBlockTree(block).Return(nil)

				mockFinalityGadget := NewMockFinalityGadget(ctrl)
				mockFinalityGadget.EXPECT().
					VerifyBlockJustification(blockHeader.Hash(), expectedJustification).
					Return(expectedJustification, nil)

				mockBlockState.EXPECT().
					SetJustification(blockHeader.Hash(), expectedJustification).
					Return(nil)

				mockBlockState.EXPECT().GetRuntime(&common.Hash{2}).
					Return(nil, mockError)

				return chainProcessor{
					blockState:     mockBlockState,
					finalityGadget: mockFinalityGadget,
				}
			},
			blockData: &types.BlockData{
				Hash:          common.Hash{1},
				Justification: &[]byte{1, 2, 3},
			},
			errSentinel: mockError,
			errString:   "getting runtime instance: mock test error",
		},
		"has header and body - trie state error": {
			chainProcessorBuilder: func(ctrl *gomock.Controller) chainProcessor {
				expectedJustification := []byte{1, 2, 3}
				blockHeader := types.Header{
					ParentHash: common.Hash{2},
					StateRoot:  common.Hash{3},
				}
				block := &types.Block{
					Header: blockHeader,
				}
				mockBlockState := NewMockBlockState(ctrl)
				mockBlockState.EXPECT().HasHeader(common.Hash{1}).Return(true, nil)
				mockBlockState.EXPECT().HasBlockBody(common.Hash{1}).Return(true, nil)
				mockBlockState.EXPECT().GetBlockByHash(common.Hash{1}).Return(block, nil)
				mockBlockState.EXPECT().AddBlockToBlockTree(block).Return(nil)

				mockFinalityGadget := NewMockFinalityGadget(ctrl)
				mockFinalityGadget.EXPECT().
					VerifyBlockJustification(blockHeader.Hash(), expectedJustification).
					Return(expectedJustification, nil)

				mockBlockState.EXPECT().
					SetJustification(blockHeader.Hash(), expectedJustification).
					Return(nil)

				mockInstance := NewMockInstance(ctrl)
				mockBlockState.EXPECT().GetRuntime(&common.Hash{2}).
					Return(mockInstance, nil)
				const stateVersion = trie.V0
				mockInstance.EXPECT().StateVersion().Return(stateVersion)

				storageState := NewMockStorageState(ctrl)
				storageState.EXPECT().TrieState(&common.Hash{3}, stateVersion).
					Return(nil, mockError)

				return chainProcessor{
					blockState:     mockBlockState,
					finalityGadget: mockFinalityGadget,
					storageState:   storageState,
				}
			},
			blockData: &types.BlockData{
				Hash:          common.Hash{1},
				Justification: &[]byte{1, 2, 3},
			},
			errSentinel: mockError,
			errString:   "running trie state: mock test error",
		},
		"has header and body - block import error": {
			chainProcessorBuilder: func(ctrl *gomock.Controller) chainProcessor {
				expectedJustification := []byte{1, 2, 3}
				blockHeader := types.Header{
					ParentHash: common.Hash{2},
					StateRoot:  common.Hash{3},
				}
				block := &types.Block{
					Header: blockHeader,
				}
				mockBlockState := NewMockBlockState(ctrl)
				mockBlockState.EXPECT().HasHeader(common.Hash{1}).Return(true, nil)
				mockBlockState.EXPECT().HasBlockBody(common.Hash{1}).Return(true, nil)
				mockBlockState.EXPECT().GetBlockByHash(common.Hash{1}).Return(block, nil)
				mockBlockState.EXPECT().AddBlockToBlockTree(block).Return(nil)

				mockFinalityGadget := NewMockFinalityGadget(ctrl)
				mockFinalityGadget.EXPECT().
					VerifyBlockJustification(blockHeader.Hash(), expectedJustification).
					Return(expectedJustification, nil)

				mockBlockState.EXPECT().
					SetJustification(blockHeader.Hash(), expectedJustification).
					Return(nil)

				mockInstance := NewMockInstance(ctrl)
				mockBlockState.EXPECT().GetRuntime(&common.Hash{2}).
					Return(mockInstance, nil)
				const stateVersion = trie.V0
				mockInstance.EXPECT().StateVersion().Return(stateVersion)

				storageState := NewMockStorageState(ctrl)
				someTrie := trie.NewEmptyTrie()
				someTrie.Put([]byte{1}, []byte{2}, stateVersion)
				trieState := storage.NewTrieState(someTrie)
				storageState.EXPECT().TrieState(&common.Hash{3}, stateVersion).
					Return(trieState, nil)

				blockImportHandler := NewMockBlockImportHandler(ctrl)
				blockImportHandler.EXPECT().HandleBlockImport(block, trieState).
					Return(mockError)

				return chainProcessor{
					blockState:         mockBlockState,
					finalityGadget:     mockFinalityGadget,
					storageState:       storageState,
					blockImportHandler: blockImportHandler,
				}
			},
			blockData: &types.BlockData{
				Hash:          common.Hash{1},
				Justification: &[]byte{1, 2, 3},
			},
			errSentinel: mockError,
			errString:   "handling block import: mock test error",
		},
		"has header and body - success": {
			chainProcessorBuilder: func(ctrl *gomock.Controller) chainProcessor {
				expectedJustification := []byte{1, 2, 3}
				blockHeader := types.Header{
					ParentHash: common.Hash{2},
					StateRoot:  common.Hash{3},
				}
				block := &types.Block{
					Header: blockHeader,
				}
				mockBlockState := NewMockBlockState(ctrl)
				mockBlockState.EXPECT().HasHeader(common.Hash{1}).Return(true, nil)
				mockBlockState.EXPECT().HasBlockBody(common.Hash{1}).Return(true, nil)
				mockBlockState.EXPECT().GetBlockByHash(common.Hash{1}).Return(block, nil)
				mockBlockState.EXPECT().AddBlockToBlockTree(block).Return(nil)

				mockFinalityGadget := NewMockFinalityGadget(ctrl)
				mockFinalityGadget.EXPECT().
					VerifyBlockJustification(blockHeader.Hash(), expectedJustification).
					Return(expectedJustification, nil)

				mockBlockState.EXPECT().
					SetJustification(blockHeader.Hash(), expectedJustification).
					Return(nil)

				mockInstance := NewMockInstance(ctrl)
				mockBlockState.EXPECT().GetRuntime(&common.Hash{2}).
					Return(mockInstance, nil)
				const stateVersion = trie.V0
				mockInstance.EXPECT().StateVersion().Return(stateVersion)

				storageState := NewMockStorageState(ctrl)
				someTrie := trie.NewEmptyTrie()
				someTrie.Put([]byte{1}, []byte{2}, stateVersion)
				trieState := storage.NewTrieState(someTrie)
				storageState.EXPECT().TrieState(&common.Hash{3}, stateVersion).
					Return(trieState, nil)

				blockImportHandler := NewMockBlockImportHandler(ctrl)
				blockImportHandler.EXPECT().HandleBlockImport(block, trieState).
					Return(nil)

				return chainProcessor{
					blockState:         mockBlockState,
					finalityGadget:     mockFinalityGadget,
					storageState:       storageState,
					blockImportHandler: blockImportHandler,
				}
			},
			blockData: &types.BlockData{
				Hash:          common.Hash{1},
				Justification: &[]byte{1, 2, 3},
			},
		},
		"block data has header and body - verify block error": {
			chainProcessorBuilder: func(ctrl *gomock.Controller) chainProcessor {
				blockState := NewMockBlockState(ctrl)
				blockState.EXPECT().HasHeader(common.Hash{1}).Return(false, nil)
				blockState.EXPECT().HasBlockBody(common.Hash{1}).Return(true, nil)

				babeVerifier := NewMockBabeVerifier(ctrl)
				babeVerifier.EXPECT().VerifyBlock(&types.Header{ParentHash: common.Hash{2}}).
					Return(mockError)

				return chainProcessor{
					blockState:   blockState,
					babeVerifier: babeVerifier,
				}
			},
			blockData: &types.BlockData{
				Header: &types.Header{ParentHash: common.Hash{2}},
				Body:   &types.Body{},
				Hash:   common.Hash{1},
			},
			errSentinel: mockError,
			errString:   "babe verifying block: mock test error",
		},
		"block data has header and body - handle block error": {
			chainProcessorBuilder: func(ctrl *gomock.Controller) chainProcessor {
				blockState := NewMockBlockState(ctrl)
				blockState.EXPECT().HasHeader(common.Hash{1}).Return(false, nil)
				blockState.EXPECT().HasBlockBody(common.Hash{1}).Return(true, nil)

				babeVerifier := NewMockBabeVerifier(ctrl)
				babeVerifier.EXPECT().VerifyBlock(&types.Header{ParentHash: common.Hash{2}}).
					Return(nil)

				blockState.EXPECT().GetHeader(common.Hash{2}).
					Return(nil, mockError)

				return chainProcessor{
					blockState:   blockState,
					babeVerifier: babeVerifier,
				}
			},
			blockData: &types.BlockData{
				Header: &types.Header{ParentHash: common.Hash{2}},
				Body:   &types.Body{},
				Hash:   common.Hash{1},
			},
			errSentinel: errFailedToGetParent,
			errString:   "handling block: failed to get parent header: mock test error",
		},
		"has no header - handle justification error": {
			chainProcessorBuilder: func(ctrl *gomock.Controller) chainProcessor {
				blockState := NewMockBlockState(ctrl)
				blockState.EXPECT().HasHeader(common.Hash{1}).Return(false, nil)
				blockState.EXPECT().HasBlockBody(common.Hash{1}).Return(true, nil)

				finalityGadget := NewMockFinalityGadget(ctrl)
				header := types.Header{ParentHash: common.Hash{2}}
				headerHash := header.Hash()
				finalityGadget.EXPECT().
					VerifyBlockJustification(headerHash, []byte{1, 2, 3}).
					Return(nil, mockError)

				return chainProcessor{
					blockState:     blockState,
					finalityGadget: finalityGadget,
				}
			},
			blockData: &types.BlockData{
				Header:        &types.Header{ParentHash: common.Hash{2}},
				Hash:          common.Hash{1},
				Justification: &[]byte{1, 2, 3},
			},
			errSentinel: mockError,
			errString: "handling justification: " +
				"verifying block number 0 justification: mock test error",
		},
		"compare and set block data error": {
			chainProcessorBuilder: func(ctrl *gomock.Controller) chainProcessor {
				blockState := NewMockBlockState(ctrl)
				blockState.EXPECT().HasHeader(common.Hash{1}).Return(false, nil)
				blockState.EXPECT().HasBlockBody(common.Hash{1}).Return(true, nil)

				blockState.EXPECT().CompareAndSetBlockData(&types.BlockData{Hash: common.Hash{1}}).
					Return(mockError)

				return chainProcessor{
					blockState: blockState,
				}
			},
			blockData: &types.BlockData{
				Hash: common.Hash{1},
			},
			errSentinel: mockError,
			errString: "comparing and setting block data: " +
				"mock test error",
		},
		"shortest success code path": {
			chainProcessorBuilder: func(ctrl *gomock.Controller) chainProcessor {
				blockState := NewMockBlockState(ctrl)
				blockState.EXPECT().HasHeader(common.Hash{1}).Return(false, nil)
				blockState.EXPECT().HasBlockBody(common.Hash{1}).Return(true, nil)

				blockState.EXPECT().CompareAndSetBlockData(&types.BlockData{Hash: common.Hash{1}}).
					Return(nil)

				return chainProcessor{
					blockState: blockState,
				}
			},
			blockData: &types.BlockData{
				Hash: common.Hash{1},
			},
		},
		"longest success code path": {
			chainProcessorBuilder: func(ctrl *gomock.Controller) chainProcessor {
				expectedBlockData := &types.BlockData{
					Header: &types.Header{
						Number:     6,
						ParentHash: common.Hash{2},
					},
					Body:          &types.Body{{3}},
					Hash:          common.Hash{1},
					Justification: &[]byte{4},
				}

				blockState := NewMockBlockState(ctrl)
				blockState.EXPECT().HasHeader(expectedBlockData.Hash).Return(false, nil)
				blockState.EXPECT().HasBlockBody(expectedBlockData.Hash).Return(true, nil)

				babeVerifier := NewMockBabeVerifier(ctrl)
				blockDataHeader := &types.Header{
					Number:     6,
					ParentHash: common.Hash{2},
				}
				babeVerifier.EXPECT().VerifyBlock(blockDataHeader).Return(nil)

				transactionState := NewMockTransactionState(ctrl)
				transactionState.EXPECT().RemoveExtrinsic(types.Extrinsic{3})

				parentStateRoot := trie.EmptyHash
				parentHeader := &types.Header{StateRoot: parentStateRoot}
				blockState.EXPECT().GetHeader(common.Hash{2}).
					Return(parentHeader, nil)
				parentHeaderHash := parentHeader.Hash()
				instance := NewMockInstance(ctrl)
				blockState.EXPECT().GetRuntime(&parentHeaderHash).
					Return(instance, nil)
				instance.EXPECT().StateVersion().Return(trie.V0)

				storageState := NewMockStorageState(ctrl)
				lockCall := storageState.EXPECT().Lock()
				storageState.EXPECT().Unlock().After(lockCall)
				trieState := storage.NewTrieState(nil)
				trieStateCall := storageState.EXPECT().
					TrieState(&parentStateRoot, trie.V0).
					Return(trieState, nil).After(lockCall)

				instance.EXPECT().SetContextStorage(trieState).After(trieStateCall)
				block := &types.Block{
					Header: *expectedBlockData.Header,
					Body:   *expectedBlockData.Body,
				}
				instance.EXPECT().ExecuteBlock(block).Return(nil, nil)

				blockImportHandler := NewMockBlockImportHandler(ctrl)
				blockImportHandler.EXPECT().HandleBlockImport(block, trieState).
					Return(nil)

				telemetryClient := NewMockClient(ctrl)
				blockHeaderHash := expectedBlockData.Header.Hash()
				blockNumber := expectedBlockData.Header.Number
				telemetryMessage := telemetry.NewBlockImport(
					&blockHeaderHash, blockNumber, "NetworkInitialSync")
				telemetryClient.EXPECT().SendMessage(telemetryMessage)

				finalityGadget := NewMockFinalityGadget(ctrl)
				finalityGadget.EXPECT().
					VerifyBlockJustification(blockHeaderHash, []byte{4}).
					Return([]byte{5}, nil)
				blockState.EXPECT().SetJustification(blockHeaderHash, []byte{5}).
					Return(nil)

				blockState.EXPECT().CompareAndSetBlockData(expectedBlockData).
					Return(nil)

				return chainProcessor{
					blockState:         blockState,
					babeVerifier:       babeVerifier,
					transactionState:   transactionState,
					storageState:       storageState,
					blockImportHandler: blockImportHandler,
					telemetry:          telemetryClient,
					finalityGadget:     finalityGadget,
				}
			},
			blockData: &types.BlockData{
				Header: &types.Header{
					Number:     6,
					ParentHash: common.Hash{2},
				},
				Body:          &types.Body{{3}},
				Hash:          common.Hash{1},
				Justification: &[]byte{4},
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

			assert.ErrorIs(t, err, tt.errSentinel)
			if tt.errSentinel != nil {
				assert.EqualError(t, err, tt.errString)
			}
		})
	}
}

func Test_chainProcessor_processReadyBlocks(t *testing.T) {
	t.Parallel()

	mockError := errors.New("test mock error")

	tests := map[string]struct {
		chainProcessorBuilder func(ctrl *gomock.Controller, done chan<- struct{}) chainProcessor
		blockData             *types.BlockData
	}{
		"context canceled": {
			chainProcessorBuilder: func(ctrl *gomock.Controller, done chan<- struct{}) chainProcessor {
				defer close(done)
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				<-ctx.Done()
				return chainProcessor{
					ctx:         ctx,
					cancel:      cancel,
					readyBlocks: newBlockQueue(1),
				}
			},
			blockData: &types.BlockData{
				Hash: common.Hash{1},
			},
		},
		"process block data success": {
			chainProcessorBuilder: func(ctrl *gomock.Controller, done chan<- struct{}) chainProcessor {
				mockBlockState := NewMockBlockState(ctrl)
				mockBlockState.EXPECT().HasHeader(common.Hash{1}).Return(false, nil)
				mockBlockState.EXPECT().HasBlockBody(common.Hash{1}).Return(false, nil)
				mockBlockState.EXPECT().CompareAndSetBlockData(&types.BlockData{
					Hash: common.Hash{1},
				}).DoAndReturn(func(*types.BlockData) error {
					close(done)
					return nil
				})

				ctx, cancel := context.WithCancel(context.Background())
				return chainProcessor{
					ctx:         ctx,
					cancel:      cancel,
					readyBlocks: newBlockQueue(1),
					blockState:  mockBlockState,
				}
			},
			blockData: &types.BlockData{
				Hash: common.Hash{1},
			},
		},
		"process block data error": {
			chainProcessorBuilder: func(ctrl *gomock.Controller, done chan<- struct{}) chainProcessor {
				mockBlockState := NewMockBlockState(ctrl)
				mockBlockState.EXPECT().HasHeader(common.Hash{1}).Return(true, nil)
				mockBlockState.EXPECT().HasBlockBody(common.Hash{1}).Return(true, nil)
				mockBlockState.EXPECT().GetBlockByHash(common.Hash{1}).
					DoAndReturn(func(_ common.Hash) (block *types.Block, err error) {
						defer close(done)
						return nil, mockError
					})

				ctx, cancel := context.WithCancel(context.Background())
				return chainProcessor{
					ctx:         ctx,
					cancel:      cancel,
					readyBlocks: newBlockQueue(1),
					blockState:  mockBlockState,
				}
			},
			blockData: &types.BlockData{
				Hash: common.Hash{1},
			},
		},
		"add block error": {
			chainProcessorBuilder: func(ctrl *gomock.Controller, done chan<- struct{}) chainProcessor {
				mockBlockState := NewMockBlockState(ctrl)
				mockBlockState.EXPECT().HasHeader(common.Hash{1}).Return(false, nil)
				mockBlockState.EXPECT().HasBlockBody(common.Hash{1}).Return(false, nil)

				babeVerifier := NewMockBabeVerifier(ctrl)
				babeVerifier.EXPECT().
					VerifyBlock(&types.Header{ParentHash: common.Hash{2}}).
					Return(nil)

				transactionState := NewMockTransactionState(ctrl)
				transactionState.EXPECT().RemoveExtrinsic(types.Extrinsic{3})

				mockBlockState.EXPECT().GetHeader(common.Hash{2}).
					Return(nil, mockError)

				pendingBlocks := NewMockDisjointBlockSet(ctrl)
				block := &types.Block{
					Header: types.Header{ParentHash: common.Hash{2}},
					Body:   types.Body{{3}},
				}
				pendingBlocks.EXPECT().addBlock(block).
					DoAndReturn(func(block *types.Block) error {
						defer close(done)
						return mockError
					})

				ctx, cancel := context.WithCancel(context.Background())
				return chainProcessor{
					ctx:              ctx,
					cancel:           cancel,
					readyBlocks:      newBlockQueue(1),
					blockState:       mockBlockState,
					babeVerifier:     babeVerifier,
					transactionState: transactionState,
					pendingBlocks:    pendingBlocks,
				}
			},
			blockData: &types.BlockData{
				Hash:   common.Hash{1},
				Header: &types.Header{ParentHash: common.Hash{2}},
				Body:   &types.Body{{3}},
			},
		},
		"add block success": {
			chainProcessorBuilder: func(ctrl *gomock.Controller, done chan<- struct{}) chainProcessor {
				mockBlockState := NewMockBlockState(ctrl)
				mockBlockState.EXPECT().HasHeader(common.Hash{1}).Return(false, nil)
				mockBlockState.EXPECT().HasBlockBody(common.Hash{1}).Return(false, nil)

				babeVerifier := NewMockBabeVerifier(ctrl)
				babeVerifier.EXPECT().
					VerifyBlock(&types.Header{ParentHash: common.Hash{2}}).
					Return(nil)

				transactionState := NewMockTransactionState(ctrl)
				transactionState.EXPECT().RemoveExtrinsic(types.Extrinsic{3})

				mockBlockState.EXPECT().GetHeader(common.Hash{2}).
					Return(nil, mockError)

				pendingBlocks := NewMockDisjointBlockSet(ctrl)
				block := &types.Block{
					Header: types.Header{ParentHash: common.Hash{2}},
					Body:   types.Body{{3}},
				}
				pendingBlocks.EXPECT().addBlock(block).
					DoAndReturn(func(block *types.Block) error {
						defer close(done)
						return nil
					})

				ctx, cancel := context.WithCancel(context.Background())
				return chainProcessor{
					ctx:              ctx,
					cancel:           cancel,
					readyBlocks:      newBlockQueue(1),
					blockState:       mockBlockState,
					babeVerifier:     babeVerifier,
					transactionState: transactionState,
					pendingBlocks:    pendingBlocks,
				}
			},
			blockData: &types.BlockData{
				Hash:   common.Hash{1},
				Header: &types.Header{ParentHash: common.Hash{2}},
				Body:   &types.Body{{3}},
			},
		},
	}
	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			lastMockCalled := make(chan struct{})

			processor := tt.chainProcessorBuilder(ctrl, lastMockCalled)

			go processor.processReadyBlocks()

			processor.readyBlocks.push(tt.blockData)
			<-lastMockCalled
			processor.cancel()
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
