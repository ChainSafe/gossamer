// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package sync

import (
	"errors"
	"testing"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func Test_bootstrapSyncer_handleWorkerResult(t *testing.T) {
	t.Parallel()
	mockError := errors.New("mock testing error")

	tests := map[string]struct {
		blockStateBuilder func(ctrl *gomock.Controller) BlockState
		worker            *worker
		wantWorkerToRetry *worker
		err               error
	}{
		"nil_worker.err_returns_nil": {
			blockStateBuilder: func(ctrl *gomock.Controller) BlockState {
				return NewMockBlockState(ctrl)
			},
			worker: &worker{},
		},
		"best_block_header_error": {
			blockStateBuilder: func(ctrl *gomock.Controller) BlockState {
				mockBlockState := NewMockBlockState(ctrl)
				mockBlockState.EXPECT().BestBlockHeader().Return(nil,
					mockError)
				return mockBlockState
			},
			worker: &worker{
				err:          &workerError{},
				targetNumber: uintPtr(0),
			},
			err: mockError,
		},
		"targetNumber_<_bestBlockHeader_number_returns_nil": {
			blockStateBuilder: func(ctrl *gomock.Controller) BlockState {
				mockBlockState := NewMockBlockState(ctrl)
				mockBlockState.EXPECT().BestBlockHeader().Return(&types.Header{Number: 2}, nil)
				return mockBlockState
			},
			worker: &worker{
				err:          &workerError{},
				targetNumber: uintPtr(0),
			},
		},
		"targetNumber_>_bestBlockHeader_number_worker_errUnknownParent,_error_GetHighestFinalisedHeader": {
			blockStateBuilder: func(ctrl *gomock.Controller) BlockState {
				mockBlockState := NewMockBlockState(ctrl)
				mockBlockState.EXPECT().BestBlockHeader().Return(&types.Header{Number: 2}, nil)
				mockBlockState.EXPECT().GetHighestFinalisedHeader().Return(nil, mockError)
				return mockBlockState
			},
			worker: &worker{
				err:          &workerError{err: errUnknownParent},
				targetNumber: uintPtr(3),
			},
			err: mockError,
		},
		"targetNumber_>_bestBlockHeader_number_worker_errUnknownParent_returns_worker": {
			blockStateBuilder: func(ctrl *gomock.Controller) BlockState {
				mockBlockState := NewMockBlockState(ctrl)
				mockBlockState.EXPECT().BestBlockHeader().Return(&types.Header{Number: 2}, nil)
				mockBlockState.EXPECT().GetHighestFinalisedHeader().Return(&types.Header{Number: 1}, nil)
				return mockBlockState
			},
			worker: &worker{
				err:          &workerError{err: errUnknownParent},
				targetNumber: uintPtr(3),
			},
			wantWorkerToRetry: &worker{
				startNumber:  uintPtr(1),
				targetNumber: uintPtr(3),
			},
		},
		"targetNumber_>_bestBlockHeader_number_returns_worker": {
			blockStateBuilder: func(ctrl *gomock.Controller) BlockState {
				mockBlockState := NewMockBlockState(ctrl)
				mockBlockState.EXPECT().BestBlockHeader().Return(&types.Header{Number: 2}, nil)
				return mockBlockState
			},
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
	for testName, tt := range tests {
		tt := tt
		t.Run(testName, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			s := &bootstrapSyncer{
				blockState: tt.blockStateBuilder(ctrl),
			}
			gotWorkerToRetry, err := s.handleWorkerResult(tt.worker)
			assert.ErrorIs(t, err, tt.err)
			assert.Equal(t, tt.wantWorkerToRetry, gotWorkerToRetry)
		})
	}
}
