// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package sync

import (
	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/dot/sync/mocks"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/golang/mock/gomock"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/stretchr/testify/assert"
	"math/big"
	"testing"
)

func TestNewService(t *testing.T) {
	ctrl := gomock.NewController(t)

	type args struct {
		cfg *Config
	}
	tests := []struct {
		name string
		args args
		want *Service
		err  error
	}{
		{
			name: "nil Network",
			args: args{cfg: &Config{}},
			err:  errNilNetwork,
		},
		{
			name: "nil BlockState",
			args: args{cfg: &Config{
				Network: NewMockNetwork(ctrl),
			}},
			err: errNilBlockState,
		},
		{
			name: "nil StorageState",
			args: args{cfg: &Config{
				Network:    NewMockNetwork(ctrl),
				BlockState: NewMockBlockState(ctrl),
			}},
			err: errNilStorageState,
		},
		{
			name: "nil FinalityGadget",
			args: args{cfg: &Config{
				Network:      NewMockNetwork(ctrl),
				BlockState:   NewMockBlockState(ctrl),
				StorageState: NewMockStorageState(ctrl),
			}},
			err: errNilFinalityGadget,
		},
		{
			name: "nil TransactionState",
			args: args{cfg: &Config{
				Network:        NewMockNetwork(ctrl),
				BlockState:     NewMockBlockState(ctrl),
				StorageState:   NewMockStorageState(ctrl),
				FinalityGadget: NewMockFinalityGadget(ctrl),
			}},
			err: errNilTransactionState,
		},
		{
			name: "nil Verifier",
			args: args{cfg: &Config{
				Network:          NewMockNetwork(ctrl),
				BlockState:       NewMockBlockState(ctrl),
				StorageState:     NewMockStorageState(ctrl),
				FinalityGadget:   NewMockFinalityGadget(ctrl),
				TransactionState: NewMockTransactionState(ctrl),
			}},
			err: errNilVerifier,
		},
		{
			name: "nil BlockImportHandler",
			args: args{cfg: &Config{
				Network:          NewMockNetwork(ctrl),
				BlockState:       NewMockBlockState(ctrl),
				StorageState:     NewMockStorageState(ctrl),
				FinalityGadget:   NewMockFinalityGadget(ctrl),
				TransactionState: NewMockTransactionState(ctrl),
				BabeVerifier:     NewMockBabeVerifier(ctrl),
			}},
			err: errNilBlockImportHandler,
		},
		{
			name: "working example",
			args: args{cfg: &Config{
				Network:            NewMockNetwork(ctrl),
				BlockState:         newMockBlockState(ctrl),
				StorageState:       NewMockStorageState(ctrl),
				FinalityGadget:     NewMockFinalityGadget(ctrl),
				TransactionState:   NewMockTransactionState(ctrl),
				BabeVerifier:       NewMockBabeVerifier(ctrl),
				BlockImportHandler: NewMockBlockImportHandler(ctrl),
			}},
			want: &Service{},
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
			}
		})
	}
}

func newMockBlockState(ctrl *gomock.Controller) BlockState {
	mock := NewMockBlockState(ctrl)
	mock.EXPECT().GetFinalisedNotifierChannel().Return(make(chan *types.FinalisationInfo))
	return mock
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
			name: "test",
			fields: fields{
				blockState:     &blockState,
				chainSync:      nil,
				chainProcessor: nil,
				network:        nil,
			},
			args: args{
				from: "1",
				msg: &network.BlockAnnounceMessage{
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
