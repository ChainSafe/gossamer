// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package statementdistribution

import (
	"testing"

	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newBitVec(t *testing.T, values ...bool) parachaintypes.BitVec {
	bv, err := parachaintypes.NewBitVec(values)
	require.NoError(t, err)
	return bv
}

func TestReceivedManifests_candidateStatementFilter(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		setupReceived map[parachaintypes.CandidateHash]manifestSummary
		candidateHash parachaintypes.CandidateHash
		wantFilter    *statementFilter
	}{
		{
			name:          "nil_received",
			setupReceived: nil,
			candidateHash: parachaintypes.CandidateHash{Value: common.Hash{1, 2, 3}},
			wantFilter:    nil,
		},
		{
			name:          "candidate_not_in_received",
			setupReceived: map[parachaintypes.CandidateHash]manifestSummary{},
			candidateHash: parachaintypes.CandidateHash{Value: common.Hash{1, 2, 3}},
			wantFilter:    nil,
		},
		{
			name: "candidate_exists",
			setupReceived: map[parachaintypes.CandidateHash]manifestSummary{
				{Value: common.Hash{1, 2, 3}}: {
					statementKnowledge: statementFilter{
						secondedInGroup:  newBitVec(t, true, false, true, false),
						validatedInGroup: newBitVec(t, false, true, false, true),
					},
				},
			},
			candidateHash: parachaintypes.CandidateHash{Value: common.Hash{1, 2, 3}},
			wantFilter: &statementFilter{
				secondedInGroup:  newBitVec(t, true, false, true, false),
				validatedInGroup: newBitVec(t, false, true, false, true),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			rm := &receivedManifests{
				received: tt.setupReceived,
			}
			got := rm.candidateStatementFilter(tt.candidateHash)

			if tt.wantFilter == nil {
				assert.Nil(t, got)
			} else {
				assert.NotNil(t, got)
				assert.Equal(t, tt.wantFilter, got)
			}
		})
	}
}

func TestReceivedManifests_importReceived(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                string
		setupReceived       map[parachaintypes.CandidateHash]manifestSummary
		setupSecondedCounts map[parachaintypes.GroupIndex][]uint
		groupSize           uint
		secondingLimit      uint
		candidateHash       parachaintypes.CandidateHash
		summary             manifestSummary
		wantErr             error
		wantReceived        map[parachaintypes.CandidateHash]manifestSummary
		wantSecondedCounts  map[parachaintypes.GroupIndex][]uint
	}{
		{
			name:                "new_candidate_within_limits",
			setupReceived:       map[parachaintypes.CandidateHash]manifestSummary{},
			setupSecondedCounts: map[parachaintypes.GroupIndex][]uint{},
			groupSize:           3,
			secondingLimit:      2,
			candidateHash:       parachaintypes.CandidateHash{Value: common.Hash{1, 2, 3}},
			summary: manifestSummary{
				claimedParentHash: common.Hash{4, 5, 6},
				claimedGroupIndex: 1,
				statementKnowledge: statementFilter{
					secondedInGroup:  newBitVec(t, true, false, false),
					validatedInGroup: newBitVec(t, false, true, false),
				},
			},
			wantErr: nil,
			wantReceived: map[parachaintypes.CandidateHash]manifestSummary{
				{Value: common.Hash{1, 2, 3}}: {
					claimedParentHash: common.Hash{4, 5, 6},
					claimedGroupIndex: 1,
					statementKnowledge: statementFilter{
						secondedInGroup:  newBitVec(t, true, false, false),
						validatedInGroup: newBitVec(t, false, true, false),
					},
				},
			},
			wantSecondedCounts: map[parachaintypes.GroupIndex][]uint{
				1: {1, 0, 0},
			},
		},
		{
			name:          "new_candidate_exceeds_seconding_limit",
			setupReceived: map[parachaintypes.CandidateHash]manifestSummary{},
			setupSecondedCounts: map[parachaintypes.GroupIndex][]uint{
				1: {2, 0, 0}, // validator 0 already has 2 seconded statements
			},
			groupSize:      3,
			secondingLimit: 2,
			candidateHash:  parachaintypes.CandidateHash{Value: common.Hash{1, 2, 3}},
			summary: manifestSummary{
				claimedParentHash: common.Hash{4, 5, 6},
				claimedGroupIndex: 1,
				statementKnowledge: statementFilter{
					secondedInGroup:  newBitVec(t, true, false, false),
					validatedInGroup: newBitVec(t, false, true, false),
				},
			},
			wantErr:      errManifestImportOverflow,
			wantReceived: map[parachaintypes.CandidateHash]manifestSummary{},
			wantSecondedCounts: map[parachaintypes.GroupIndex][]uint{
				1: {2, 0, 0}, // unchanged
			},
		},
		{
			name: "existing_candidate_with_conflicting_group_index",
			setupReceived: map[parachaintypes.CandidateHash]manifestSummary{
				{Value: common.Hash{1, 2, 3}}: {
					claimedParentHash: common.Hash{4, 5, 6},
					claimedGroupIndex: 1,
					statementKnowledge: statementFilter{
						secondedInGroup:  newBitVec(t, true, false, false),
						validatedInGroup: newBitVec(t, false, true, false),
					},
				},
			},
			setupSecondedCounts: map[parachaintypes.GroupIndex][]uint{
				1: {1, 0, 0},
			},
			groupSize:      3,
			secondingLimit: 2,
			candidateHash:  parachaintypes.CandidateHash{Value: common.Hash{1, 2, 3}},
			summary: manifestSummary{
				claimedParentHash: common.Hash{4, 5, 6},
				claimedGroupIndex: 2, // Different group index
				statementKnowledge: statementFilter{
					secondedInGroup:  newBitVec(t, true, false, false),
					validatedInGroup: newBitVec(t, false, true, false),
				},
			},
			wantErr: errManifestImportConflicting,
			wantReceived: map[parachaintypes.CandidateHash]manifestSummary{
				{Value: common.Hash{1, 2, 3}}: {
					claimedParentHash: common.Hash{4, 5, 6},
					claimedGroupIndex: 1,
					statementKnowledge: statementFilter{
						secondedInGroup:  newBitVec(t, true, false, false),
						validatedInGroup: newBitVec(t, false, true, false),
					},
				},
			},
			wantSecondedCounts: map[parachaintypes.GroupIndex][]uint{
				1: {1, 0, 0}, // unchanged
			},
		},
		{
			name: "existing_candidate_with_non_superset_seconded_statements",
			setupReceived: map[parachaintypes.CandidateHash]manifestSummary{
				{Value: common.Hash{1, 2, 3}}: {
					claimedParentHash: common.Hash{4, 5, 6},
					claimedGroupIndex: 1,
					statementKnowledge: statementFilter{
						secondedInGroup:  newBitVec(t, true, false, false),
						validatedInGroup: newBitVec(t, false, true, false),
					},
				},
			},
			setupSecondedCounts: map[parachaintypes.GroupIndex][]uint{
				1: {1, 0, 0},
			},
			groupSize:      3,
			secondingLimit: 2,
			candidateHash:  parachaintypes.CandidateHash{Value: common.Hash{1, 2, 3}},
			summary: manifestSummary{
				claimedParentHash: common.Hash{4, 5, 6},
				claimedGroupIndex: 1,
				statementKnowledge: statementFilter{
					secondedInGroup:  newBitVec(t, false, true, false), // Not a superset
					validatedInGroup: newBitVec(t, false, true, false),
				},
			},
			wantErr: errManifestImportConflicting,
			wantReceived: map[parachaintypes.CandidateHash]manifestSummary{
				{Value: common.Hash{1, 2, 3}}: {
					claimedParentHash: common.Hash{4, 5, 6},
					claimedGroupIndex: 1,
					statementKnowledge: statementFilter{
						secondedInGroup:  newBitVec(t, true, false, false),
						validatedInGroup: newBitVec(t, false, true, false),
					},
				},
			},
			wantSecondedCounts: map[parachaintypes.GroupIndex][]uint{
				1: {1, 0, 0}, // unchanged
			},
		},
		{
			name: "existing_candidate_with_non_superset_validated_statements",
			setupReceived: map[parachaintypes.CandidateHash]manifestSummary{
				{Value: common.Hash{1, 2, 3}}: {
					claimedParentHash: common.Hash{4, 5, 6},
					claimedGroupIndex: 1,
					statementKnowledge: statementFilter{
						secondedInGroup:  newBitVec(t, true, false, false),
						validatedInGroup: newBitVec(t, false, true, false),
					},
				},
			},
			setupSecondedCounts: map[parachaintypes.GroupIndex][]uint{
				1: {1, 0, 0},
			},
			groupSize:      3,
			secondingLimit: 2,
			candidateHash:  parachaintypes.CandidateHash{Value: common.Hash{1, 2, 3}},
			summary: manifestSummary{
				claimedParentHash: common.Hash{4, 5, 6},
				claimedGroupIndex: 1,
				statementKnowledge: statementFilter{
					secondedInGroup:  newBitVec(t, true, false, false),
					validatedInGroup: newBitVec(t, false, false, true), // Not a superset
				},
			},
			wantErr: errManifestImportConflicting,
			wantReceived: map[parachaintypes.CandidateHash]manifestSummary{
				{Value: common.Hash{1, 2, 3}}: {
					claimedParentHash: common.Hash{4, 5, 6},
					claimedGroupIndex: 1,
					statementKnowledge: statementFilter{
						secondedInGroup:  newBitVec(t, true, false, false),
						validatedInGroup: newBitVec(t, false, true, false),
					},
				},
			},
			wantSecondedCounts: map[parachaintypes.GroupIndex][]uint{
				1: {1, 0, 0}, // unchanged
			},
		},
		{
			name: "successful_import",
			setupReceived: map[parachaintypes.CandidateHash]manifestSummary{
				{Value: common.Hash{1, 2, 3}}: {
					claimedParentHash: common.Hash{4, 5, 6},
					claimedGroupIndex: 1,
					statementKnowledge: statementFilter{
						secondedInGroup:  newBitVec(t, true, false, false),
						validatedInGroup: newBitVec(t, false, true, false),
					},
				},
			},
			setupSecondedCounts: map[parachaintypes.GroupIndex][]uint{
				1: {1, 0, 0},
			},
			groupSize:      3,
			secondingLimit: 2,
			candidateHash:  parachaintypes.CandidateHash{Value: common.Hash{1, 2, 3}},
			summary: manifestSummary{
				claimedParentHash: common.Hash{4, 5, 6},
				claimedGroupIndex: 1,
				statementKnowledge: statementFilter{
					secondedInGroup:  newBitVec(t, true, false, false),
					validatedInGroup: newBitVec(t, false, true, false),
				},
			},
			wantErr: nil,
			wantReceived: map[parachaintypes.CandidateHash]manifestSummary{
				{Value: common.Hash{1, 2, 3}}: {
					claimedParentHash: common.Hash{4, 5, 6},
					claimedGroupIndex: 1,
					statementKnowledge: statementFilter{
						secondedInGroup:  newBitVec(t, true, false, false),
						validatedInGroup: newBitVec(t, false, true, false),
					},
				},
			},
			wantSecondedCounts: map[parachaintypes.GroupIndex][]uint{
				1: {2, 0, 0},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			rm := &receivedManifests{
				received:       make(map[parachaintypes.CandidateHash]manifestSummary),
				secondedCounts: make(map[parachaintypes.GroupIndex][]uint),
			}

			for k, v := range tt.setupReceived {
				rm.received[k] = v
			}
			for k, v := range tt.setupSecondedCounts {
				rm.secondedCounts[k] = make([]uint, len(v))
				copy(rm.secondedCounts[k], v)
			}

			err := rm.importReceived(tt.groupSize, tt.secondingLimit, tt.candidateHash, tt.summary)

			assert.Equal(t, tt.wantErr, err)
			assert.Equal(t, tt.wantReceived, rm.received)
			assert.Equal(t, tt.wantSecondedCounts, rm.secondedCounts)
		})
	}
}

func TestReceivedManifests_updatingEnsureWithinSecondingLimit(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		secondedCounts map[parachaintypes.GroupIndex][]uint
		groupIndex     parachaintypes.GroupIndex
		groupSize      uint
		secondingLimit uint
		newSeconded    parachaintypes.BitVec
		want           bool
		wantCounts     []uint
	}{
		{
			name:           "zero_seconding_limit_always_returns_false",
			secondedCounts: map[parachaintypes.GroupIndex][]uint{},
			groupIndex:     1,
			groupSize:      3,
			secondingLimit: 0,
			newSeconded:    newBitVec(t, true, false, false),
			want:           false,
			wantCounts:     nil,
		},
		{
			name:           "new_group_with_counts_within_limit",
			secondedCounts: map[parachaintypes.GroupIndex][]uint{},
			groupIndex:     1,
			groupSize:      3,
			secondingLimit: 2,
			newSeconded:    newBitVec(t, true, false, false),
			want:           true,
			wantCounts:     []uint{1, 0, 0},
		},
		{
			name: "existing_counts_within_limit",
			secondedCounts: map[parachaintypes.GroupIndex][]uint{
				1: {1, 1, 0},
			},
			groupIndex:     1,
			groupSize:      3,
			secondingLimit: 2,
			newSeconded:    newBitVec(t, true, true, false),
			want:           true,
			wantCounts:     []uint{2, 2, 0},
		},
		{
			name: "existing_counts_would_exceed_limit",
			secondedCounts: map[parachaintypes.GroupIndex][]uint{
				1: {1, 0, 2},
			},
			groupIndex:     1,
			groupSize:      3,
			secondingLimit: 2,
			newSeconded:    newBitVec(t, true, false, true),
			want:           false,
			wantCounts:     []uint{1, 0, 2}, // should not be modified
		},
		{
			name: "one_validator_would_exceed_another_would_not",
			secondedCounts: map[parachaintypes.GroupIndex][]uint{
				1: {2, 1, 0},
			},
			groupIndex:     1,
			groupSize:      3,
			secondingLimit: 2,
			newSeconded:    newBitVec(t, true, true, false),
			want:           false,
			wantCounts:     []uint{2, 1, 0}, // should not be modified
		},
		{
			name:           "bit_vector_longer_than_group_size",
			secondedCounts: map[parachaintypes.GroupIndex][]uint{},
			groupIndex:     1,
			groupSize:      3,
			secondingLimit: 2,
			newSeconded:    newBitVec(t, true, false, false, true),
			want:           true,
			wantCounts:     []uint{1, 0, 0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := updatingEnsureWithinSecondingLimit(
				tt.secondedCounts,
				tt.groupIndex,
				tt.groupSize,
				tt.secondingLimit,
				tt.newSeconded,
			)

			assert.Equal(t, tt.want, got)
			assert.Equal(t, tt.wantCounts, tt.secondedCounts[tt.groupIndex])
		})
	}
}
