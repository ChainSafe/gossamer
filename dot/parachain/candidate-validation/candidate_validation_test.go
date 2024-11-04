// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package candidatevalidation

import (
	"context"
	"fmt"
	"os"
	"testing"

	parachainruntime "github.com/ChainSafe/gossamer/dot/parachain/runtime"
	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

var (
	povHashMismatch  = PoVHashMismatch
	paramsTooLarge   = ParamsTooLarge
	codeHashMismatch = CodeHashMismatch
	badSignature     = BadSignature
	invalidOutputs   = InvalidOutputs
	badParent        = BadParent
)

func createTestCandidateReceiptAndValidationCodeWParaId(t *testing.T, id parachaintypes.ParaID) (
	parachaintypes.CandidateReceipt, parachaintypes.ValidationCode) {
	t.Helper()
	// this wasm was achieved by building polkadot's adder test parachain
	runtimeFilePath := "./testdata/test_parachain_adder.wasm"
	validationCodeBytes, err := os.ReadFile(runtimeFilePath)
	require.NoError(t, err)

	validationCode := parachaintypes.ValidationCode(validationCodeBytes)

	validationCodeHashV := common.MustBlake2bHash(validationCodeBytes)

	collatorKeypair, err := sr25519.GenerateKeypair()
	require.NoError(t, err)

	descriptor := makeValidCandidateDescriptor(t, id,
		common.MustHexToHash("0xded542bacb3ca6c033a57676f94ae7c8f36834511deb44e3164256fd3b1c0de0"),
		common.MustHexToHash("0x690d8f252ef66ab0f969c3f518f90012b849aa5ac94e1752c5e5ae5a8996de37"),
		common.MustHexToHash("0xb608991ffc48dd405fd4b10e92eaebe2b5a2eedf44d0c3efb8997fdee8bebed9"),
		parachaintypes.ValidationCodeHash(validationCodeHashV),
		common.MustHexToHash("0x657a011336002a7f2acd5db97d34b9b703c04cadcb63ad82c7658b04fb42f3de"),
		common.MustHexToHash("0xc07f658163e93c45a6f0288d229698f09c1252e41076f4caa71c8cbc12f118a1"), *collatorKeypair)

	candidateReceipt := parachaintypes.CandidateReceipt{
		Descriptor:      descriptor,
		CommitmentsHash: common.MustHexToHash("0x4ddce2e9ed80f386cdbba4b42f5de76957d5fbf9f093258d6048e9218d1fe98d"),
	}

	return candidateReceipt, validationCode
}

func makeValidCandidateDescriptor(t *testing.T, paraID parachaintypes.ParaID, relayParent common.Hash,
	persistedValidationDataHash common.Hash, povHash common.Hash,
	validationCodeHash parachaintypes.ValidationCodeHash, paraHead common.Hash, erasureRoot common.Hash,
	collator sr25519.Keypair,
) parachaintypes.CandidateDescriptor {
	collatorID, err := sr25519.NewPublicKey(collator.Public().Encode())
	require.NoError(t, err)

	descriptor := parachaintypes.CandidateDescriptor{
		ParaID:                      paraID,
		RelayParent:                 relayParent,
		Collator:                    collatorID.AsBytes(),
		PersistedValidationDataHash: persistedValidationDataHash,
		PovHash:                     povHash,
		ErasureRoot:                 erasureRoot,
		ParaHead:                    paraHead,
		ValidationCodeHash:          validationCodeHash,
	}
	payload, err := descriptor.CreateSignaturePayload()
	require.NoError(t, err)

	signatureBytes, err := collator.Sign(payload)
	require.NoError(t, err)

	signature := [sr25519.SignatureLength]byte{}
	copy(signature[:], signatureBytes)

	descriptor.Signature = signature

	return descriptor
}

type HeadDataInAdderParachain struct {
	Number     uint64
	ParentHash [32]byte
	PostState  [32]byte
}

type BlockDataInAdderParachain struct {
	State uint64
	Add   uint64
}

func TestCandidateValidation_wasm_invalid_magic_number(t *testing.T) {
	validationCode := parachaintypes.ValidationCode{1, 2, 3, 4, 5, 6, 7, 8}
	parachainRuntimeInstance, err := parachainruntime.SetupVM(validationCode)
	require.EqualError(t, err, "creating instance: creating runtime instance: invalid magic number")
	require.Emptyf(t, parachainRuntimeInstance, "parachainRuntimeInstance should be empty")
}

func TestCandidateValidation_processMessageValidateFromExhaustive(t *testing.T) {
	t.Parallel()

	candidateReceipt, validationCode := createTestCandidateReceiptAndValidationCodeWParaId(t, 1000)
	candidateReceipt2 := candidateReceipt
	candidateReceipt2.Descriptor.PovHash = common.MustHexToHash(
		"0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef")

	bd, err := scale.Marshal(BlockDataInAdderParachain{
		State: uint64(1),
		Add:   uint64(1),
	})
	require.NoError(t, err)

	pov := parachaintypes.PoV{
		BlockData: bd,
	}

	encodedState, err := scale.Marshal(uint64(1))
	require.NoError(t, err)
	postState, err := common.Keccak256(encodedState)
	require.NoError(t, err)

	hd, err := scale.Marshal(HeadDataInAdderParachain{
		Number:     uint64(1),
		ParentHash: common.MustHexToHash("0x0102030405060708090001020304050607080900010203040506070809000102"),
		PostState:  postState,
	})
	require.NoError(t, err)

	overseerToSubsystem := make(chan any)
	sender := make(chan parachaintypes.OverseerFuncRes[ValidationResult])
	candidateValidationSubsystem := CandidateValidation{
		pvfHost: newValidationHost(),
	}

	t.Cleanup(candidateValidationSubsystem.Stop)

	ctx := context.Background()
	go candidateValidationSubsystem.Run(ctx, overseerToSubsystem)

	tests := map[string]struct {
		msg  ValidateFromExhaustive
		want parachaintypes.OverseerFuncRes[ValidationResult]
	}{
		"invalid_pov_hash": {
			msg: ValidateFromExhaustive{
				PersistedValidationData: parachaintypes.PersistedValidationData{
					ParentHead:             parachaintypes.HeadData{Data: hd},
					RelayParentNumber:      uint32(1),
					RelayParentStorageRoot: common.MustHexToHash("0x50c969706800c0e9c3c4565dc2babb25e4a73d1db0dee1bcf7745535a32e7ca1"),
					MaxPovSize:             uint32(2048),
				},
				ValidationCode:   validationCode,
				CandidateReceipt: candidateReceipt2,
				PoV:              pov,
				Ch:               sender,
			},
			want: parachaintypes.OverseerFuncRes[ValidationResult]{
				Data: ValidationResult{
					Invalid: &povHashMismatch,
				},
				Err: nil,
			},
		},
		"invalid_pov_size": {
			msg: ValidateFromExhaustive{
				PersistedValidationData: parachaintypes.PersistedValidationData{
					ParentHead:             parachaintypes.HeadData{Data: hd},
					RelayParentNumber:      uint32(1),
					RelayParentStorageRoot: common.MustHexToHash("0x50c969706800c0e9c3c4565dc2babb25e4a73d1db0dee1bcf7745535a32e7ca1"),
					MaxPovSize:             uint32(10),
				},
				ValidationCode:   validationCode,
				CandidateReceipt: candidateReceipt,
				PoV:              pov,
				Ch:               sender,
			},
			want: parachaintypes.OverseerFuncRes[ValidationResult]{
				Data: ValidationResult{
					Invalid: &paramsTooLarge,
				},
			},
		},
		"code_mismatch": {
			msg: ValidateFromExhaustive{
				PersistedValidationData: parachaintypes.PersistedValidationData{
					ParentHead:             parachaintypes.HeadData{Data: hd},
					RelayParentNumber:      uint32(1),
					RelayParentStorageRoot: common.MustHexToHash("0x50c969706800c0e9c3c4565dc2babb25e4a73d1db0dee1bcf7745535a32e7ca1"),
					MaxPovSize:             uint32(2048),
				},
				ValidationCode:   []byte{1, 2, 3, 4, 5, 6, 7, 8},
				CandidateReceipt: candidateReceipt,
				PoV:              pov,
				Ch:               sender,
			},
			want: parachaintypes.OverseerFuncRes[ValidationResult]{
				Data: ValidationResult{
					Invalid: &codeHashMismatch,
				},
			},
		},
		"happy_path": {
			msg: ValidateFromExhaustive{
				PersistedValidationData: parachaintypes.PersistedValidationData{
					ParentHead:             parachaintypes.HeadData{Data: hd},
					RelayParentNumber:      uint32(1),
					RelayParentStorageRoot: common.MustHexToHash("0x50c969706800c0e9c3c4565dc2babb25e4a73d1db0dee1bcf7745535a32e7ca1"),
					MaxPovSize:             uint32(2048),
				},
				ValidationCode:   validationCode,
				CandidateReceipt: candidateReceipt,
				PoV:              pov,
				Ch:               sender,
			},
			want: parachaintypes.OverseerFuncRes[ValidationResult]{
				Data: ValidationResult{
					Valid: &Valid{
						CandidateCommitments: parachaintypes.CandidateCommitments{
							HeadData: parachaintypes.HeadData{Data: []byte{2, 0, 0, 0, 0, 0, 0, 0, 123,
								207, 206, 8, 219, 227, 136, 82, 236, 169, 14, 100, 45, 100, 31, 177, 154, 160, 220, 245,
								59, 106, 76, 168, 122, 109, 164, 169, 22, 46, 144, 39, 103, 92, 31, 78, 66, 72, 252, 64,
								24, 194, 129, 162, 128, 1, 77, 147, 200, 229, 189, 242, 111, 198, 236, 139, 16, 143, 19,
								245, 113, 233, 138, 210}},
							HrmpWatermark: 1,
						},
						PersistedValidationData: parachaintypes.PersistedValidationData{
							ParentHead: parachaintypes.HeadData{Data: []byte{1, 0, 0, 0, 0, 0, 0, 0, 1,
								2, 3, 4, 5, 6, 7, 8, 9, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 0, 1, 2, 3, 4, 5, 6, 7,
								8, 9, 0, 1, 2, 48, 246, 146, 178, 86, 226, 64, 9,
								188, 179, 77, 14, 232, 77, 167, 60, 41, 138, 250, 204, 9, 36, 224, 17, 5, 226, 235,
								15, 1, 168, 127, 226}},
							RelayParentNumber: 1,
							RelayParentStorageRoot: common.MustHexToHash(
								"0x50c969706800c0e9c3c4565dc2babb25e4a73d1db0dee1bcf7745535a32e7ca1"),
							MaxPovSize: 2048,
						},
					},
				},
			},
		},
	}
	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			overseerToSubsystem <- tt.msg
			result := <-sender
			require.Equal(t, tt.want, result)
		})
	}
}

func TestCandidateValidation_processMessageValidateFromChainState(t *testing.T) {
	t.Parallel()

	candidateReceipt, validationCode := createTestCandidateReceiptAndValidationCodeWParaId(t, 1000)
	candidateReceipt2 := candidateReceipt
	candidateReceipt2.Descriptor.PovHash = common.MustHexToHash(
		"0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef")
	candidateReceipt2.Descriptor.ParaID = 2

	candidateReceipt3 := candidateReceipt
	candidateReceipt3.Descriptor.ParaID = 3

	candidateReceipt4 := candidateReceipt
	candidateReceipt4.Descriptor.ParaID = 4
	candidateReceipt4.Descriptor.ValidationCodeHash = parachaintypes.ValidationCodeHash(common.MustHexToHash(
		"0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"))

	candidateReceipt5 := candidateReceipt
	candidateReceipt5.Descriptor.ParaID = 5

	candidateReceipt6, _ := createTestCandidateReceiptAndValidationCodeWParaId(t, 6)

	candidateReceipt7 := candidateReceipt
	candidateReceipt7.Descriptor.ParaID = 7

	// NOTE: adder parachain internally compares postState with bd.State in it's validate_block,
	// so following is necessary.
	encodedState, err := scale.Marshal(uint64(1))
	require.NoError(t, err)
	postState, err := common.Keccak256(encodedState)
	require.NoError(t, err)

	hd, err := scale.Marshal(HeadDataInAdderParachain{
		Number:     uint64(1),
		ParentHash: common.MustHexToHash("0x0102030405060708090001020304050607080900010203040506070809000102"),
		PostState:  postState,
	})
	require.NoError(t, err)

	expectedPersistedValidationData := parachaintypes.PersistedValidationData{
		ParentHead:             parachaintypes.HeadData{Data: hd},
		RelayParentNumber:      uint32(1),
		RelayParentStorageRoot: common.MustHexToHash("0x50c969706800c0e9c3c4565dc2babb25e4a73d1db0dee1bcf7745535a32e7ca1"),
		MaxPovSize:             uint32(2048),
	}

	expectedPersistedValidationDataSmallMax := parachaintypes.PersistedValidationData{
		ParentHead:             parachaintypes.HeadData{Data: hd},
		RelayParentNumber:      uint32(1),
		RelayParentStorageRoot: common.MustHexToHash("0x50c969706800c0e9c3c4565dc2babb25e4a73d1db0dee1bcf7745535a32e7ca1"),
		MaxPovSize:             uint32(10),
	}

	validCandidateCommitments := parachaintypes.CandidateCommitments{
		HeadData: parachaintypes.HeadData{Data: []byte{2, 0, 0, 0, 0, 0, 0, 0, 123, 207, 206, 8, 219, 227,
			136, 82, 236, 169, 14, 100, 45, 100, 31, 177, 154, 160, 220, 245, 59, 106, 76, 168, 122, 109,
			164, 169, 22, 46, 144, 39, 103, 92, 31, 78, 66, 72, 252, 64, 24, 194, 129, 162, 128, 1, 77, 147,
			200, 229, 189, 242, 111, 198, 236, 139, 16, 143, 19, 245, 113, 233, 138, 210}},
		ProcessedDownwardMessages: 0,
		HrmpWatermark:             1,
	}

	bd, err := scale.Marshal(BlockDataInAdderParachain{
		State: uint64(1),
		Add:   uint64(1),
	})
	require.NoError(t, err)
	pov := parachaintypes.PoV{
		BlockData: bd,
	}

	setup := func(
		t *testing.T,
		msg *ValidateFromChainState,
		configMockInstance func(*MockInstance),
	) (*CandidateValidation, chan any) {

		ctrl := gomock.NewController(t)
		mockInstance := NewMockInstance(ctrl)
		configMockInstance(mockInstance)

		msg.Ch = make(chan parachaintypes.OverseerFuncRes[ValidationResult])

		mockBlockState := NewMockBlockState(ctrl)
		mockBlockState.EXPECT().GetRuntime(common.MustHexToHash(
			"0xded542bacb3ca6c033a57676f94ae7c8f36834511deb44e3164256fd3b1c0de0")).Return(mockInstance, nil).Times(1)

		cv := CandidateValidation{
			pvfHost:    newValidationHost(),
			BlockState: mockBlockState,
		}

		t.Cleanup(cv.Stop)
		toSubsystem := make(chan any)
		return &cv, toSubsystem
	}

	tests := map[string]struct {
		msg                ValidateFromChainState
		want               *ValidationResult
		expectedError      error
		configMockInstance func(*MockInstance)
	}{
		"invalid_pov_hash": {
			msg: ValidateFromChainState{
				CandidateReceipt: candidateReceipt2,
				Pov:              pov,
			},
			want: &ValidationResult{
				Invalid: &povHashMismatch,
			},
			configMockInstance: func(mi *MockInstance) {
				mi.EXPECT().
					ParachainHostPersistedValidationData(
						parachaintypes.ParaID(2),
						gomock.AssignableToTypeOf(parachaintypes.OccupiedCoreAssumption{})).
					Return(&expectedPersistedValidationData, nil)
				mi.EXPECT().
					ParachainHostValidationCode(parachaintypes.ParaID(2),
						gomock.AssignableToTypeOf(parachaintypes.OccupiedCoreAssumption{})).
					Return(&validationCode, nil)
			},
		},
		"invalid_pov_size": {
			msg: ValidateFromChainState{
				CandidateReceipt: candidateReceipt3,
				Pov:              pov,
			},
			want: &ValidationResult{
				Invalid: &paramsTooLarge,
			},
			configMockInstance: func(mi *MockInstance) {
				mi.EXPECT().
					ParachainHostPersistedValidationData(
						parachaintypes.ParaID(3),
						gomock.AssignableToTypeOf(parachaintypes.OccupiedCoreAssumption{})).
					Return(&expectedPersistedValidationDataSmallMax, nil)
				mi.EXPECT().
					ParachainHostValidationCode(parachaintypes.ParaID(3),
						gomock.AssignableToTypeOf(parachaintypes.OccupiedCoreAssumption{})).
					Return(&validationCode, nil)
			},
		},
		"code_mismatch": {
			msg: ValidateFromChainState{
				CandidateReceipt: candidateReceipt4,
				Pov:              pov,
			},
			want: &ValidationResult{
				Invalid: &codeHashMismatch,
			},
			configMockInstance: func(mi *MockInstance) {
				mi.EXPECT().
					ParachainHostPersistedValidationData(
						parachaintypes.ParaID(4),
						gomock.AssignableToTypeOf(parachaintypes.OccupiedCoreAssumption{})).
					Return(&expectedPersistedValidationData, nil)
				mi.EXPECT().
					ParachainHostValidationCode(parachaintypes.ParaID(4),
						gomock.AssignableToTypeOf(parachaintypes.OccupiedCoreAssumption{})).
					Return(&validationCode, nil)
			},
		},
		"bad_signature": {
			msg: ValidateFromChainState{
				CandidateReceipt: candidateReceipt5,
				Pov:              pov,
			},
			want: &ValidationResult{
				Invalid: &badSignature,
			},
			configMockInstance: func(mi *MockInstance) {
				mi.EXPECT().
					ParachainHostPersistedValidationData(
						parachaintypes.ParaID(5),
						gomock.AssignableToTypeOf(parachaintypes.OccupiedCoreAssumption{})).
					Return(&expectedPersistedValidationData, nil)
				mi.EXPECT().
					ParachainHostValidationCode(parachaintypes.ParaID(5),
						gomock.AssignableToTypeOf(parachaintypes.OccupiedCoreAssumption{})).
					Return(&validationCode, nil)
			},
		},
		"invalid_outputs": {
			msg: ValidateFromChainState{
				CandidateReceipt: candidateReceipt6,
				Pov:              pov,
			},
			want: &ValidationResult{
				Invalid: &invalidOutputs,
			},
			configMockInstance: func(mi *MockInstance) {
				mi.EXPECT().
					ParachainHostPersistedValidationData(parachaintypes.ParaID(6),
						gomock.AssignableToTypeOf(parachaintypes.OccupiedCoreAssumption{})).
					Return(&expectedPersistedValidationData, nil)
				mi.EXPECT().
					ParachainHostValidationCode(parachaintypes.ParaID(6),
						gomock.AssignableToTypeOf(parachaintypes.OccupiedCoreAssumption{})).
					Return(&validationCode, nil)
				mi.EXPECT().ParachainHostCheckValidationOutputs(parachaintypes.ParaID(6),
					validCandidateCommitments).Return(false, nil)
			},
		},
		"bad_parent": {
			msg: ValidateFromChainState{
				CandidateReceipt: candidateReceipt7,
				Pov:              pov,
			},
			want: &ValidationResult{
				Invalid: &badParent,
			},
			configMockInstance: func(mi *MockInstance) {
				mi.EXPECT().
					ParachainHostPersistedValidationData(parachaintypes.ParaID(7), gomock.AssignableToTypeOf(parachaintypes.
						OccupiedCoreAssumption{})).
					Return(nil, nil)
				mi.EXPECT().
					ParachainHostValidationCode(parachaintypes.ParaID(7),
						gomock.AssignableToTypeOf(parachaintypes.OccupiedCoreAssumption{})).
					Return(&validationCode, nil)
			},
		},
		"happy_path": {
			msg: ValidateFromChainState{
				CandidateReceipt: candidateReceipt,
				Pov:              pov,
			},
			want: &ValidationResult{
				Valid: &Valid{
					CandidateCommitments: validCandidateCommitments,
					PersistedValidationData: parachaintypes.PersistedValidationData{
						ParentHead: parachaintypes.HeadData{Data: []byte{1, 0, 0, 0, 0, 0, 0, 0, 1,
							2, 3, 4, 5, 6, 7, 8, 9, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 0, 1, 2, 3, 4, 5, 6, 7,
							8, 9, 0, 1, 2, 48, 246, 146, 178, 86, 226, 64, 9,
							188, 179, 77, 14, 232, 77, 167, 60, 41, 138, 250, 204, 9, 36, 224, 17, 5, 226, 235,
							15, 1, 168, 127, 226}},
						RelayParentNumber: 1,
						RelayParentStorageRoot: common.MustHexToHash(
							"0x50c969706800c0e9c3c4565dc2babb25e4a73d1db0dee1bcf7745535a32e7ca1"),
						MaxPovSize: 2048,
					},
				},
			},
			configMockInstance: func(mi *MockInstance) {
				mi.EXPECT().
					ParachainHostPersistedValidationData(
						parachaintypes.ParaID(1000),
						gomock.AssignableToTypeOf(parachaintypes.OccupiedCoreAssumption{})).
					Return(&expectedPersistedValidationData, nil)
				mi.EXPECT().
					ParachainHostValidationCode(parachaintypes.ParaID(1000),
						gomock.AssignableToTypeOf(parachaintypes.OccupiedCoreAssumption{})).
					Return(&validationCode, nil)
				mi.EXPECT().ParachainHostCheckValidationOutputs(parachaintypes.ParaID(1000),
					validCandidateCommitments).Return(true, nil)

			},
		},
	}
	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			candidateValidationSubsystem, toSubsystem := setup(t, &tt.msg, tt.configMockInstance)

			go candidateValidationSubsystem.Run(context.Background(), toSubsystem)

			toSubsystem <- tt.msg
			result := <-tt.msg.Ch
			require.Equal(t, tt.want, &result.Data)
		})
	}
}

func Test_precheckPvF(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	candidate, validationCode := createTestCandidateReceiptAndValidationCodeWParaId(t, 1000)

	mockInstance := NewMockInstance(ctrl)
	mockInstance.EXPECT().ParachainHostValidationCodeByHash(common.Hash(candidate.Descriptor.ValidationCodeHash)).
		Return(&validationCode, nil)
	mockInstance.EXPECT().ParachainHostSessionIndexForChild().Return(parachaintypes.SessionIndex(1), nil)

	executionParams := parachaintypes.ExecutorParams{}
	timeout := parachaintypes.PvfPrepTimeout{
		PvfPrepTimeoutKind: func() parachaintypes.PvfPrepTimeoutKind {
			kind := parachaintypes.NewPvfPrepTimeoutKind()
			if err := kind.SetValue(parachaintypes.Precheck{}); err != nil {
				panic(err)
			}
			return kind
		}(),
		Millisec: 1000,
	}
	timeoutParam := parachaintypes.NewExecutorParam()
	err := timeoutParam.SetValue(timeout)
	require.NoError(t, err)
	executionParams = append(executionParams, timeoutParam)
	mockInstance.EXPECT().ParachainHostSessionExecutorParams(parachaintypes.SessionIndex(1)).Return(&executionParams,
		nil)

	mockInstanceExecutorError := NewMockInstance(ctrl)
	mockInstanceExecutorError.EXPECT().ParachainHostValidationCodeByHash(common.MustHexToHash("0x04")).Return(
		&parachaintypes.ValidationCode{}, nil)
	mockInstanceExecutorError.EXPECT().ParachainHostSessionIndexForChild().Return(parachaintypes.SessionIndex(2), nil)
	mockInstanceExecutorError.EXPECT().ParachainHostSessionExecutorParams(parachaintypes.SessionIndex(2)).Return(
		nil, fmt.Errorf("executor params not found"))

	mockInstanceShortTimeout := NewMockInstance(ctrl)
	mockInstanceShortTimeout.EXPECT().ParachainHostValidationCodeByHash(common.MustHexToHash("0x0404")).Return(
		&validationCode, nil)
	mockInstanceShortTimeout.EXPECT().ParachainHostSessionIndexForChild().Return(parachaintypes.SessionIndex(3), nil)
	executionParamsShortTimeout := parachaintypes.ExecutorParams{}
	timeoutShort := parachaintypes.PvfPrepTimeout{
		PvfPrepTimeoutKind: func() parachaintypes.PvfPrepTimeoutKind {
			kind := parachaintypes.NewPvfPrepTimeoutKind()
			if err := kind.SetValue(parachaintypes.Precheck{}); err != nil {
				panic(err)
			}
			return kind
		}(),
		Millisec: 1,
	}
	timeoutShortParam := parachaintypes.NewExecutorParam()
	err = timeoutShortParam.SetValue(timeoutShort)
	require.NoError(t, err)
	executionParamsShortTimeout = append(executionParamsShortTimeout, timeoutShortParam)
	mockInstanceShortTimeout.EXPECT().ParachainHostSessionExecutorParams(parachaintypes.SessionIndex(3)).Return(
		&executionParamsShortTimeout, nil)

	mockBlockState := NewMockBlockState(ctrl)
	mockBlockState.EXPECT().GetRuntime(common.MustHexToHash("0x01")).Return(nil, fmt.Errorf("runtime not found"))
	mockBlockState.EXPECT().GetRuntime(common.MustHexToHash("0x02")).Return(mockInstance, nil)
	mockBlockState.EXPECT().GetRuntime(common.MustHexToHash("0x03")).Return(mockInstanceExecutorError, nil)
	mockBlockState.EXPECT().GetRuntime(common.MustHexToHash("0x04")).Return(mockInstanceShortTimeout, nil)

	tests := map[string]struct {
		msg            PreCheck
		expectedResult PreCheckOutcome
		expectedError  error
	}{
		"validation_code_not_found": {
			msg: PreCheck{
				RelayParent: common.MustHexToHash("0x01"),
			},
			expectedResult: PreCheckOutcomeFailed,
			expectedError:  fmt.Errorf("failed to get runtime instance: runtime not found"),
		},
		"invalid_executor_params": {
			msg: PreCheck{
				RelayParent:        common.MustHexToHash("0x03"),
				ValidationCodeHash: parachaintypes.ValidationCodeHash(common.MustHexToHash("0x04")),
			},
			expectedResult: PreCheckOutcomeInvalid,
			expectedError: fmt.Errorf("failed to acquire params for the session, thus voting against: " +
				"executor params not found"),
		},
		"precheck_timeout": {
			msg: PreCheck{
				RelayParent:        common.MustHexToHash("0x04"),
				ValidationCodeHash: parachaintypes.ValidationCodeHash(common.MustHexToHash("0x0404")),
			},
			expectedResult: PreCheckOutcomeFailed,
			expectedError:  fmt.Errorf("failed to precheck: failed to create a new worker: precheck timed out"),
		},
		"happy_path": {
			msg: PreCheck{
				RelayParent:        common.MustHexToHash("0x02"),
				ValidationCodeHash: candidate.Descriptor.ValidationCodeHash,
			},
			expectedResult: PreCheckOutcomeValid,
		},
	}
	for name, tt := range tests {
		tt := tt

		t.Run(name, func(t *testing.T) {
			t.Parallel()

			candidateValidationSubsystem := CandidateValidation{
				pvfHost:    newValidationHost(),
				BlockState: mockBlockState,
			}
			result, err := candidateValidationSubsystem.precheckPvF(tt.msg.RelayParent, tt.msg.ValidationCodeHash)
			require.Equal(t, tt.expectedResult, result)
			if tt.expectedError != nil {
				require.EqualError(t, err, tt.expectedError.Error())
			} else {
				require.NoError(t, err)
			}
		})
	}
}
