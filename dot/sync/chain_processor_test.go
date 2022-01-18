// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package sync

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/runtime/mocks"
	"github.com/ChainSafe/gossamer/lib/runtime/storage"
	"github.com/ChainSafe/gossamer/lib/trie"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"math/big"
	"reflect"
	"testing"
)

func Test_chainProcessor_handleBlock(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockBlockState := NewMockBlockState(ctrl)
	mockHeader := &types.Header{
		Number:    big.NewInt(0),
		StateRoot: trie.EmptyHash,
	}
	mockHeaderHash := mockHeader.Hash()
	mockBlock := &types.Block{
		Header: types.Header{
			Number: big.NewInt(0),
		},
		Body: types.Body{},
	}
	mockTrieState, _ := storage.NewTrieState(nil)
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

	type fields struct {
		blockState         BlockState
		storageState       StorageState
		blockImportHandler BlockImportHandler
	}
	type args struct {
		block *types.Block
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		err    error
	}{
		{
			name: "nil block",
			err:  errors.New("block or body is nil"),
		},
		{
			name: "base case",
			fields: fields{
				blockState:         mockBlockState,
				storageState:       mockStorageState,
				blockImportHandler: mockBlockImportHandler,
			},
			args: args{
				block: &types.Block{
					Header: types.Header{
						Number: big.NewInt(0),
					},
					Body: types.Body{},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &chainProcessor{
				blockState:         tt.fields.blockState,
				storageState:       tt.fields.storageState,
				blockImportHandler: tt.fields.blockImportHandler,
			}
			err := s.handleBlock(tt.args.block)
			if tt.err != nil {
				assert.EqualError(t, err, tt.err.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func Test_chainProcessor_handleBody(t *testing.T) {
	var testExtrinsic = []types.Extrinsic{{1, 2, 3}, {7, 8, 9, 0}, {0xa, 0xb}}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
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
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
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
				Number: big.NewInt(0),
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
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
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
		if bytes.Compare(justification, []byte(`xx`)) == 0 {
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
					Number: big.NewInt(0),
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
					Number: big.NewInt(0),
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
					Number: big.NewInt(0),
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

func Test_chainProcessor_processBlockData(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockBlockState := NewMockBlockState(ctrl)
	mockBlockState.EXPECT().HasHeader(gomock.AssignableToTypeOf(common.Hash{})).DoAndReturn(func(hash common.
		Hash) (bool, error) {
		if hash.IsEmpty() {
			return false, nil
		}
		return true, nil
	}).Times(5)
	mockBlockState.EXPECT().HasBlockBody(gomock.AssignableToTypeOf(common.Hash{})).DoAndReturn(func(hash common.
		Hash) (bool, error) {
		if hash.IsEmpty() {
			return false, nil
		}
		return true, nil
	}).Times(5)
	mockBlockState.EXPECT().CompareAndSetBlockData(gomock.AssignableToTypeOf(&types.BlockData{})).Times(3)
	mockBlockState.EXPECT().GetBlockByHash(gomock.AssignableToTypeOf(common.Hash{})).DoAndReturn(func(hash common.
		Hash) (*types.Block, error) {
		if hash.IsEmpty() {
			return nil, nil
		}
		num := big.NewInt(0)
		num.SetBytes(hash[0:1])
		block := &types.Block{
			Header: types.Header{
				Number: num,
			},
		}
		return block, nil
	}).Times(2)
	mockBlockState.EXPECT().AddBlockToBlockTree(gomock.AssignableToTypeOf(&types.Block{})).DoAndReturn(func(
		block *types.Block) error {
		if block.Header.Number.Cmp(big.NewInt(1)) == 0 {
			return errors.New("fake error adding block")
		}
		return nil
	}).Times(2)
	mockBlockState.EXPECT().GetHeader(gomock.AssignableToTypeOf(common.Hash{})).Return(&types.Header{
		Number:    big.NewInt(0),
		StateRoot: common.MustHexToHash("0x03170a2e7597b7b7e3d84c05391d139a62b157e78786d8c082f29dcf4c111314"),
	}, nil).Times(2)
	mockInstance := mocks.NewMockInstance(ctrl)
	mockTrieState, _ := storage.NewTrieState(nil)
	mockInstance.EXPECT().SetContextStorage(mockTrieState).Times(2)
	mockInstance.EXPECT().ExecuteBlock(gomock.AssignableToTypeOf(&types.Block{})).Times(2)
	mockBlockState.EXPECT().GetRuntime(gomock.AssignableToTypeOf(&common.Hash{})).Return(mockInstance, nil).Times(2)

	mockStorageState := NewMockStorageState(ctrl)
	mockStorageState.EXPECT().TrieState(gomock.AssignableToTypeOf(&common.Hash{})).DoAndReturn(func(hash *common.
		Hash) (*storage.TrieState, error) {
		return storage.NewTrieState(nil)

	}).Times(3)
	mockStorageState.EXPECT().Lock().Times(2)
	mockStorageState.EXPECT().Unlock().Times(2)

	mockBlockImportHandler := NewMockBlockImportHandler(ctrl)
	mockBlockImportHandler.EXPECT().HandleBlockImport(gomock.AssignableToTypeOf(&types.Block{}),
		gomock.AssignableToTypeOf(&storage.TrieState{})).Times(3)

	mockBabeVerifier := NewMockBabeVerifier(ctrl)
	mockBabeVerifier.EXPECT().VerifyBlock(gomock.AssignableToTypeOf(&types.Header{})).Times(2)

	mockJustification := []byte{0, 1, 2}
	mockFinalityGadget := NewMockFinalityGadget(ctrl)
	mockFinalityGadget.EXPECT().VerifyBlockJustification(gomock.AssignableToTypeOf(common.Hash{}),
		gomock.AssignableToTypeOf([]byte{}))
	mockBlockState.EXPECT().SetJustification(gomock.AssignableToTypeOf(common.Hash{}),
		gomock.AssignableToTypeOf([]byte{}))

	type fields struct {
		ctx                context.Context
		cancel             context.CancelFunc
		readyBlocks        *blockQueue
		pendingBlocks      DisjointBlockSet
		blockState         BlockState
		storageState       StorageState
		transactionState   TransactionState
		babeVerifier       BabeVerifier
		finalityGadget     FinalityGadget
		blockImportHandler BlockImportHandler
	}
	type args struct {
		bd *types.BlockData
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		err    error
	}{
		{
			name: "nil BlockData",
			err:  errors.New("got nil BlockData"),
		},
		{
			name: "has header/body false",
			args: args{bd: &types.BlockData{}},
			fields: fields{
				blockState: mockBlockState,
			},
		},
		{
			name: "error adding block data",
			args: args{bd: &types.BlockData{
				Hash: common.MustHexToHash("0x010203"),
			}},
			fields: fields{
				blockState: mockBlockState,
			},
			err: errors.New("fake error adding block"),
		},
		{
			name: "handle block import",
			args: args{bd: &types.BlockData{
				Hash: common.MustHexToHash("0x020203"),
			}},
			fields: fields{
				blockState:         mockBlockState,
				storageState:       mockStorageState,
				blockImportHandler: mockBlockImportHandler,
			},
		},
		{
			name: "handle header",
			args: args{bd: &types.BlockData{
				Header: &types.Header{
					Number: big.NewInt(0),
				},
				Body: &types.Body{},
			}},
			fields: fields{
				blockState:         mockBlockState,
				storageState:       mockStorageState,
				blockImportHandler: mockBlockImportHandler,
				babeVerifier:       mockBabeVerifier,
			},
		},
		{
			name: "handle justification",
			args: args{bd: &types.BlockData{
				Header: &types.Header{
					Number: big.NewInt(0),
				},
				Body:          &types.Body{},
				Justification: &mockJustification,
			}},
			fields: fields{
				blockState:         mockBlockState,
				storageState:       mockStorageState,
				blockImportHandler: mockBlockImportHandler,
				babeVerifier:       mockBabeVerifier,
				finalityGadget:     mockFinalityGadget,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &chainProcessor{
				ctx:                tt.fields.ctx,
				cancel:             tt.fields.cancel,
				readyBlocks:        tt.fields.readyBlocks,
				pendingBlocks:      tt.fields.pendingBlocks,
				blockState:         tt.fields.blockState,
				storageState:       tt.fields.storageState,
				transactionState:   tt.fields.transactionState,
				babeVerifier:       tt.fields.babeVerifier,
				finalityGadget:     tt.fields.finalityGadget,
				blockImportHandler: tt.fields.blockImportHandler,
			}
			err := s.processBlockData(tt.args.bd)
			if tt.err != nil {
				assert.EqualError(t, err, tt.err.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func Test_chainProcessor_processReadyBlocks(t *testing.T) {
	type fields struct {
		ctx                context.Context
		cancel             context.CancelFunc
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
		name   string
		fields fields
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &chainProcessor{
				ctx:                tt.fields.ctx,
				cancel:             tt.fields.cancel,
				readyBlocks:        tt.fields.readyBlocks,
				pendingBlocks:      tt.fields.pendingBlocks,
				blockState:         tt.fields.blockState,
				storageState:       tt.fields.storageState,
				transactionState:   tt.fields.transactionState,
				babeVerifier:       tt.fields.babeVerifier,
				finalityGadget:     tt.fields.finalityGadget,
				blockImportHandler: tt.fields.blockImportHandler,
			}
			fmt.Printf("s %v\n", s)
		})
	}
}

func Test_chainProcessor_start(t *testing.T) {
	type fields struct {
		ctx                context.Context
		cancel             context.CancelFunc
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
		name   string
		fields fields
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &chainProcessor{
				ctx:                tt.fields.ctx,
				cancel:             tt.fields.cancel,
				readyBlocks:        tt.fields.readyBlocks,
				pendingBlocks:      tt.fields.pendingBlocks,
				blockState:         tt.fields.blockState,
				storageState:       tt.fields.storageState,
				transactionState:   tt.fields.transactionState,
				babeVerifier:       tt.fields.babeVerifier,
				finalityGadget:     tt.fields.finalityGadget,
				blockImportHandler: tt.fields.blockImportHandler,
			}
			fmt.Printf("s %v\n", s)
		})
	}
}

func Test_chainProcessor_stop(t *testing.T) {
	type fields struct {
		ctx                context.Context
		cancel             context.CancelFunc
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
		name   string
		fields fields
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &chainProcessor{
				ctx:                tt.fields.ctx,
				cancel:             tt.fields.cancel,
				readyBlocks:        tt.fields.readyBlocks,
				pendingBlocks:      tt.fields.pendingBlocks,
				blockState:         tt.fields.blockState,
				storageState:       tt.fields.storageState,
				transactionState:   tt.fields.transactionState,
				babeVerifier:       tt.fields.babeVerifier,
				finalityGadget:     tt.fields.finalityGadget,
				blockImportHandler: tt.fields.blockImportHandler,
			}
			fmt.Printf("s %v\n", s)
		})
	}
}

func Test_newChainProcessor(t *testing.T) {
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
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := newChainProcessor(tt.args.readyBlocks, tt.args.pendingBlocks, tt.args.blockState, tt.args.storageState, tt.args.transactionState, tt.args.babeVerifier, tt.args.finalityGadget, tt.args.blockImportHandler); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("newChainProcessor() = %v, want %v", got, tt.want)
			}
		})
	}
}
