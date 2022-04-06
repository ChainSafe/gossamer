package sync

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_newWorkerState(t *testing.T) {
	tests := []struct {
		name string
		want *workerState
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, newWorkerState(), "newWorkerState()")
		})
	}
}

func Test_uintPtr(t *testing.T) {
	type args struct {
		n uint
	}
	tests := []struct {
		name string
		args args
		want *uint
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, uintPtr(tt.args.n), "uintPtr(%v)", tt.args.n)
		})
	}
}

func Test_workerState_add(t *testing.T) {
	type fields struct {
		ctx        context.Context
		cancel     context.CancelFunc
		Mutex      sync.Mutex
		nextWorker uint64
		workers    map[uint64]*worker
	}
	type args struct {
		w *worker
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &workerState{
				ctx:        tt.fields.ctx,
				cancel:     tt.fields.cancel,
				Mutex:      tt.fields.Mutex,
				nextWorker: tt.fields.nextWorker,
				workers:    tt.fields.workers,
			}
			s.add(tt.args.w)
		})
	}
}

func Test_workerState_delete(t *testing.T) {
	type fields struct {
		ctx        context.Context
		cancel     context.CancelFunc
		Mutex      sync.Mutex
		nextWorker uint64
		workers    map[uint64]*worker
	}
	type args struct {
		id uint64
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &workerState{
				ctx:        tt.fields.ctx,
				cancel:     tt.fields.cancel,
				Mutex:      tt.fields.Mutex,
				nextWorker: tt.fields.nextWorker,
				workers:    tt.fields.workers,
			}
			s.delete(tt.args.id)
		})
	}
}

func Test_workerState_reset(t *testing.T) {
	type fields struct {
		ctx        context.Context
		cancel     context.CancelFunc
		Mutex      sync.Mutex
		nextWorker uint64
		workers    map[uint64]*worker
	}
	tests := []struct {
		name   string
		fields fields
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &workerState{
				ctx:        tt.fields.ctx,
				cancel:     tt.fields.cancel,
				Mutex:      tt.fields.Mutex,
				nextWorker: tt.fields.nextWorker,
				workers:    tt.fields.workers,
			}
			s.reset()
		})
	}
}
