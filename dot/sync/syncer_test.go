// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package sync

import (
	"errors"
	"sync"
	"testing"

	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/golang/mock/gomock"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/assert"
)

func TestNewService(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		cfgBuilder func(ctrl *gomock.Controller) *Config
		want       *Service
		err        error
	}{
		{
			name: "working_example",
			cfgBuilder: func(ctrl *gomock.Controller) *Config {
				blockState := NewMockBlockState(ctrl)
				blockState.EXPECT().GetFinalisedNotifierChannel().
					Return(make(chan *types.FinalisationInfo))
				return &Config{
					BlockState: blockState,
				}
			},
			want: &Service{},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)

			config := tt.cfgBuilder(ctrl)
			mockReqRes := NewMockRequestMaker(ctrl)

			got, err := NewService(config, mockReqRes)
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
	t.Parallel()

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
			name: "working_example",
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
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
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
	header := types.NewHeader(common.Hash{}, common.Hash{}, common.Hash{}, 1,
		scale.VaryingDataTypeSlice{})

	mock.EXPECT().setBlockAnnounce(peer.ID("1"), header).Return(nil).AnyTimes()
	mock.EXPECT().setPeerHead(peer.ID("1"), common.Hash{}, uint(0)).Return(nil).AnyTimes()
	mock.EXPECT().syncState().Return(bootstrap).AnyTimes()
	mock.EXPECT().start().AnyTimes()
	mock.EXPECT().stop().AnyTimes()
	mock.EXPECT().getHighestBlock().Return(uint(2), nil).AnyTimes()

	return mock
}

func Test_Service_HandleBlockAnnounceHandshake(t *testing.T) {
	t.Parallel()

	errTest := errors.New("test error")

	testCases := map[string]struct {
		serviceBuilder func(ctrl *gomock.Controller) Service
		from           peer.ID
		message        *network.BlockAnnounceHandshake
		errWrapped     error
		errMessage     string
	}{
		"success": {
			serviceBuilder: func(ctrl *gomock.Controller) Service {
				chainSync := NewMockChainSync(ctrl)
				chainSync.EXPECT().setPeerHead(peer.ID("abc"), common.Hash{1}, uint(2)).
					Return(nil)
				return Service{
					chainSync: chainSync,
				}
			},
			from: peer.ID("abc"),
			message: &network.BlockAnnounceHandshake{
				BestBlockHash:   common.Hash{1},
				BestBlockNumber: 2,
			},
		},
		"failure": {
			serviceBuilder: func(ctrl *gomock.Controller) Service {
				chainSync := NewMockChainSync(ctrl)
				chainSync.EXPECT().setPeerHead(peer.ID("abc"), common.Hash{1}, uint(2)).
					Return(errTest)
				return Service{
					chainSync: chainSync,
				}
			},
			from: peer.ID("abc"),
			message: &network.BlockAnnounceHandshake{
				BestBlockHash:   common.Hash{1},
				BestBlockNumber: 2,
			},
			errWrapped: errTest,
			errMessage: "test error",
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)

			service := testCase.serviceBuilder(ctrl)

			err := service.HandleBlockAnnounceHandshake(testCase.from, testCase.message)

			assert.ErrorIs(t, err, testCase.errWrapped)
			if testCase.errWrapped != nil {
				assert.EqualError(t, err, testCase.errMessage)
			}
		})
	}
}

func TestService_IsSynced(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		serviceBuilder func(ctrl *gomock.Controller) Service
		synced         bool
	}{
		"tip": {
			serviceBuilder: func(ctrl *gomock.Controller) Service {
				chainSync := NewMockChainSync(ctrl)
				chainSync.EXPECT().syncState().Return(tip)
				return Service{
					chainSync: chainSync,
				}
			},
			synced: true,
		},
		"not_tip": {
			serviceBuilder: func(ctrl *gomock.Controller) Service {
				chainSync := NewMockChainSync(ctrl)
				chainSync.EXPECT().syncState().Return(bootstrap)
				return Service{
					chainSync: chainSync,
				}
			},
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)

			service := testCase.serviceBuilder(ctrl)

			synced := service.IsSynced()

			assert.Equal(t, testCase.synced, synced)
		})
	}
}

func TestService_Start(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)

	var allCalled sync.WaitGroup

	chainSync := NewMockChainSync(ctrl)
	allCalled.Add(1)
	chainSync.EXPECT().start().DoAndReturn(func() {
		allCalled.Done()
	})

	chainProcessor := NewMockChainProcessor(ctrl)
	allCalled.Add(1)
	chainProcessor.EXPECT().processReadyBlocks().DoAndReturn(func() {
		allCalled.Done()
	})

	service := Service{
		chainSync:      chainSync,
		chainProcessor: chainProcessor,
	}

	err := service.Start()
	allCalled.Wait()
	assert.NoError(t, err)
}

func TestService_Stop(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)

	chainSync := NewMockChainSync(ctrl)
	chainSync.EXPECT().stop()
	chainProcessor := NewMockChainProcessor(ctrl)
	chainProcessor.EXPECT().stop()

	service := &Service{
		chainSync:      chainSync,
		chainProcessor: chainProcessor,
	}

	err := service.Stop()
	assert.NoError(t, err)
}

func Test_reverseBlockData(t *testing.T) {
	t.Parallel()

	type args struct {
		data []*types.BlockData
	}
	tests := []struct {
		name     string
		args     args
		expected args
	}{
		{
			name: "working_example",
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
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			reverseBlockData(tt.args.data)
			assert.Equal(t, tt.expected.data, tt.args.data)
		})
	}
}

func TestService_HighestBlock(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)

	chainSync := NewMockChainSync(ctrl)
	chainSync.EXPECT().getHighestBlock().Return(uint(2), nil)

	service := &Service{
		chainSync: chainSync,
	}
	highestBlock := service.HighestBlock()
	const expected = uint(2)
	assert.Equal(t, expected, highestBlock)
}
