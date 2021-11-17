// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package sync

import (
	"errors"
	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/dot/state"
	"github.com/ChainSafe/gossamer/dot/sync/mocks"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"math/big"
	"testing"
)

func TestNewService(t *testing.T) {
	cfg := &Config{}
	testDatadirPath := t.TempDir()

	cfg.Network = newMockNetwork()

	type args struct {
		cfg *Config
	}
	scfg := state.Config{
		Path:     testDatadirPath,
		LogLevel: log.Info,
	}
	stateSrvc := state.NewService(scfg)
	stateSrvc.UseMemDB()

	gen, genTrie, genHeader := newTestGenesisWithTrieAndHeader(t)
	err := stateSrvc.Initialise(gen, genHeader, genTrie)
	require.NoError(t, err)

	err = stateSrvc.Start()
	require.NoError(t, err)

	cfg.BlockState = stateSrvc.Block

	cfg.StorageState = stateSrvc.Storage

	cfg.TransactionState = stateSrvc.Transaction

	cfg.BabeVerifier = newMockBabeVerifier()

	cfg.FinalityGadget = newMockFinalityGadget()

	cfg.BlockImportHandler = new(mocks.BlockImportHandler)

	tests := []struct {
		name    string
		args    args
		want    *Service
		err error
	}{
		{
			name:    "empty config",
			args:    args{cfg: &Config{}},
			err: errors.New("cannot have nil Network"),
		},
		{
			name:    "working example",
			args:    args{cfg: cfg},
			want:    &Service{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewService(tt.args.cfg)
			if tt.err != nil {
				assert.EqualError(t, err, tt.err.Error())
			} else {
				assert.NoError(t, err)
			}

			if tt.want != nil {
				assert.NotNil(t, got)
			} else {
				assert.Nil(t, got)
			}
		})
	}
}

func TestService_HandleBlockAnnounce(t *testing.T) {

	blockState := mocks.BlockState{}

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
		{
			name:    "test",
			fields:  fields{
				blockState:     &blockState,
				chainSync:      nil,
				chainProcessor: nil,
				network:        nil,
			},
			args:    args{
				from: "1",
				msg:  &network.BlockAnnounceMessage{
					ParentHash:     common.Hash{},
					Number:         big.NewInt(1),
					StateRoot:      common.Hash{},
					ExtrinsicsRoot: common.Hash{},
					Digest:         scale.VaryingDataTypeSlice{},
					BestBlock:      false,
				},
			},
			wantErr: false,
		},
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