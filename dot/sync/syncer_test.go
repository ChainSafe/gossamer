// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package sync

import (
	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/libp2p/go-libp2p-core/peer"
	"reflect"
	"testing"
)

func TestNewService(t *testing.T) {
	type args struct {
		cfg *Config
	}
	tests := []struct {
		name    string
		args    args
		want    *Service
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewService(tt.args.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewService() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewService() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestService_HandleBlockAnnounce(t *testing.T) {
	type fields struct {
		blockState     BlockState
		chainSync      ChainSync
		chainProcessor ChainProcessor
		network        Network
	}
	type args struct {
		from peer.ID
		msg  *network.BlockAnnounceMessage
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
			s := &Service{
				blockState:     tt.fields.blockState,
				chainSync:      tt.fields.chainSync,
				chainProcessor: tt.fields.chainProcessor,
				network:        tt.fields.network,
			}
			if err := s.HandleBlockAnnounce(tt.args.from, tt.args.msg); (err != nil) != tt.wantErr {
				t.Errorf("HandleBlockAnnounce() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestService_HandleBlockAnnounceHandshake(t *testing.T) {
	type fields struct {
		blockState     BlockState
		chainSync      ChainSync
		chainProcessor ChainProcessor
		network        Network
	}
	type args struct {
		from peer.ID
		msg  *network.BlockAnnounceHandshake
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
			s := &Service{
				blockState:     tt.fields.blockState,
				chainSync:      tt.fields.chainSync,
				chainProcessor: tt.fields.chainProcessor,
				network:        tt.fields.network,
			}
			if err := s.HandleBlockAnnounceHandshake(tt.args.from, tt.args.msg); (err != nil) != tt.wantErr {
				t.Errorf("HandleBlockAnnounceHandshake() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestService_IsSynced(t *testing.T) {
	type fields struct {
		blockState     BlockState
		chainSync      ChainSync
		chainProcessor ChainProcessor
		network        Network
	}
	tests := []struct {
		name   string
		fields fields
		want   bool
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
			if got := s.IsSynced(); got != tt.want {
				t.Errorf("IsSynced() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestService_Start(t *testing.T) {
	type fields struct {
		blockState     BlockState
		chainSync      ChainSync
		chainProcessor ChainProcessor
		network        Network
	}
	tests := []struct {
		name    string
		fields  fields
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
			if err := s.Start(); (err != nil) != tt.wantErr {
				t.Errorf("Start() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestService_Stop(t *testing.T) {
	type fields struct {
		blockState     BlockState
		chainSync      ChainSync
		chainProcessor ChainProcessor
		network        Network
	}
	tests := []struct {
		name    string
		fields  fields
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
			if err := s.Stop(); (err != nil) != tt.wantErr {
				t.Errorf("Stop() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_reverseBlockData(t *testing.T) {
	type args struct {
		data []*types.BlockData
	}
	tests := []struct {
		name string
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
		})
	}
}