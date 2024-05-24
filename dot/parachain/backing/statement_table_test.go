package backing

import (
	"testing"

	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/lib/common"
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

	attested, err := data.attested(validityThreshold)
	require.ErrorIs(t, err, errNotEnoughValidityVotes)
	require.Nil(t, attested)

	data.validityVotes = map[parachaintypes.ValidatorIndex]validityVoteWithSign{
		1: {validityVote: issued, signature: parachaintypes.ValidatorSignature{1}},
		2: {validityVote: valid, signature: parachaintypes.ValidatorSignature{1, 2}},
		3: {validityVote: valid, signature: parachaintypes.ValidatorSignature{1, 2, 3}},
	}

	expectedAttestedCandidate := &attestedCandidate{
		groupID:                   1,
		committedCandidateReceipt: committedCandidateReceipt,
		validityAttestations: func() []validatorIndexWithAttestation {
			var attestations []validatorIndexWithAttestation

			// validity vote: issued
			vote1 := parachaintypes.NewValidityAttestation()
			err := vote1.Set(parachaintypes.Implicit(
				parachaintypes.ValidatorSignature{1},
			))
			require.NoError(t, err)
			attest1 := validatorIndexWithAttestation{
				validatorIndex:      1,
				validityAttestation: vote1,
			}

			// validity vote: valid
			vote2 := parachaintypes.NewValidityAttestation()
			err = vote2.Set(parachaintypes.Explicit(
				parachaintypes.ValidatorSignature{1, 2},
			))
			require.NoError(t, err)
			attest2 := validatorIndexWithAttestation{
				validatorIndex:      2,
				validityAttestation: vote2,
			}

			// validity vote: valid
			vote3 := parachaintypes.NewValidityAttestation()
			err = vote3.Set(parachaintypes.Explicit(
				parachaintypes.ValidatorSignature{1, 2, 3},
			))
			require.NoError(t, err)
			attest3 := validatorIndexWithAttestation{
				validatorIndex:      3,
				validityAttestation: vote3,
			}

			return append(attestations, attest1, attest2, attest3)
		}(),
	}

	attested, err = data.attested(validityThreshold)
	require.NoError(t, err)
	require.Equal(t, expectedAttestedCandidate, attested)
}

func TestStatementTable_attestedCandidate(t *testing.T) {
	t.Parallel()

	type args struct {
		candidateHash       parachaintypes.CandidateHash
		tableContext        *tableContext
		minimumBackingVotes uint32
	}
	tests := []struct {
		name    string
		table   *statementTable
		args    args
		want    *attestedCandidate
		wantErr error
	}{
		{
			name: "candidate_votes_not_available_for_given_candidate_hash",
			table: &statementTable{
				candidateVotes: map[parachaintypes.CandidateHash]*candidateData{},
			},
			args: args{
				candidateHash: dummyCandidateHash(t),
			},
			wantErr: errCandidateDataNotFound,
		},
		{
			name: "not_enough_validity_votes",
			table: &statementTable{
				candidateVotes: map[parachaintypes.CandidateHash]*candidateData{
					dummyCandidateHash(t): {
						groupID:       1,
						validityVotes: map[parachaintypes.ValidatorIndex]validityVoteWithSign{},
					},
				},
			},
			args: args{
				candidateHash: dummyCandidateHash(t),
				tableContext: &tableContext{
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

func TestStatementTable_importStatement(t *testing.T) {
	t.Parallel()
	committedCandidate := getDummyCommittedCandidateReceipt(t)

	testCases := []struct {
		description             string
		statementVDT            parachaintypes.StatementVDT
		detectedMisbehaviourLen int
	}{
		{
			description: "seconded_statement",
			statementVDT: func() parachaintypes.StatementVDT {
				secondedStatement := parachaintypes.NewStatementVDT()
				err := secondedStatement.Set(parachaintypes.Seconded(committedCandidate))
				require.NoError(t, err)

				return secondedStatement
			}(),
			detectedMisbehaviourLen: 1,
		},
		{
			description: "valid_statement",
			statementVDT: func() parachaintypes.StatementVDT {
				candidateHash, err := parachaintypes.GetCandidateHash(committedCandidate)
				require.NoError(t, err)

				validStatement := parachaintypes.NewStatementVDT()
				err = validStatement.Set(parachaintypes.Valid(candidateHash))
				require.NoError(t, err)

				return validStatement
			}(),
			detectedMisbehaviourLen: 0,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.description, func(t *testing.T) {
			t.Parallel()

			tableCtx := &tableContext{}
			signedStatement := parachaintypes.SignedFullStatement{
				Payload: tc.statementVDT,
			}

			table := newTable(tableConfig{})

			summary, err := table.importStatement(tableCtx, signedStatement)
			require.NoError(t, err)
			require.Nil(t, summary)

			require.Len(t, table.detectedMisbehaviour, tc.detectedMisbehaviourLen)
		})
	}
}

func TestStatementTable_importCandidate(t *testing.T) {

	authority := parachaintypes.ValidatorIndex(10)
	candidate := getDummyCommittedCandidateReceipt(t)
	var signature parachaintypes.ValidatorSignature

	var tempSignature = common.MustHexToBytes("0xc67cb93bf0a36fcee3d29de8a6a69a759659680acf486475e0a2552a5fbed87e45adce5f290698d8596095722b33599227f7461f51af8617c8be74b894cf1b86") //nolint:lll
	copy(signature[:], tempSignature)

	tableCtx := &tableContext{}

	statementSeconded := parachaintypes.NewStatementVDT()
	err := statementSeconded.Set(parachaintypes.Seconded(candidate))
	require.NoError(t, err)

	// candidateHash, err := parachaintypes.GetCandidateHash(candidate)
	require.NoError(t, err)

	table := newTable(tableConfig{})

	/*
		// validator not present in group of parachain validator
		summary, misehaviour, err := table.importCandidate(authority, candidate, signature, tableCtx)
		require.NoError(t, err)
		require.Nil(t, summary)
		require.Equal(t, parachaintypes.UnauthorizedStatement{
			Payload:        statementSeconded,
			ValidatorIndex: authority,
			Signature:      signature,
		}, misehaviour)
	*/

	//no proposals available from the validator
	tableCtx = &tableContext{
		groups: map[parachaintypes.ParaID][]parachaintypes.ValidatorIndex{
			1: {10},
		},
	}
	/*
		summary, misbehaviour, err := table.importCandidate(authority, candidate, signature, tableCtx)
		require.NoError(t, err)
		require.Nil(t, misbehaviour)
		require.Equal(t, &Summary{
			Candidate:     candidateHash,
			GroupID:       parachaintypes.ParaID(candidate.Descriptor.ParaID),
			ValidityVotes: 1,
		}, summary)

	*/

	// proposal already exists
	oldCandidateHash := parachaintypes.CandidateHash{Value: getDummyHash(t, 4)}
	oldCandidate := parachaintypes.CommittedCandidateReceipt{}
	oldSign := parachaintypes.ValidatorSignature{1, 2, 3}
	table.authorityData[authority] = []proposal{
		{
			candidateHash: oldCandidateHash,
			signature:     oldSign,
		},
	}

	table.candidateVotes[oldCandidateHash] = &candidateData{
		candidate: oldCandidate,
	}

	summary, misbehaviour, err := table.importCandidate(authority, candidate, signature, tableCtx)
	require.NoError(t, err)
	require.Nil(t, summary)
	require.Equal(t, parachaintypes.MultipleCandidates{
		First: parachaintypes.CommittedCandidateReceiptAndSign{
			CommittedCandidateReceipt: oldCandidate,
			Signature:                 oldSign,
		},
		Second: parachaintypes.CommittedCandidateReceiptAndSign{
			CommittedCandidateReceipt: candidate,
			Signature:                 signature,
		},
	}, misbehaviour)

}
