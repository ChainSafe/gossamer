// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package sync

import (
	"testing"

	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/golang/mock/gomock"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/stretchr/testify/assert"
)

func TestNewService(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

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
				BlockState:         newMockBlockState(ctrl, 0, 0, 0, 0, 0, 1, 0, 0, 0),
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

func TestService_HandleBlockAnnounce(t *testing.T) {
	ctrl := gomock.NewController(t)

	type fields struct {
		chainSync ChainSync
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
			name: "working example",
			fields: fields{
				chainSync: newMockChainSync(ctrl),
			},
			args: args{
				from: peer.ID("1"),
				msg: &network.BlockAnnounceMessage{
					ParentHash:     common.Hash{},
					Number:         1,
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
				chainSync: tt.fields.chainSync,
			}
			if err := s.HandleBlockAnnounce(tt.args.from, tt.args.msg); (err != nil) != tt.wantErr {
				t.Errorf("HandleBlockAnnounce() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func newMockChainSync(ctrl *gomock.Controller) ChainSync {
	mock := NewMockChainSync(ctrl)
	header, _ := types.NewHeader(common.Hash{}, common.Hash{}, common.Hash{}, 1,
		scale.VaryingDataTypeSlice{})

	mock.EXPECT().setBlockAnnounce(peer.ID("1"), header).Return(nil).AnyTimes()
	mock.EXPECT().setPeerHead(peer.ID("1"), common.Hash{}, uint(0)).Return(nil).AnyTimes()
	mock.EXPECT().syncState().Return(bootstrap).AnyTimes()
	mock.EXPECT().start().AnyTimes()
	mock.EXPECT().stop().AnyTimes()
	mock.EXPECT().getHighestBlock().Return(uint(2), nil).AnyTimes()

	return mock
}

func newMockChainProcessor(ctrl *gomock.Controller) ChainProcessor {
	mock := NewMockChainProcessor(ctrl)

	mock.EXPECT().stop().AnyTimes()

	return mock
}

func TestService_HandleBlockAnnounceHandshake(t *testing.T) {
	ctrl := gomock.NewController(t)

	type fields struct {
		chainSync ChainSync
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
		{
			name: "working example",
			fields: fields{
				chainSync: newMockChainSync(ctrl),
			},
			args: args{
				from: peer.ID("1"),
				msg: &network.BlockAnnounceHandshake{
					Roles:           0,
					BestBlockNumber: 0,
					BestBlockHash:   common.Hash{},
					GenesisHash:     common.Hash{},
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Service{
				chainSync: tt.fields.chainSync,
			}
			if err := s.HandleBlockAnnounceHandshake(tt.args.from, tt.args.msg); (err != nil) != tt.wantErr {
				t.Errorf("HandleBlockAnnounceHandshake() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestService_IsSynced(t *testing.T) {
	ctrl := gomock.NewController(t)

	type fields struct {
		chainSync ChainSync
	}
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{
			name: "working example",
			fields: fields{
				chainSync: newMockChainSync(ctrl),
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Service{
				chainSync: tt.fields.chainSync,
			}
			if got := s.IsSynced(); got != tt.want {
				t.Errorf("IsSynced() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestService_Start(t *testing.T) {
	ctrl := gomock.NewController(t)

	tests := []struct {
		name    string
		wantErr bool
	}{
		{
			name:    "working example",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockChainProcessor := NewMockChainProcessor(ctrl)
			mockChainProcessor.EXPECT().start()
			s := &Service{
				chainSync:      newMockChainSync(ctrl),
				chainProcessor: mockChainProcessor,
			}
			if err := s.Start(); (err != nil) != tt.wantErr {
				t.Errorf("Start() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestService_Stop(t *testing.T) {
	ctrl := gomock.NewController(t)

	type fields struct {
		chainSync      ChainSync
		chainProcessor ChainProcessor
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "working example",
			fields: fields{
				chainSync:      newMockChainSync(ctrl),
				chainProcessor: newMockChainProcessor(ctrl),
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Service{
				chainSync:      tt.fields.chainSync,
				chainProcessor: tt.fields.chainProcessor,
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
		name     string
		args     args
		expected args
	}{
		{
			name: "working example",
			args: args{data: []*types.BlockData{
				{
					Hash: common.MustHexToHash("0x01"),
				},
				{
					Hash: common.MustHexToHash("0x02"),
				}}},
			expected: args{data: []*types.BlockData{{
				Hash: common.MustHexToHash("0x02"),
			}, {
				Hash: common.MustHexToHash("0x01"),
			}},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reverseBlockData(tt.args.data)
			assert.Equal(t, tt.expected.data, tt.args.data)
		})
	}
}

func TestService_HighestBlock(t *testing.T) {
	ctrl := gomock.NewController(t)
	tests := []struct {
		name      string
		chainSync ChainSync
		want      uint
	}{
		{
			name:      "base case",
			chainSync: newMockChainSync(ctrl),
			want:      uint(2),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Service{
				chainSync: tt.chainSync,
			}
			got := s.HighestBlock()
			assert.Equal(t, tt.want, got)
		})
	}
}
