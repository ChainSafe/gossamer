// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package backing

import (
	"errors"
	"testing"

	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"
)

func TestHandleGetBackedCandidatesMessage(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		description                string
		perRelayParent             func() map[common.Hash]*perRelayParentState
		expectedBackableCandidates int
	}{
		{
			description: "relay_parent_is_out_of_view",
			perRelayParent: func() map[common.Hash]*perRelayParentState {
				return map[common.Hash]*perRelayParentState{}
			},
			expectedBackableCandidates: 0,
		},
		{
			description: "relay_parent_state_is_nil",
			perRelayParent: func() map[common.Hash]*perRelayParentState {
				return map[common.Hash]*perRelayParentState{
					getDummyHash(t, 2): nil,
				}
			},
			expectedBackableCandidates: 0,
		},
		{
			description: "error_getting_attested_candidate",
			perRelayParent: func() map[common.Hash]*perRelayParentState {
				ctrl := gomock.NewController(t)
				mockTable := NewMockTable(ctrl)

				mockTable.EXPECT().attestedCandidate(
					gomock.AssignableToTypeOf(parachaintypes.CandidateHash{}),
					gomock.AssignableToTypeOf(new(tableContext)),
					gomock.AssignableToTypeOf(uint32(0)),
				).Return(nil, errors.New("could not get attested candidate from table"))

				return map[common.Hash]*perRelayParentState{
					getDummyHash(t, 2): {
						table: mockTable,
					},
				}
			},
			expectedBackableCandidates: 0,
		},
		{
			description: "get_1_backable_candidate",
			perRelayParent: func() map[common.Hash]*perRelayParentState {
				ctrl := gomock.NewController(t)
				mockTable := NewMockTable(ctrl)

				implicitAttestation := parachaintypes.ValidityAttestation{}
				err := implicitAttestation.SetValue(parachaintypes.Implicit([64]byte{1}))
				require.NoError(t, err)

				explicitAttestation := parachaintypes.ValidityAttestation{}
				err = explicitAttestation.SetValue(parachaintypes.Explicit([64]byte{2}))
				require.NoError(t, err)

				attested := &attestedCandidate{
					groupID:                   parachaintypes.GroupIndex(10),
					committedCandidateReceipt: getDummyCommittedCandidateReceipt(t),
					validityAttestations: []validatorIndexWithAttestation{
						{
							validatorIndex:      parachaintypes.ValidatorIndex(1),
							validityAttestation: implicitAttestation,
						},
						{
							validatorIndex:      parachaintypes.ValidatorIndex(2),
							validityAttestation: explicitAttestation,
						},
					},
				}

				mockTable.EXPECT().attestedCandidate(
					gomock.AssignableToTypeOf(parachaintypes.CandidateHash{}),
					gomock.AssignableToTypeOf(new(tableContext)),
					gomock.AssignableToTypeOf(uint32(0)),
				).Return(attested, nil)

				return map[common.Hash]*perRelayParentState{
					getDummyHash(t, 2): {
						table: mockTable,
						tableContext: tableContext{
							groups: map[parachaintypes.CoreIndex][]parachaintypes.ValidatorIndex{
								{Index: 10}: {1, 2, 3},
							},
						},
					},
					getDummyHash(t, 3): {},
				}
			},
			expectedBackableCandidates: 1,
		},
		{
			description: "get_2_backable_candidate",
			perRelayParent: func() map[common.Hash]*perRelayParentState {
				ctrl := gomock.NewController(t)
				mockTable := NewMockTable(ctrl)

				implicitAttestation := parachaintypes.ValidityAttestation{}
				err := implicitAttestation.SetValue(parachaintypes.Implicit([64]byte{1}))
				require.NoError(t, err)

				explicitAttestation := parachaintypes.ValidityAttestation{}
				err = explicitAttestation.SetValue(parachaintypes.Explicit([64]byte{2}))
				require.NoError(t, err)

				attested := &attestedCandidate{
					groupID:                   parachaintypes.GroupIndex(10),
					committedCandidateReceipt: getDummyCommittedCandidateReceipt(t),
					validityAttestations: []validatorIndexWithAttestation{
						{
							validatorIndex:      parachaintypes.ValidatorIndex(1),
							validityAttestation: implicitAttestation,
						},
						{
							validatorIndex:      parachaintypes.ValidatorIndex(2),
							validityAttestation: explicitAttestation,
						},
					},
				}

				mockTable.EXPECT().attestedCandidate(
					gomock.AssignableToTypeOf(parachaintypes.CandidateHash{}),
					gomock.AssignableToTypeOf(new(tableContext)),
					gomock.AssignableToTypeOf(uint32(0)),
				).Return(attested, nil).Times(2)

				return map[common.Hash]*perRelayParentState{
					getDummyHash(t, 2): {
						table: mockTable,
						tableContext: tableContext{
							groups: map[parachaintypes.CoreIndex][]parachaintypes.ValidatorIndex{
								{Index: 10}: {1, 2, 3},
							},
						},
					},
					getDummyHash(t, 1): {
						table: mockTable,
						tableContext: tableContext{
							groups: map[parachaintypes.CoreIndex][]parachaintypes.ValidatorIndex{
								{Index: 10}: {1, 2, 3},
							},
						},
					},
				}
			},
			expectedBackableCandidates: 2,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.description, func(t *testing.T) {
			t.Parallel()

			resCh := make(chan []*parachaintypes.BackedCandidate)
			defer close(resCh)

			requestedCandidates := GetBackableCandidatesMessage{
				Candidates: map[parachaintypes.ParaID][]*parachaintypes.CandidateHashAndRelayParent{
					1: {
						{
							CandidateHash:        parachaintypes.CandidateHash{Value: [32]byte{4}},
							CandidateRelayParent: getDummyHash(t, 2),
						},
						{
							CandidateHash:        parachaintypes.CandidateHash{Value: [32]byte{5}},
							CandidateRelayParent: getDummyHash(t, 1),
						},
						{
							CandidateHash:        parachaintypes.CandidateHash{Value: [32]byte{6}},
							CandidateRelayParent: getDummyHash(t, 3),
						},
					},
				},
			}

			cb := CandidateBacking{
				perRelayParent: tc.perRelayParent(),
			}

			paraToBackable := cb.handleGetBackableCandidatesMessage(requestedCandidates.Candidates)

			backableCandidates := paraToBackable[1]
			require.Len(t, backableCandidates, tc.expectedBackableCandidates)

		})
	}

}
