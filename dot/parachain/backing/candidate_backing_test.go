package backing

import (
	"testing"

	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/scale"
	gomock "github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func getDummyHash(num byte) common.Hash {
	hash := common.Hash{}
	for i := 0; i < 32; i++ {
		hash[i] = num
	}
	return hash
}

func getDummySeconded(t *testing.T) parachaintypes.Seconded {
	hash5 := getDummyHash(5)

	var collatorID parachaintypes.CollatorID
	tempCollatID, err := common.HexToBytes("0x48215b9d322601e5b1a95164cea0dc4626f545f98343d07f1551eb9543c4b147")
	require.NoError(t, err)
	copy(collatorID[:], tempCollatID)

	var collatorSignature parachaintypes.CollatorSignature
	tempSignature, err := common.HexToBytes("0xc67cb93bf0a36fcee3d29de8a6a69a759659680acf486475e0a2552a5fbed87e45adce5f290698d8596095722b33599227f7461f51af8617c8be74b894cf1b86") //nolint:lll
	require.NoError(t, err)
	copy(collatorSignature[:], tempSignature)

	seconded := parachaintypes.Seconded{
		Descriptor: parachaintypes.CandidateDescriptor{
			ParaID:                      uint32(1),
			RelayParent:                 hash5,
			Collator:                    collatorID,
			PersistedValidationDataHash: hash5,
			PovHash:                     hash5,
			ErasureRoot:                 hash5,
			Signature:                   collatorSignature,
			ParaHead:                    hash5,
			ValidationCodeHash:          parachaintypes.ValidationCodeHash(hash5),
		},
		Commitments: parachaintypes.CandidateCommitments{
			UpwardMessages:    []parachaintypes.UpwardMessage{{1, 2, 3}},
			NewValidationCode: &parachaintypes.ValidationCode{1, 2, 3},
			HeadData: parachaintypes.HeadData{
				Data: []byte{1, 2, 3},
			},
			ProcessedDownwardMessages: uint32(5),
			HrmpWatermark:             uint32(0),
		},
	}

	return seconded
}

func TestImportStatement(t *testing.T) {
	seconded := getDummySeconded(t)

	statementVDTSeconded := parachaintypes.NewStatementVDT()
	err := statementVDTSeconded.Set(seconded)
	require.NoError(t, err)

	candidateHash := parachaintypes.CandidateHash{
		Value: common.MustBlake2bHash(scale.MustMarshal(seconded)),
	}

	statementVDTValid := parachaintypes.NewStatementVDT()
	err = statementVDTValid.Set(parachaintypes.Valid{})
	require.NoError(t, err)

	testCases := []struct {
		description            string
		rpState                perRelayParentState
		perCandidate           map[parachaintypes.CandidateHash]perCandidateState
		signedStatementWithPVD SignedFullStatementWithPVD
		summary                *Summary
		err                    string
	}{
		{
			description:            "statementVDT not set",
			rpState:                perRelayParentState{},
			signedStatementWithPVD: SignedFullStatementWithPVD{},
			summary:                nil,
			err:                    "getting value from statementVDT:",
		},
		{
			description: "statementVDT in not seconded",
			rpState: func() perRelayParentState {
				ctrl := gomock.NewController(t)
				mockTable := NewMockTable(ctrl)

				mockTable.EXPECT().importStatement(
					gomock.AssignableToTypeOf(new(TableContext)),
					gomock.AssignableToTypeOf(SignedFullStatementWithPVD{}),
				).Return(new(Summary), nil)

				return perRelayParentState{
					Table: mockTable,
				}
			}(),
			signedStatementWithPVD: SignedFullStatementWithPVD{
				SignedFullStatement: parachaintypes.UncheckedSignedFullStatement{
					Payload: statementVDTValid,
				},
			},
			summary: new(Summary),
			err:     "",
		},
		{
			description: "invalid persisted validation data",
			rpState:     perRelayParentState{},
			signedStatementWithPVD: SignedFullStatementWithPVD{
				SignedFullStatement: parachaintypes.UncheckedSignedFullStatement{
					Payload: statementVDTSeconded,
				},
			},
			summary: nil,
			err:     "persisted validation data is nil",
		},
		{
			description: "statement is 'seconded' and candidate is known",
			rpState: func() perRelayParentState {
				ctrl := gomock.NewController(t)
				mockTable := NewMockTable(ctrl)

				mockTable.EXPECT().importStatement(
					gomock.AssignableToTypeOf(new(TableContext)),
					gomock.AssignableToTypeOf(SignedFullStatementWithPVD{}),
				).Return(new(Summary), nil)

				return perRelayParentState{
					Table: mockTable,
				}
			}(),
			perCandidate: map[parachaintypes.CandidateHash]perCandidateState{
				candidateHash: {
					persistedValidationData: parachaintypes.PersistedValidationData{
						ParentHead: parachaintypes.HeadData{
							Data: []byte{1, 2, 3},
						},
					},
					SecondedLocally: false,
					ParaID:          1,
					RelayParent:     getDummyHash(5),
				},
			},
			signedStatementWithPVD: SignedFullStatementWithPVD{
				SignedFullStatement: parachaintypes.UncheckedSignedFullStatement{
					Payload: statementVDTSeconded,
				},
				PersistedValidationData: &parachaintypes.PersistedValidationData{
					ParentHead: parachaintypes.HeadData{
						Data: []byte{1, 2, 3},
					},
					RelayParentNumber:      5,
					RelayParentStorageRoot: getDummyHash(5),
					MaxPovSize:             3,
				},
			},
			summary: new(Summary),
			err:     "",
		},
		{
			description: "statement is 'seconded' and candidate is unknown",
			rpState: func() perRelayParentState {
				ctrl := gomock.NewController(t)
				mockTable := NewMockTable(ctrl)

				mockTable.EXPECT().importStatement(
					gomock.AssignableToTypeOf(new(TableContext)),
					gomock.AssignableToTypeOf(SignedFullStatementWithPVD{}),
				).Return(new(Summary), nil)

				return perRelayParentState{
					Table: mockTable,
				}
			}(),
			perCandidate: map[parachaintypes.CandidateHash]perCandidateState{},
			signedStatementWithPVD: SignedFullStatementWithPVD{
				SignedFullStatement: parachaintypes.UncheckedSignedFullStatement{
					Payload: statementVDTSeconded,
				},
				PersistedValidationData: &parachaintypes.PersistedValidationData{
					ParentHead: parachaintypes.HeadData{
						Data: []byte{1, 2, 3},
					},
					RelayParentNumber:      5,
					RelayParentStorageRoot: getDummyHash(5),
					MaxPovSize:             3,
				},
			},
			summary: new(Summary),
			err:     "",
		},
	}

	for _, c := range testCases {
		c := c
		t.Run(c.description, func(t *testing.T) {
			t.Parallel()

			subSystemToOverseer := make(chan<- any)

			summary, err := c.rpState.importStatement(subSystemToOverseer, c.signedStatementWithPVD, c.perCandidate)
			if c.summary == nil {
				require.Nil(t, summary)
			} else {
				require.Equal(t, c.summary, summary)
			}

			if c.err == "" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, c.err)
			}
		})
	}
}

/*
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
*/
