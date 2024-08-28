// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package candidatevalidation

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/dot/parachain/pvf"
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

	povHashMismatch := pvf.PoVHashMismatch
	paramsTooLarge := pvf.ParamsTooLarge
	codeHashMismatch := pvf.CodeHashMismatch
	paraHedHashMismatch := pvf.ParaHeadHashMismatch
	commitmentsHashMismatch := pvf.CommitmentsHashMismatch
	executionError := pvf.ExecutionError

	pvfHost := pvf.NewValidationHost()

	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)

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

	tests := map[string]struct {
		validationTask *pvf.ValidationTask
		want           *pvf.ValidationResult
		isValid        bool
	}{
		"invalid_pov_hash": {
			validationTask: &pvf.ValidationTask{
				PersistedValidationData: parachaintypes.PersistedValidationData{
					ParentHead:             parachaintypes.HeadData{Data: hd},
					RelayParentNumber:      uint32(1),
					RelayParentStorageRoot: common.MustHexToHash("0x50c969706800c0e9c3c4565dc2babb25e4a73d1db0dee1bcf7745535a32e7ca1"),
					MaxPovSize:             uint32(2048),
				},
				CandidateReceipt:   &candidateReceipt2,
				PoV:                pov,
				PvfExecTimeoutKind: parachaintypes.PvfExecTimeoutKind{},
				ValidationCode:     &validationCode,
			},
			want: &pvf.ValidationResult{
				InvalidResult: &povHashMismatch,
			},
			isValid: false,
		},
		"invalid_pov_size": {
			validationTask: &pvf.ValidationTask{
				PersistedValidationData: parachaintypes.PersistedValidationData{
					ParentHead:             parachaintypes.HeadData{Data: hd},
					RelayParentNumber:      uint32(1),
					RelayParentStorageRoot: common.MustHexToHash("0x50c969706800c0e9c3c4565dc2babb25e4a73d1db0dee1bcf7745535a32e7ca1"),
					MaxPovSize:             uint32(10),
				},
				ValidationCode:   &validationCode,
				CandidateReceipt: &candidateReceipt,
				PoV:              pov,
			},
			want: &pvf.ValidationResult{
				InvalidResult: &paramsTooLarge,
			},
		},
		"code_mismatch": {
			validationTask: &pvf.ValidationTask{
				PersistedValidationData: parachaintypes.PersistedValidationData{
					ParentHead:             parachaintypes.HeadData{Data: hd},
					RelayParentNumber:      uint32(1),
					RelayParentStorageRoot: common.MustHexToHash("0x50c969706800c0e9c3c4565dc2babb25e4a73d1db0dee1bcf7745535a32e7ca1"),
					MaxPovSize:             uint32(2048),
				},
				ValidationCode:   &parachaintypes.ValidationCode{1, 2, 3, 4, 5, 6, 7, 8},
				CandidateReceipt: &candidateReceipt,
				PoV:              pov,
			},
			want: &pvf.ValidationResult{
				InvalidResult: &codeHashMismatch,
			},
			isValid: false,
		},
		"wasm_error_unreachable": {
			validationTask: &pvf.ValidationTask{
				PersistedValidationData: parachaintypes.PersistedValidationData{
					MaxPovSize: uint32(2048),
				},
				ValidationCode:   &validationCode,
				CandidateReceipt: &candidateReceipt,
				PoV:              pov,
			},
			want: &pvf.ValidationResult{
				InvalidResult: &executionError,
			},
		},
		"para_head_hash_mismatch": {
			validationTask: &pvf.ValidationTask{
				PersistedValidationData: parachaintypes.PersistedValidationData{
					ParentHead:             parachaintypes.HeadData{Data: hd},
					RelayParentNumber:      uint32(1),
					RelayParentStorageRoot: common.MustHexToHash("0x50c969706800c0e9c3c4565dc2babb25e4a73d1db0dee1bcf7745535a32e7ca1"),
					MaxPovSize:             uint32(2048),
				},
				ValidationCode:   &validationCode,
				CandidateReceipt: &candidateReceiptParaHeadMismatch,
				PoV:              pov,
			},
			want: &pvf.ValidationResult{
				InvalidResult: &paraHedHashMismatch,
			},
			isValid: false,
		},
		"commitments_hash_mismatch": {
			validationTask: &pvf.ValidationTask{
				PersistedValidationData: parachaintypes.PersistedValidationData{
					ParentHead:             parachaintypes.HeadData{Data: hd},
					RelayParentNumber:      uint32(1),
					RelayParentStorageRoot: common.MustHexToHash("0x50c969706800c0e9c3c4565dc2babb25e4a73d1db0dee1bcf7745535a32e7ca1"),
					MaxPovSize:             uint32(2048),
				},
				ValidationCode:   &validationCode,
				CandidateReceipt: &candidateReceiptCommitmentsMismatch,
				PoV:              pov,
			},
			want: &pvf.ValidationResult{
				InvalidResult: &commitmentsHashMismatch,
			},
			isValid: false,
		},
		"happy_path": {
			validationTask: &pvf.ValidationTask{
				PersistedValidationData: parachaintypes.PersistedValidationData{
					ParentHead:             parachaintypes.HeadData{Data: hd},
					RelayParentNumber:      uint32(1),
					RelayParentStorageRoot: common.MustHexToHash("0x50c969706800c0e9c3c4565dc2babb25e4a73d1db0dee1bcf7745535a32e7ca1"),
					MaxPovSize:             uint32(2048),
				},
				ValidationCode:   &validationCode,
				CandidateReceipt: &candidateReceipt,
				PoV:              pov,
			},
			want: &pvf.ValidationResult{
				ValidResult: &pvf.ValidValidationResult{
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

			taskResult := make(chan *pvf.ValidationTaskResult)
			defer close(taskResult)
			//tt.validationTask.ResultCh = taskResult

			go pvfHost.Validate(tt.validationTask)

			result := <-taskResult
			require.Equal(t, tt.want, result.Result)
			require.Equal(t, tt.isValid, result.Result.IsValid())
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
	povHashMismatch := pvf.PoVHashMismatch
	paramsTooLarge := pvf.ParamsTooLarge
	codeHashMismatch := pvf.CodeHashMismatch

	candidateReceipt, validationCode := createTestCandidateReceiptAndValidationCode(t)
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

	toSubsystem := make(chan any)
	sender := make(chan parachaintypes.OverseerFuncRes[pvf.ValidationResult])
	stopChan := make(chan struct{})
	candidateValidationSubsystem := CandidateValidation{
		OverseerToSubsystem: toSubsystem,
		stopChan:            stopChan,
		pvfHost:             pvf.NewValidationHost(),
	}
	defer candidateValidationSubsystem.Stop()

	ctx := context.Background()
	go candidateValidationSubsystem.Run(ctx, toSubsystem)

	tests := map[string]struct {
		msg  ValidateFromExhaustive
		want parachaintypes.OverseerFuncRes[pvf.ValidationResult]
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
			want: parachaintypes.OverseerFuncRes[pvf.ValidationResult]{
				Data: pvf.ValidationResult{
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
			want: parachaintypes.OverseerFuncRes[pvf.ValidationResult]{
				Data: pvf.ValidationResult{
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
			want: parachaintypes.OverseerFuncRes[pvf.ValidationResult]{
				Data: pvf.ValidationResult{
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
			want: parachaintypes.OverseerFuncRes[pvf.ValidationResult]{
				Data: pvf.ValidationResult{
					ValidResult: &pvf.ValidValidationResult{
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
	paramsTooLarge := pvf.ParamsTooLarge
	povHashMismatch := pvf.PoVHashMismatch
	codeHashMismatch := pvf.CodeHashMismatch
	badSignature := pvf.BadSignature

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
		expectedError *pvf.ReasonForInvalidity
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

	povHashMismatch := pvf.PoVHashMismatch
	paramsTooLarge := pvf.ParamsTooLarge
	codeHashMismatch := pvf.CodeHashMismatch
	badSignature := pvf.BadSignature

	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)

	//NOTE: adder parachain internally compares postState with bd.State in it's validate_block,
	//so following is necessary.
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

	mockInstance := NewMockInstance(ctrl)
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

	mockBlockState := NewMockBlockState(ctrl)
	mockBlockState.EXPECT().GetRuntime(common.MustHexToHash(
		"0xded542bacb3ca6c033a57676f94ae7c8f36834511deb44e3164256fd3b1c0de0")).Return(mockInstance, nil).Times(5)

	bd, err := scale.Marshal(BlockDataInAdderParachain{
		State: uint64(1),
		Add:   uint64(1),
	})
	require.NoError(t, err)
	pov := parachaintypes.PoV{
		BlockData: bd,
	}

	toSubsystem := make(chan any)
	stopChan := make(chan struct{})
	candidateValidationSubsystem := CandidateValidation{
		OverseerToSubsystem: toSubsystem,
		stopChan:            stopChan,
		pvfHost:             pvf.NewValidationHost(),
		BlockState:          mockBlockState,
	}
	defer candidateValidationSubsystem.Stop()

	candidateValidationSubsystem.Run(context.Background(), nil)

	tests := map[string]struct {
		msg           ValidateFromChainState
		want          *pvf.ValidationResult
		expectedError error
	}{
		"invalid_pov_hash": {
			msg: ValidateFromChainState{
				CandidateReceipt: candidateReceipt2,
				Pov:              pov,
			},
			want: &pvf.ValidationResult{
				InvalidResult: &povHashMismatch,
			},
		},
		"invalid_pov_size": {
			msg: ValidateFromChainState{
				CandidateReceipt: candidateReceipt3,
				Pov:              pov,
			},
			want: &pvf.ValidationResult{
				InvalidResult: &paramsTooLarge,
			},
		},
		"code_mismatch": {
			msg: ValidateFromChainState{
				CandidateReceipt: candidateReceipt4,
				Pov:              pov,
			},
			want: &pvf.ValidationResult{
				InvalidResult: &codeHashMismatch,
			},
		},
		"bad_signature": {
			msg: ValidateFromChainState{
				CandidateReceipt: candidateReceipt5,
				Pov:              pov,
			},
			want: &pvf.ValidationResult{
				InvalidResult: &badSignature,
			},
		},
		"happy_path": {
			msg: ValidateFromChainState{
				CandidateReceipt: candidateReceipt,
				Pov:              pov,
			},
			want: &pvf.ValidationResult{
				ValidResult: &pvf.ValidValidationResult{
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
			sender := make(chan parachaintypes.OverseerFuncRes[pvf.ValidationResult])
			defer close(sender)
			tt.msg.Ch = sender
			time.Sleep(100 * time.Millisecond)
			toSubsystem <- tt.msg
			time.Sleep(100 * time.Millisecond)
			result := <-sender
			require.Equal(t, tt.want, &result.Data)
		})
	}
}
