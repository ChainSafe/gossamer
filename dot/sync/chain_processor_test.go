// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package sync

import (
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
		body *types.Body
	}
	tests := []struct {
		name   string
		fields fields
		args   args
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

func Test_chainProcessor_handleHeader(t *testing.T) {
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
		header *types.Header
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
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
			if err := s.handleHeader(tt.args.header); (err != nil) != tt.wantErr {
				t.Errorf("handleHeader() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_chainProcessor_handleJustification(t *testing.T) {
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
		header        *types.Header
		justification []byte
	}
	tests := []struct {
		name   string
		fields fields
		args   args
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

func Test_chainProcessor_processBlockData(t *testing.T) {
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
		name    string
		fields  fields
		args    args
		wantErr bool
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
			if err := s.processBlockData(tt.args.bd); (err != nil) != tt.wantErr {
				t.Errorf("processBlockData() error = %v, wantErr %v", err, tt.wantErr)
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
