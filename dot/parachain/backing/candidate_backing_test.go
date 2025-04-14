// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package backing

import (
	_ "embed"
	"errors"
	"testing"

	availabilitystore "github.com/ChainSafe/gossamer/dot/parachain/availability-store"
	candidatevalidation "github.com/ChainSafe/gossamer/dot/parachain/candidate-validation"
	prospectiveparachains "github.com/ChainSafe/gossamer/dot/parachain/prospective-parachains/messages"
	provisionermessages "github.com/ChainSafe/gossamer/dot/parachain/provisioner/messages"
	statementdistributionmessages "github.com/ChainSafe/gossamer/dot/parachain/statement-distribution/messages"
	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"
)

var tempSignature = common.MustHexToBytes("0xc67cb93bf0a36fcee3d29de8a6a69a759659680acf486475e0a2552a5fbed87e45adce5f290698d8596095722b33599227f7461f51af8617c8be74b894cf1b86") //nolint:lll

func getDummyHash(t *testing.T, num byte) common.Hash {
	t.Helper()
	hash := common.Hash{}
	for i := 0; i < 32; i++ {
		hash[i] = num
	}
	return hash
}

func getDummyCommittedCandidateReceipt(t *testing.T) parachaintypes.CommittedCandidateReceiptV2 {
	t.Helper()
	hash5 := getDummyHash(t, 6)

	var collatorID parachaintypes.CollatorID
	tempCollatID := common.MustHexToBytes("0x48215b9d322601e5b1a95164cea0dc4626f545f98343d07f1551eb9543c4b147")
	copy(collatorID[:], tempCollatID)

	var collatorSignature parachaintypes.CollatorSignature
	copy(collatorSignature[:], tempSignature)

	ccr := parachaintypes.CommittedCandidateReceipt{
		Descriptor: parachaintypes.CandidateDescriptor{
			ParaID:                      1,
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

	return ccr.V2()
}

func mockOverseer(t *testing.T, subsystemToOverseer chan any) {
	t.Helper()
	for data := range subsystemToOverseer {
		switch data := data.(type) {
		case prospectiveparachains.IntroduceSecondedCandidate:
			data.Response <- true
		case provisionermessages.ProvisionableData,
			prospectiveparachains.CandidateBacked,
			statementdistributionmessages.Backed:
			continue
		default:
			t.Errorf("unknown type: %T\n", data)
		}
	}
}

func secondedSignedFullStatementWithPVD(
	t *testing.T,
	statementVDTSeconded parachaintypes.StatementVDT,
) parachaintypes.SignedFullStatementWithPVD {
	t.Helper()
	return parachaintypes.SignedFullStatementWithPVD{
		SignedFullStatement: parachaintypes.SignedFullStatement{
			Payload: statementVDTSeconded,
		},
		PersistedValidationData: &parachaintypes.PersistedValidationData{
			ParentHead: parachaintypes.HeadData{
				Data: []byte{1, 2, 3},
			},
			RelayParentNumber:      5,
			RelayParentStorageRoot: getDummyHash(t, 5),
			MaxPovSize:             3,
		},
	}
}

func claimQueueTestData(t *testing.T) map[parachaintypes.CoreIndex][]parachaintypes.ParaID {
	t.Helper()

	return map[parachaintypes.CoreIndex][]parachaintypes.ParaID{
		{Index: 0}: {1},
		{Index: 1}: {2},
	}
}

func validatorToGroupTestData(t *testing.T) map[parachaintypes.ValidatorIndex]parachaintypes.GroupIndex {
	t.Helper()

	return map[parachaintypes.ValidatorIndex]parachaintypes.GroupIndex{
		0: 0,
		1: 1,
		2: 0,
		3: 0,
		5: 0,
	}
}

func groupRotationInfoTestData(t *testing.T) parachaintypes.GroupRotationInfo {
	t.Helper()

	return parachaintypes.GroupRotationInfo{
		SessionStartBlock:      0,
		GroupRotationFrequency: 100,
		Now:                    1,
	}
}

func TestImportStatement(t *testing.T) {
	t.Parallel()

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
					table:             mockTable,
					numOfCores:        2,
					claimQueue:        claimQueueTestData(t),
					validatorToGroup:  validatorToGroupTestData(t),
					groupRotationInfo: groupRotationInfoTestData(t),
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
					table:             mockTable,
					numOfCores:        2,
					claimQueue:        claimQueueTestData(t),
					validatorToGroup:  validatorToGroupTestData(t),
					groupRotationInfo: groupRotationInfoTestData(t),
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
					table:             mockTable,
					numOfCores:        2,
					claimQueue:        claimQueueTestData(t),
					validatorToGroup:  validatorToGroupTestData(t),
					groupRotationInfo: groupRotationInfoTestData(t),
				}
			},
			perCandidate:           map[parachaintypes.CandidateHash]*perCandidateState{},
			signedStatementWithPVD: secondedSignedFullStatementWithPVD(t, statementVDTSeconded),
			mockOverseer: func(t *testing.T, subSystemToOverseer chan any) {
				v := <-subSystemToOverseer
				introduce, ok := v.(prospectiveparachains.IntroduceSecondedCandidate)
				require.True(t, ok)

				introduce.Response <- true
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

func mustHexTo32BArray(t *testing.T, inputHex string) (outputArray [sr25519.PublicKeyLength]byte) {
	t.Helper()
	copy(outputArray[:], common.MustHexToBytes(inputHex))
	return outputArray
}

func dummyValidityAttestation(t *testing.T, value string) parachaintypes.ValidityAttestation {
	t.Helper()

	var validatorSignature parachaintypes.ValidatorSignature
	copy(validatorSignature[:], tempSignature)

	vdt := parachaintypes.NewValidityAttestation()
	switch value {
	case "implicit":
		err := vdt.SetValue(parachaintypes.Implicit(validatorSignature))
		require.NoError(t, err)
	case "explicit":
		err := vdt.SetValue(parachaintypes.Explicit(validatorSignature))
		require.NoError(t, err)
	default:
		require.Fail(t, "invalid value")
	}
	return vdt
}

func dummyTableContext(t *testing.T) tableContext {
	t.Helper()

	return tableContext{
		validator: &parachaintypes.Validator{
			Index: 1,
		},
		groups: map[parachaintypes.CoreIndex][]parachaintypes.ValidatorIndex{
			{Index: 1}: {1, 2, 3},
			{Index: 2}: {4, 5, 6},
			{Index: 3}: {7, 8, 9},
		},
		validators: []parachaintypes.ValidatorID{
			mustHexTo32BArray(t, "0xa262f83b46310770ae8d092147176b8b25e8855bcfbbe701d346b10db0c5385d"),
			mustHexTo32BArray(t, "0x804b9df571e2b744d65eca2d4c59eb8e4345286c00389d97bfc1d8d13aa6e57e"),
			mustHexTo32BArray(t, "0x4eb63e4aad805c06dc924e2f19b1dde7faf507e5bb3c1838d6a3cfc10e84fe72"),
			mustHexTo32BArray(t, "0x74c337d57035cd6b7718e92a0d8ea6ef710da8ab1215a057c40c4ef792155a68"),
		},
	}
}

func rpStateForSuccessfulPostImportStatement(t *testing.T, ctrl *gomock.Controller) perRelayParentState {
	t.Helper()

	attestedToReturn := attestedCandidate{
		groupID:                   3,
		committedCandidateReceipt: getDummyCommittedCandidateReceipt(t),
		validityAttestations: []validatorIndexWithAttestation{
			{
				validatorIndex:      7,
				validityAttestation: dummyValidityAttestation(t, "implicit"),
			},
			{
				validatorIndex:      8,
				validityAttestation: dummyValidityAttestation(t, "explicit"),
			},
			{
				validatorIndex:      9,
				validityAttestation: dummyValidityAttestation(t, "implicit"),
			},
		},
	}

	mockTable := NewMockTable(ctrl)

	mockTable.EXPECT().drainMisbehaviors().
		Return(map[parachaintypes.ValidatorIndex][]parachaintypes.Misbehaviour{})
	mockTable.EXPECT().attestedCandidate(
		gomock.AssignableToTypeOf(parachaintypes.CandidateHash{}),
		gomock.AssignableToTypeOf(new(tableContext)),
		gomock.AssignableToTypeOf(uint32(0)),
	).Return(&attestedToReturn, nil)

	return perRelayParentState{
		table:        mockTable,
		tableContext: dummyTableContext(t),
		backed:       map[parachaintypes.CandidateHash]bool{},
	}
}

func TestPostImportStatement(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		description string
		rpState     func(*gomock.Controller) perRelayParentState
		summary     *Summary
		validate    func(t *testing.T, subSystemToOverseer chan any)
	}{
		{
			description: "nil_summary",
			rpState: func(ctrl *gomock.Controller) perRelayParentState {
				mockTable := NewMockTable(ctrl)
				mockTable.EXPECT().drainMisbehaviors().Return(map[parachaintypes.ValidatorIndex][]parachaintypes.Misbehaviour{})

				return perRelayParentState{
					relayParent: common.Hash{0x01},
					table:       mockTable,
					backed:      make(map[parachaintypes.CandidateHash]bool),
				}
			},
			summary: nil,
			validate: func(t *testing.T, subSystemToOverseer chan any) {
				require.Empty(t, subSystemToOverseer)
			},
		},
		{
			description: "candidate_not_attested",
			rpState: func(ctrl *gomock.Controller) perRelayParentState {
				mockTable := NewMockTable(ctrl)
				mockTable.EXPECT().drainMisbehaviors().
					Return(map[parachaintypes.ValidatorIndex][]parachaintypes.Misbehaviour{})
				mockTable.EXPECT().attestedCandidate(
					gomock.AssignableToTypeOf(parachaintypes.CandidateHash{}),
					gomock.AssignableToTypeOf(new(tableContext)),
					gomock.AssignableToTypeOf(uint32(0)),
				).Return(nil, errors.New("could not get attested candidate from table"))

				return perRelayParentState{
					relayParent: common.Hash{0x01},
					table:       mockTable,
					backed:      make(map[parachaintypes.CandidateHash]bool),
				}
			},
			summary: &Summary{},
			validate: func(t *testing.T, subSystemToOverseer chan any) {
				require.Empty(t, subSystemToOverseer)
			},
		},
		{
			description: "candidate_already_backed",
			rpState: func(ctrl *gomock.Controller) perRelayParentState {

				candidate := getDummyCommittedCandidateReceipt(t)
				candidateHash, err := parachaintypes.GetCandidateHash(candidate)
				require.NoError(t, err)

				mockTable := NewMockTable(ctrl)
				mockTable.EXPECT().drainMisbehaviors().
					Return(map[parachaintypes.ValidatorIndex][]parachaintypes.Misbehaviour{})
				mockTable.EXPECT().attestedCandidate(
					gomock.AssignableToTypeOf(parachaintypes.CandidateHash{}),
					gomock.AssignableToTypeOf(new(tableContext)),
					gomock.AssignableToTypeOf(uint32(0)),
				).Return(&attestedCandidate{
					groupID:                   4,
					committedCandidateReceipt: candidate,
				}, nil)
				return perRelayParentState{
					relayParent: common.Hash{0x01},
					table:       mockTable,
					backed: map[parachaintypes.CandidateHash]bool{
						candidateHash: true,
					},
				}
			},
			summary: &Summary{},
			validate: func(t *testing.T, subSystemToOverseer chan any) {
				require.Empty(t, subSystemToOverseer)
			},
		},
		{
			description: "successful_post_import_statement",
			rpState: func(ctrl *gomock.Controller) perRelayParentState {
				return rpStateForSuccessfulPostImportStatement(t, ctrl)
			},
			summary: &Summary{},
			validate: func(t *testing.T, subSystemToOverseer chan any) {
				require.Len(t, subSystemToOverseer, 2)
				require.IsType(t, prospectiveparachains.CandidateBacked{}, <-subSystemToOverseer)
				require.IsType(t, statementdistributionmessages.Backed{}, <-subSystemToOverseer)
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.description, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			subSystemToOverseer := make(chan any, 10)
			defer close(subSystemToOverseer)

			rpState := tc.rpState(ctrl)
			rpState.postImportStatement(subSystemToOverseer, tc.summary)
			tc.validate(t, subSystemToOverseer)
		})
	}
}

func TestKickOffValidationWork(t *testing.T) {
	t.Parallel()

	attesting := attestingData{
		candidate: getDummyCommittedCandidateReceipt(t).ToPlain(),
	}

	hash, err := attesting.candidate.Hash()
	require.NoError(t, err)

	candidateHash := parachaintypes.CandidateHash{Value: hash}

	testCases := []struct {
		description     string
		rpState         perRelayParentState
		processChannels func(chan any, chan relayParentAndCommand)
	}{
		{
			description: "already_issued_statement_for_candidate",
			rpState: perRelayParentState{
				tableContext: tableContext{validator: &parachaintypes.Validator{Disabled: false}},
				issuedStatements: map[parachaintypes.CandidateHash]bool{
					candidateHash: true,
				},
			},
			processChannels: func(chan any, chan relayParentAndCommand) {},
		},
		{
			description: "not_issued_statement_but_waiting_for_validation",
			rpState: perRelayParentState{
				tableContext:     tableContext{validator: &parachaintypes.Validator{Disabled: false}},
				issuedStatements: map[parachaintypes.CandidateHash]bool{},
				awaitingValidation: map[parachaintypes.CandidateHash]bool{
					candidateHash: true,
				},
			},
			processChannels: func(subSystemToOverseer chan any, cmdCh chan relayParentAndCommand) {
				for {
					select {
					case data := <-subSystemToOverseer:
						val, ok := data.(parachaintypes.AvailabilityDistributionMessageFetchPoV)
						if !ok {
							t.Errorf("invalid overseer message type: %T\n", data)
						}
						val.PovCh <- parachaintypes.OverseerFuncRes[parachaintypes.PoV]{
							Err: parachaintypes.ErrFetchPoV,
						}
					case <-cmdCh:
					}
				}
			},
		},
	}

	for _, c := range testCases {
		c := c
		t.Run(c.description, func(t *testing.T) {
			t.Parallel()

			subSystemToOverseer := make(chan any)
			chRelayParentAndCommand := make(chan relayParentAndCommand)
			pvd := parachaintypes.PersistedValidationData{}

			go c.processChannels(subSystemToOverseer, chRelayParentAndCommand)

			err := c.rpState.kickOffValidationWork(nil, subSystemToOverseer, chRelayParentAndCommand, pvd, attesting)
			require.NoError(t, err)
		})
	}
}

func TestValidateAndMakeAvailable(t *testing.T) {
	t.Parallel()

	var pvd parachaintypes.PersistedValidationData
	candidateReceipt := getDummyCommittedCandidateReceipt(t).ToPlain()

	hash, err := candidateReceipt.Hash()
	require.NoError(t, err)

	candidateHash := parachaintypes.CandidateHash{Value: hash}

	testCases := []struct {
		description    string
		rpState        perRelayParentState
		expectedErr    string
		mockOverseer   func(ch chan any)
		mockBlockState func() *MockBlockState
	}{
		{
			description: "validation_process_already_started_for_candidate",
			rpState: perRelayParentState{
				assignedCore:     &parachaintypes.CoreIndex{Index: 1},
				issuedStatements: map[parachaintypes.CandidateHash]bool{},
				awaitingValidation: map[parachaintypes.CandidateHash]bool{
					candidateHash: true,
				},
			},
			expectedErr:  "",
			mockOverseer: func(ch chan any) {},
			mockBlockState: func() *MockBlockState {
				ctrl := gomock.NewController(t)
				mockBlockstate := NewMockBlockState(ctrl)
				return mockBlockstate
			},
		},
		{
			description: "unable_to_get_validation_code",
			rpState: perRelayParentState{
				assignedCore:       &parachaintypes.CoreIndex{Index: 1},
				issuedStatements:   map[parachaintypes.CandidateHash]bool{},
				awaitingValidation: map[parachaintypes.CandidateHash]bool{},
			},
			expectedErr:  "getting validation code by hash: ",
			mockOverseer: func(ch chan any) {},
			mockBlockState: func() *MockBlockState {
				ctrl := gomock.NewController(t)

				mockRuntime := NewMockInstance(ctrl)
				mockRuntime.EXPECT().ParachainHostValidationCodeByHash(gomock.AssignableToTypeOf(common.Hash{})).
					Return(nil, errors.New("mock error getting validation code"))

				mockBlockstate := NewMockBlockState(ctrl)
				mockBlockstate.EXPECT().GetRuntime(gomock.AssignableToTypeOf(common.Hash{})).Return(mockRuntime, nil)
				return mockBlockstate
			},
		},
		{
			description: "unable_to_get_validation_result",
			rpState: perRelayParentState{
				executorParams:     &parachaintypes.ExecutorParams{},
				assignedCore:       &parachaintypes.CoreIndex{Index: 1},
				issuedStatements:   map[parachaintypes.CandidateHash]bool{},
				awaitingValidation: map[parachaintypes.CandidateHash]bool{},
			},
			expectedErr: "getting validation result: ",
			mockOverseer: func(ch chan any) {
				for data := range ch {
					switch data := data.(type) {
					case candidatevalidation.ValidateFromExhaustive:
						data.Ch <- parachaintypes.OverseerFuncRes[candidatevalidation.ValidationResult]{
							Err: errors.New("mock error getting validation result"),
						}
					default:
						t.Errorf("invalid overseer message type: %T\n", data)
					}
				}
			},
			mockBlockState: func() *MockBlockState {
				ctrl := gomock.NewController(t)

				mockRuntime := NewMockInstance(ctrl)
				mockRuntime.EXPECT().ParachainHostValidationCodeByHash(gomock.AssignableToTypeOf(common.Hash{})).
					Return(&parachaintypes.ValidationCode{1, 2, 3}, nil)

				mockBlockstate := NewMockBlockState(ctrl)
				mockBlockstate.EXPECT().GetRuntime(gomock.AssignableToTypeOf(common.Hash{})).Return(mockRuntime, nil)
				return mockBlockstate
			},
		},
		{
			description: "validation_result_is_invalid",
			rpState: perRelayParentState{
				executorParams:     &parachaintypes.ExecutorParams{},
				assignedCore:       &parachaintypes.CoreIndex{Index: 1},
				issuedStatements:   map[parachaintypes.CandidateHash]bool{},
				awaitingValidation: map[parachaintypes.CandidateHash]bool{},
			},
			expectedErr: "",
			mockOverseer: func(ch chan any) {
				for data := range ch {
					switch data := data.(type) {
					case candidatevalidation.ValidateFromExhaustive:
						data.Ch <- parachaintypes.OverseerFuncRes[candidatevalidation.ValidationResult]{
							Data: candidatevalidation.ValidationResult{
								Invalid: candidatevalidation.ExecutionError.Ptr(),
							},
						}
					default:
						t.Errorf("invalid overseer message type: %T\n", data)
					}
				}
			},
			mockBlockState: func() *MockBlockState {
				ctrl := gomock.NewController(t)

				mockRuntime := NewMockInstance(ctrl)
				mockRuntime.EXPECT().ParachainHostValidationCodeByHash(gomock.AssignableToTypeOf(common.Hash{})).
					Return(&parachaintypes.ValidationCode{1, 2, 3}, nil)

				mockBlockstate := NewMockBlockState(ctrl)
				mockBlockstate.EXPECT().GetRuntime(gomock.AssignableToTypeOf(common.Hash{})).Return(mockRuntime, nil)
				return mockBlockstate
			},
		},
		{
			description: "validation_result_is_valid",
			rpState: perRelayParentState{
				executorParams:     &parachaintypes.ExecutorParams{},
				assignedCore:       &parachaintypes.CoreIndex{Index: 1},
				issuedStatements:   map[parachaintypes.CandidateHash]bool{},
				awaitingValidation: map[parachaintypes.CandidateHash]bool{},
			},
			expectedErr: "",
			mockOverseer: func(ch chan any) {
				for data := range ch {
					switch data := data.(type) {
					case candidatevalidation.ValidateFromExhaustive:
						data.Ch <- parachaintypes.OverseerFuncRes[candidatevalidation.ValidationResult]{
							Data: candidatevalidation.ValidationResult{
								Valid: &candidatevalidation.Valid{},
							},
						}
					case availabilitystore.StoreAvailableData:
						data.Sender <- errInvalidErasureRoot
					default:
						t.Errorf("invalid overseer message type: %T\n", data)
					}
				}
			},
			mockBlockState: func() *MockBlockState {
				ctrl := gomock.NewController(t)

				mockRuntime := NewMockInstance(ctrl)
				mockRuntime.EXPECT().ParachainHostValidationCodeByHash(gomock.AssignableToTypeOf(common.Hash{})).
					Return(&parachaintypes.ValidationCode{1, 2, 3}, nil)

				mockBlockstate := NewMockBlockState(ctrl)
				mockBlockstate.EXPECT().GetRuntime(gomock.AssignableToTypeOf(common.Hash{})).Return(mockRuntime, nil)
				return mockBlockstate
			},
		},
	}

	for _, c := range testCases {
		c := c
		t.Run(c.description, func(t *testing.T) {
			t.Parallel()

			subSystemToOverseer := make(chan any)
			chRelayParentAndCommand := make(chan relayParentAndCommand)

			go c.mockOverseer(subSystemToOverseer)
			go func(chRelayParentAndCommand chan relayParentAndCommand) {
				<-chRelayParentAndCommand
			}(chRelayParentAndCommand)

			rpState := c.rpState
			rpState.relayParent = getDummyHash(t, 5)

			err := rpState.validateAndMakeAvailable(
				c.mockBlockState(),
				subSystemToOverseer,
				chRelayParentAndCommand,
				candidateReceipt,
				pvd,
				parachaintypes.PoV{},
				2,
				attest,
				candidateHash,
			)

			if c.expectedErr != "" {
				require.ErrorContains(t, err, c.expectedErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestHandleStatementMessage(t *testing.T) {
	t.Parallel()

	relayParent := getDummyHash(t, 5)
	chRelayParentAndCommand := make(chan relayParentAndCommand)

	dummyCCR := getDummyCommittedCandidateReceipt(t)
	seconded := parachaintypes.Seconded(dummyCCR)

	statementVDTSeconded := parachaintypes.NewStatementVDT()
	err := statementVDTSeconded.SetValue(seconded)
	require.NoError(t, err)

	candidateHash, err := parachaintypes.GetCandidateHash(dummyCCR)
	require.NoError(t, err)

	statementVDTValid := parachaintypes.NewStatementVDT()
	err = statementVDTValid.SetValue(parachaintypes.Valid(candidateHash))
	require.NoError(t, err)

	testCases := []struct {
		description            string
		perRelayParent         func() map[common.Hash]*perRelayParentState
		perCandidate           map[parachaintypes.CandidateHash]*perCandidateState
		signedStatementWithPVD parachaintypes.SignedFullStatementWithPVD
		err                    string
	}{
		{
			description: "unknown_relay_parent",
			perRelayParent: func() map[common.Hash]*perRelayParentState {
				return map[common.Hash]*perRelayParentState{}
			},
			signedStatementWithPVD: parachaintypes.SignedFullStatementWithPVD{},
			err:                    errStatementForUnknownRelayParent.Error(),
		},
		{
			description: "nil_relay_parent",
			perRelayParent: func() map[common.Hash]*perRelayParentState {
				return map[common.Hash]*perRelayParentState{
					relayParent: nil,
				}
			},
			signedStatementWithPVD: parachaintypes.SignedFullStatementWithPVD{},
			err:                    errNilRelayParentState.Error(),
		},
		{
			description: "getting_error_importing_statement",
			perRelayParent: func() map[common.Hash]*perRelayParentState {
				return map[common.Hash]*perRelayParentState{
					relayParent: {},
				}
			},
			signedStatementWithPVD: parachaintypes.SignedFullStatementWithPVD{},
			err:                    scale.ErrUnsupportedVaryingDataTypeValue.Error(),
		},
		{
			description: "getting_nil_summary_of_import_statement",
			perRelayParent: func() map[common.Hash]*perRelayParentState {
				ctrl := gomock.NewController(t)
				mockTable := NewMockTable(ctrl)

				mockTable.EXPECT().importStatement(
					gomock.AssignableToTypeOf(new(tableContext)),
					gomock.AssignableToTypeOf(parachaintypes.GroupIndex(0)),
					gomock.AssignableToTypeOf(parachaintypes.SignedFullStatement{}),
				).Return(nil, nil)
				mockTable.EXPECT().drainMisbehaviors().
					Return(map[parachaintypes.ValidatorIndex][]parachaintypes.Misbehaviour{})

				return map[common.Hash]*perRelayParentState{
					relayParent: {
						table:             mockTable,
						numOfCores:        2,
						claimQueue:        claimQueueTestData(t),
						validatorToGroup:  validatorToGroupTestData(t),
						groupRotationInfo: groupRotationInfoTestData(t),
					},
				}
			},
			signedStatementWithPVD: parachaintypes.SignedFullStatementWithPVD{
				SignedFullStatement: parachaintypes.SignedFullStatement{
					Payload: statementVDTValid,
				},
			},
			err: "",
		},
		{
			description: "paraId_is_not_assigned_to_the_local_validator",
			perRelayParent: func() map[common.Hash]*perRelayParentState {
				ctrl := gomock.NewController(t)
				mockTable := NewMockTable(ctrl)

				mockTable.EXPECT().importStatement(
					gomock.AssignableToTypeOf(new(tableContext)),
					gomock.AssignableToTypeOf(parachaintypes.GroupIndex(0)),
					gomock.AssignableToTypeOf(parachaintypes.SignedFullStatement{}),
				).Return(&Summary{
					GroupID: 4,
				}, nil)
				mockTable.EXPECT().drainMisbehaviors().
					Return(map[parachaintypes.ValidatorIndex][]parachaintypes.Misbehaviour{})
				mockTable.EXPECT().attestedCandidate(
					gomock.AssignableToTypeOf(parachaintypes.CandidateHash{}),
					gomock.AssignableToTypeOf(new(tableContext)),
					gomock.AssignableToTypeOf(uint32(0)),
				).Return(nil, errors.New("could not get attested candidate from table"))

				return map[common.Hash]*perRelayParentState{
					relayParent: {
						table:        mockTable,
						assignedCore: &parachaintypes.CoreIndex{Index: 1},
						fallbacks: map[parachaintypes.CandidateHash]attestingData{
							candidateHash: {},
						},
						tableContext:      dummyTableContext(t),
						numOfCores:        2,
						claimQueue:        claimQueueTestData(t),
						validatorToGroup:  validatorToGroupTestData(t),
						groupRotationInfo: groupRotationInfoTestData(t),
					},
				}
			},
			signedStatementWithPVD: parachaintypes.SignedFullStatementWithPVD{
				SignedFullStatement: parachaintypes.SignedFullStatement{
					Payload: statementVDTValid,
				},
			},
			err: "",
		},
		{
			description: "statementVDT_set_to_valid_and_candidate_not_in_fallbacks",
			perRelayParent: func() map[common.Hash]*perRelayParentState {
				ctrl := gomock.NewController(t)
				mockTable := NewMockTable(ctrl)

				mockTable.EXPECT().importStatement(
					gomock.AssignableToTypeOf(new(tableContext)),
					gomock.AssignableToTypeOf(parachaintypes.GroupIndex(0)),
					gomock.AssignableToTypeOf(parachaintypes.SignedFullStatement{}),
				).Return(&Summary{
					Candidate: candidateHash,
					GroupID:   4,
				}, nil)
				mockTable.EXPECT().drainMisbehaviors().
					Return(map[parachaintypes.ValidatorIndex][]parachaintypes.Misbehaviour{})
				mockTable.EXPECT().attestedCandidate(
					gomock.AssignableToTypeOf(parachaintypes.CandidateHash{}),
					gomock.AssignableToTypeOf(new(tableContext)),
					gomock.AssignableToTypeOf(uint32(0)),
				).Return(new(attestedCandidate), nil)

				return map[common.Hash]*perRelayParentState{
					relayParent: {
						assignedCore:      &parachaintypes.CoreIndex{Index: 4},
						table:             mockTable,
						backed:            map[parachaintypes.CandidateHash]bool{},
						fallbacks:         map[parachaintypes.CandidateHash]attestingData{},
						numOfCores:        2,
						claimQueue:        claimQueueTestData(t),
						validatorToGroup:  validatorToGroupTestData(t),
						groupRotationInfo: groupRotationInfoTestData(t),
					},
				}
			},
			signedStatementWithPVD: parachaintypes.SignedFullStatementWithPVD{
				SignedFullStatement: parachaintypes.SignedFullStatement{
					Payload: statementVDTValid,
				},
			},
			err: errFallbackNotAvailable.Error(),
		},
		{
			description: "statementVDT_set_to_valid_also_same_validatorIndex_in_tableContext_and_signedStatement",
			perRelayParent: func() map[common.Hash]*perRelayParentState {
				ctrl := gomock.NewController(t)
				mockTable := NewMockTable(ctrl)

				mockTable.EXPECT().importStatement(
					gomock.AssignableToTypeOf(new(tableContext)),
					gomock.AssignableToTypeOf(parachaintypes.GroupIndex(0)),
					gomock.AssignableToTypeOf(parachaintypes.SignedFullStatement{}),
				).Return(&Summary{
					Candidate: candidateHash,
					GroupID:   4,
				}, nil)
				mockTable.EXPECT().drainMisbehaviors().
					Return(map[parachaintypes.ValidatorIndex][]parachaintypes.Misbehaviour{})
				mockTable.EXPECT().attestedCandidate(
					gomock.AssignableToTypeOf(parachaintypes.CandidateHash{}),
					gomock.AssignableToTypeOf(new(tableContext)),
					gomock.AssignableToTypeOf(uint32(0)),
				).Return(new(attestedCandidate), nil)

				return map[common.Hash]*perRelayParentState{
					relayParent: {
						assignedCore: &parachaintypes.CoreIndex{Index: 4},
						table:        mockTable,
						tableContext: dummyTableContext(t),
						backed:       map[parachaintypes.CandidateHash]bool{},
						fallbacks: map[parachaintypes.CandidateHash]attestingData{
							candidateHash: {},
						},
						numOfCores:        2,
						claimQueue:        claimQueueTestData(t),
						validatorToGroup:  validatorToGroupTestData(t),
						groupRotationInfo: groupRotationInfoTestData(t),
					},
				}
			},
			signedStatementWithPVD: parachaintypes.SignedFullStatementWithPVD{
				SignedFullStatement: parachaintypes.SignedFullStatement{
					Payload:        statementVDTValid,
					ValidatorIndex: 1,
				},
			},
			err: "",
		},
		{
			description: "statementVDT_set_to_valid_and_validation_job_already_running_for_candidate",
			perRelayParent: func() map[common.Hash]*perRelayParentState {
				ctrl := gomock.NewController(t)
				mockTable := NewMockTable(ctrl)

				mockTable.EXPECT().importStatement(
					gomock.AssignableToTypeOf(new(tableContext)),
					gomock.AssignableToTypeOf(parachaintypes.GroupIndex(0)),
					gomock.AssignableToTypeOf(parachaintypes.SignedFullStatement{}),
				).Return(&Summary{
					Candidate: candidateHash,
					GroupID:   4,
				}, nil)
				mockTable.EXPECT().drainMisbehaviors().
					Return(map[parachaintypes.ValidatorIndex][]parachaintypes.Misbehaviour{})
				mockTable.EXPECT().attestedCandidate(
					gomock.AssignableToTypeOf(parachaintypes.CandidateHash{}),
					gomock.AssignableToTypeOf(new(tableContext)),
					gomock.AssignableToTypeOf(uint32(0)),
				).Return(new(attestedCandidate), nil)

				return map[common.Hash]*perRelayParentState{
					relayParent: {
						assignedCore: &parachaintypes.CoreIndex{Index: 4},
						table:        mockTable,
						tableContext: dummyTableContext(t),
						backed:       map[parachaintypes.CandidateHash]bool{},
						fallbacks: map[parachaintypes.CandidateHash]attestingData{
							candidateHash: {},
						},
						awaitingValidation: map[parachaintypes.CandidateHash]bool{
							candidateHash: true,
						},
						numOfCores:        2,
						claimQueue:        claimQueueTestData(t),
						validatorToGroup:  validatorToGroupTestData(t),
						groupRotationInfo: groupRotationInfoTestData(t),
					},
				}
			},
			signedStatementWithPVD: parachaintypes.SignedFullStatementWithPVD{
				SignedFullStatement: parachaintypes.SignedFullStatement{
					Payload:        statementVDTValid,
					ValidatorIndex: 2,
				},
			},
			err: "",
		},
		{
			description: "statementVDT_set_to_valid_and_start_validation_job_for_candidate",
			perRelayParent: func() map[common.Hash]*perRelayParentState {
				ctrl := gomock.NewController(t)
				mockTable := NewMockTable(ctrl)

				mockTable.EXPECT().importStatement(
					gomock.AssignableToTypeOf(new(tableContext)),
					gomock.AssignableToTypeOf(parachaintypes.GroupIndex(0)),
					gomock.AssignableToTypeOf(parachaintypes.SignedFullStatement{}),
				).Return(&Summary{
					Candidate: candidateHash,
					GroupID:   4,
				}, nil)
				mockTable.EXPECT().drainMisbehaviors().
					Return(map[parachaintypes.ValidatorIndex][]parachaintypes.Misbehaviour{})
				mockTable.EXPECT().attestedCandidate(
					gomock.AssignableToTypeOf(parachaintypes.CandidateHash{}),
					gomock.AssignableToTypeOf(new(tableContext)),
					gomock.AssignableToTypeOf(uint32(0)),
				).Return(new(attestedCandidate), nil)

				return map[common.Hash]*perRelayParentState{
					relayParent: {
						assignedCore: &parachaintypes.CoreIndex{Index: 4},
						table:        mockTable,
						tableContext: dummyTableContext(t),
						backed:       map[parachaintypes.CandidateHash]bool{},
						fallbacks: map[parachaintypes.CandidateHash]attestingData{
							candidateHash: {
								candidate: getDummyCommittedCandidateReceipt(t).ToPlain(),
							},
						},
						awaitingValidation: map[parachaintypes.CandidateHash]bool{},
						issuedStatements: map[parachaintypes.CandidateHash]bool{
							candidateHash: true,
						},
						numOfCores:        2,
						claimQueue:        claimQueueTestData(t),
						validatorToGroup:  validatorToGroupTestData(t),
						groupRotationInfo: groupRotationInfoTestData(t),
					},
				}
			},

			perCandidate: map[parachaintypes.CandidateHash]*perCandidateState{
				candidateHash: {},
			},
			signedStatementWithPVD: parachaintypes.SignedFullStatementWithPVD{
				SignedFullStatement: parachaintypes.SignedFullStatement{
					Payload:        statementVDTValid,
					ValidatorIndex: 2,
				},
			},
			err: "",
		},
		{
			description: "statementVDT_set_to_seconded_and_error_getting_candidate_from_table",
			perRelayParent: func() map[common.Hash]*perRelayParentState {
				ctrl := gomock.NewController(t)
				mockTable := NewMockTable(ctrl)

				mockTable.EXPECT().importStatement(
					gomock.AssignableToTypeOf(new(tableContext)),
					gomock.AssignableToTypeOf(parachaintypes.GroupIndex(0)),
					gomock.AssignableToTypeOf(parachaintypes.SignedFullStatement{}),
				).Return(&Summary{
					Candidate: candidateHash,
					GroupID:   4,
				}, nil)
				mockTable.EXPECT().drainMisbehaviors().
					Return(map[parachaintypes.ValidatorIndex][]parachaintypes.Misbehaviour{})
				mockTable.EXPECT().attestedCandidate(
					gomock.AssignableToTypeOf(parachaintypes.CandidateHash{}),
					gomock.AssignableToTypeOf(new(tableContext)),
					gomock.AssignableToTypeOf(uint32(0)),
				).Return(new(attestedCandidate), nil)
				mockTable.EXPECT().getCommittedCandidateReceipt(
					gomock.AssignableToTypeOf(parachaintypes.CandidateHash{}),
				).Return(
					parachaintypes.CommittedCandidateReceiptV2{},
					errors.New("could not get candidate from table"),
				)

				return map[common.Hash]*perRelayParentState{
					relayParent: {
						assignedCore: &parachaintypes.CoreIndex{Index: 4},
						table:        mockTable,
						backed: map[parachaintypes.CandidateHash]bool{
							candidateHash: true,
						},
						fallbacks: map[parachaintypes.CandidateHash]attestingData{},
						issuedStatements: map[parachaintypes.CandidateHash]bool{
							candidateHash: true,
						},
						numOfCores:        2,
						claimQueue:        claimQueueTestData(t),
						validatorToGroup:  validatorToGroupTestData(t),
						groupRotationInfo: groupRotationInfoTestData(t),
					},
				}
			},
			signedStatementWithPVD: secondedSignedFullStatementWithPVD(t, statementVDTSeconded),
			err:                    "could not get candidate from table",
		},
		{
			description: "statementVDT_set_to_seconded_and_successfully_get_candidate_from_table",
			perRelayParent: func() map[common.Hash]*perRelayParentState {
				ctrl := gomock.NewController(t)
				mockTable := NewMockTable(ctrl)

				mockTable.EXPECT().importStatement(
					gomock.AssignableToTypeOf(new(tableContext)),
					gomock.AssignableToTypeOf(parachaintypes.GroupIndex(0)),
					gomock.AssignableToTypeOf(parachaintypes.SignedFullStatement{}),
				).Return(&Summary{
					Candidate: candidateHash,
					GroupID:   4,
				}, nil)
				mockTable.EXPECT().drainMisbehaviors().
					Return(map[parachaintypes.ValidatorIndex][]parachaintypes.Misbehaviour{})
				mockTable.EXPECT().attestedCandidate(
					gomock.AssignableToTypeOf(parachaintypes.CandidateHash{}),
					gomock.AssignableToTypeOf(new(tableContext)),
					gomock.AssignableToTypeOf(uint32(0)),
				).Return(new(attestedCandidate), nil)
				mockTable.EXPECT().getCommittedCandidateReceipt(
					gomock.AssignableToTypeOf(parachaintypes.CandidateHash{}),
				).Return(dummyCCR, nil)

				return map[common.Hash]*perRelayParentState{
					relayParent: {
						assignedCore: &parachaintypes.CoreIndex{Index: 4},
						table:        mockTable,
						tableContext: tableContext{validator: &parachaintypes.Validator{Disabled: false}},
						backed: map[parachaintypes.CandidateHash]bool{
							candidateHash: true,
						},
						fallbacks: map[parachaintypes.CandidateHash]attestingData{},
						issuedStatements: map[parachaintypes.CandidateHash]bool{
							candidateHash: true,
						},
						numOfCores:        2,
						claimQueue:        claimQueueTestData(t),
						validatorToGroup:  validatorToGroupTestData(t),
						groupRotationInfo: groupRotationInfoTestData(t),
					},
				}

			},
			signedStatementWithPVD: secondedSignedFullStatementWithPVD(t, statementVDTSeconded),
			err:                    "",
		},
	}
	for _, c := range testCases {
		c := c
		t.Run(c.description, func(t *testing.T) {
			t.Parallel()

			subSystemToOverseer := make(chan any)

			backing := CandidateBacking{
				SubSystemToOverseer: subSystemToOverseer,
				perRelayParent:      c.perRelayParent(),
				perCandidate: func() map[parachaintypes.CandidateHash]*perCandidateState {
					if c.perCandidate == nil {
						return map[parachaintypes.CandidateHash]*perCandidateState{}
					}
					return c.perCandidate
				}(),
			}

			defer close(subSystemToOverseer)
			go mockOverseer(t, subSystemToOverseer)

			err := backing.handleStatementMessage(relayParent, c.signedStatementWithPVD, chRelayParentAndCommand)
			if c.err == "" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, c.err)
			}
		})
	}
}
