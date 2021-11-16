// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package sync

import (
	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"math/big"
	"reflect"
	"testing"
)

func TestService_CreateBlockResponse(t *testing.T) {
	type fields struct {
		blockState     BlockState
		chainSync      ChainSync
		chainProcessor ChainProcessor
		network        Network
	}
	type args struct {
		req *network.BlockRequestMessage
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *network.BlockResponseMessage
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Service{
				blockState:     tt.fields.blockState,
				chainSync:      tt.fields.chainSync,
				chainProcessor: tt.fields.chainProcessor,
				network:        tt.fields.network,
			}
			got, err := s.CreateBlockResponse(tt.args.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateBlockResponse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CreateBlockResponse() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestService_checkOrGetDescendantHash(t *testing.T) {
	type fields struct {
		blockState     BlockState
		chainSync      ChainSync
		chainProcessor ChainProcessor
		network        Network
	}
	type args struct {
		ancestor         common.Hash
		descendant       *common.Hash
		descendantNumber *big.Int
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    common.Hash
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Service{
				blockState:     tt.fields.blockState,
				chainSync:      tt.fields.chainSync,
				chainProcessor: tt.fields.chainProcessor,
				network:        tt.fields.network,
			}
			got, err := s.checkOrGetDescendantHash(tt.args.ancestor, tt.args.descendant, tt.args.descendantNumber)
			if (err != nil) != tt.wantErr {
				t.Errorf("checkOrGetDescendantHash() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("checkOrGetDescendantHash() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestService_getBlockData(t *testing.T) {
	type fields struct {
		blockState     BlockState
		chainSync      ChainSync
		chainProcessor ChainProcessor
		network        Network
	}
	type args struct {
		hash          common.Hash
		requestedData byte
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *types.BlockData
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Service{
				blockState:     tt.fields.blockState,
				chainSync:      tt.fields.chainSync,
				chainProcessor: tt.fields.chainProcessor,
				network:        tt.fields.network,
			}
			got, err := s.getBlockData(tt.args.hash, tt.args.requestedData)
			if (err != nil) != tt.wantErr {
				t.Errorf("getBlockData() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getBlockData() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestService_getBlockDataByNumber(t *testing.T) {
	type fields struct {
		blockState     BlockState
		chainSync      ChainSync
		chainProcessor ChainProcessor
		network        Network
	}
	type args struct {
		num           *big.Int
		requestedData byte
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *types.BlockData
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Service{
				blockState:     tt.fields.blockState,
				chainSync:      tt.fields.chainSync,
				chainProcessor: tt.fields.chainProcessor,
				network:        tt.fields.network,
			}
			got, err := s.getBlockDataByNumber(tt.args.num, tt.args.requestedData)
			if (err != nil) != tt.wantErr {
				t.Errorf("getBlockDataByNumber() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getBlockDataByNumber() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestService_handleAscendingByNumber(t *testing.T) {
	type fields struct {
		blockState     BlockState
		chainSync      ChainSync
		chainProcessor ChainProcessor
		network        Network
	}
	type args struct {
		start         uint64
		end           uint64
		requestedData byte
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *network.BlockResponseMessage
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Service{
				blockState:     tt.fields.blockState,
				chainSync:      tt.fields.chainSync,
				chainProcessor: tt.fields.chainProcessor,
				network:        tt.fields.network,
			}
			got, err := s.handleAscendingByNumber(tt.args.start, tt.args.end, tt.args.requestedData)
			if (err != nil) != tt.wantErr {
				t.Errorf("handleAscendingByNumber() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("handleAscendingByNumber() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestService_handleAscendingRequest(t *testing.T) {
	type fields struct {
		blockState     BlockState
		chainSync      ChainSync
		chainProcessor ChainProcessor
		network        Network
	}
	type args struct {
		req *network.BlockRequestMessage
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *network.BlockResponseMessage
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Service{
				blockState:     tt.fields.blockState,
				chainSync:      tt.fields.chainSync,
				chainProcessor: tt.fields.chainProcessor,
				network:        tt.fields.network,
			}
			got, err := s.handleAscendingRequest(tt.args.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("handleAscendingRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("handleAscendingRequest() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestService_handleChainByHash(t *testing.T) {
	type fields struct {
		blockState     BlockState
		chainSync      ChainSync
		chainProcessor ChainProcessor
		network        Network
	}
	type args struct {
		ancestor      common.Hash
		descendant    common.Hash
		max           uint32
		requestedData byte
		direction     network.SyncDirection
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *network.BlockResponseMessage
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Service{
				blockState:     tt.fields.blockState,
				chainSync:      tt.fields.chainSync,
				chainProcessor: tt.fields.chainProcessor,
				network:        tt.fields.network,
			}
			got, err := s.handleChainByHash(tt.args.ancestor, tt.args.descendant, tt.args.max, tt.args.requestedData, tt.args.direction)
			if (err != nil) != tt.wantErr {
				t.Errorf("handleChainByHash() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("handleChainByHash() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestService_handleDescendingByNumber(t *testing.T) {
	type fields struct {
		blockState     BlockState
		chainSync      ChainSync
		chainProcessor ChainProcessor
		network        Network
	}
	type args struct {
		start         uint64
		end           uint64
		requestedData byte
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *network.BlockResponseMessage
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Service{
				blockState:     tt.fields.blockState,
				chainSync:      tt.fields.chainSync,
				chainProcessor: tt.fields.chainProcessor,
				network:        tt.fields.network,
			}
			got, err := s.handleDescendingByNumber(tt.args.start, tt.args.end, tt.args.requestedData)
			if (err != nil) != tt.wantErr {
				t.Errorf("handleDescendingByNumber() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("handleDescendingByNumber() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestService_handleDescendingRequest(t *testing.T) {
	type fields struct {
		blockState     BlockState
		chainSync      ChainSync
		chainProcessor ChainProcessor
		network        Network
	}
	type args struct {
		req *network.BlockRequestMessage
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *network.BlockResponseMessage
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Service{
				blockState:     tt.fields.blockState,
				chainSync:      tt.fields.chainSync,
				chainProcessor: tt.fields.chainProcessor,
				network:        tt.fields.network,
			}
			got, err := s.handleDescendingRequest(tt.args.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("handleDescendingRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("handleDescendingRequest() got = %v, want %v", got, tt.want)
			}
		})
	}
}