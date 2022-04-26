// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package sync

import (
	"reflect"
	"testing"

	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func Test_tipSyncer_handleNewPeerState(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)

	mockBlockState := NewMockBlockState(ctrl)
	mockBlockState.EXPECT().GetHighestFinalisedHeader().Return(&types.Header{
		Number: 2,
	}, nil).Times(2)

	type fields struct {
		blockState    BlockState
		pendingBlocks DisjointBlockSet
		readyBlocks   *blockQueue
	}
	type args struct {
		ps *peerState
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   *worker
		err    error
	}{
		{
			name: "peer state number < final block number",
			fields: fields{
				blockState: mockBlockState,
			},
			args: args{ps: &peerState{number: 1}},
			want: nil,
		},
		{
			name: "base state",
			fields: fields{
				blockState: mockBlockState,
			},
			args: args{ps: &peerState{number: 3}},
			want: &worker{
				startNumber:  uintPtr(3),
				targetNumber: uintPtr(3),
				requestData:  bootstrapRequestData,
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s := &tipSyncer{
				blockState:    tt.fields.blockState,
				pendingBlocks: tt.fields.pendingBlocks,
				readyBlocks:   tt.fields.readyBlocks,
			}
			got, err := s.handleNewPeerState(tt.args.ps)
			if tt.err != nil {
				assert.EqualError(t, err, tt.err.Error())
			} else {
				assert.NoError(t, err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("handleNewPeerState() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_tipSyncer_handleTick(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)

	mockDisjointBlockSet := NewMockDisjointBlockSet(ctrl)
	mockDisjointBlockSet.EXPECT().size().Return(1).Times(2)
	pendingBlock2 := &pendingBlock{
		number: 3,
	}
	pendingBlock3 := &pendingBlock{
		hash:   common.Hash{},
		number: 4,
		header: &types.Header{
			Number: 4,
		},
	}
	pendingBlock4 := &pendingBlock{
		hash:   common.Hash{},
		number: 5,
		header: &types.Header{
			Number: 5,
		},
		body: &types.Body{},
	}
	mockDisjointBlockSet.EXPECT().getBlocks().Return([]*pendingBlock{
		{
			number: 2,
		},
		pendingBlock2,
		pendingBlock3,
		pendingBlock4,
	})
	mockDisjointBlockSet.EXPECT().removeBlock(common.Hash{})

	mockBlockState := NewMockBlockState(ctrl)
	mockBlockState.EXPECT().GetHighestFinalisedHeader().Return(&types.Header{
		Number: 2,
	}, nil)
	mockBlockState.EXPECT().HasHeader(common.Hash{}).Return(false, nil)

	type fields struct {
		blockState    BlockState
		pendingBlocks DisjointBlockSet
		readyBlocks   *blockQueue
	}
	tests := []struct {
		name   string
		fields fields
		want   []*worker
		err    error
	}{
		{
			name: "base case",
			fields: fields{
				pendingBlocks: mockDisjointBlockSet,
				blockState:    mockBlockState,
				readyBlocks:   newBlockQueue(3),
			},
			want: []*worker{
				{
					startNumber:  uintPtr(3),
					targetNumber: uintPtr(2),
					targetHash: common.Hash{5, 189, 204, 69, 79, 96, 160, 141, 66, 125, 5, 231, 241,
						159, 36, 15, 220, 57, 31, 87, 10, 183, 111, 203, 150, 236, 202, 11, 88, 35, 211, 191},
					pendingBlock: pendingBlock2,
					requestData:  bootstrapRequestData,
					direction:    network.Descending,
				},
				{
					startNumber:  uintPtr(4),
					targetNumber: uintPtr(4),
					pendingBlock: pendingBlock3,
					requestData:  network.RequestedDataBody + network.RequestedDataJustification,
				},
				{
					startNumber:  uintPtr(4),
					targetNumber: uintPtr(2),
					direction:    network.Descending,
					pendingBlock: pendingBlock4,
					requestData:  bootstrapRequestData,
				},
			},
			err: nil,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s := &tipSyncer{
				blockState:    tt.fields.blockState,
				pendingBlocks: tt.fields.pendingBlocks,
				readyBlocks:   tt.fields.readyBlocks,
			}
			got, err := s.handleTick()
			if tt.err != nil {
				assert.EqualError(t, err, tt.err.Error())
			} else {
				assert.NoError(t, err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("handleTick() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_tipSyncer_handleWorkerResult(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)

	mockBlockState := NewMockBlockState(ctrl)
	mockBlockState.EXPECT().GetHighestFinalisedHeader().Return(&types.Header{
		Number: 2,
	}, nil).Times(4)

	type fields struct {
		blockState BlockState
	}
	type args struct {
		res *worker
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   *worker
		err    error
	}{
		{
			name: "worker error is nil",
			args: args{res: &worker{}},
			want: nil,
			err:  nil,
		},
		{
			name: "worker error is error unknown parent",
			args: args{res: &worker{
				err: &workerError{
					err: errUnknownParent,
				},
			}},
			want: nil,
			err:  nil,
		},
		{
			name: "ascending, target number < finalised number",
			fields: fields{
				blockState: mockBlockState,
			},
			args: args{res: &worker{
				targetNumber: uintPtr(1),
				direction:    network.Ascending,
				err:          &workerError{},
			}},
			want: nil,
			err:  nil,
		},
		{
			name: "ascending, start number < finalised number",
			fields: fields{
				blockState: mockBlockState,
			},
			args: args{res: &worker{
				startNumber:  uintPtr(1),
				targetNumber: uintPtr(3),
				direction:    network.Ascending,
				err:          &workerError{},
			}},
			want: &worker{
				startNumber:  uintPtr(3),
				targetNumber: uintPtr(3),
			},
			err: nil,
		},
		{
			name: "descending, start number < finalised number",
			fields: fields{
				blockState: mockBlockState,
			},
			args: args{res: &worker{
				startNumber: uintPtr(1),
				direction:   network.Descending,
				err:         &workerError{},
			}},
			want: nil,
			err:  nil,
		},
		{
			name: "descending, target number < finalised number",
			fields: fields{
				blockState: mockBlockState,
			},
			args: args{res: &worker{
				startNumber:  uintPtr(3),
				targetNumber: uintPtr(1),
				direction:    network.Descending,
				err:          &workerError{},
			}},
			want: &worker{
				startNumber:  uintPtr(3),
				targetNumber: uintPtr(3),
				direction:    network.Descending,
			},
			err: nil,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s := &tipSyncer{
				blockState: tt.fields.blockState,
			}
			got, err := s.handleWorkerResult(tt.args.res)
			if tt.err != nil {
				assert.EqualError(t, err, tt.err.Error())
			} else {
				assert.NoError(t, err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("handleWorkerResult() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_tipSyncer_hasCurrentWorker(t *testing.T) {
	t.Parallel()

	testWorker1 := &worker{
		direction:    network.Ascending,
		targetNumber: uintPtr(3),
		startNumber:  uintPtr(3),
	}
	testWorker2 := &worker{
		direction:    network.Ascending,
		targetNumber: uintPtr(3),
		startNumber:  uintPtr(1),
	}
	testWorker3 := &worker{
		startNumber:  uintPtr(3),
		targetNumber: uintPtr(3),
		direction:    network.Descending,
	}
	testWorker4 := &worker{
		startNumber:  uintPtr(3),
		targetNumber: uintPtr(1),
		direction:    network.Descending,
	}

	type args struct {
		w       *worker
		workers map[uint64]*worker
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "worker nil",
			want: true,
		},
		{
			name: "ascending, false",
			args: args{
				w: &worker{
					direction:    network.Ascending,
					startNumber:  uintPtr(2),
					targetNumber: uintPtr(2),
				},
				workers: map[uint64]*worker{
					1: testWorker1,
				},
			},
			want: false,
		},
		{
			name: "ascending, true",
			args: args{
				w: &worker{
					direction:    network.Ascending,
					startNumber:  uintPtr(2),
					targetNumber: uintPtr(2),
				},
				workers: map[uint64]*worker{
					1: testWorker2,
				},
			},
			want: true,
		},
		{
			name: "descending, false",
			args: args{
				w: &worker{
					direction:    network.Descending,
					startNumber:  uintPtr(2),
					targetNumber: uintPtr(2),
				},
				workers: map[uint64]*worker{
					1: testWorker3,
				},
			},
			want: false,
		},
		{
			name: "descending, true",
			args: args{
				w: &worker{
					direction:    network.Descending,
					startNumber:  uintPtr(2),
					targetNumber: uintPtr(2),
				},
				workers: map[uint64]*worker{
					1: testWorker4,
				},
			},
			want: true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ti := &tipSyncer{}
			if got := ti.hasCurrentWorker(tt.args.w, tt.args.workers); got != tt.want {
				t.Errorf("hasCurrentWorker() = %v, want %v", got, tt.want)
			}
		})
	}
}
