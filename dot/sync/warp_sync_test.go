// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package sync

import (
	"testing"

	"github.com/ChainSafe/gossamer/dot/network/messages"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestNextActions(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockBlockState := NewMockBlockState(ctrl)

	genesisHeader := &types.Header{
		Number: 1,
	}
	mockBlockState.EXPECT().GetHighestFinalisedHeader().Return(genesisHeader, nil).AnyTimes()

	tc := map[string]struct {
		phase                WarpSyncPhase
		lastBlock            *types.Header
		expectedRequestType  interface{}
		expectedResponseType interface{}
	}{
		"warp_sync_phase": {
			phase:                WarpProof,
			expectedRequestType:  &messages.WarpProofRequest{},
			expectedResponseType: &messages.WarpSyncProof{},
		},
		"target_block_phase": {
			phase:                TargetBlock,
			expectedRequestType:  &messages.BlockRequestMessage{},
			expectedResponseType: &messages.BlockResponseMessage{},
		},
	}

	for name, c := range tc {
		t.Run(name, func(t *testing.T) {
			strategy := NewWarpSyncStrategy(&WarpSyncConfig{
				BlockState: mockBlockState,
			})

			strategy.phase = c.phase

			tasks, err := strategy.NextActions()
			require.NoError(t, err)
			require.Equal(t, 1, len(tasks), "expected 1 task")

			task := tasks[0]
			require.IsType(t, c.expectedRequestType, task.request)
			require.IsType(t, c.expectedResponseType, task.response)
		})
	}
}
