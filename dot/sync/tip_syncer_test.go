// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package sync

import (
	"testing"

	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func Test_tipSyncer_handleNewPeerState(t *testing.T) {
	t.Parallel()

	type fields struct {
		blockStateBuilder func(ctrl *gomock.Controller) BlockState
		pendingBlocks     DisjointBlockSet
		readyBlocks       *blockQueue
	}
	tests := map[string]struct {
		fields    fields
		peerState *peerState
		want      *worker
		err       error
	}{
		"peer state number < final block number": {
			fields: fields{
				blockStateBuilder: func(ctrl *gomock.Controller) BlockState {
					mockBlockState := NewMockBlockState(ctrl)
					mockBlockState.EXPECT().GetHighestFinalisedHeader().Return(&types.Header{
						Number: 2,
					}, nil)
					return mockBlockState
				},
			},
			peerState: &peerState{number: 1},
			want:      nil,
		},
		"base state": {
			fields: fields{
				blockStateBuilder: func(ctrl *gomock.Controller) BlockState {
					mockBlockState := NewMockBlockState(ctrl)
					mockBlockState.EXPECT().GetHighestFinalisedHeader().Return(&types.Header{
						Number: 2,
					}, nil)
					return mockBlockState
				},
			},
			peerState: &peerState{number: 3},
			want: &worker{
				startNumber:  uintPtr(3),
				targetNumber: uintPtr(3),
				requestData:  bootstrapRequestData,
			},
		},
	}
	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			s := &tipSyncer{
				blockState:    tt.fields.blockStateBuilder(ctrl),
				pendingBlocks: tt.fields.pendingBlocks,
				readyBlocks:   tt.fields.readyBlocks,
			}
			got, err := s.handleNewPeerState(tt.peerState)
			if tt.err != nil {
				assert.EqualError(t, err, tt.err.Error())
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_tipSyncer_handleTick(t *testing.T) {
	t.Parallel()

	type fields struct {
		blockStateBuilder    func(ctrl *gomock.Controller) BlockState
		pendingBlocksBuilder func(ctrl *gomock.Controller) DisjointBlockSet
		readyBlocks          *blockQueue
	}
	tests := map[string]struct {
		fields fields
		want   []*worker
		err    error
	}{
		"base case": {
			fields: fields{
				pendingBlocksBuilder: func(ctrl *gomock.Controller) DisjointBlockSet {
					mockDisjointBlockSet := NewMockDisjointBlockSet(ctrl)
					mockDisjointBlockSet.EXPECT().size().Return(1).Times(2)
					mockDisjointBlockSet.EXPECT().getBlocks().Return([]*pendingBlock{
						{number: 2},
						{number: 3},
						{number: 4,
							header: &types.Header{
								Number: 4,
							},
						},
						{number: 5,
							header: &types.Header{
								Number: 5,
							},
							body: &types.Body{},
						},
					})
					mockDisjointBlockSet.EXPECT().removeBlock(common.Hash{})
					return mockDisjointBlockSet
				},
				blockStateBuilder: func(ctrl *gomock.Controller) BlockState {
					mockBlockState := NewMockBlockState(ctrl)
					mockBlockState.EXPECT().GetHighestFinalisedHeader().Return(&types.Header{
						Number: 2,
					}, nil)
					mockBlockState.EXPECT().HasHeader(common.Hash{}).Return(false, nil)
					return mockBlockState
				},
				readyBlocks: newBlockQueue(3),
			},
			want: []*worker{
				{
					startNumber:  uintPtr(3),
					targetNumber: uintPtr(2),
					targetHash: common.Hash{5, 189, 204, 69, 79, 96, 160, 141, 66, 125, 5, 231, 241,
						159, 36, 15, 220, 57, 31, 87, 10, 183, 111, 203, 150, 236, 202, 11, 88, 35, 211, 191},
					pendingBlock: &pendingBlock{number: 3},
					requestData:  bootstrapRequestData,
					direction:    network.Descending,
				},
				{
					startNumber:  uintPtr(4),
					targetNumber: uintPtr(4),
					pendingBlock: &pendingBlock{
						number: 4,
						header: &types.Header{
							Number: 4,
						},
					},
					requestData: network.RequestedDataBody + network.RequestedDataJustification,
				},
				{
					startNumber:  uintPtr(4),
					targetNumber: uintPtr(2),
					direction:    network.Descending,
					pendingBlock: &pendingBlock{
						number: 5,
						header: &types.Header{
							Number: 5,
						},
						body: &types.Body{},
					},
					requestData: bootstrapRequestData,
				},
			},
			err: nil,
		},
	}
	for name, tt := range tests {
		tt := tt
		ctrl := gomock.NewController(t)
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			s := &tipSyncer{
				blockState:    tt.fields.blockStateBuilder(ctrl),
				pendingBlocks: tt.fields.pendingBlocksBuilder(ctrl),
				readyBlocks:   tt.fields.readyBlocks,
			}
			got, err := s.handleTick()
			if tt.err != nil {
				assert.EqualError(t, err, tt.err.Error())
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_tipSyncer_handleWorkerResult(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		blockStateBuilder func(ctrl *gomock.Controller) BlockState
		res               *worker
		want              *worker
		err               error
	}{
		"worker error is nil": {
			blockStateBuilder: func(ctrl *gomock.Controller) BlockState {
				return NewMockBlockState(ctrl)
			},
			res:  &worker{},
			want: nil,
			err:  nil,
		},
		"worker error is error unknown parent": {
			blockStateBuilder: func(ctrl *gomock.Controller) BlockState {
				return NewMockBlockState(ctrl)
			},
			res: &worker{
				err: &workerError{
					err: errUnknownParent,
				},
			},
			want: nil,
			err:  nil,
		},
		"ascending, target number < finalised number": {
			blockStateBuilder: func(ctrl *gomock.Controller) BlockState {
				mockBlockState := NewMockBlockState(ctrl)
				mockBlockState.EXPECT().GetHighestFinalisedHeader().Return(&types.Header{
					Number: 2,
				}, nil)
				return mockBlockState
			},
			res: &worker{
				targetNumber: uintPtr(1),
				direction:    network.Ascending,
				err:          &workerError{},
			},
			want: nil,
			err:  nil,
		},
		"ascending, start number < finalised number": {
			blockStateBuilder: func(ctrl *gomock.Controller) BlockState {
				mockBlockState := NewMockBlockState(ctrl)
				mockBlockState.EXPECT().GetHighestFinalisedHeader().Return(&types.Header{
					Number: 2,
				}, nil)
				return mockBlockState
			},
			res: &worker{
				startNumber:  uintPtr(1),
				targetNumber: uintPtr(3),
				direction:    network.Ascending,
				err:          &workerError{},
			},
			want: &worker{
				startNumber:  uintPtr(3),
				targetNumber: uintPtr(3),
			},
			err: nil,
		},
		"descending, start number < finalised number": {
			blockStateBuilder: func(ctrl *gomock.Controller) BlockState {
				mockBlockState := NewMockBlockState(ctrl)
				mockBlockState.EXPECT().GetHighestFinalisedHeader().Return(&types.Header{
					Number: 2,
				}, nil)
				return mockBlockState
			},
			res: &worker{
				startNumber: uintPtr(1),
				direction:   network.Descending,
				err:         &workerError{},
			},
			want: nil,
			err:  nil,
		},
		"descending, target number < finalised number": {
			blockStateBuilder: func(ctrl *gomock.Controller) BlockState {
				mockBlockState := NewMockBlockState(ctrl)
				mockBlockState.EXPECT().GetHighestFinalisedHeader().Return(&types.Header{
					Number: 2,
				}, nil)
				return mockBlockState
			},
			res: &worker{
				startNumber:  uintPtr(3),
				targetNumber: uintPtr(1),
				direction:    network.Descending,
				err:          &workerError{},
			},
			want: &worker{
				startNumber:  uintPtr(3),
				targetNumber: uintPtr(3),
				direction:    network.Descending,
			},
			err: nil,
		},
	}
	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			s := &tipSyncer{
				blockState: tt.blockStateBuilder(ctrl),
			}
			got, err := s.handleWorkerResult(tt.res)
			if tt.err != nil {
				assert.EqualError(t, err, tt.err.Error())
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_tipSyncer_hasCurrentWorker(t *testing.T) {
	t.Parallel()

	type args struct {
		w       *worker
		workers map[uint64]*worker
	}
	tests := map[string]struct {
		args args
		want bool
	}{
		"worker nil": {
			want: true,
		},
		"ascending, false": {
			args: args{
				w: &worker{
					direction:    network.Ascending,
					startNumber:  uintPtr(2),
					targetNumber: uintPtr(2),
				},
				workers: map[uint64]*worker{
					1: {
						direction:    network.Ascending,
						targetNumber: uintPtr(3),
						startNumber:  uintPtr(3),
					},
				},
			},
			want: false,
		},
		"ascending, true": {
			args: args{
				w: &worker{
					direction:    network.Ascending,
					startNumber:  uintPtr(2),
					targetNumber: uintPtr(2),
				},
				workers: map[uint64]*worker{
					1: {
						direction:    network.Ascending,
						targetNumber: uintPtr(3),
						startNumber:  uintPtr(1),
					},
				},
			},
			want: true,
		},
		"descending, false": {
			args: args{
				w: &worker{
					direction:    network.Descending,
					startNumber:  uintPtr(2),
					targetNumber: uintPtr(2),
				},
				workers: map[uint64]*worker{
					1: {
						startNumber:  uintPtr(3),
						targetNumber: uintPtr(3),
						direction:    network.Descending,
					},
				},
			},
			want: false,
		},
		"descending, true": {
			args: args{
				w: &worker{
					direction:    network.Descending,
					startNumber:  uintPtr(2),
					targetNumber: uintPtr(2),
				},
				workers: map[uint64]*worker{
					1: {
						startNumber:  uintPtr(3),
						targetNumber: uintPtr(1),
						direction:    network.Descending,
					},
				},
			},
			want: true,
		},
	}
	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			s := &tipSyncer{}
			got := s.hasCurrentWorker(tt.args.w, tt.args.workers)
			assert.Equal(t, tt.want, got)
		})
	}
}
