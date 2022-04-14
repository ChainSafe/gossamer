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

	mock.EXPECT().BestBlockHeader().Return(&types.Header{Number: 2}, nil)

	mock.EXPECT().GetHighestFinalisedHeader().Return(&types.Header{Number: 1}, nil).AnyTimes()

	return mock
}

func Test_bootstrapSyncer_handleWorkerResult(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)

	tests := map[string]struct {
		blockState        BlockState
		worker            *worker
		wantWorkerToRetry *worker
		wantErr           bool
	}{
		"nil worker.err returns nil": {
			worker: &worker{},
		},
		"targetNumber < bestBlockHeader number, returns nil": {
			blockState: newMockBlockStateForBootstrapSyncerTest(ctrl),
			worker: &worker{
				err:          &workerError{},
				targetNumber: uintPtr(0),
			},
		},
		"targetNumber > bestBlockHeader number, worker errUnknownParent, returns worker": {
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
		"targetNumber > bestBlockHeader number, returns worker": {
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
	for testName, tt := range tests {
		t.Run(testName, func(t *testing.T) {
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
