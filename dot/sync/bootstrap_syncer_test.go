// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package sync

import (
	"errors"
	"github.com/ChainSafe/gossamer/dot/types"
	"testing"

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
		"nil worker.err returns nil": {
			blockStateBuilder: func(ctrl *gomock.Controller) BlockState {
				return NewMockBlockState(ctrl)
			},
			worker: &worker{},
		},
		"best block header error": {
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
		"targetNumber < bestBlockHeader number returns nil": {
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
		"targetNumber > bestBlockHeader number worker errUnknownParent, error GetHighestFinalisedHeader": {
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
		"targetNumber > bestBlockHeader number worker errUnknownParent returns worker": {
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
		"targetNumber > bestBlockHeader number returns worker": {
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
