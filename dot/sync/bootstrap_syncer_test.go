// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package sync

import (
	"testing"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func newMockBlockStateForBootstrapSyncerTest(ctrl *gomock.Controller) BlockState {
	mock := NewMockBlockState(ctrl)

	//BestBlockHeader() (*types.Header, error)
	mock.EXPECT().BestBlockHeader().Return(&types.Header{Number: 2}, nil)

	mock.EXPECT().GetHighestFinalisedHeader().Return(&types.Header{Number: 1}, nil).AnyTimes()

	// mock.EXPECT().GetHashByNumber(gomock.AssignableToTypeOf(uint(0))).DoAndReturn(func(
	// 	number uint) (common.Hash, error) {
	// 	return common.Hash{}, nil
	// }).AnyTimes()

	// mock.EXPECT().IsDescendantOf(gomock.AssignableToTypeOf(common.Hash{}),
	// 	gomock.AssignableToTypeOf(common.Hash{})).Return(true, nil).AnyTimes()

	// mock.EXPECT().GetHeader(gomock.AssignableToTypeOf(common.Hash{})).DoAndReturn(func(hash common.Hash) (*types.
	// 	Header, error) {
	// 	header := &types.Header{
	// 		Number: uint(hash[0]),
	// 	}
	// 	return header, nil
	// }).AnyTimes()

	return mock
}

func Test_bootstrapSyncer_handleWorkerResult(t *testing.T) {
	ctrl := gomock.NewController(t)

	tests := []struct {
		name              string
		blockState        BlockState
		worker            *worker
		wantWorkerToRetry *worker
		wantErr           bool
	}{
		{
			name:   "nil worker.err returns nil",
			worker: &worker{},
		},
		{
			name:       "targetNumber < bestBlockHeader number, returns nil",
			blockState: newMockBlockStateForBootstrapSyncerTest(ctrl),
			worker: &worker{
				err:          &workerError{},
				targetNumber: uintPtr(0),
			},
		},
		{
			name:       "targetNumber > bestBlockHeader number, worker errUnknownParent, returns worker",
			blockState: newMockBlockStateForBootstrapSyncerTest(ctrl),
			worker: &worker{
				err:          &workerError{err: errUnknownParent},
				targetNumber: uintPtr(3),
			},
			wantWorkerToRetry: &worker{
				startNumber:  uintPtr(1),
				targetNumber: uintPtr(3),
			},
		},
		{
			name:       "targetNumber > bestBlockHeader number, returns worker",
			blockState: newMockBlockStateForBootstrapSyncerTest(ctrl),
			worker: &worker{
				err:          &workerError{},
				targetNumber: uintPtr(3),
			},
			wantWorkerToRetry: &worker{
				startNumber:  uintPtr(3),
				targetNumber: uintPtr(3),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &bootstrapSyncer{
				blockState: tt.blockState,
			}
			gotWorkerToRetry, err := s.handleWorkerResult(tt.worker)
			if (err != nil) != tt.wantErr {
				t.Errorf("bootstrapSyncer.handleWorkerResult() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			assert.Equal(t, tt.wantWorkerToRetry, gotWorkerToRetry)
		})
	}
}
