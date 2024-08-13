// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package candidatevalidation

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	parachainruntime "github.com/ChainSafe/gossamer/dot/parachain/runtime"
	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func createTestCandidateReceiptAndValidationCode(t *testing.T) (
	parachaintypes.CandidateReceipt, parachaintypes.ValidationCode) {
	// this wasm was achieved by building polkadot's adder test parachain
	runtimeFilePath := "./testdata/test_parachain_adder.wasm"
	validationCodeBytes, err := os.ReadFile(runtimeFilePath)
	require.NoError(t, err)

	validationCode := parachaintypes.ValidationCode(validationCodeBytes)

	validationCodeHashV := common.MustBlake2bHash(validationCodeBytes)

	collatorKeypair, err := sr25519.GenerateKeypair()
	require.NoError(t, err)

	descriptor := makeValidCandidateDescriptor(t, 1000,
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

func makeValidCandidateDescriptor(t *testing.T, paraID uint32, relayParent common.Hash,
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

func TestValidateFromChainState(t *testing.T) {

	candidateReceipt, validationCode := createTestCandidateReceiptAndValidationCode(t)

	bd, err := scale.Marshal(BlockDataInAdderParachain{
		State: uint64(1),
		Add:   uint64(1),
	})
	require.NoError(t, err)

	pov := parachaintypes.PoV{
		BlockData: bd,
	}

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

	ctrl := gomock.NewController(t)

	mockInstance := NewMockRuntimeInstance(ctrl)
	mockInstance.EXPECT().
		ParachainHostPersistedValidationData(
			uint32(1000),
			gomock.AssignableToTypeOf(parachaintypes.OccupiedCoreAssumption{})).
		Return(&expectedPersistedValidationData, nil)
	mockInstance.EXPECT().
		ParachainHostValidationCode(uint32(1000), gomock.AssignableToTypeOf(parachaintypes.OccupiedCoreAssumption{})).
		Return(&validationCode, nil)

	validationResult, err := validateFromChainState(
		mockInstance, pov, candidateReceipt)
	require.NoError(t, err)
	require.True(t, validationResult.IsValid())
	require.Equal(t, expectedPersistedValidationData, validationResult.ValidResult.PersistedValidationData)
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

func TestCandidateValidation_validateFromExhaustive(t *testing.T) {
	t.Parallel()
	candidateReceipt, validationCode := createTestCandidateReceiptAndValidationCode(t)
	candidateReceipt2 := candidateReceipt
	candidateReceipt2.Descriptor.PovHash = common.MustHexToHash(
		"0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef")
	candidateReceiptParaHeadMismatch := candidateReceipt
	candidateReceiptParaHeadMismatch.Descriptor.ParaHead = common.MustHexToHash(
		"0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef")
	candidateReceiptCommitmentsMismatch := candidateReceipt
	candidateReceiptCommitmentsMismatch.CommitmentsHash = common.MustHexToHash(
		"0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef")
	testValidationHost, err := parachainruntime.SetupVM(validationCode)
	require.NoError(t, err)

	povHashMismatch := PoVHashMismatch
	paramsTooLarge := ParamsTooLarge
	codeHashMismatch := CodeHashMismatch
	paraHedHashMismatch := ParaHeadHashMismatch
	commitmentsHashMismatch := CommitmentsHashMismatch

	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)

	mockValidationHost := NewMockValidationHost(ctrl)
	mockValidationHost.EXPECT().ValidateBlock(gomock.Any()).Return(nil, parachainruntime.ErrHardTimeout).AnyTimes()

	bd, err := scale.Marshal(BlockDataInAdderParachain{
		State: uint64(1),
		Add:   uint64(1),
	})
	require.NoError(t, err)
	pov := parachaintypes.PoV{
		BlockData: bd,
	}

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

	type args struct {
		validationHost          parachainruntime.ValidationHost
		persistedValidationData parachaintypes.PersistedValidationData
		validationCode          parachaintypes.ValidationCode
		candidateReceipt        parachaintypes.CandidateReceipt
		pov                     parachaintypes.PoV
	}
	tests := map[string]struct {
		args          args
		want          *ValidationResult
		expectedError error
		isValid       bool
	}{
		"invalid_pov_hash": {
			args: args{
				persistedValidationData: parachaintypes.PersistedValidationData{
					ParentHead:             parachaintypes.HeadData{Data: hd},
					RelayParentNumber:      uint32(1),
					RelayParentStorageRoot: common.MustHexToHash("0x50c969706800c0e9c3c4565dc2babb25e4a73d1db0dee1bcf7745535a32e7ca1"),
					MaxPovSize:             uint32(2048),
				},
				validationCode:   validationCode,
				candidateReceipt: candidateReceipt2,
				pov:              pov,
			},
			want: &ValidationResult{
				InvalidResult: &povHashMismatch,
			},
			isValid: false,
		},
		"invalid_pov_size": {
			args: args{
				persistedValidationData: parachaintypes.PersistedValidationData{
					ParentHead:             parachaintypes.HeadData{Data: hd},
					RelayParentNumber:      uint32(1),
					RelayParentStorageRoot: common.MustHexToHash("0x50c969706800c0e9c3c4565dc2babb25e4a73d1db0dee1bcf7745535a32e7ca1"),
					MaxPovSize:             uint32(10),
				},
				validationCode:   validationCode,
				candidateReceipt: candidateReceipt,
				pov:              pov,
			},
			want: &ValidationResult{
				InvalidResult: &paramsTooLarge,
			},
		},
		"code_mismatch": {
			args: args{
				persistedValidationData: parachaintypes.PersistedValidationData{
					ParentHead:             parachaintypes.HeadData{Data: hd},
					RelayParentNumber:      uint32(1),
					RelayParentStorageRoot: common.MustHexToHash("0x50c969706800c0e9c3c4565dc2babb25e4a73d1db0dee1bcf7745535a32e7ca1"),
					MaxPovSize:             uint32(2048),
				},
				validationCode:   []byte{1, 2, 3, 4, 5, 6, 7, 8},
				candidateReceipt: candidateReceipt,
				pov:              pov,
			},
			want: &ValidationResult{
				InvalidResult: &codeHashMismatch,
			},
			isValid: false,
		},
		"mock_test": {
			args: args{
				persistedValidationData: parachaintypes.PersistedValidationData{
					ParentHead:             parachaintypes.HeadData{Data: hd},
					RelayParentNumber:      uint32(1),
					RelayParentStorageRoot: common.MustHexToHash("0x50c969706800c0e9c3c4565dc2babb25e4a73d1db0dee1bcf7745535a32e7ca1"),
					MaxPovSize:             uint32(2048),
				},
				validationHost:   mockValidationHost,
				validationCode:   validationCode,
				candidateReceipt: candidateReceipt,
				pov:              pov,
			},
			want:          nil,
			expectedError: fmt.Errorf("executing validate_block: %w", parachainruntime.ErrHardTimeout),
		},
		"wasm_error_unreachable": {
			args: args{
				persistedValidationData: parachaintypes.PersistedValidationData{
					MaxPovSize: uint32(2048),
				},
				validationHost:   testValidationHost,
				validationCode:   validationCode,
				candidateReceipt: candidateReceipt,
				pov:              pov,
			},
			want: nil,
			expectedError: errors.New("executing validate_block: running runtime function: wasm error: unreachable" +
				"\nwasm stack trace:\n\t.rust_begin_unwind(i32)\n\t._ZN4core9panicking9panic_fmt17h55a9886e2bf4227aE(" +
				"i32,i32)\n\t\t0xcbc: /rustc/1c42cb4ef0544fbfaa500216e53382d6b079c001/library/core/src/panicking." +
				"rs:67:14\n\t._ZN4core6result13unwrap_failed17h18cc772327ac51f6E(i32,i32,i32,i32," +
				"i32)\n\t\t0xfe9: /rustc/1c42cb4ef0544fbfaa500216e53382d6b079c001/library/core/src/result." +
				"rs:1651:5\n\t.validate_block(i32,i32) i64"),
		},
		"para_head_hash_mismatch": {
			args: args{
				persistedValidationData: parachaintypes.PersistedValidationData{
					ParentHead:             parachaintypes.HeadData{Data: hd},
					RelayParentNumber:      uint32(1),
					RelayParentStorageRoot: common.MustHexToHash("0x50c969706800c0e9c3c4565dc2babb25e4a73d1db0dee1bcf7745535a32e7ca1"),
					MaxPovSize:             uint32(2048),
				},
				validationHost:   testValidationHost,
				validationCode:   validationCode,
				candidateReceipt: candidateReceiptParaHeadMismatch,
				pov:              pov,
			},
			want: &ValidationResult{
				InvalidResult: &paraHedHashMismatch,
			},
			isValid: false,
		},
		"commitments_hash_mismatch": {
			args: args{
				persistedValidationData: parachaintypes.PersistedValidationData{
					ParentHead:             parachaintypes.HeadData{Data: hd},
					RelayParentNumber:      uint32(1),
					RelayParentStorageRoot: common.MustHexToHash("0x50c969706800c0e9c3c4565dc2babb25e4a73d1db0dee1bcf7745535a32e7ca1"),
					MaxPovSize:             uint32(2048),
				},
				validationHost:   testValidationHost,
				validationCode:   validationCode,
				candidateReceipt: candidateReceiptCommitmentsMismatch,
				pov:              pov,
			},
			want: &ValidationResult{
				InvalidResult: &commitmentsHashMismatch,
			},
			isValid: false,
		},
		"happy_path": {
			args: args{
				persistedValidationData: parachaintypes.PersistedValidationData{
					ParentHead:             parachaintypes.HeadData{Data: hd},
					RelayParentNumber:      uint32(1),
					RelayParentStorageRoot: common.MustHexToHash("0x50c969706800c0e9c3c4565dc2babb25e4a73d1db0dee1bcf7745535a32e7ca1"),
					MaxPovSize:             uint32(2048),
				},
				validationHost:   testValidationHost,
				validationCode:   validationCode,
				candidateReceipt: candidateReceipt,
				pov:              pov,
			},
			want: &ValidationResult{
				ValidResult: &ValidValidationResult{
					CandidateCommitments: parachaintypes.CandidateCommitments{
						UpwardMessages:     nil,
						HorizontalMessages: nil,
						NewValidationCode:  nil,
						HeadData: parachaintypes.HeadData{Data: []byte{2, 0, 0, 0, 0, 0, 0, 0, 123, 207, 206, 8, 219, 227,
							136, 82, 236, 169, 14, 100, 45, 100, 31, 177, 154, 160, 220, 245, 59, 106, 76, 168, 122, 109,
							164, 169, 22, 46, 144, 39, 103, 92, 31, 78, 66, 72, 252, 64, 24, 194, 129, 162, 128, 1, 77, 147,
							200, 229, 189, 242, 111, 198, 236, 139, 16, 143, 19, 245, 113, 233, 138, 210}},
						ProcessedDownwardMessages: 0,
						HrmpWatermark:             1,
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
			isValid: true,
		},
	}
	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got, err := validateFromExhaustive(tt.args.validationHost, tt.args.persistedValidationData,
				tt.args.validationCode,
				tt.args.candidateReceipt, tt.args.pov)
			if tt.expectedError != nil {
				require.EqualError(t, err, tt.expectedError.Error())
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, tt.want, got)
			if got != nil {
				require.Equal(t, tt.isValid, got.IsValid())
			}
		})
	}
}

func TestCandidateValidation_wasm_invalid_magic_number(t *testing.T) {
	validationCode := parachaintypes.ValidationCode{1, 2, 3, 4, 5, 6, 7, 8}
	parachainRuntimeInstance, err := parachainruntime.SetupVM(validationCode)
	require.EqualError(t, err, "creating instance: creating runtime instance: invalid magic number")
	require.Emptyf(t, parachainRuntimeInstance, "parachainRuntimeInstance should be empty")
}

func TestCandidateValidation_processMessageValidateFromExhaustive(t *testing.T) {
	povHashMismatch := PoVHashMismatch
	paramsTooLarge := ParamsTooLarge
	codeHashMismatch := CodeHashMismatch

	candidateReceipt, validationCode := createTestCandidateReceiptAndValidationCode(t)
	candidateReceipt2 := candidateReceipt
	candidateReceipt2.Descriptor.PovHash = common.MustHexToHash(
		"0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef")
	testValidationHost, err := parachainruntime.SetupVM(validationCode)
	require.NoError(t, err)

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

	toSubsystem := make(chan any)
	sender := make(chan parachaintypes.OverseerFuncRes[ValidationResult])
	candidateValidationSubsystem := CandidateValidation{
		OverseerToSubsystem: toSubsystem,
		ValidationHost:      testValidationHost,
	}
	defer candidateValidationSubsystem.Stop()

	ctx, cancel := context.WithCancel(context.Background())
	go candidateValidationSubsystem.Run(ctx, cancel, nil, nil)

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
					InvalidResult: &povHashMismatch,
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
					InvalidResult: &paramsTooLarge,
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
					InvalidResult: &codeHashMismatch,
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
					ValidResult: &ValidValidationResult{
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
		t.Run(name, func(t *testing.T) {
			time.Sleep(100 * time.Millisecond)
			toSubsystem <- tt.msg
			time.Sleep(100 * time.Millisecond)
			result := <-sender
			require.Equal(t, tt.want, result)
		})
	}
}

func Test_performBasicChecks(t *testing.T) {
	paramsTooLarge := ParamsTooLarge
	povHashMismatch := PoVHashMismatch
	codeHashMismatch := CodeHashMismatch
	badSignature := BadSignature

	pov := parachaintypes.PoV{
		BlockData: []byte{1, 2, 3, 4, 5, 6, 7, 8},
	}
	povHash, err := pov.Hash()
	pov2 := parachaintypes.PoV{
		BlockData: []byte{1, 1, 1, 1, 1},
	}
	validationCode := parachaintypes.ValidationCode{1, 2, 3}
	validationCodeHash := validationCode.Hash()

	require.NoError(t, err)
	collatorKeypair, err := sr25519.GenerateKeypair()
	require.NoError(t, err)
	collatorID, err := sr25519.NewPublicKey(collatorKeypair.Public().Encode())
	require.NoError(t, err)

	candidate := parachaintypes.CandidateDescriptor{
		Collator:           collatorID.AsBytes(),
		PovHash:            povHash,
		ValidationCodeHash: validationCodeHash,
	}
	candidate2 := candidate

	payload, err := candidate.CreateSignaturePayload()
	require.NoError(t, err)

	signatureBytes, err := collatorKeypair.Sign(payload)
	require.NoError(t, err)

	signature := [sr25519.SignatureLength]byte{}
	copy(signature[:], signatureBytes)

	signature2Bytes, err := collatorKeypair.Sign([]byte{1, 2, 3, 4, 5, 6, 7, 8})
	require.NoError(t, err)
	signature2 := [sr25519.SignatureLength]byte{}
	copy(signature2[:], signature2Bytes)

	candidate.Signature = parachaintypes.CollatorSignature(signature)
	candidate2.Signature = parachaintypes.CollatorSignature(signature2)

	type args struct {
		candidate          *parachaintypes.CandidateDescriptor
		maxPoVSize         uint32
		pov                parachaintypes.PoV
		validationCodeHash parachaintypes.ValidationCodeHash
	}
	tests := map[string]struct {
		args          args
		expectedError *ReasonForInvalidity
	}{
		"params_too_large": {
			args: args{
				candidate:  &candidate,
				maxPoVSize: 2,
				pov:        pov,
			},
			expectedError: &paramsTooLarge,
		},
		"invalid_pov_hash": {
			args: args{
				candidate:  &candidate,
				maxPoVSize: 1024,
				pov:        pov2,
			},
			expectedError: &povHashMismatch,
		},
		"invalid_code_hash": {
			args: args{
				candidate:          &candidate,
				maxPoVSize:         1024,
				pov:                pov,
				validationCodeHash: parachaintypes.ValidationCodeHash{1, 2, 3},
			},
			expectedError: &codeHashMismatch,
		},
		"invalid_signature": {
			args: args{
				candidate:          &candidate2,
				maxPoVSize:         1024,
				pov:                pov,
				validationCodeHash: validationCodeHash,
			},
			expectedError: &badSignature,
		},
		"happy_path": {
			args: args{
				candidate:          &candidate,
				maxPoVSize:         1024,
				pov:                pov,
				validationCodeHash: validationCodeHash,
			},
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			validationError, _ := performBasicChecks(tt.args.candidate, tt.args.maxPoVSize, tt.args.pov,
				tt.args.validationCodeHash)
			if tt.expectedError != nil {
				require.EqualError(t, validationError, tt.expectedError.Error())
			} else {
				require.Nil(t, validationError)
			}
		})
	}
}

func TestCandidateValidation_validateFromChainState(t *testing.T) {
	t.Parallel()
	candidateReceipt, validationCode := createTestCandidateReceiptAndValidationCode(t)
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

	povHashMismatch := PoVHashMismatch
	paramsTooLarge := ParamsTooLarge
	codeHashMismatch := CodeHashMismatch
	badSignature := BadSignature

	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)

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

	mockInstance := NewMockRuntimeInstance(ctrl)
	mockInstance.EXPECT().
		ParachainHostPersistedValidationData(
			uint32(1000),
			gomock.AssignableToTypeOf(parachaintypes.OccupiedCoreAssumption{})).
		Return(&expectedPersistedValidationData, nil)
	mockInstance.EXPECT().
		ParachainHostValidationCode(uint32(1000), gomock.AssignableToTypeOf(parachaintypes.OccupiedCoreAssumption{})).
		Return(&validationCode, nil)

	mockInstance.EXPECT().
		ParachainHostPersistedValidationData(
			uint32(2),
			gomock.AssignableToTypeOf(parachaintypes.OccupiedCoreAssumption{})).
		Return(&expectedPersistedValidationData, nil)
	mockInstance.EXPECT().
		ParachainHostValidationCode(uint32(2), gomock.AssignableToTypeOf(parachaintypes.OccupiedCoreAssumption{})).
		Return(&validationCode, nil)

	mockInstance.EXPECT().
		ParachainHostPersistedValidationData(
			uint32(3),
			gomock.AssignableToTypeOf(parachaintypes.OccupiedCoreAssumption{})).
		Return(&expectedPersistedValidationDataSmallMax, nil)
	mockInstance.EXPECT().
		ParachainHostValidationCode(uint32(3), gomock.AssignableToTypeOf(parachaintypes.OccupiedCoreAssumption{})).
		Return(&validationCode, nil)

	mockInstance.EXPECT().
		ParachainHostPersistedValidationData(
			uint32(4),
			gomock.AssignableToTypeOf(parachaintypes.OccupiedCoreAssumption{})).
		Return(&expectedPersistedValidationData, nil)
	mockInstance.EXPECT().
		ParachainHostValidationCode(uint32(4), gomock.AssignableToTypeOf(parachaintypes.OccupiedCoreAssumption{})).
		Return(&validationCode, nil)

	mockInstance.EXPECT().
		ParachainHostPersistedValidationData(
			uint32(5),
			gomock.AssignableToTypeOf(parachaintypes.OccupiedCoreAssumption{})).
		Return(&expectedPersistedValidationData, nil)
	mockInstance.EXPECT().
		ParachainHostValidationCode(uint32(5), gomock.AssignableToTypeOf(parachaintypes.OccupiedCoreAssumption{})).
		Return(&validationCode, nil)

	bd, err := scale.Marshal(BlockDataInAdderParachain{
		State: uint64(1),
		Add:   uint64(1),
	})
	require.NoError(t, err)
	pov := parachaintypes.PoV{
		BlockData: bd,
	}

	tests := map[string]struct {
		candidateReceipt parachaintypes.CandidateReceipt
		want             *ValidationResult
		expectedError    error
	}{
		"invalid_pov_hash": {
			candidateReceipt: candidateReceipt2,
			want: &ValidationResult{
				InvalidResult: &povHashMismatch,
			},
		},
		"invalid_pov_size": {
			candidateReceipt: candidateReceipt3,
			want: &ValidationResult{
				InvalidResult: &paramsTooLarge,
			},
		},
		"code_mismatch": {
			candidateReceipt: candidateReceipt4,
			want: &ValidationResult{
				InvalidResult: &codeHashMismatch,
			},
		},
		"bad_signature": {
			candidateReceipt: candidateReceipt5,
			want: &ValidationResult{
				InvalidResult: &badSignature,
			},
		},
		"happy_path": {
			candidateReceipt: candidateReceipt,
			want: &ValidationResult{
				ValidResult: &ValidValidationResult{
					CandidateCommitments: parachaintypes.CandidateCommitments{
						HeadData: parachaintypes.HeadData{Data: []byte{2, 0, 0, 0, 0, 0, 0, 0, 123, 207, 206, 8, 219, 227,
							136, 82, 236, 169, 14, 100, 45, 100, 31, 177, 154, 160, 220, 245, 59, 106, 76, 168, 122, 109,
							164, 169, 22, 46, 144, 39, 103, 92, 31, 78, 66, 72, 252, 64, 24, 194, 129, 162, 128, 1, 77, 147,
							200, 229, 189, 242, 111, 198, 236, 139, 16, 143, 19, 245, 113, 233, 138, 210}},
						ProcessedDownwardMessages: 0,
						HrmpWatermark:             1,
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
	}
	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got, err := validateFromChainState(mockInstance, pov, tt.candidateReceipt)
			if tt.expectedError != nil {
				require.EqualError(t, err, tt.expectedError.Error())
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, *tt.want, *got)
		})
	}
}
