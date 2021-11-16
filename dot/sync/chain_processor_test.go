// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package sync

import (
	"context"
	"fmt"
	"github.com/ChainSafe/gossamer/dot/types"
	"reflect"
	"testing"
)

func Test_chainProcessor_handleBlock(t *testing.T) {
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
		block *types.Block
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
			if err := s.handleBlock(tt.args.block); (err != nil) != tt.wantErr {
				t.Errorf("handleBlock() error = %v, wantErr %v", err, tt.wantErr)
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