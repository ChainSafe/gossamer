// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package sync

import (
	"reflect"
	"testing"
)

func Test_newTipSyncer(t *testing.T) {
	type args struct {
		blockState    BlockState
		pendingBlocks DisjointBlockSet
		readyBlocks   *blockQueue
	}
	tests := []struct {
		name string
		args args
		want *tipSyncer
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := newTipSyncer(tt.args.blockState, tt.args.pendingBlocks, tt.args.readyBlocks); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("newTipSyncer() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_tipSyncer_handleNewPeerState(t *testing.T) {
	type fields struct {
		blockState    BlockState
		pendingBlocks DisjointBlockSet
		readyBlocks   *blockQueue
	}
	type args struct {
		ps *peerState
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
			s := &tipSyncer{
				blockState:    tt.fields.blockState,
				pendingBlocks: tt.fields.pendingBlocks,
				readyBlocks:   tt.fields.readyBlocks,
			}
			got, err := s.handleNewPeerState(tt.args.ps)
			if (err != nil) != tt.wantErr {
				t.Errorf("handleNewPeerState() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("handleNewPeerState() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_tipSyncer_handleTick(t *testing.T) {
	type fields struct {
		blockState    BlockState
		pendingBlocks DisjointBlockSet
		readyBlocks   *blockQueue
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
			s := &tipSyncer{
				blockState:    tt.fields.blockState,
				pendingBlocks: tt.fields.pendingBlocks,
				readyBlocks:   tt.fields.readyBlocks,
			}
			got, err := s.handleTick()
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

func Test_tipSyncer_handleWorkerResult(t *testing.T) {
	type fields struct {
		blockState    BlockState
		pendingBlocks DisjointBlockSet
		readyBlocks   *blockQueue
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
			s := &tipSyncer{
				blockState:    tt.fields.blockState,
				pendingBlocks: tt.fields.pendingBlocks,
				readyBlocks:   tt.fields.readyBlocks,
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

func Test_tipSyncer_hasCurrentWorker(t *testing.T) {
	type fields struct {
		blockState    BlockState
		pendingBlocks DisjointBlockSet
		readyBlocks   *blockQueue
	}
	type args struct {
		w       *worker
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
			ti := &tipSyncer{
				blockState:    tt.fields.blockState,
				pendingBlocks: tt.fields.pendingBlocks,
				readyBlocks:   tt.fields.readyBlocks,
			}
			if got := ti.hasCurrentWorker(tt.args.w, tt.args.workers); got != tt.want {
				t.Errorf("hasCurrentWorker() = %v, want %v", got, tt.want)
			}
		})
	}
}