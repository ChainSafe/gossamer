package backing

import (
	"testing"

	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/stretchr/testify/require"
)

func TestCoreIndexFromStatement(t *testing.T) {
	candidate := getDummyCommittedCandidateReceipt(t)
	candidateHash, err := parachaintypes.GetCandidateHash(candidate)
	require.NoError(t, err)

	testCases := []struct {
		description       string
		statementWithPVD  parachaintypes.SignedFullStatementWithPVD
		expectedCoreIndex *parachaintypes.CoreIndex
		err               string
	}{
		{
			description: "valid_backing_statement",
			statementWithPVD: func() parachaintypes.SignedFullStatementWithPVD {
				statement := parachaintypes.NewStatementVDT()
				err := statement.SetValue(parachaintypes.Valid(candidateHash))
				require.NoError(t, err)

				statementWithPVD := parachaintypes.SignedFullStatementWithPVD{
					SignedFullStatement: parachaintypes.SignedFullStatement{
						Payload:        statement,
						ValidatorIndex: 2,
					},
				}
				return statementWithPVD
			}(),
			expectedCoreIndex: &parachaintypes.CoreIndex{Index: 0},
			err:               "",
		},
		{
			description: "seconded_backing_statement_and_invalid_core",
			statementWithPVD: func() parachaintypes.SignedFullStatementWithPVD {
				statement := parachaintypes.NewStatementVDT()
				err := statement.SetValue(parachaintypes.Seconded(candidate))
				require.NoError(t, err)

				statementWithPVD := parachaintypes.SignedFullStatementWithPVD{
					SignedFullStatement: parachaintypes.SignedFullStatement{
						Payload:        statement,
						ValidatorIndex: 1,
					},
				}
				return statementWithPVD
			}(),
			expectedCoreIndex: nil,
			err:               "core is not assigned to the paraID",
		},
		{
			description: "seconded_backing_statement_and_valid_core",
			statementWithPVD: func() parachaintypes.SignedFullStatementWithPVD {
				candidate.Descriptor.ParaID = 2
				statement := parachaintypes.NewStatementVDT()
				err := statement.SetValue(parachaintypes.Seconded(candidate))
				require.NoError(t, err)

				statementWithPVD := parachaintypes.SignedFullStatementWithPVD{
					SignedFullStatement: parachaintypes.SignedFullStatement{
						Payload:        statement,
						ValidatorIndex: 1,
					},
				}
				return statementWithPVD
			}(),
			expectedCoreIndex: &parachaintypes.CoreIndex{Index: 1},
			err:               "",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.description, func(t *testing.T) {
			rpState := perRelayParentState{
				numOfCores:        2,
				claimQueue:        claimQueueTestData(t),
				validatorToGroup:  validatorToGroupTestData(t),
				groupRotationInfo: groupRotationInfoTestData(t),
			}

			coreIndex, err := rpState.coreIndexFromStatement(tc.statementWithPVD)
			if tc.err != "" {
				require.ErrorContains(t, err, tc.err)
			} else {
				require.NoError(t, err)
			}

			require.Equal(t, tc.expectedCoreIndex, coreIndex)
		})
	}
}
