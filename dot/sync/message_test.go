// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package sync

import (
	"errors"
	"reflect"
	"testing"

	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/common/variadic"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestService_CreateBlockResponse(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockBlockState := NewMockBlockState(ctrl)
	mockBlockState.EXPECT().BestBlockNumber().Return(uint(1), nil).Times(8)
	mockBlockState.EXPECT().GetHashByNumber(gomock.Any()).DoAndReturn(func(uint) (
		common.Hash, error) {
		return common.Hash{1, 2}, nil
	}).Times(5)
	mockBlockState.EXPECT().IsDescendantOf(gomock.AssignableToTypeOf(common.Hash{}),
		gomock.AssignableToTypeOf(common.Hash{})).Return(true, nil).Times(4)
	mockBlockState.EXPECT().GetHeader(gomock.AssignableToTypeOf(common.Hash{})).Return(&types.Header{
		Number: 1,
	}, nil).Times(4)
	mockBlockState.EXPECT().SubChain(gomock.AssignableToTypeOf(common.Hash{}),
		gomock.AssignableToTypeOf(common.Hash{})).Return([]common.Hash{{1, 2}}, nil).Times(4)
	mockBlockState.EXPECT().GetHeaderByNumber(uint(1)).Return(&types.Header{
		Number: 1,
	}, nil)

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
		name   string
		fields fields
		args   args
		want   *network.BlockResponseMessage
		err    error
	}{
		{
			name: "invalid block request",
			args: args{req: &network.BlockRequestMessage{}},
			err:  ErrInvalidBlockRequest,
		},
		{
			name: "ascending request, nil startHash, nil endHash",
			fields: fields{
				blockState: mockBlockState,
			},
			args: args{req: &network.BlockRequestMessage{
				StartingBlock: *variadic.MustNewUint32OrHash(0),
				Direction:     network.Ascending,
			}},
			want: &network.BlockResponseMessage{BlockData: []*types.BlockData{{
				Hash: common.Hash{1, 2},
			}}},
		},
		{
			name: "ascending request, start number higher",
			fields: fields{
				blockState: mockBlockState,
			},
			args: args{req: &network.BlockRequestMessage{
				StartingBlock: *variadic.MustNewUint32OrHash(2),
				Direction:     network.Ascending,
			}},
			err:  errRequestStartTooHigh,
			want: nil,
		},
		{
			name: "ascending request, endHash not nil",
			fields: fields{
				blockState: mockBlockState,
			},
			args: args{req: &network.BlockRequestMessage{
				StartingBlock: *variadic.MustNewUint32OrHash(0),
				EndBlockHash:  &common.Hash{1, 2, 3},
				Direction:     network.Ascending,
			}},
			want: &network.BlockResponseMessage{BlockData: []*types.BlockData{{
				Hash: common.Hash{1, 2},
			}}},
		},
		{
			name: "descending request, nil startHash, nil endHash",
			fields: fields{
				blockState: mockBlockState,
			},
			args: args{req: &network.BlockRequestMessage{
				StartingBlock: *variadic.MustNewUint32OrHash(0),
				Direction:     network.Descending,
			}},
			want: &network.BlockResponseMessage{BlockData: []*types.BlockData{}},
		},
		{
			name: "descending request, start number higher",
			fields: fields{
				blockState: mockBlockState,
			},
			args: args{req: &network.BlockRequestMessage{
				StartingBlock: *variadic.MustNewUint32OrHash(2),
				Direction:     network.Descending,
			}},
			err: nil,
			want: &network.BlockResponseMessage{BlockData: []*types.BlockData{{
				Hash: common.Hash{1, 2},
			}}},
		},
		{
			name: "descending request, endHash not nil",
			fields: fields{
				blockState: mockBlockState,
			},
			args: args{req: &network.BlockRequestMessage{
				StartingBlock: *variadic.MustNewUint32OrHash(0),
				EndBlockHash:  &common.Hash{1, 2, 3},
				Direction:     network.Descending,
			}},
			want: &network.BlockResponseMessage{BlockData: []*types.BlockData{{
				Hash: common.Hash{1, 2},
			}}},
		},
		{
			name: "ascending request, startHash, nil endHash",
			fields: fields{
				blockState: mockBlockState,
			},
			args: args{req: &network.BlockRequestMessage{
				StartingBlock: *variadic.MustNewUint32OrHash(common.Hash{}),
				Direction:     network.Ascending,
			}},
			want: &network.BlockResponseMessage{BlockData: []*types.BlockData{{
				Hash: common.Hash{1, 2},
			}}},
		},
		{
			name: "descending request, startHash, nil endHash",
			fields: fields{
				blockState: mockBlockState,
			},
			args: args{req: &network.BlockRequestMessage{
				StartingBlock: *variadic.MustNewUint32OrHash(common.Hash{}),
				Direction:     network.Descending,
			}},
			want: &network.BlockResponseMessage{BlockData: []*types.BlockData{{
				Hash: common.Hash{1, 2},
			}}},
		},
		{
			name: "invalid direction",
			args: args{req: &network.BlockRequestMessage{
				Direction: network.SyncDirection(3),
			}},
			err: errInvalidRequestDirection,
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
			got, err := s.CreateBlockResponse(tt.args.req)
			if tt.err != nil {
				assert.EqualError(t, err, tt.err.Error())
			} else {
				assert.NoError(t, err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CreateBlockResponse() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func newMockBlockStateForMessageTest(ctrl *gomock.Controller) BlockState {
	mock := NewMockBlockState(ctrl)

	mock.EXPECT().GetHashByNumber(gomock.AssignableToTypeOf(uint(0))).DoAndReturn(func(
		number uint) (common.Hash, error) {
		return common.Hash{}, nil
	}).AnyTimes()

	mock.EXPECT().IsDescendantOf(gomock.AssignableToTypeOf(common.Hash{}),
		gomock.AssignableToTypeOf(common.Hash{})).Return(true, nil).AnyTimes()

	mock.EXPECT().GetHeader(gomock.AssignableToTypeOf(common.Hash{})).DoAndReturn(func(hash common.Hash) (*types.
		Header, error) {
		header := &types.Header{
			Number: uint(hash[0]),
		}
		return header, nil
	}).AnyTimes()

	return mock
}

func TestService_checkOrGetDescendantHash(t *testing.T) {
	ctrl := gomock.NewController(t)
	type fields struct {
		blockState     BlockState
		chainSync      ChainSync
		chainProcessor ChainProcessor
		network        Network
	}
	type args struct {
		ancestor         common.Hash
		descendant       *common.Hash
		descendantNumber uint
	}
	tests := []struct {
		name          string
		fields        fields
		args          args
		want          common.Hash
		expectedError error
	}{
		{
			name: "nil descendant",
			fields: fields{
				blockState: newMockBlockStateForMessageTest(ctrl)},
			args: args{ancestor: common.Hash{}, descendant: nil, descendantNumber: 1},
		},
		{
			name: "not nil descendant",
			fields: fields{
				blockState: newMockBlockStateForMessageTest(ctrl)},
			args: args{ancestor: common.Hash{0}, descendant: &common.Hash{1, 2}, descendantNumber: 1},
			want: common.Hash{1, 2},
		},
		{
			name: "descendant greater than header",
			fields: fields{
				blockState: newMockBlockStateForMessageTest(ctrl)},
			args:          args{ancestor: common.Hash{2}, descendant: &common.Hash{1, 2}, descendantNumber: 1},
			want:          common.Hash{},
			expectedError: errors.New("invalid request, descendant number 2 is higher than ancestor 1"),
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
			got, err := s.checkOrGetDescendantHash(tt.args.ancestor, tt.args.descendant, tt.args.descendantNumber)
			if tt.expectedError != nil {
				assert.EqualError(t, err, tt.expectedError.Error())
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("checkOrGetDescendantHash() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestService_getBlockData(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockBlockState := NewMockBlockState(ctrl)
	mockBlockState.EXPECT().GetHeader(common.Hash{}).Return(nil, errors.New("empty hash"))
	mockBlockState.EXPECT().GetHeader(common.Hash{1}).Return(&types.Header{
		Number: 2,
	}, nil)
	mockBlockState.EXPECT().GetBlockBody(common.Hash{}).Return(nil, errors.New("empty hash"))
	mockBlockState.EXPECT().GetBlockBody(common.Hash{1}).Return(&types.Body{[]byte{1}}, nil)
	mockBlockState.EXPECT().GetReceipt(common.Hash{1}).Return([]byte{1}, nil)
	mockBlockState.EXPECT().GetMessageQueue(common.Hash{2}).Return([]byte{2}, nil)
	mockBlockState.EXPECT().GetJustification(common.Hash{3}).Return([]byte{3}, nil)

	type fields struct {
		blockState BlockState
	}
	type args struct {
		hash          common.Hash
		requestedData byte
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   *types.BlockData
		err    error
	}{
		{
			name: "requestedData 0",
			args: args{
				hash:          common.Hash{},
				requestedData: 0,
			},
			want: &types.BlockData{},
		},
		{
			name: "requestedData RequestedDataHeader, error",
			fields: fields{
				blockState: mockBlockState,
			},
			args: args{
				hash:          common.Hash{0},
				requestedData: network.RequestedDataHeader,
			},
			want: &types.BlockData{
				Hash: common.Hash{},
			},
		},
		{
			name: "requestedData RequestedDataHeader",
			fields: fields{
				blockState: mockBlockState,
			},
			args: args{
				hash:          common.Hash{1},
				requestedData: network.RequestedDataHeader,
			},
			want: &types.BlockData{
				Hash: common.Hash{1},
				Header: &types.Header{
					Number: 2,
				},
			},
		},
		{
			name: "requestedData RequestedDataBody, error",
			fields: fields{
				blockState: mockBlockState,
			},
			args: args{
				hash:          common.Hash{},
				requestedData: network.RequestedDataBody,
			},
			want: &types.BlockData{
				Hash: common.Hash{},
			},
		},
		{
			name: "requestedData RequestedDataBody",
			fields: fields{
				blockState: mockBlockState,
			},
			args: args{
				hash:          common.Hash{1},
				requestedData: network.RequestedDataBody,
			},
			want: &types.BlockData{
				Hash: common.Hash{1},
				Body: &types.Body{[]byte{1}},
			},
		},
		{
			name: "requestedData RequestedDataReceipt",
			fields: fields{
				blockState: mockBlockState,
			},
			args: args{
				hash:          common.Hash{1},
				requestedData: network.RequestedDataReceipt,
			},
			want: &types.BlockData{
				Hash:    common.Hash{1},
				Receipt: &[]byte{1},
			},
		},
		{
			name: "requestedData RequestedDataMessageQueue",
			fields: fields{
				blockState: mockBlockState,
			},
			args: args{
				hash:          common.Hash{2},
				requestedData: network.RequestedDataMessageQueue,
			},
			want: &types.BlockData{
				Hash:         common.Hash{2},
				MessageQueue: &[]byte{2},
			},
		},
		{
			name: "requestedData RequestedDataJustification",
			fields: fields{
				blockState: mockBlockState,
			},
			args: args{
				hash:          common.Hash{3},
				requestedData: network.RequestedDataJustification,
			},
			want: &types.BlockData{
				Hash:          common.Hash{3},
				Justification: &[]byte{3},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Service{
				blockState: tt.fields.blockState,
			}
			got, err := s.getBlockData(tt.args.hash, tt.args.requestedData)
			if tt.err != nil {
				assert.EqualError(t, err, tt.err.Error())
			} else {
				assert.NoError(t, err)
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
		num           uint
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
		start         uint
		end           uint
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
		max           uint
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
			got, err := s.handleChainByHash(tt.args.ancestor, tt.args.descendant, tt.args.max, tt.args.requestedData,
				tt.args.direction)
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
		start         uint
		end           uint
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
