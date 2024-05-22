// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package candidatevalidation

import (
	"context"
	"errors"
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
	collatorID, err := sr25519.NewPublicKey(collatorKeypair.Public().Encode())
	require.NoError(t, err)

	candidateReceipt := parachaintypes.CandidateReceipt{
		Descriptor: parachaintypes.CandidateDescriptor{
			ParaID:                      uint32(1000),
			RelayParent:                 common.MustHexToHash("0xded542bacb3ca6c033a57676f94ae7c8f36834511deb44e3164256fd3b1c0de0"), //nolint:lll
			Collator:                    collatorID.AsBytes(),
			PersistedValidationDataHash: common.MustHexToHash("0x690d8f252ef66ab0f969c3f518f90012b849aa5ac94e1752c5e5ae5a8996de37"), //nolint:lll
			PovHash:                     common.MustHexToHash("0xb608991ffc48dd405fd4b10e92eaebe2b5a2eedf44d0c3efb8997fdee8bebed9"), //nolint:lll
			ErasureRoot:                 common.MustHexToHash("0xc07f658163e93c45a6f0288d229698f09c1252e41076f4caa71c8cbc12f118a1"), //nolint:lll
			ParaHead:                    common.MustHexToHash("0x9a8a7107426ef873ab89fc8af390ec36bdb2f744a9ff71ad7f18a12d55a7f4f5"), //nolint:lll
			ValidationCodeHash:          parachaintypes.ValidationCodeHash(validationCodeHashV),
		},

		CommitmentsHash: common.MustHexToHash("0xa54a8dce5fd2a27e3715f99e4241f674a48f4529f77949a4474f5b283b823535"),
	}

	payload, err := candidateReceipt.Descriptor.CreateSignaturePayload()
	require.NoError(t, err)

	signatureBytes, err := collatorKeypair.Sign(payload)
	require.NoError(t, err)

	signature := [sr25519.SignatureLength]byte{}
	copy(signature[:], signatureBytes)

	candidateReceipt.Descriptor.Signature = parachaintypes.CollatorSignature(signature)

	return candidateReceipt, validationCode
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
	mockInstance.EXPECT().
		ParachainHostCheckValidationOutputs(uint32(1000), gomock.AssignableToTypeOf(parachaintypes.CandidateCommitments{})).
		Return(true, nil)

	mockPoVRequestor := NewMockPoVRequestor(ctrl)
	mockPoVRequestor.EXPECT().
		RequestPoV(common.MustHexToHash("0xb608991ffc48dd405fd4b10e92eaebe2b5a2eedf44d0c3efb8997fdee8bebed9")).Return(pov)

	candidateCommitments, persistedValidationData, isValid, err := validateFromChainState(
		mockInstance, mockPoVRequestor, candidateReceipt)
	require.NoError(t, err)
	require.True(t, isValid)
	require.NotNil(t, candidateCommitments)
	require.Equal(t, expectedPersistedValidationData, *persistedValidationData)
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
		persistedValidationData parachaintypes.PersistedValidationData
		validationCode          parachaintypes.ValidationCode
		candidateReceipt        parachaintypes.CandidateReceipt
		pov                     parachaintypes.PoV
	}
	tests := map[string]struct {
		args          args
		want          *parachainruntime.ValidationResult
		expectedError error
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
			expectedError: ErrValidationPoVHashMismatch,
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
			expectedError: errors.New("validation parameters are too large, limit: 10, got: 17"),
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
			expectedError: ErrValidationCodeMismatch,
		},
		"happy_path": {
			args: args{
				persistedValidationData: parachaintypes.PersistedValidationData{
					ParentHead:             parachaintypes.HeadData{Data: hd},
					RelayParentNumber:      uint32(1),
					RelayParentStorageRoot: common.MustHexToHash("0x50c969706800c0e9c3c4565dc2babb25e4a73d1db0dee1bcf7745535a32e7ca1"),
					MaxPovSize:             uint32(2048),
				},
				validationCode:   validationCode,
				candidateReceipt: candidateReceipt,
				pov:              pov,
			},
			want: &parachainruntime.ValidationResult{
				HeadData: parachaintypes.HeadData{Data: []byte{2, 0, 0, 0, 0, 0, 0, 0, 123, 207, 206, 8, 219, 227,
					136, 82, 236, 169, 14, 100, 45, 100, 31, 177, 154, 160, 220, 245, 59, 106, 76, 168, 122, 109,
					164, 169, 22, 46, 144, 39, 103, 92, 31, 78, 66, 72, 252, 64, 24, 194, 129, 162, 128, 1, 77, 147,
					200, 229, 189, 242, 111, 198, 236, 139, 16, 143, 19, 245, 113, 233, 138, 210}},
				ProcessedDownwardMessages: 0,
				HrmpWatermark:             1,
			},
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := validateFromExhaustive(tt.args.persistedValidationData, tt.args.validationCode,
				tt.args.candidateReceipt, tt.args.pov)
			if tt.expectedError != nil {
				require.EqualError(t, err, tt.expectedError.Error())
			} else {
				require.NoError(t, err)
			}

			require.Equal(t, tt.want, got)
		})
	}
}

func TestCandidateValidation_processMessageValidateFromExhaustive(t *testing.T) {
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
	sender := make(chan ValidationResultMessage)
	stopChan := make(chan struct{})
	candidateValidationSubsystem := CandidateValidation{
		OverseerToSubsystem: toSubsystem,
		stopChan:            stopChan,
	}
	defer candidateValidationSubsystem.Stop()

	candidateValidationSubsystem.Run(context.Background(), nil, nil)

	tests := map[string]struct {
		msg  ValidateFromExhaustive
		want ValidationResultMessage
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
				Pov:              pov,
				Sender:           sender,
			},
			want: ValidationResultMessage{
				ValidationFailed: ErrValidationPoVHashMismatch.Error(),
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
				Pov:              pov,
				Sender:           sender,
			},
			want: ValidationResultMessage{
				ValidationFailed: "validation parameters are too large, limit: 10, got: 17",
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
				Pov:              pov,
				Sender:           sender,
			},
			want: ValidationResultMessage{
				ValidationFailed: ErrValidationCodeMismatch.Error(),
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
				Pov:              pov,
				Sender:           sender,
			},
			want: ValidationResultMessage{
				ValidationResult: parachainruntime.ValidationResult{
					HeadData: parachaintypes.HeadData{Data: []byte{2, 0, 0, 0, 0, 0, 0, 0, 123, 207, 206, 8, 219, 227,
						136, 82, 236, 169, 14, 100, 45, 100, 31, 177, 154, 160, 220, 245, 59, 106, 76, 168, 122, 109,
						164, 169, 22, 46, 144, 39, 103, 92, 31, 78, 66, 72, 252, 64, 24, 194, 129, 162, 128, 1, 77, 147,
						200, 229, 189, 242, 111, 198, 236, 139, 16, 143, 19, 245, 113, 233, 138, 210}},
					ProcessedDownwardMessages: 0,
					HrmpWatermark:             1,
				},
				ValidationFailed: "",
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
