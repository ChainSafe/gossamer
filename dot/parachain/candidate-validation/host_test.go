package candidatevalidation

import (
	"testing"

	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/stretchr/testify/require"
)

func TestHost_validate(t *testing.T) {
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

	povHashMismatch := PoVHashMismatch
	paramsTooLarge := ParamsTooLarge
	codeHashMismatch := CodeHashMismatch
	paraHedHashMismatch := ParaHeadHashMismatch
	commitmentsHashMismatch := CommitmentsHashMismatch
	executionError := ExecutionError

	pvfHost := newValidationHost()

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
		validationTask *ValidationTask
		want           *ValidationResult
		isValid        bool
	}{
		"invalid_pov_hash": {
			validationTask: &ValidationTask{
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
			want: &ValidationResult{
				InvalidResult: &povHashMismatch,
			},
			isValid: false,
		},
		"invalid_pov_size": {
			validationTask: &ValidationTask{
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
			want: &ValidationResult{
				InvalidResult: &paramsTooLarge,
			},
		},
		"code_mismatch": {
			validationTask: &ValidationTask{
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
			want: &ValidationResult{
				InvalidResult: &codeHashMismatch,
			},
			isValid: false,
		},
		"wasm_error_unreachable": {
			validationTask: &ValidationTask{
				PersistedValidationData: parachaintypes.PersistedValidationData{
					MaxPovSize: uint32(2048),
				},
				ValidationCode:   &validationCode,
				CandidateReceipt: &candidateReceipt,
				PoV:              pov,
			},
			want: &ValidationResult{
				InvalidResult: &executionError,
			},
		},
		"para_head_hash_mismatch": {
			validationTask: &ValidationTask{
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
			want: &ValidationResult{
				InvalidResult: &paraHedHashMismatch,
			},
			isValid: false,
		},
		"commitments_hash_mismatch": {
			validationTask: &ValidationTask{
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
			want: &ValidationResult{
				InvalidResult: &commitmentsHashMismatch,
			},
			isValid: false,
		},
		"happy_path": {
			validationTask: &ValidationTask{
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
			want: &ValidationResult{
				ValidResult: &Valid{
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

			taskResult, err := pvfHost.validate(tt.validationTask)

			require.NoError(t, err)
			require.Equal(t, tt.want, taskResult)
			require.Equal(t, tt.isValid, taskResult.IsValid())
		})
	}
}

func TestHost_performBasicChecks(t *testing.T) {
	t.Parallel()
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
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			validationError, internalError := performBasicChecks(tt.args.candidate, tt.args.maxPoVSize, tt.args.pov,
				tt.args.validationCodeHash)
			require.NoError(t, internalError)
			if tt.expectedError != nil {
				require.EqualError(t, validationError, tt.expectedError.Error())
			} else {
				require.Nil(t, validationError)
			}
		})
	}
}
