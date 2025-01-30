package backing

import (
	"testing"

	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"
)

func TestImportStatement(t *testing.T) {
	statementVDTValid := parachaintypes.NewStatementVDT()
	err := statementVDTValid.SetValue(parachaintypes.Valid{})
	require.NoError(t, err)

	dummyCCR := getDummyCommittedCandidateReceipt(t)
	seconded := parachaintypes.Seconded(dummyCCR)

	candidateHash, err := parachaintypes.GetCandidateHash(dummyCCR)
	require.NoError(t, err)

	statementVDTSeconded := parachaintypes.NewStatementVDT()
	err = statementVDTSeconded.SetValue(seconded)
	require.NoError(t, err)

	testCases := []struct {
		description            string
		rpState                func() perRelayParentState
		perCandidate           map[parachaintypes.CandidateHash]*perCandidateState
		signedStatementWithPVD parachaintypes.SignedFullStatementWithPVD
		mockOverseer           func(*testing.T, chan any)
	}{
		{
			description: "statement_is_not_seconded",
			rpState: func() perRelayParentState {
				ctrl := gomock.NewController(t)
				mockTable := NewMockTable(ctrl)

				mockTable.EXPECT().importStatement(
					gomock.AssignableToTypeOf(new(tableContext)),
					gomock.AssignableToTypeOf(parachaintypes.GroupIndex(0)),
					gomock.AssignableToTypeOf(parachaintypes.SignedFullStatement{}),
				).Return(new(Summary), nil)

				return perRelayParentState{
					table: mockTable,
				}
			},
			signedStatementWithPVD: parachaintypes.SignedFullStatementWithPVD{
				SignedFullStatement: parachaintypes.SignedFullStatement{
					Payload: statementVDTValid,
				},
			},
			mockOverseer: func(*testing.T, chan any) {},
		},
		{
			description: "seconded_statement_known_candidate",
			rpState: func() perRelayParentState {
				ctrl := gomock.NewController(t)
				mockTable := NewMockTable(ctrl)

				mockTable.EXPECT().importStatement(
					gomock.AssignableToTypeOf(new(tableContext)),
					gomock.AssignableToTypeOf(parachaintypes.GroupIndex(0)),
					gomock.AssignableToTypeOf(parachaintypes.SignedFullStatement{}),
				).Return(new(Summary), nil)

				return perRelayParentState{
					table: mockTable,
				}
			},
			perCandidate: map[parachaintypes.CandidateHash]*perCandidateState{
				candidateHash: {
					persistedValidationData: parachaintypes.PersistedValidationData{
						ParentHead: parachaintypes.HeadData{
							Data: []byte{1, 2, 3},
						},
					},
					secondedLocally: false,
					relayParent:     getDummyHash(t, 5),
				},
			},
			signedStatementWithPVD: secondedSignedFullStatementWithPVD(t, statementVDTSeconded),
			mockOverseer:           func(*testing.T, chan any) {},
		},
		{
			description: "seconded_statement_unknown_candidate",
			rpState: func() perRelayParentState {
				ctrl := gomock.NewController(t)
				mockTable := NewMockTable(ctrl)

				mockTable.EXPECT().importStatement(
					gomock.AssignableToTypeOf(new(tableContext)),
					gomock.AssignableToTypeOf(parachaintypes.GroupIndex(0)),
					gomock.AssignableToTypeOf(parachaintypes.SignedFullStatement{}),
				).Return(new(Summary), nil)

				return perRelayParentState{
					table: mockTable,
				}
			},
			perCandidate:           map[parachaintypes.CandidateHash]*perCandidateState{},
			signedStatementWithPVD: secondedSignedFullStatementWithPVD(t, statementVDTSeconded),
			mockOverseer: func(t *testing.T, subSystemToOverseer chan any) {
				v := <-subSystemToOverseer
				introduce, ok := v.(parachaintypes.ProspectiveParachainsMessageIntroduceCandidate)
				require.True(t, ok)

				introduce.Ch <- nil
			},
		},
	}

	for _, c := range testCases {
		c := c
		t.Run(c.description, func(t *testing.T) {
			t.Parallel()

			subSystemToOverseer := make(chan any)
			defer close(subSystemToOverseer)

			go c.mockOverseer(t, subSystemToOverseer)

			rpState := c.rpState()
			summary, err := rpState.importStatement(subSystemToOverseer, c.signedStatementWithPVD, c.perCandidate)
			require.Equal(t, new(Summary), summary)
			require.NoError(t, err)
		})
	}
}
