// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package backing

import (
	"errors"
	"testing"

	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/lib/common"
	gomock "github.com/golang/mock/gomock"
)

func TestHandleGetBackedCandidatesMessage(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		description    string
		perRelayParent func() map[common.Hash]*perRelayParentState
	}{
		{
			description: "relay_parent_is_out_of_view",
			perRelayParent: func() map[common.Hash]*perRelayParentState {
				return map[common.Hash]*perRelayParentState{}
			},
		},
		{
			description: "relay_parent_state_is_nil",
			perRelayParent: func() map[common.Hash]*perRelayParentState {
				return map[common.Hash]*perRelayParentState{
					getDummyHash(t, 2): nil,
				}
			},
		},
		{
			description: "error_getting_attested_candidate",
			perRelayParent: func() map[common.Hash]*perRelayParentState {
				ctrl := gomock.NewController(t)
				mockTable := NewMockTable(ctrl)

				mockTable.EXPECT().attestedCandidate(
					gomock.AssignableToTypeOf(parachaintypes.CandidateHash{}),
					gomock.AssignableToTypeOf(new(TableContext)),
				).Return(nil, errors.New("could not get attested candidate from table"))

				return map[common.Hash]*perRelayParentState{
					getDummyHash(t, 2): {
						table: mockTable,
					},
				}
			},
		},
		{
			description: "attested_candidate_is_nil",
			perRelayParent: func() map[common.Hash]*perRelayParentState {
				ctrl := gomock.NewController(t)
				mockTable := NewMockTable(ctrl)

				mockTable.EXPECT().attestedCandidate(
					gomock.AssignableToTypeOf(parachaintypes.CandidateHash{}),
					gomock.AssignableToTypeOf(new(TableContext)),
				).Return(nil, nil)

				return map[common.Hash]*perRelayParentState{
					getDummyHash(t, 2): {
						table: mockTable,
					},
				}
			},
		},
		{
			description: "attested_candidate_is_not_nil",
			perRelayParent: func() map[common.Hash]*perRelayParentState {
				ctrl := gomock.NewController(t)
				mockTable := NewMockTable(ctrl)

				mockTable.EXPECT().attestedCandidate(
					gomock.AssignableToTypeOf(parachaintypes.CandidateHash{}),
					gomock.AssignableToTypeOf(new(TableContext)),
				).Return(new(AttestedCandidate), nil)

				return map[common.Hash]*perRelayParentState{
					getDummyHash(t, 2): {
						table: mockTable,
					},
				}
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.description, func(t *testing.T) {
			t.Parallel()

			resCh := make(chan []*parachaintypes.BackedCandidate)
			defer close(resCh)

			requestedCandidates := GetBackedCandidatesMessage{
				Candidates: []*CandidateHashAndRelayParent{
					{
						CandidateHash:        dummyCandidateHash(t),
						CandidateRelayParent: getDummyHash(t, 2),
					},
				},
				ResCh: resCh,
			}

			go func(resCh chan []*parachaintypes.BackedCandidate) {
				<-resCh
			}(resCh)

			cb := CandidateBacking{
				perRelayParent: tc.perRelayParent(),
			}

			cb.handleGetBackedCandidatesMessage(requestedCandidates)
		})
	}

}
