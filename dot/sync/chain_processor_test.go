// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package sync

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/runtime/mocks"
	"github.com/ChainSafe/gossamer/lib/runtime/storage"
	"github.com/ChainSafe/gossamer/lib/trie"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_chainProcessor_handleBlock_baseCase(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)

	block := &types.Block{
		Header: types.Header{
			Number: 0,
		},
		Body: types.Body{},
	}

	mockBlockState := NewMockBlockState(ctrl)
	mockHeader := &types.Header{
		Number:    0,
		StateRoot: trie.EmptyHash,
	}
	mockHeaderHash := mockHeader.Hash()
	mockBlock := &types.Block{
		Header: types.Header{
			Number: 0,
		},
		Body: types.Body{},
	}
	mockTrieState, err := storage.NewTrieState(nil)
	require.NoError(t, err)
	mockBlockState.EXPECT().GetHeader(common.Hash{}).Return(mockHeader, nil)
	mockInstance := mocks.NewMockInstance(ctrl)
	mockInstance.EXPECT().SetContextStorage(mockTrieState)
	mockInstance.EXPECT().ExecuteBlock(mockBlock)
	mockBlockState.EXPECT().GetRuntime(&mockHeaderHash).Return(mockInstance, nil)

	mockStorageState := NewMockStorageState(ctrl)
	mockStorageState.EXPECT().Lock()
	mockStorageState.EXPECT().Unlock()
	mockStorageState.EXPECT().TrieState(&trie.EmptyHash).Return(mockTrieState, nil)

	mockBlockImportHandler := NewMockBlockImportHandler(ctrl)
	mockBlockImportHandler.EXPECT().HandleBlockImport(mockBlock, mockTrieState).Return(nil)

	mockTelemetry := NewMockClient(ctrl)
	mockTelemetry.EXPECT().SendMessage(gomock.Any())

	s := &chainProcessor{
		blockState:         mockBlockState,
		storageState:       mockStorageState,
		blockImportHandler: mockBlockImportHandler,
		telemetry:          mockTelemetry,
	}
	err = s.handleBlock(block)
	assert.NoError(t, err)
}

func Test_chainProcessor_handleBlock_nilBlock(t *testing.T) {
	t.Parallel()

	s := &chainProcessor{}
	err := s.handleBlock(nil)
	assert.EqualError(t, err, errors.New("block or body is nil").Error())
}

func Test_chainProcessor_handleBody(t *testing.T) {
	t.Parallel()

	var testExtrinsic = []types.Extrinsic{{1, 2, 3}, {7, 8, 9, 0}, {0xa, 0xb}}

	ctrl := gomock.NewController(t)

	mockTransactionState := NewMockTransactionState(ctrl)
	mockTransactionState.EXPECT().RemoveExtrinsic(testExtrinsic[0])
	mockTransactionState.EXPECT().RemoveExtrinsic(testExtrinsic[1])
	mockTransactionState.EXPECT().RemoveExtrinsic(testExtrinsic[2])

	testBody := types.NewBody(testExtrinsic)

	type fields struct {
		transactionState TransactionState
	}
	type args struct {
		body *types.Body
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			name: "base case",
			fields: fields{
				transactionState: mockTransactionState,
			},
			args: args{body: testBody},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &chainProcessor{
				transactionState: tt.fields.transactionState,
			}
			s.handleBody(tt.args.body)
		})
	}
}

func Test_chainProcessor_handleHeader(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)

	mockBabeVerifier := NewMockBabeVerifier(ctrl)
	mockBabeVerifier.EXPECT().VerifyBlock(gomock.AssignableToTypeOf(&types.Header{})).DoAndReturn(func(h *types.
		Header) error {
		if h == nil {
			return errors.New("nil header")
		}
		return nil
	}).Times(2)

	type fields struct {
		babeVerifier BabeVerifier
	}
	type args struct {
		header *types.Header
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		err    error
	}{
		{
			name: "nil header",
			fields: fields{
				babeVerifier: mockBabeVerifier,
			},
			err: errors.New("could not verify block: nil header"),
		},
		{
			name: "base case",
			fields: fields{
				babeVerifier: mockBabeVerifier,
			},
			args: args{header: &types.Header{
				Number: 0,
			}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &chainProcessor{
				babeVerifier: tt.fields.babeVerifier,
			}
			err := s.handleHeader(tt.args.header)
			if tt.err != nil {
				assert.EqualError(t, err, tt.err.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func Test_chainProcessor_handleJustification(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)

	mockFinalityGadget := NewMockFinalityGadget(ctrl)
	mockFinalityGadget.EXPECT().VerifyBlockJustification(gomock.AssignableToTypeOf(common.Hash{}),
		gomock.AssignableToTypeOf([]byte{})).DoAndReturn(func(_ common.Hash, justification []byte) error {
		if len(justification) < 2 {
			return errors.New("error")
		}
		return nil
	}).Times(3)
	mockBlockState := NewMockBlockState(ctrl)
	mockBlockState.EXPECT().SetJustification(gomock.AssignableToTypeOf(common.Hash{}),
		gomock.AssignableToTypeOf([]byte{})).DoAndReturn(func(_ common.Hash, justification []byte) error {
		if bytes.Equal(justification, []byte(`xx`)) {
			return errors.New("fake error")
		}
		return nil
	}).Times(2)

	type fields struct {
		blockState     BlockState
		finalityGadget FinalityGadget
	}
	type args struct {
		header        *types.Header
		justification []byte
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			name: "nil justification and header",
		},
		{
			name: "invalid justification",
			fields: fields{
				finalityGadget: mockFinalityGadget,
			},
			args: args{
				header: &types.Header{
					Number: 0,
				},
				justification: []byte(`x`),
			},
		},
		{
			name: "set justification error",
			fields: fields{
				blockState:     mockBlockState,
				finalityGadget: mockFinalityGadget,
			},
			args: args{
				header: &types.Header{
					Number: 0,
				},
				justification: []byte(`xx`),
			},
		},
		{
			name: "base case set",
			fields: fields{
				blockState:     mockBlockState,
				finalityGadget: mockFinalityGadget,
			},
			args: args{
				header: &types.Header{
					Number: 0,
				},
				justification: []byte(`1234`),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &chainProcessor{
				blockState:     tt.fields.blockState,
				finalityGadget: tt.fields.finalityGadget,
			}
			s.handleJustification(tt.args.header, tt.args.justification)
		})
	}
}

func Test_chainProcessor_processBlockData_nilBlockData(t *testing.T) {
	t.Parallel()

	s := &chainProcessor{}
	err := s.processBlockData(nil)
	assert.EqualError(t, err, errors.New("got nil BlockData").Error())
}

func Test_chainProcessor_processBlockData_hasHeaderBodyFalse(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	mockBlockState := NewMockBlockState(ctrl)
	mockBlockState.EXPECT().HasHeader(common.Hash{}).Return(false, nil)
	mockBlockState.EXPECT().HasBlockBody(common.Hash{}).Return(false, nil)
	mockBlockState.EXPECT().CompareAndSetBlockData(&types.BlockData{})

	s := &chainProcessor{
		blockState: mockBlockState,
	}
	err := s.processBlockData(&types.BlockData{})
	assert.NoError(t, err)
}

func Test_chainProcessor_processBlockData_errorAddingBlock(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	mockBlockState := NewMockBlockState(ctrl)
	mockBlockState.EXPECT().HasHeader(common.Hash{1, 2, 3}).Return(true, nil)
	mockBlockState.EXPECT().HasBlockBody(common.Hash{1, 2, 3}).Return(true, nil)
	mockBlockState.EXPECT().GetBlockByHash(common.Hash{1, 2, 3}).Return(&types.Block{
		Header: types.Header{
			Number: uint(1),
		},
	}, nil)
	mockBlockState.EXPECT().AddBlockToBlockTree(&types.Block{Header: types.Header{Number: 1}}).Return(
		errors.New("fake error adding block"))

	s := &chainProcessor{
		blockState: mockBlockState,
	}

	blockData := types.BlockData{
		Hash: common.MustHexToHash("0x010203"),
	}
	err := s.processBlockData(&blockData)
	assert.EqualError(t, err, errors.New("fake error adding block").Error())
}

func Test_chainProcessor_processBlockData_handleBlockImport(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	mockTrieState, err := storage.NewTrieState(nil)
	require.NoError(t, err)

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
	mockBlockImportHandler.EXPECT().HandleBlockImport(&types.Block{Header: types.Header{Number: 1}},
		mockTrieState)

	s := &chainProcessor{
		blockState:         mockBlockState,
		storageState:       mockStorageState,
		blockImportHandler: mockBlockImportHandler,
	}
	blockData := types.BlockData{
		Hash: common.MustHexToHash("0x010203"),
	}

	err = s.processBlockData(&blockData)
	assert.NoError(t, err)
}

func Test_chainProcessor_processBlockData_handleHeader(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	stateRootHash := common.MustHexToHash("0x03170a2e7597b7b7e3d84c05391d139a62b157e78786d8c082f29dcf4c111314")
	mockTrieState, err := storage.NewTrieState(nil)
	require.NoError(t, err)

	mockBlockState := NewMockBlockState(ctrl)
	mockBlockState.EXPECT().HasHeader(common.Hash{}).Return(false, nil)
	mockBlockState.EXPECT().HasBlockBody(common.Hash{}).Return(false, nil)
	mockBlockState.EXPECT().GetHeader(common.Hash{}).Return(&types.Header{
		Number:    0,
		StateRoot: stateRootHash,
	}, nil)
	mockBlockState.EXPECT().CompareAndSetBlockData(&types.BlockData{Header: &types.Header{}, Body: &types.Body{}})

	runtimeHash := common.MustHexToHash("0x7db9db5ed9967b80143100189ba69d9e4deab85ac3570e5df25686cabe32964a")
	mockInstance := mocks.NewMockInstance(ctrl)
	mockInstance.EXPECT().SetContextStorage(mockTrieState)
	mockInstance.EXPECT().ExecuteBlock(&types.Block{Header: types.Header{}, Body: types.Body{}})
	mockBlockState.EXPECT().GetRuntime(&runtimeHash).Return(mockInstance, nil)

	mockBabeVerifier := NewMockBabeVerifier(ctrl)
	mockBabeVerifier.EXPECT().VerifyBlock(&types.Header{})

	mockStorageState := NewMockStorageState(ctrl)
	mockStorageState.EXPECT().Lock()
	mockStorageState.EXPECT().TrieState(&stateRootHash).Return(mockTrieState, nil)
	mockStorageState.EXPECT().Unlock()

	mockBlockImportHandler := NewMockBlockImportHandler(ctrl)
	mockBlockImportHandler.EXPECT().HandleBlockImport(&types.Block{
		Header: types.Header{}, Body: types.Body{}}, mockTrieState)

	mockTelemetry := NewMockClient(ctrl)
	mockTelemetry.EXPECT().SendMessage(gomock.Any()).AnyTimes()

	s := &chainProcessor{
		blockState:         mockBlockState,
		storageState:       mockStorageState,
		babeVerifier:       mockBabeVerifier,
		blockImportHandler: mockBlockImportHandler,
		telemetry:          mockTelemetry,
	}
	blockData := &types.BlockData{
		Header: &types.Header{
			Number: 0,
		},
		Body: &types.Body{},
	}
	err = s.processBlockData(blockData)
	assert.NoError(t, err)
}

func Test_chainProcessor_processBlockData_handleJustification(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	justification := []byte{0, 1, 2}
	stateRootHash := common.MustHexToHash("0x03170a2e7597b7b7e3d84c05391d139a62b157e78786d8c082f29dcf4c111314")
	mockTrieState, err := storage.NewTrieState(nil)
	require.NoError(t, err)

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

	runtimeHash := common.MustHexToHash("0x7db9db5ed9967b80143100189ba69d9e4deab85ac3570e5df25686cabe32964a")
	mockInstance := mocks.NewMockInstance(ctrl)
	mockInstance.EXPECT().SetContextStorage(mockTrieState)
	mockInstance.EXPECT().ExecuteBlock(&types.Block{Header: types.Header{}, Body: types.Body{}})
	mockBlockState.EXPECT().GetRuntime(&runtimeHash).Return(mockInstance, nil)

	mockBabeVerifier := NewMockBabeVerifier(ctrl)
	mockBabeVerifier.EXPECT().VerifyBlock(&types.Header{})

	mockStorageState := NewMockStorageState(ctrl)
	mockStorageState.EXPECT().Lock()
	mockStorageState.EXPECT().TrieState(&stateRootHash).Return(mockTrieState, nil)
	mockStorageState.EXPECT().Unlock()

	mockBlockImportHandler := NewMockBlockImportHandler(ctrl)
	mockBlockImportHandler.EXPECT().HandleBlockImport(
		&types.Block{Header: types.Header{}, Body: types.Body{}}, mockTrieState)

	mockTelemetry := NewMockClient(ctrl)
	mockTelemetry.EXPECT().SendMessage(gomock.Any()).AnyTimes()

	mockFinalityGadget := NewMockFinalityGadget(ctrl)
	mockFinalityGadget.EXPECT().VerifyBlockJustification(
		common.MustHexToHash("0xdcdd89927d8a348e00257e1ecc8617f45edb5118efff3ea2f9961b2ad9b7690a"), justification)

	s := &chainProcessor{
		blockState:         mockBlockState,
		storageState:       mockStorageState,
		babeVerifier:       mockBabeVerifier,
		blockImportHandler: mockBlockImportHandler,
		finalityGadget:     mockFinalityGadget,
		telemetry:          mockTelemetry,
	}

	blockData := &types.BlockData{
		Header: &types.Header{
			Number: 0,
		},
		Body:          &types.Body{},
		Justification: &justification,
	}
	err = s.processBlockData(blockData)
	assert.NoError(t, err)
}

func Test_chainProcessor_processReadyBlocks(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	mockBlockState := NewMockBlockState(ctrl)
	mockBlockState.EXPECT().HasHeader(common.Hash{}).Return(false, nil)
	mockBlockState.EXPECT().HasBlockBody(common.Hash{}).Return(false, nil)
	mockBlockState.EXPECT().CompareAndSetBlockData(&types.BlockData{})

	tests := []struct {
		name       string
		blockState BlockState
	}{
		{
			name:       "base case",
			blockState: mockBlockState,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			readyBlock := newBlockQueue(5)

			s := &chainProcessor{
				ctx:         ctx,
				cancel:      cancel,
				readyBlocks: readyBlock,
				blockState:  tt.blockState,
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

	ctrl := gomock.NewController(t)

	mockReadyBlock := newBlockQueue(5)
	mockDisjointBlockSet := NewMockDisjointBlockSet(ctrl)
	mockBlockState := NewMockBlockState(ctrl)
	mockStorageState := NewMockStorageState(ctrl)
	mockTransactionState := NewMockTransactionState(ctrl)
	mockBabeVerifier := NewMockBabeVerifier(ctrl)
	mockFinalityGadget := NewMockFinalityGadget(ctrl)
	mockBlockImportHandler := NewMockBlockImportHandler(ctrl)

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
			name: "base case",
			args: args{},
			want: &chainProcessor{},
		},
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
		t.Run(tt.name, func(t *testing.T) {
			got := newChainProcessor(tt.args.readyBlocks, tt.args.pendingBlocks, tt.args.blockState,
				tt.args.storageState, tt.args.transactionState, tt.args.babeVerifier, tt.args.finalityGadget,
				tt.args.blockImportHandler, nil)
			assert.NotEmpty(t, got.ctx)
			assert.NotEmpty(t, got.cancel)
			assert.Equal(t, tt.want.readyBlocks, got.readyBlocks)
			assert.Equal(t, tt.want.pendingBlocks, got.pendingBlocks)
			assert.Equal(t, tt.want.blockState, got.blockState)
			assert.Equal(t, tt.want.storageState, got.storageState)
			assert.Equal(t, tt.want.transactionState, got.transactionState)
			assert.Equal(t, tt.want.babeVerifier, got.babeVerifier)
			assert.Equal(t, tt.want.finalityGadget, got.finalityGadget)
			assert.Equal(t, tt.want.blockImportHandler, got.blockImportHandler)
		})
	}
}
