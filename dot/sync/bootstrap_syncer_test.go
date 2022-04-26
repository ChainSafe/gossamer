// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package sync

import (
	"testing"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func Test_bootstrapSyncer_handleWorkerResult(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)

	blockStateBuilderEmpty := func(ctrl *gomock.Controller) BlockState {
		return NewMockBlockState(ctrl)
	}

	blockStateBuilder := func(ctrl *gomock.Controller) BlockState {
		mockBlockState := NewMockBlockState(ctrl)
		mockBlockState.EXPECT().BestBlockHeader().Return(&types.Header{Number: 2}, nil)
		return mockBlockState
	}

	blockStateBuilderWithFinalised := func(ctrl *gomock.Controller) BlockState {
		mockBlockState := NewMockBlockState(ctrl)
		mockBlockState.EXPECT().BestBlockHeader().Return(&types.Header{Number: 2}, nil)
		mockBlockState.EXPECT().GetHighestFinalisedHeader().Return(&types.Header{Number: 1}, nil)
		return mockBlockState
	}

	tests := map[string]struct {
		blockStateBuilder func(ctrl *gomock.Controller) BlockState
		worker            *worker
		wantWorkerToRetry *worker
		err               error
	}{
		"nil worker.err returns nil": {
			blockStateBuilder: blockStateBuilderEmpty,
			worker:            &worker{},
		},
		"targetNumber < bestBlockHeader number returns nil": {
			blockStateBuilder: blockStateBuilder,
			worker: &worker{
				err:          &workerError{},
				targetNumber: uintPtr(0),
			},
		},
		"targetNumber > bestBlockHeader number worker errUnknownParent returns worker": {
			blockStateBuilder: blockStateBuilderWithFinalised,
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
			blockStateBuilder: blockStateBuilder,
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
			s := &bootstrapSyncer{
				blockState: tt.blockStateBuilder(ctrl),
			}
			gotWorkerToRetry, err := s.handleWorkerResult(tt.worker)
			assert.ErrorIs(t, err, tt.err)
			assert.Equal(t, tt.wantWorkerToRetry, gotWorkerToRetry)
		})
	}
}
