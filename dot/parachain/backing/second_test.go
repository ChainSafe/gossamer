// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package backing

import (
	"testing"

	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/stretchr/testify/require"
)

func TestHandleSecondMessage(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		description      string
		cb               *CandidateBacking
		candidateReceipt parachaintypes.CandidateReceipt
		pvd              parachaintypes.PersistedValidationData
		err              error
	}{
		{
			description:      "wrong_persisted_validation_data_for_seconding_candidate",
			cb:               &CandidateBacking{},
			candidateReceipt: parachaintypes.CandidateReceipt{},
			pvd:              parachaintypes.PersistedValidationData{},
			err:              errWrongPVDForSecondingCandidate,
		},
		{
			description:      "unknown_relay_parent_for_seconding_candidate",
			cb:               &CandidateBacking{},
			candidateReceipt: dummyCandidateReceipt(t),
			pvd:              dummyPVD(t),
			err:              errUnknownRelayParentForSecondingCandidate,
		},
		{
			description: "parachain_outside_assignment_for_seconding",
			cb: &CandidateBacking{
				perRelayParent: map[common.Hash]*perRelayParentState{
					getDummyHash(t, 6): {
						Assignment: 10,
					},
				},
			},
			candidateReceipt: dummyCandidateReceipt(t),
			pvd:              dummyPVD(t),
			err:              errParaOutsideAssignmentForSeconding,
		},
		{
			description: "already_signed_valid_statement_for_candidate",
			cb: &CandidateBacking{
				perRelayParent: map[common.Hash]*perRelayParentState{
					getDummyHash(t, 6): {
						Assignment: 1,
						issuedStatements: map[parachaintypes.CandidateHash]bool{
							dummyCandidateHash(t): true,
						},
					},
				},
			},
			candidateReceipt: dummyCandidateReceipt(t),
			pvd:              dummyPVD(t),
			err:              errAlreadySignedValidStatement,
		},
		{
			description: "kick_off_background_validation_with_intent_to_second",
			cb: &CandidateBacking{
				perRelayParent: map[common.Hash]*perRelayParentState{
					getDummyHash(t, 6): {
						Assignment: 1,
						AwaitingValidation: map[parachaintypes.CandidateHash]bool{
							dummyCandidateHash(t): true,
						},
					},
				},
			},
			candidateReceipt: dummyCandidateReceipt(t),
			pvd:              dummyPVD(t),
			err:              nil,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.description, func(t *testing.T) {
			t.Parallel()

			err := tc.cb.handleSecondMessage(tc.candidateReceipt, tc.pvd, parachaintypes.PoV{}, nil)
			if err == nil {
				require.NoError(t, err)
			} else {
				require.ErrorIs(t, err, tc.err)
			}
		})
	}
}

func dummyPVD(t *testing.T) parachaintypes.PersistedValidationData {
	t.Helper()

	return parachaintypes.PersistedValidationData{
		ParentHead: parachaintypes.HeadData{
			Data: []byte{1, 2, 3},
		},
		RelayParentNumber:      5,
		RelayParentStorageRoot: getDummyHash(t, 5),
		MaxPovSize:             3,
	}
}

func dummyCandidateReceipt(t *testing.T) parachaintypes.CandidateReceipt {
	t.Helper()

	cr := getDummyCommittedCandidateReceipt(t).ToPlain()

	// blake2bhash of PVD in dummyPVD(t *testing.T) function
	cr.Descriptor.PersistedValidationDataHash =
		common.MustHexToHash("0x3544fbcdcb094751a5e044a30b994b2586ffc0b50e8b88c381461fe023a7242f")

	return cr
}

func dummyCandidateHash(t *testing.T) parachaintypes.CandidateHash {
	t.Helper()

	cr := dummyCandidateReceipt(t)
	hash, err := cr.Hash()
	require.NoError(t, err)

	return parachaintypes.CandidateHash{Value: hash}
}
