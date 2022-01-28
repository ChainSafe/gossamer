// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package sync

import (
	"math/big"
	"reflect"
	"testing"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func Test_bootstrapSyncer_handleNewPeerState(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	m := NewMockBlockState(ctrl)
	m.EXPECT().BestBlockHeader().Return(&types.Header{
		Number: big.NewInt(1),
	}, nil).AnyTimes()

	type fields struct {
		blockState BlockState
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
			name:   "peer state number less than header number",
			fields: fields{blockState: m},
			args: args{ps: &peerState{
				number: big.NewInt(0),
			}},
			want: nil,
		},
		{
			name:   "peer state number greater than header number",
			fields: fields{blockState: m},
			args: args{ps: &peerState{
				number: big.NewInt(2),
			}},
			want: &worker{
				startNumber:  big.NewInt(2),
				targetNumber: big.NewInt(2),
				requestData:  bootstrapRequestData,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			s := &bootstrapSyncer{
				blockState: tt.fields.blockState,
			}
			got, err := s.handleNewPeerState(tt.args.ps)
			if tt.err != nil {
				assert.EqualError(t, err, tt.err.Error())
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_bootstrapSyncer_handleTick(t *testing.T) {
	type fields struct {
		blockState BlockState
	}
	tests := []struct {
		name    string
		fields  fields
		want    []*worker
		wantErr bool
	}{
		{
			name:    "base case",
			fields:  fields{},
			want:    nil,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bo := &bootstrapSyncer{
				blockState: tt.fields.blockState,
			}
			got, err := bo.handleTick()
			if (err != nil) != tt.wantErr {
				t.Errorf("handleTick() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_bootstrapSyncer_handleWorkerResult(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	m := NewMockBlockState(ctrl)
	m.EXPECT().BestBlockHeader().Return(&types.Header{
		Number: big.NewInt(1),
	}, nil).Times(3)
	m.EXPECT().GetHighestFinalisedHeader().Return(&types.Header{Number: big.NewInt(0)}, nil)

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
			name:   "res nil error",
			fields: fields{},
			args:   args{res: &worker{}},
			want:   nil,
		},
		{
			name:   "targetNumber less than header number",
			fields: fields{blockState: m},
			args: args{res: &worker{
				targetNumber: big.NewInt(0),
				err:          &workerError{},
			}},
			want: nil,
		},
		{
			name:   "targetNumber greater than header number",
			fields: fields{blockState: m},
			args: args{res: &worker{
				targetNumber: big.NewInt(2),
				err:          &workerError{},
			}},
			want: &worker{
				startNumber:  big.NewInt(2),
				targetNumber: big.NewInt(2),
			},
		},
		{
			name:   "error unknown parent",
			fields: fields{blockState: m},
			args: args{res: &worker{
				targetNumber: big.NewInt(2),
				err: &workerError{
					err: errUnknownParent,
				},
			}},
			want: &worker{
				startNumber:  big.NewInt(0),
				targetNumber: big.NewInt(2),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &bootstrapSyncer{
				blockState: tt.fields.blockState,
			}
			got, err := s.handleWorkerResult(tt.args.res)
			if tt.err != nil {
				assert.EqualError(t, err, tt.err.Error())
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.want, got)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("handleWorkerResult() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_bootstrapSyncer_hasCurrentWorker(t *testing.T) {
	type args struct {
		workers map[uint64]*worker
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "expect false",
			want: false,
		},
		{
			name: "expect true",
			args: args{
				workers: map[uint64]*worker{
					0: &worker{},
				},
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bo := &bootstrapSyncer{}
			if got := bo.hasCurrentWorker(nil, tt.args.workers); got != tt.want {
				t.Errorf("hasCurrentWorker() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_newBootstrapSyncer(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	m := NewMockBlockState(ctrl)

	type args struct {
		blockState BlockState
	}
	tests := []struct {
		name string
		args args
		want *bootstrapSyncer
	}{
		{
			name: "base case",
			want: &bootstrapSyncer{},
		},
		{
			name: "with block state",
			args: args{
				blockState: m,
			},
			want: &bootstrapSyncer{blockState: m},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := newBootstrapSyncer(tt.args.blockState)
			assert.Equal(t, tt.want, got)
		})
	}
}
