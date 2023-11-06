package backing

import (
	"testing"

	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/lib/common"
	gomock "github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func TestProcessOverseerMessageStatement(t *testing.T) {

	t.Parallel()

	testRelayParent := common.Hash{}

	statementVDTSeconded := parachaintypes.NewStatementVDT()
	err := statementVDTSeconded.Set(parachaintypes.Seconded{})
	require.NoError(t, err)

	statementVDTValid := parachaintypes.NewStatementVDT()
	err = statementVDTValid.Set(parachaintypes.Valid{})
	require.NoError(t, err)

	testCases := []struct {
		description    string
		msg            any
		perRelayParent map[common.Hash]perRelayParentState
		perCandidate   map[parachaintypes.CandidateHash]perCandidateState
		errString      string
	}{
		{
			description:    "unknown relay parent",
			msg:            StatementMessage{},
			perRelayParent: map[common.Hash]perRelayParentState{},
			errString:      ErrStatementForUnknownRelayParent.Error(),
		},
		{
			description: "statementVDT not set",
			msg: StatementMessage{
				RelayParent: testRelayParent,
			},
			perRelayParent: map[common.Hash]perRelayParentState{
				testRelayParent: {},
			},
			errString: "importing statement: getting value from statementVDT:",
		},
		{
			description: "invalid persisted validation data",
			msg: StatementMessage{
				RelayParent: testRelayParent,
				SignedFullStatement: SignedFullStatementWithPVD{
					SignedFullStatement: parachaintypes.UncheckedSignedFullStatement{
						Payload: statementVDTSeconded,
					},
					PersistedValidationData: nil,
				},
			},
			perRelayParent: map[common.Hash]perRelayParentState{
				testRelayParent: {},
			},
			errString: "persisted validation data is nil",
		},
		{
			description: "invalid summery of import statement",
			msg: StatementMessage{
				RelayParent: testRelayParent,
				SignedFullStatement: SignedFullStatementWithPVD{
					SignedFullStatement: parachaintypes.UncheckedSignedFullStatement{
						Payload: statementVDTSeconded,
					},
					PersistedValidationData: &parachaintypes.PersistedValidationData{
						ParentHead:             parachaintypes.HeadData{Data: []byte{0x01}},
						RelayParentNumber:      2,
						RelayParentStorageRoot: testRelayParent,
						MaxPovSize:             3,
					},
				},
			},
			perRelayParent: map[common.Hash]perRelayParentState{
				testRelayParent: func() perRelayParentState {
					ctrl := gomock.NewController(t)
					mockTable := NewMockTable(ctrl)

					mockTable.EXPECT().importStatement(
						gomock.AssignableToTypeOf(new(TableContext)),
						gomock.AssignableToTypeOf(SignedFullStatementWithPVD{}),
					).Return(nil, nil)

					mockTable.EXPECT().drainMisbehaviors().Return([]parachaintypes.PDMisbehaviorReport{})

					return perRelayParentState{
						Table: mockTable,
					}
				}(),
			},
			perCandidate: make(map[parachaintypes.CandidateHash]perCandidateState),
			errString:    "",
		},
		{
			description: "happy path",
			msg: StatementMessage{
				RelayParent: testRelayParent,
				SignedFullStatement: SignedFullStatementWithPVD{
					SignedFullStatement: parachaintypes.UncheckedSignedFullStatement{
						Payload: statementVDTSeconded,
					},
					PersistedValidationData: &parachaintypes.PersistedValidationData{
						ParentHead:             parachaintypes.HeadData{Data: []byte{0x01}},
						RelayParentNumber:      2,
						RelayParentStorageRoot: testRelayParent,
						MaxPovSize:             3,
					},
				},
			},
			perRelayParent: map[common.Hash]perRelayParentState{
				testRelayParent: func() perRelayParentState {
					ctrl := gomock.NewController(t)
					mockTable := NewMockTable(ctrl)

					mockTable.EXPECT().importStatement(
						gomock.AssignableToTypeOf(new(TableContext)),
						gomock.AssignableToTypeOf(SignedFullStatementWithPVD{}),
					).Return(&Summary{Candidate: parachaintypes.CandidateHash{
						Value: common.MustHexToHash("0xc52534b8a49be30506fb5b214e4f588e58f5ff9feeafcdab85c8ccf66ad28e6b"),
					}}, nil)

					mockTable.EXPECT().attestedCandidate(
						gomock.AssignableToTypeOf(new(parachaintypes.CandidateHash)),
						gomock.AssignableToTypeOf(new(TableContext)),
					).Return(new(AttestedCandidate), nil)

					mockTable.EXPECT().drainMisbehaviors().Return([]parachaintypes.PDMisbehaviorReport{})

					mockTable.EXPECT().getCandidate(
						gomock.AssignableToTypeOf(parachaintypes.CandidateHash{}),
					).Return(new(parachaintypes.CommittedCandidateReceipt), nil)

					return perRelayParentState{
						Table:              mockTable,
						backed:             make(map[parachaintypes.CandidateHash]bool),
						fallbacks:          make(map[parachaintypes.CandidateHash]AttestingData),
						AwaitingValidation: make(map[parachaintypes.CandidateHash]bool),
					}
				}(),
			},
			perCandidate: make(map[parachaintypes.CandidateHash]perCandidateState),
			errString:    "",
		},
	}
	for _, c := range testCases {
		c := c
		t.Run(c.description, func(t *testing.T) {
			t.Parallel()

			cb := CandidateBacking{
				perRelayParent:      c.perRelayParent,
				SubSystemToOverseer: make(chan<- any, 1),
				perCandidate:        c.perCandidate,
			}

			chRelayParentAndCommand := make(chan RelayParentAndCommand)

			err := cb.processOverseerMessage(c.msg, chRelayParentAndCommand)

			if c.errString == "" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, c.errString)
			}
		})
	}
}
