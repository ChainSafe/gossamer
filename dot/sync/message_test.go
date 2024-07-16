// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package sync

import (
	"errors"
	"testing"

	"github.com/ChainSafe/gossamer/dot/network/messages"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/common/variadic"
	lrucache "github.com/ChainSafe/gossamer/lib/utils/lru-cache"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestService_CreateBlockResponse(t *testing.T) {
	t.Parallel()

	type args struct {
		req *messages.BlockRequestMessage
	}
	tests := map[string]struct {
		blockStateBuilder func(ctrl *gomock.Controller) BlockState
		args              args
		want              *messages.BlockResponseMessage
		err               error
	}{
		"invalid_block_request": {
			blockStateBuilder: func(ctrl *gomock.Controller) BlockState {
				mockBlockState := NewMockBlockState(ctrl)
				return mockBlockState
			},
			args: args{req: &messages.BlockRequestMessage{}},
			err:  ErrInvalidBlockRequest,
		},
		"ascending_request_nil_startHash": {
			blockStateBuilder: func(ctrl *gomock.Controller) BlockState {
				mockBlockState := NewMockBlockState(ctrl)
				mockBlockState.EXPECT().BestBlockNumber().Return(uint(1), nil)
				mockBlockState.EXPECT().GetHashByNumber(uint(1)).Return(common.Hash{1, 2}, nil)
				return mockBlockState
			},
			args: args{req: &messages.BlockRequestMessage{
				StartingBlock: *variadic.MustNewUint32OrHash(uint32(0)),
				Direction:     messages.Ascending,
			}},
			want: &messages.BlockResponseMessage{BlockData: []*types.BlockData{{
				Hash: common.Hash{1, 2},
			}}},
		},
		"ascending_request_start_number_higher": {
			blockStateBuilder: func(ctrl *gomock.Controller) BlockState {
				mockBlockState := NewMockBlockState(ctrl)
				mockBlockState.EXPECT().BestBlockNumber().Return(uint(1), nil)
				return mockBlockState
			},
			args: args{req: &messages.BlockRequestMessage{
				StartingBlock: *variadic.MustNewUint32OrHash(2),
				Direction:     messages.Ascending,
			}},
			err:  errRequestStartTooHigh,
			want: nil,
		},
		"descending_request_nil_startHash": {
			blockStateBuilder: func(ctrl *gomock.Controller) BlockState {
				mockBlockState := NewMockBlockState(ctrl)
				mockBlockState.EXPECT().BestBlockNumber().Return(uint(1), nil)
				return mockBlockState
			},
			args: args{req: &messages.BlockRequestMessage{
				StartingBlock: *variadic.MustNewUint32OrHash(0),
				Direction:     messages.Descending,
			}},
			want: &messages.BlockResponseMessage{BlockData: []*types.BlockData{}},
		},
		"descending_request_start_number_higher": {
			blockStateBuilder: func(ctrl *gomock.Controller) BlockState {
				mockBlockState := NewMockBlockState(ctrl)
				mockBlockState.EXPECT().BestBlockNumber().Return(uint(1), nil)
				mockBlockState.EXPECT().GetHashByNumber(uint(1)).Return(common.Hash{1, 2}, nil)
				return mockBlockState
			},
			args: args{req: &messages.BlockRequestMessage{
				StartingBlock: *variadic.MustNewUint32OrHash(2),
				Direction:     messages.Descending,
			}},
			err: nil,
			want: &messages.BlockResponseMessage{BlockData: []*types.BlockData{{
				Hash: common.Hash{1, 2},
			}}},
		},
		"ascending_request_startHash": {
			blockStateBuilder: func(ctrl *gomock.Controller) BlockState {
				mockBlockState := NewMockBlockState(ctrl)
				mockBlockState.EXPECT().GetHeader(common.Hash{}).Return(&types.Header{
					Number: 1,
				}, nil)
				mockBlockState.EXPECT().BestBlockNumber().Return(uint(2), nil)
				mockBlockState.EXPECT().GetHashByNumber(uint(2)).Return(common.Hash{1, 2, 3}, nil)
				mockBlockState.EXPECT().IsDescendantOf(common.Hash{}, common.Hash{1, 2, 3}).Return(true,
					nil)
				mockBlockState.EXPECT().Range(common.Hash{}, common.Hash{1, 2, 3}).Return([]common.Hash{{1,
					2}},
					nil)
				return mockBlockState
			},
			args: args{req: &messages.BlockRequestMessage{
				StartingBlock: *variadic.MustNewUint32OrHash(common.Hash{}),
				Direction:     messages.Ascending,
			}},
			want: &messages.BlockResponseMessage{BlockData: []*types.BlockData{{
				Hash: common.Hash{1, 2},
			}}},
		},
		"descending_request_startHash": {
			blockStateBuilder: func(ctrl *gomock.Controller) BlockState {
				mockBlockState := NewMockBlockState(ctrl)
				mockBlockState.EXPECT().GetHeader(common.Hash{}).Return(&types.Header{
					Number: 1,
				}, nil)
				mockBlockState.EXPECT().GetHeaderByNumber(uint(1)).Return(&types.Header{
					Number: 1,
				}, nil)
				mockBlockState.EXPECT().Range(common.MustHexToHash(
					"0x6443a0b46e0412e626363028115a9f2cf963eeed526b8b33e5316f08b50d0dc3"),
					common.Hash{}).Return([]common.Hash{{1, 2}}, nil)
				return mockBlockState
			},
			args: args{req: &messages.BlockRequestMessage{
				StartingBlock: *variadic.MustNewUint32OrHash(common.Hash{}),
				Direction:     messages.Descending,
			}},
			want: &messages.BlockResponseMessage{BlockData: []*types.BlockData{{
				Hash: common.Hash{1, 2},
			}}},
		},
		"invalid_direction": {
			blockStateBuilder: func(ctrl *gomock.Controller) BlockState {
				return nil
			},
			args: args{
				req: &messages.BlockRequestMessage{
					StartingBlock: *variadic.MustNewUint32OrHash(common.Hash{}),
					Direction:     messages.SyncDirection(3),
				}},
			err: errInvalidRequestDirection,
		},
	}
	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			s := &Service{
				blockState:            tt.blockStateBuilder(ctrl),
				seenBlockSyncRequests: lrucache.NewLRUCache[common.Hash, uint](100),
			}
			got, err := s.CreateBlockResponse(peer.ID("alice"), tt.args.req)
			if tt.err != nil {
				assert.EqualError(t, err, tt.err.Error())
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestService_checkOrGetDescendantHash(t *testing.T) {
	t.Parallel()

	type args struct {
		ancestor         common.Hash
		descendant       *common.Hash
		descendantNumber uint
	}
	tests := map[string]struct {
		name              string
		blockStateBuilder func(ctrl *gomock.Controller) BlockState
		args              args
		want              common.Hash
		expectedError     error
	}{
		"nil_descendant": {
			blockStateBuilder: func(ctrl *gomock.Controller) BlockState {
				mockStateBuilder := NewMockBlockState(ctrl)
				mockStateBuilder.EXPECT().GetHashByNumber(uint(1)).Return(common.Hash{}, nil)
				mockStateBuilder.EXPECT().IsDescendantOf(common.Hash{}, common.Hash{}).Return(true, nil)
				return mockStateBuilder
			},
			args: args{ancestor: common.Hash{}, descendant: nil, descendantNumber: 1},
		},
		"not_nil_descendant": {
			blockStateBuilder: func(ctrl *gomock.Controller) BlockState {
				mockBlockState := NewMockBlockState(ctrl)
				mockBlockState.EXPECT().GetHeader(common.Hash{}).Return(&types.Header{}, nil)
				mockBlockState.EXPECT().IsDescendantOf(common.Hash{}, common.Hash{1, 2}).Return(true, nil)
				return mockBlockState
			},
			args: args{ancestor: common.Hash{0}, descendant: &common.Hash{1, 2}, descendantNumber: 1},
			want: common.Hash{1, 2},
		},
		"descendant_greater_than_header": {
			blockStateBuilder: func(ctrl *gomock.Controller) BlockState {
				mockBlockState := NewMockBlockState(ctrl)
				mockBlockState.EXPECT().GetHeader(common.Hash{2}).Return(&types.Header{
					Number: 2,
				}, nil)
				return mockBlockState
			},
			args:          args{ancestor: common.Hash{2}, descendant: &common.Hash{1, 2}, descendantNumber: 1},
			want:          common.Hash{},
			expectedError: errors.New("invalid request, descendant number 1 is lower than ancestor 2"),
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			s := &Service{
				blockState: tt.blockStateBuilder(ctrl),
			}
			got, err := s.checkOrGetDescendantHash(tt.args.ancestor, tt.args.descendant, tt.args.descendantNumber)
			if tt.expectedError != nil {
				assert.EqualError(t, err, tt.expectedError.Error())
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestService_getBlockData(t *testing.T) {
	t.Parallel()

	type args struct {
		hash          common.Hash
		requestedData byte
	}
	tests := map[string]struct {
		blockStateBuilder func(ctrl *gomock.Controller) BlockState
		args              args
		want              *types.BlockData
		err               error
	}{
		"requestedData_0": {
			blockStateBuilder: func(ctrl *gomock.Controller) BlockState {
				return nil
			},
			args: args{
				hash:          common.Hash{},
				requestedData: 0,
			},
			want: &types.BlockData{},
		},
		"requestedData_RequestedDataHeader_error": {
			blockStateBuilder: func(ctrl *gomock.Controller) BlockState {
				mockBlockState := NewMockBlockState(ctrl)
				mockBlockState.EXPECT().GetHeader(common.Hash{}).Return(nil, errors.New("empty hash"))
				return mockBlockState
			},
			args: args{
				hash:          common.Hash{0},
				requestedData: messages.RequestedDataHeader,
			},
			want: &types.BlockData{
				Hash: common.Hash{},
			},
		},
		"requestedData_RequestedDataHeader": {
			blockStateBuilder: func(ctrl *gomock.Controller) BlockState {
				mockBlockState := NewMockBlockState(ctrl)
				mockBlockState.EXPECT().GetHeader(common.Hash{1}).Return(&types.Header{
					Number: 2,
				}, nil)
				return mockBlockState
			},
			args: args{
				hash:          common.Hash{1},
				requestedData: messages.RequestedDataHeader,
			},
			want: &types.BlockData{
				Hash: common.Hash{1},
				Header: &types.Header{
					Number: 2,
				},
			},
		},
		"requestedData_RequestedDataBody_error": {
			blockStateBuilder: func(ctrl *gomock.Controller) BlockState {
				mockBlockState := NewMockBlockState(ctrl)
				mockBlockState.EXPECT().GetBlockBody(common.Hash{}).Return(nil, errors.New("empty hash"))
				return mockBlockState
			},

			args: args{
				hash:          common.Hash{},
				requestedData: messages.RequestedDataBody,
			},
			want: &types.BlockData{
				Hash: common.Hash{},
			},
		},
		"requestedData_RequestedDataBody": {
			blockStateBuilder: func(ctrl *gomock.Controller) BlockState {
				mockBlockState := NewMockBlockState(ctrl)
				mockBlockState.EXPECT().GetBlockBody(common.Hash{1}).Return(&types.Body{[]byte{1}}, nil)
				return mockBlockState
			},
			args: args{
				hash:          common.Hash{1},
				requestedData: messages.RequestedDataBody,
			},
			want: &types.BlockData{
				Hash: common.Hash{1},
				Body: &types.Body{[]byte{1}},
			},
		},
		"requestedData_RequestedDataReceipt": {
			blockStateBuilder: func(ctrl *gomock.Controller) BlockState {
				mockBlockState := NewMockBlockState(ctrl)
				mockBlockState.EXPECT().GetReceipt(common.Hash{1}).Return([]byte{1}, nil)
				return mockBlockState
			},
			args: args{
				hash:          common.Hash{1},
				requestedData: messages.RequestedDataReceipt,
			},
			want: &types.BlockData{
				Hash:    common.Hash{1},
				Receipt: &[]byte{1},
			},
		},
		"requestedData_RequestedDataMessageQueue": {
			blockStateBuilder: func(ctrl *gomock.Controller) BlockState {
				mockBlockState := NewMockBlockState(ctrl)
				mockBlockState.EXPECT().GetMessageQueue(common.Hash{2}).Return([]byte{2}, nil)
				return mockBlockState
			},
			args: args{
				hash:          common.Hash{2},
				requestedData: messages.RequestedDataMessageQueue,
			},
			want: &types.BlockData{
				Hash:         common.Hash{2},
				MessageQueue: &[]byte{2},
			},
		},
		"requestedData_RequestedDataJustification": {
			blockStateBuilder: func(ctrl *gomock.Controller) BlockState {
				mockBlockState := NewMockBlockState(ctrl)
				mockBlockState.EXPECT().GetJustification(common.Hash{3}).Return([]byte{3}, nil)
				return mockBlockState
			},
			args: args{
				hash:          common.Hash{3},
				requestedData: messages.RequestedDataJustification,
			},
			want: &types.BlockData{
				Hash:          common.Hash{3},
				Justification: &[]byte{3},
			},
		},
	}
	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			s := &Service{
				blockState: tt.blockStateBuilder(ctrl),
			}
			got, err := s.getBlockData(tt.args.hash, tt.args.requestedData)
			if tt.err != nil {
				assert.EqualError(t, err, tt.err.Error())
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.want, got)
		})
	}
}
