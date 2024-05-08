package backing

import (
	"testing"

	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/stretchr/testify/require"
)

func TestCandidateData_attested(t *testing.T) {
	committedCandidateReceipt := getDummyCommittedCandidateReceipt(t)
	validityThreshold := uint(2)
	data := candidateData{
		groupID:       1,
		candidate:     committedCandidateReceipt,
		validityVotes: map[parachaintypes.ValidatorIndex]validityVoteWithSign{},
	}

	attestedCandidate, err := data.attested(validityThreshold)
	require.ErrorIs(t, err, errNotEnoughValidityVotes)
	require.Nil(t, attestedCandidate)

	data.validityVotes = map[parachaintypes.ValidatorIndex]validityVoteWithSign{
		1: {validityVote: issued, signature: parachaintypes.ValidatorSignature{1}},
		2: {validityVote: valid, signature: parachaintypes.ValidatorSignature{1, 2}},
		3: {validityVote: valid, signature: parachaintypes.ValidatorSignature{1, 2, 3}},
	}

	expectedAttestedCandidate := &AttestedCandidate{
		GroupID:   1,
		Candidate: committedCandidateReceipt,
		ValidityAttestations: func() []validityAttestation {
			var attestations []validityAttestation

			// validity vote: issued
			vote1 := parachaintypes.NewValidityAttestation()
			err := vote1.Set(parachaintypes.Implicit(
				parachaintypes.ValidatorSignature{1},
			))
			require.NoError(t, err)
			attest1 := validityAttestation{
				ValidatorIndex:      1,
				ValidityAttestation: vote1,
			}

			// validity vote: valid
			vote2 := parachaintypes.NewValidityAttestation()
			err = vote2.Set(parachaintypes.Explicit(
				parachaintypes.ValidatorSignature{1, 2},
			))
			require.NoError(t, err)
			attest2 := validityAttestation{
				ValidatorIndex:      2,
				ValidityAttestation: vote2,
			}

			// validity vote: valid
			vote3 := parachaintypes.NewValidityAttestation()
			err = vote3.Set(parachaintypes.Explicit(
				parachaintypes.ValidatorSignature{1, 2, 3},
			))
			require.NoError(t, err)
			attest3 := validityAttestation{
				ValidatorIndex:      3,
				ValidityAttestation: vote3,
			}

			return append(attestations, attest1, attest2, attest3)
		}(),
	}

	attestedCandidate, err = data.attested(validityThreshold)
	require.NoError(t, err)
	require.Equal(t, expectedAttestedCandidate, attestedCandidate)
}

func TestStatementTable_attestedCandidate(t *testing.T) {
	t.Parallel()

	type args struct {
		candidateHash       parachaintypes.CandidateHash
		tableContext        *TableContext
		minimumBackingVotes uint32
	}
	tests := []struct {
		name    string
		table   *statementTable
		args    args
		want    *AttestedCandidate
		wantErr error
	}{
		{
			name: "candidate_votes_not_available_for_given_candidate_hash",
			table: &statementTable{
				candidateVotes: map[parachaintypes.CandidateHash]candidateData{},
			},
			args: args{
				candidateHash: dummyCandidateHash(t),
			},
			wantErr: errCandidateDataNotFound,
		},
		{
			name: "not_enough_validity_votes",
			table: &statementTable{
				candidateVotes: map[parachaintypes.CandidateHash]candidateData{
					dummyCandidateHash(t): {
						groupID:       1,
						validityVotes: map[parachaintypes.ValidatorIndex]validityVoteWithSign{},
					},
				},
			},
			args: args{
				candidateHash: dummyCandidateHash(t),
				tableContext: &TableContext{
					groups: map[parachaintypes.ParaID][]parachaintypes.ValidatorIndex{
						1: {1, 2, 3},
						2: {4, 5, 6},
						3: {7, 8, 9},
					},
				},
				minimumBackingVotes: 2,
			},
			wantErr: errNotEnoughValidityVotes,
		},
		// Positive test case is not added here because it is already covered in TestCandidateData_attested.
		// Context: When there are enough validity votes available, attested method of candidateData is called.
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			attestedCandidate, err := tt.table.attestedCandidate(
				tt.args.candidateHash, tt.args.tableContext, tt.args.minimumBackingVotes)
			require.ErrorIs(t, err, tt.wantErr)
			require.Nil(t, attestedCandidate)
		})
	}
}
