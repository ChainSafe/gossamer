// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package sync

import (
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"math/big"
	"reflect"
	"testing"
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
			assert.Equal(t, got, tt.want)
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
		// TODO: Add test cases.
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
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("handleTick() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_bootstrapSyncer_handleWorkerResult(t *testing.T) {
	type fields struct {
		blockState BlockState
	}
	type args struct {
		res *worker
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *worker
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &bootstrapSyncer{
				blockState: tt.fields.blockState,
			}
			got, err := s.handleWorkerResult(tt.args.res)
			if (err != nil) != tt.wantErr {
				t.Errorf("handleWorkerResult() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("handleWorkerResult() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_bootstrapSyncer_hasCurrentWorker(t *testing.T) {
	type fields struct {
		blockState BlockState
	}
	type args struct {
		in0     *worker
		workers map[uint64]*worker
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bo := &bootstrapSyncer{
				blockState: tt.fields.blockState,
			}
			if got := bo.hasCurrentWorker(tt.args.in0, tt.args.workers); got != tt.want {
				t.Errorf("hasCurrentWorker() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_newBootstrapSyncer(t *testing.T) {
	type args struct {
		blockState BlockState
	}
	tests := []struct {
		name string
		args args
		want *bootstrapSyncer
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := newBootstrapSyncer(tt.args.blockState); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("newBootstrapSyncer() = %v, want %v", got, tt.want)
			}
		})
	}
}
