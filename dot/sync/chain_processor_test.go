// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package sync

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/dot/telemetry"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/runtime/storage"
	"github.com/ChainSafe/gossamer/lib/trie"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//go:generate mockgen -destination=mock_instance_test.go -package=$GOPACKAGE github.com/ChainSafe/gossamer/lib/runtime Instance

func Test_chainProcessor_handleBlock(t *testing.T) {
	t.Parallel()
	mockError := errors.New("test mock error")
	testHash := common.MustHexToHash("0x03170a2e7597b7b7e3d84c05391d139a62b157e78786d8c082f29dcf4c111314")
	testParentHash := common.MustHexToHash("0x7db9db5ed9967b80143100189ba69d9e4deab85ac3570e5df25686cabe32964a")
	mockTrieState, _ := storage.NewTrieState(nil)

	tests := map[string]struct {
		chainProcessorBuilder func(ctrl *gomock.Controller) chainProcessor
		block                 *types.Block
		wantErr               error
	}{
		"nil block": {
			chainProcessorBuilder: func(ctrl *gomock.Controller) chainProcessor {
				return chainProcessor{}
			},
			wantErr: errBlockOrBodyNil,
		},
		"handle getHeader error": {
			chainProcessorBuilder: func(ctrl *gomock.Controller) chainProcessor {
				return chainProcessor{
					blockState: func(ctrl *gomock.Controller) BlockState {
						mockBlockState := NewMockBlockState(ctrl)
						mockBlockState.EXPECT().GetHeader(common.Hash{}).Return(nil, mockError)
						return mockBlockState
					}(ctrl),
				}
			},
			block: &types.Block{
				Body: types.Body{},
			},
			wantErr: mockError,
		},
		"handle trieState error": {
			chainProcessorBuilder: func(ctrl *gomock.Controller) chainProcessor {
				return chainProcessor{
					blockState: func(ctrl *gomock.Controller) BlockState {
						mockBlockState := NewMockBlockState(ctrl)
						mockBlockState.EXPECT().GetHeader(common.Hash{}).Return(&types.Header{}, nil)
						return mockBlockState
					}(ctrl),
					storageState: func(ctrl *gomock.Controller) StorageState {
						mockStorageState := NewMockStorageState(ctrl)
						mockStorageState.EXPECT().Lock()
						mockStorageState.EXPECT().TrieState(&common.Hash{}).Return(nil, mockError)
						mockStorageState.EXPECT().Unlock()
						return mockStorageState
					}(ctrl),
				}
			},
			block: &types.Block{
				Body: types.Body{},
			},
			wantErr: mockError,
		},
		"handle getRuntime error": {
			chainProcessorBuilder: func(ctrl *gomock.Controller) chainProcessor {
				return chainProcessor{
					blockState: func(ctrl *gomock.Controller) BlockState {
						mockBlockState := NewMockBlockState(ctrl)
						mockBlockState.EXPECT().GetHeader(common.Hash{}).Return(&types.Header{
							StateRoot: testHash,
						}, nil)
						mockBlockState.EXPECT().GetRuntime(&testParentHash).Return(nil, mockError)
						return mockBlockState
					}(ctrl),
					storageState: func(ctrl *gomock.Controller) StorageState {
						mockStorageState := NewMockStorageState(ctrl)
						mockStorageState.EXPECT().Lock()
						mockStorageState.EXPECT().TrieState(&testHash).Return(mockTrieState, nil)
						mockStorageState.EXPECT().Unlock()
						return mockStorageState
					}(ctrl),
				}
			},
			block: &types.Block{
				Body: types.Body{},
			},
			wantErr: mockError,
		},
		"handle runtime ExecuteBlock error": {
			chainProcessorBuilder: func(ctrl *gomock.Controller) chainProcessor {
				return chainProcessor{
					blockState: func(ctrl *gomock.Controller) BlockState {
						mockBlockState := NewMockBlockState(ctrl)
						mockBlockState.EXPECT().GetHeader(common.Hash{}).Return(&types.Header{
							StateRoot: testHash,
						}, nil)
						mockInstance := NewMockInstance(ctrl)
						mockInstance.EXPECT().SetContextStorage(mockTrieState)
						mockInstance.EXPECT().ExecuteBlock(&types.Block{Body: types.Body{}}).Return(nil, mockError)
						mockBlockState.EXPECT().GetRuntime(&testParentHash).Return(mockInstance, nil)
						return mockBlockState
					}(ctrl),
					storageState: func(ctrl *gomock.Controller) StorageState {
						mockStorageState := NewMockStorageState(ctrl)
						mockStorageState.EXPECT().Lock()
						mockStorageState.EXPECT().TrieState(&testHash).Return(mockTrieState, nil)
						mockStorageState.EXPECT().Unlock()
						return mockStorageState
					}(ctrl),
				}
			},
			block: &types.Block{
				Body: types.Body{},
			},
			wantErr: mockError,
		},
		"handle block import error": {
			chainProcessorBuilder: func(ctrl *gomock.Controller) chainProcessor {
				return chainProcessor{
					blockState: func(ctrl *gomock.Controller) BlockState {
						mockBlockState := NewMockBlockState(ctrl)
						mockBlockState.EXPECT().GetHeader(common.Hash{}).Return(&types.Header{
							StateRoot: testHash,
						}, nil)
						mockInstance := NewMockInstance(ctrl)
						mockInstance.EXPECT().SetContextStorage(mockTrieState)
						mockInstance.EXPECT().ExecuteBlock(&types.Block{Body: types.Body{}}).Return(nil, nil)
						mockBlockState.EXPECT().GetRuntime(&testParentHash).Return(mockInstance, nil)
						return mockBlockState
					}(ctrl),
					storageState: func(ctrl *gomock.Controller) StorageState {
						mockStorageState := NewMockStorageState(ctrl)
						mockStorageState.EXPECT().Lock()
						mockStorageState.EXPECT().TrieState(&testHash).Return(mockTrieState, nil)
						mockStorageState.EXPECT().Unlock()
						return mockStorageState
					}(ctrl),
					blockImportHandler: func(ctrl *gomock.Controller) BlockImportHandler {
						mockBlockImportHandler := NewMockBlockImportHandler(ctrl)
						mockBlockImportHandler.EXPECT().HandleBlockImport(&types.Block{Body: types.Body{}},
							mockTrieState).Return(mockError)
						return mockBlockImportHandler
					}(ctrl),
				}
			},
			block: &types.Block{
				Body: types.Body{},
			},
			wantErr: mockError,
		},
		"base case": {
			chainProcessorBuilder: func(ctrl *gomock.Controller) chainProcessor {
				mockBlock := &types.Block{
					Body: types.Body{}, // empty slice of extrinsics
				}
				return chainProcessor{
					blockState: func(ctrl *gomock.Controller) BlockState {
						mockBlockState := NewMockBlockState(ctrl)
						mockHeader := &types.Header{
							Number:    0,
							StateRoot: trie.EmptyHash,
						}
						mockHeaderHash := mockHeader.Hash()
						mockBlockState.EXPECT().GetHeader(common.Hash{}).Return(mockHeader, nil)
						mockInstance := NewMockInstance(ctrl)
						mockInstance.EXPECT().SetContextStorage(mockTrieState)
						mockInstance.EXPECT().ExecuteBlock(mockBlock)
						mockBlockState.EXPECT().GetRuntime(&mockHeaderHash).Return(mockInstance, nil)
						return mockBlockState
					}(ctrl),
					storageState: func(ctrl *gomock.Controller) StorageState {
						mockStorageState := NewMockStorageState(ctrl)
						mockStorageState.EXPECT().Lock()
						mockStorageState.EXPECT().Unlock()
						mockStorageState.EXPECT().TrieState(&trie.EmptyHash).Return(mockTrieState, nil)
						return mockStorageState
					}(ctrl),
					blockImportHandler: func(ctrl *gomock.Controller) BlockImportHandler {
						mockBlockImportHandler := NewMockBlockImportHandler(ctrl)
						mockBlockImportHandler.EXPECT().HandleBlockImport(mockBlock, mockTrieState).Return(nil)
						return mockBlockImportHandler
					}(ctrl),
					telemetry: func(ctrl *gomock.Controller) telemetry.Client {
						mockTelemetry := NewMockClient(ctrl)
						mockTelemetry.EXPECT().SendMessage(gomock.Any())
						return mockTelemetry
					}(ctrl),
				}
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
			if tt.wantErr != nil {
				assert.ErrorContains(t, err, tt.wantErr.Error())
			}
		})
	}
}

func Test_chainProcessor_handleBody(t *testing.T) {
	t.Parallel()

	testExtrinsics := []types.Extrinsic{{1, 2, 3}, {7, 8, 9, 0}, {0xa, 0xb}}

	testBody := types.NewBody(testExtrinsics)

	tests := map[string]struct {
		chainProcessorBuilder func(ctrl *gomock.Controller) chainProcessor
		body                  *types.Body
	}{
		"base case": {
			chainProcessorBuilder: func(ctrl *gomock.Controller) chainProcessor {
				return chainProcessor{
					transactionState: func(ctrl *gomock.Controller) TransactionState {
						mockTransactionState := NewMockTransactionState(ctrl)
						mockTransactionState.EXPECT().RemoveExtrinsic(testExtrinsics[0])
						mockTransactionState.EXPECT().RemoveExtrinsic(testExtrinsics[1])
						mockTransactionState.EXPECT().RemoveExtrinsic(testExtrinsics[2])
						return mockTransactionState
					}(ctrl),
				}
			},
			body: testBody,
		},
	}
	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			s := tt.chainProcessorBuilder(ctrl)
			s.handleBody(tt.body)
		})
	}
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
				return chainProcessor{
					finalityGadget: func(ctrl *gomock.Controller) FinalityGadget {
						mockFinalityGadget := NewMockFinalityGadget(ctrl)
						mockFinalityGadget.EXPECT().VerifyBlockJustification(expectedHash, []byte(`x`)).Return(errors.New("error"))
						return mockFinalityGadget
					}(ctrl),
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
				return chainProcessor{
					blockState: func(ctrl *gomock.Controller) BlockState {
						mockBlockState := NewMockBlockState(ctrl)
						mockBlockState.EXPECT().SetJustification(expectedHash, []byte(`xx`)).Return(errors.New("fake error"))
						return mockBlockState
					}(ctrl),
					finalityGadget: func(ctrl *gomock.Controller) FinalityGadget {
						mockFinalityGadget := NewMockFinalityGadget(ctrl)
						mockFinalityGadget.EXPECT().VerifyBlockJustification(expectedHash, []byte(`xx`)).Return(nil)
						return mockFinalityGadget
					}(ctrl),
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
				return chainProcessor{
					blockState: func(ctrl *gomock.Controller) BlockState {
						mockBlockState := NewMockBlockState(ctrl)
						mockBlockState.EXPECT().SetJustification(expectedHash, []byte(`1234`)).Return(nil)
						return mockBlockState
					}(ctrl),
					finalityGadget: func(ctrl *gomock.Controller) FinalityGadget {
						mockFinalityGadget := NewMockFinalityGadget(ctrl)
						mockFinalityGadget.EXPECT().VerifyBlockJustification(expectedHash, []byte(`1234`)).Return(nil)
						return mockFinalityGadget
					}(ctrl),
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
				return chainProcessor{
					blockState: func(ctrl *gomock.Controller) BlockState {
						mockBlockState := NewMockBlockState(ctrl)
						mockBlockState.EXPECT().HasHeader(common.Hash{}).Return(false, mockError)
						return mockBlockState
					}(ctrl),
				}
			},
			blockData:     &types.BlockData{},
			expectedError: mockError,
		},
		"handle has block body error": {
			chainProcessorBuilder: func(ctrl *gomock.Controller) chainProcessor {
				return chainProcessor{
					blockState: func(ctrl *gomock.Controller) BlockState {
						mockBlockState := NewMockBlockState(ctrl)
						mockBlockState.EXPECT().HasHeader(common.Hash{}).Return(false, nil)
						mockBlockState.EXPECT().HasBlockBody(common.Hash{}).Return(false, mockError)
						return mockBlockState
					}(ctrl),
				}
			},
			blockData:     &types.BlockData{},
			expectedError: mockError,
		},
		"handle getBlockByHash error": {
			chainProcessorBuilder: func(ctrl *gomock.Controller) chainProcessor {
				return chainProcessor{
					blockState: func(ctrl *gomock.Controller) BlockState {
						mockBlockState := NewMockBlockState(ctrl)
						mockBlockState.EXPECT().HasHeader(common.Hash{}).Return(true, nil)
						mockBlockState.EXPECT().HasBlockBody(common.Hash{}).Return(true, nil)
						mockBlockState.EXPECT().GetBlockByHash(common.Hash{}).Return(nil, mockError)
						return mockBlockState
					}(ctrl),
				}
			},
			blockData:     &types.BlockData{},
			expectedError: mockError,
		},
		"handle block data justification != nil": {
			chainProcessorBuilder: func(ctrl *gomock.Controller) chainProcessor {
				mockBlock := &types.Block{
					Header: types.Header{
						Number: uint(1),
					},
				}
				return chainProcessor{
					blockState: func(ctrl *gomock.Controller) BlockState {
						mockBlockState := NewMockBlockState(ctrl)
						mockBlockState.EXPECT().HasHeader(common.Hash{}).Return(true, nil)
						mockBlockState.EXPECT().HasBlockBody(common.Hash{}).Return(true, nil)
						mockBlockState.EXPECT().GetBlockByHash(common.Hash{}).Return(mockBlock, nil)
						mockBlockState.EXPECT().AddBlockToBlockTree(&types.Block{
							Header: types.Header{Number: 1}}).Return(nil)
						mockBlockState.EXPECT().SetJustification(common.MustHexToHash(
							"0x6443a0b46e0412e626363028115a9f2cf963eeed526b8b33e5316f08b50d0dc3"), []byte{1, 2, 3})
						return mockBlockState
					}(ctrl),
					finalityGadget: func(ctrl *gomock.Controller) FinalityGadget {
						mockFinalityGadget := NewMockFinalityGadget(ctrl)
						mockFinalityGadget.EXPECT().VerifyBlockJustification(common.MustHexToHash(
							"0x6443a0b46e0412e626363028115a9f2cf963eeed526b8b33e5316f08b50d0dc3"), []byte{1, 2,
							3})
						return mockFinalityGadget
					}(ctrl),
					storageState: func(ctrl *gomock.Controller) StorageState {
						mockStorageState := NewMockStorageState(ctrl)
						mockStorageState.EXPECT().TrieState(&common.Hash{}).Return(nil, nil)
						return mockStorageState
					}(ctrl),
					blockImportHandler: func(ctrl *gomock.Controller) BlockImportHandler {
						mockBlockImportHandler := NewMockBlockImportHandler(ctrl)
						mockBlockImportHandler.EXPECT().HandleBlockImport(mockBlock,
							nil).Return(nil)
						return mockBlockImportHandler
					}(ctrl),
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
				return chainProcessor{
					blockState: func(ctrl *gomock.Controller) BlockState {
						mockBlockState := NewMockBlockState(ctrl)
						mockBlockState.EXPECT().HasHeader(common.Hash{}).Return(true, nil)
						mockBlockState.EXPECT().HasBlockBody(common.Hash{}).Return(true, nil)
						mockBlockState.EXPECT().GetBlockByHash(common.Hash{}).Return(mockBlock, nil)
						mockBlockState.EXPECT().AddBlockToBlockTree(&types.Block{
							Header: types.Header{Number: 1}}).Return(nil)
						mockBlockState.EXPECT().SetJustification(common.MustHexToHash(
							"0x6443a0b46e0412e626363028115a9f2cf963eeed526b8b33e5316f08b50d0dc3"), []byte{1, 2, 3})
						return mockBlockState
					}(ctrl),
					finalityGadget: func(ctrl *gomock.Controller) FinalityGadget {
						mockFinalityGadget := NewMockFinalityGadget(ctrl)
						mockFinalityGadget.EXPECT().VerifyBlockJustification(common.MustHexToHash(
							"0x6443a0b46e0412e626363028115a9f2cf963eeed526b8b33e5316f08b50d0dc3"), []byte{1, 2,
							3})
						return mockFinalityGadget
					}(ctrl),
					storageState: func(ctrl *gomock.Controller) StorageState {
						mockStorageState := NewMockStorageState(ctrl)
						mockStorageState.EXPECT().TrieState(&common.Hash{}).Return(nil, mockError)
						return mockStorageState
					}(ctrl),
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
				return chainProcessor{
					blockState: func(ctrl *gomock.Controller) BlockState {
						mockBlockState := NewMockBlockState(ctrl)
						mockBlockState.EXPECT().HasHeader(common.Hash{}).Return(true, nil)
						mockBlockState.EXPECT().HasBlockBody(common.Hash{}).Return(true, nil)
						mockBlockState.EXPECT().GetBlockByHash(common.Hash{}).Return(mockBlock, nil)
						mockBlockState.EXPECT().AddBlockToBlockTree(&types.Block{
							Header: types.Header{Number: 1}}).Return(nil)
						mockBlockState.EXPECT().SetJustification(common.MustHexToHash(
							"0x6443a0b46e0412e626363028115a9f2cf963eeed526b8b33e5316f08b50d0dc3"), []byte{1, 2, 3})
						return mockBlockState
					}(ctrl),
					finalityGadget: func(ctrl *gomock.Controller) FinalityGadget {
						mockFinalityGadget := NewMockFinalityGadget(ctrl)
						mockFinalityGadget.EXPECT().VerifyBlockJustification(common.MustHexToHash(
							"0x6443a0b46e0412e626363028115a9f2cf963eeed526b8b33e5316f08b50d0dc3"), []byte{1, 2,
							3})
						return mockFinalityGadget
					}(ctrl),
					storageState: func(ctrl *gomock.Controller) StorageState {
						mockStorageState := NewMockStorageState(ctrl)
						mockStorageState.EXPECT().TrieState(&common.Hash{}).Return(nil, nil)
						return mockStorageState
					}(ctrl),
					blockImportHandler: func(ctrl *gomock.Controller) BlockImportHandler {
						mockBlockImportHandler := NewMockBlockImportHandler(ctrl)
						mockBlockImportHandler.EXPECT().HandleBlockImport(mockBlock,
							nil).Return(mockError)
						return mockBlockImportHandler
					}(ctrl),
				}
			},
			blockData: &types.BlockData{
				Justification: &[]byte{1, 2, 3},
			},
		},
		"has header body false": {
			chainProcessorBuilder: func(ctrl *gomock.Controller) chainProcessor {
				return chainProcessor{
					blockState: func(ctrl *gomock.Controller) BlockState {
						mockBlockState := NewMockBlockState(ctrl)
						mockBlockState.EXPECT().HasHeader(common.Hash{}).Return(false, nil)
						mockBlockState.EXPECT().HasBlockBody(common.Hash{}).Return(false, nil)
						mockBlockState.EXPECT().CompareAndSetBlockData(&types.BlockData{}).Return(nil)
						return mockBlockState
					}(ctrl),
				}
			},
			blockData: &types.BlockData{},
		},
		"handle babe verify block error": {
			chainProcessorBuilder: func(ctrl *gomock.Controller) chainProcessor {
				return chainProcessor{
					blockState: func(ctrl *gomock.Controller) BlockState {
						mockBlockState := NewMockBlockState(ctrl)
						mockBlockState.EXPECT().HasHeader(common.Hash{}).Return(false, nil)
						mockBlockState.EXPECT().HasBlockBody(common.Hash{}).Return(false, nil)
						return mockBlockState
					}(ctrl),
					babeVerifier: func(ctrl *gomock.Controller) BabeVerifier {
						mockBabeVerifier := NewMockBabeVerifier(ctrl)
						mockBabeVerifier.EXPECT().VerifyBlock(&types.Header{}).Return(mockError)
						return mockBabeVerifier
					}(ctrl),
				}
			},
			blockData: &types.BlockData{
				Header: &types.Header{},
				Body:   &types.Body{},
			},
			expectedError: mockError,
		},
		"error adding block": {
			chainProcessorBuilder: func(ctrl *gomock.Controller) chainProcessor {
				return chainProcessor{
					blockState: func(ctrl *gomock.Controller) BlockState {
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
						return mockBlockState
					}(ctrl),
				}
			},
			blockData: &types.BlockData{
				Hash: common.Hash{1, 2, 3},
			},
			expectedError: mockError,
		},
		"handle block import": {
			chainProcessorBuilder: func(ctrl *gomock.Controller) chainProcessor {
				mockTrieState, err := storage.NewTrieState(nil)
				require.NoError(t, err)

				return chainProcessor{
					blockState: func(ctrl *gomock.Controller) BlockState {
						mockBlockState := NewMockBlockState(ctrl)
						mockBlockState.EXPECT().HasHeader(common.Hash{1, 2, 3}).Return(true, nil)
						mockBlockState.EXPECT().HasBlockBody(common.Hash{1, 2, 3}).Return(true, nil)
						mockBlockState.EXPECT().GetBlockByHash(common.Hash{1, 2, 3}).Return(&types.Block{
							Header: types.Header{
								Number: uint(1),
							},
						}, nil)
						mockBlockState.EXPECT().AddBlockToBlockTree(&types.Block{Header: types.Header{Number: 1}}).Return(nil)
						return mockBlockState
					}(ctrl),
					storageState: func(ctrl *gomock.Controller) StorageState {
						mockStorageState := NewMockStorageState(ctrl)
						mockStorageState.EXPECT().TrieState(&common.Hash{}).Return(mockTrieState, nil)
						return mockStorageState
					}(ctrl),
					blockImportHandler: func(ctrl *gomock.Controller) BlockImportHandler {
						mockBlockImportHandler := NewMockBlockImportHandler(ctrl)
						mockBlockImportHandler.EXPECT().HandleBlockImport(&types.Block{Header: types.Header{Number: 1}}, mockTrieState)
						return mockBlockImportHandler
					}(ctrl),
				}
			},
			blockData: &types.BlockData{
				Hash: common.Hash{1, 2, 3},
			},
		},
		"handle compareAndSetBlockData error": {
			chainProcessorBuilder: func(ctrl *gomock.Controller) chainProcessor {
				return chainProcessor{
					blockState: func(ctrl *gomock.Controller) BlockState {
						mockBlockState := NewMockBlockState(ctrl)
						mockBlockState.EXPECT().HasHeader(common.Hash{}).Return(false, nil)
						mockBlockState.EXPECT().HasBlockBody(common.Hash{}).Return(false, nil)
						mockBlockState.EXPECT().CompareAndSetBlockData(&types.BlockData{}).Return(mockError)
						return mockBlockState
					}(ctrl),
				}
			},
			blockData:     &types.BlockData{},
			expectedError: mockError,
		},
		"handle header": {
			chainProcessorBuilder: func(ctrl *gomock.Controller) chainProcessor {
				stateRootHash := common.MustHexToHash("0x03170a2e7597b7b7e3d84c05391d139a62b157e78786d8c082f29dcf4c111314")
				mockTrieState, err := storage.NewTrieState(nil)
				require.NoError(t, err)

				runtimeHash := common.MustHexToHash("0x7db9db5ed9967b80143100189ba69d9e4deab85ac3570e5df25686cabe32964a")
				mockInstance := NewMockInstance(ctrl)
				mockInstance.EXPECT().SetContextStorage(mockTrieState)
				mockInstance.EXPECT().ExecuteBlock(&types.Block{Header: types.Header{}, Body: types.Body{}})

				return chainProcessor{
					blockState: func(ctrl *gomock.Controller) BlockState {
						mockBlockState := NewMockBlockState(ctrl)
						mockBlockState.EXPECT().HasHeader(common.Hash{}).Return(false, nil)
						mockBlockState.EXPECT().HasBlockBody(common.Hash{}).Return(false, nil)
						mockBlockState.EXPECT().GetHeader(common.Hash{}).Return(&types.Header{
							Number:    0,
							StateRoot: stateRootHash,
						}, nil)
						mockBlockState.EXPECT().CompareAndSetBlockData(&types.BlockData{Header: &types.Header{}, Body: &types.Body{}})
						mockBlockState.EXPECT().GetRuntime(&runtimeHash).Return(mockInstance, nil)
						return mockBlockState
					}(ctrl),
					babeVerifier: func(ctrl *gomock.Controller) BabeVerifier {
						mockBabeVerifier := NewMockBabeVerifier(ctrl)
						mockBabeVerifier.EXPECT().VerifyBlock(&types.Header{})
						return mockBabeVerifier
					}(ctrl),
					storageState: func(ctrl *gomock.Controller) StorageState {
						mockStorageState := NewMockStorageState(ctrl)
						mockStorageState.EXPECT().Lock()
						mockStorageState.EXPECT().TrieState(&stateRootHash).Return(mockTrieState, nil)
						mockStorageState.EXPECT().Unlock()
						return mockStorageState
					}(ctrl),
					blockImportHandler: func(ctrl *gomock.Controller) BlockImportHandler {
						mockBlockImportHandler := NewMockBlockImportHandler(ctrl)
						mockBlockImportHandler.EXPECT().HandleBlockImport(&types.Block{
							Header: types.Header{}, Body: types.Body{}}, mockTrieState)
						return mockBlockImportHandler
					}(ctrl),
					telemetry: func(ctrl *gomock.Controller) telemetry.Client {
						mockTelemetry := NewMockClient(ctrl)
						mockTelemetry.EXPECT().SendMessage(gomock.Any()).AnyTimes()
						return mockTelemetry
					}(ctrl),
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
				mockTrieState, _ := storage.NewTrieState(nil)
				mockInstance := NewMockInstance(ctrl)
				mockInstance.EXPECT().SetContextStorage(mockTrieState)
				mockInstance.EXPECT().ExecuteBlock(&types.Block{Header: types.Header{}, Body: types.Body{}})

				return chainProcessor{
					blockState: func(ctrl *gomock.Controller) BlockState {
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

						return mockBlockState
					}(ctrl),
					babeVerifier: func(ctrl *gomock.Controller) BabeVerifier {
						mockBabeVerifier := NewMockBabeVerifier(ctrl)
						mockBabeVerifier.EXPECT().VerifyBlock(&types.Header{})
						return mockBabeVerifier
					}(ctrl),
					storageState: func(ctrl *gomock.Controller) StorageState {
						mockStorageState := NewMockStorageState(ctrl)
						mockStorageState.EXPECT().Lock()
						mockStorageState.EXPECT().TrieState(&stateRootHash).Return(mockTrieState, nil)
						mockStorageState.EXPECT().Unlock()

						return mockStorageState
					}(ctrl),
					blockImportHandler: func(ctrl *gomock.Controller) BlockImportHandler {
						mockBlockImportHandler := NewMockBlockImportHandler(ctrl)
						mockBlockImportHandler.EXPECT().HandleBlockImport(
							&types.Block{Header: types.Header{}, Body: types.Body{}}, mockTrieState)

						return mockBlockImportHandler
					}(ctrl),
					telemetry: func(ctrl *gomock.Controller) telemetry.Client {
						mockTelemetry := NewMockClient(ctrl)
						mockTelemetry.EXPECT().SendMessage(gomock.Any()).AnyTimes()
						return mockTelemetry

					}(ctrl),
					finalityGadget: func(ctrl *gomock.Controller) FinalityGadget {
						mockFinalityGadget := NewMockFinalityGadget(ctrl)
						mockFinalityGadget.EXPECT().VerifyBlockJustification(
							common.MustHexToHash("0xdcdd89927d8a348e00257e1ecc8617f45edb5118efff3ea2f9961b2ad9b7690a"), justification)
						return mockFinalityGadget
					}(ctrl),
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
			s := tt.chainProcessorBuilder(ctrl)
			err := s.processBlockData(tt.blockData)
			assert.ErrorIs(t, err, tt.expectedError)
		})
	}
}

func Test_chainProcessor_processReadyBlocks(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name              string
		blockStateBuilder func(ctrl *gomock.Controller) BlockState
	}{
		{
			name: "base case",
			blockStateBuilder: func(ctrl *gomock.Controller) BlockState {
				mockBlockState := NewMockBlockState(ctrl)
				mockBlockState.EXPECT().HasHeader(common.Hash{}).Return(false, nil)
				mockBlockState.EXPECT().HasBlockBody(common.Hash{}).Return(false, nil)
				mockBlockState.EXPECT().CompareAndSetBlockData(&types.BlockData{})
				return mockBlockState
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			ctx, cancel := context.WithCancel(context.Background())
			readyBlock := newBlockQueue(5)

			s := &chainProcessor{
				ctx:         ctx,
				cancel:      cancel,
				readyBlocks: readyBlock,
				blockState:  tt.blockStateBuilder(ctrl),
			}

			go s.processReadyBlocks()

			readyBlock.push(&types.BlockData{
				Hash: common.Hash{},
			})
			time.Sleep(time.Millisecond)

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
