// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package parachain

import (
	_ "embed"
	"errors"
	"fmt"
	"testing"

	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

//go:embed testdata/statement.yaml
var testDataStatementRaw string

var testDataStatement map[string]string

func init() {
	err := yaml.Unmarshal([]byte(testDataStatementRaw), &testDataStatement)
	if err != nil {
		fmt.Printf("Error unmarshaling test data: %s\n", err)
		return
	}
}

var ErrInvalidVayingDataTypeValue = errors.New(
	"unsupported type")

type invalidVayingDataTypeValue struct{}

func getDummyHash(num byte) common.Hash {
	hash := common.Hash{}
	for i := 0; i < 32; i++ {
		hash[i] = num
	}
	return hash
}

func TestStatementDistributionMessage(t *testing.T) {
	t.Parallel()

	var collatorSignature parachaintypes.CollatorSignature
	tempSignature := common.MustHexToBytes(testDataStatement["collatorSignature"])
	copy(collatorSignature[:], tempSignature)

	var validatorSignature parachaintypes.ValidatorSignature
	copy(validatorSignature[:], tempSignature)

	var collatorID parachaintypes.CollatorID
	tempCollatID := common.MustHexToBytes("0x48215b9d322601e5b1a95164cea0dc4626f545f98343d07f1551eb9543c4b147")
	copy(collatorID[:], tempCollatID)

	hash5 := getDummyHash(5)

	statementVDTWithValid := parachaintypes.NewStatementVDT()
	err := statementVDTWithValid.SetValue(parachaintypes.Valid{Value: hash5})
	require.NoError(t, err)

	secondedEnumValue := parachaintypes.Seconded{
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

	statementVDTWithSeconded := parachaintypes.NewStatementVDT()
	err = statementVDTWithSeconded.SetValue(secondedEnumValue)
	require.NoError(t, err)

	signedFullStatementWithValid := Statement{
		Hash: hash5,
		UncheckedSignedFullStatement: parachaintypes.UncheckedSignedFullStatement{
			Payload:        statementVDTWithValid,
			ValidatorIndex: parachaintypes.ValidatorIndex(5),
			Signature:      validatorSignature,
		},
	}

	signedFullStatementWithSeconded := Statement{
		Hash: hash5,
		UncheckedSignedFullStatement: parachaintypes.UncheckedSignedFullStatement{
			Payload:        statementVDTWithSeconded,
			ValidatorIndex: parachaintypes.ValidatorIndex(5),
			Signature:      validatorSignature,
		},
	}

	largePayload := LargePayload{
		RelayParent:   hash5,
		CandidateHash: parachaintypes.CandidateHash{Value: hash5},
		SignedBy:      parachaintypes.ValidatorIndex(5),
		Signature:     validatorSignature,
	}

	testCases := []struct {
		name          string
		enumValue     any
		encodingValue []byte
		expectedErr   error
	}{
		// expected encoding is generated by running rust test code:
		// fn statement_distribution_message_encode() {
		//     let hash1 = Hash::repeat_byte(5);
		//     let candidate_hash = CandidateHash(hash1);
		//     let statement_valid = Statement::Valid(candidate_hash);
		//     let val_sign = ValidatorSignature::from(
		//  sr25519::Signature([198, 124, 185, 59, 240, 163, 111, 206, 227,
		//  210, 157, 232, 166, 166, 154, 117, 150, 89, 104, 10, 207, 72, 100, 117, 224, 162, 85, 42, 95, 190,
		//  216, 126, 69, 173, 206, 95, 41, 6, 152, 216, 89, 96, 149, 114, 43, 51, 89, 146, 39, 247, 70, 31,
		//  81, 175, 134, 23, 200, 190, 116, 184, 148, 207, 27, 134]));
		//     let unchecked_signed_full_statement_valid = UncheckedSignedFullStatement::new(
		// statement_valid, ValidatorIndex(5), val_sign.clone());
		//     let sdm_statement_valid = StatementDistributionMessage::Statement(
		// hash1, unchecked_signed_full_statement_valid);
		//     println!("encode Statement with valid statementVDT => {:?}\n\n", sdm_statement_valid.encode());

		//     let collator_result = sr25519::Public::from_string(
		// "0x48215b9d322601e5b1a95164cea0dc4626f545f98343d07f1551eb9543c4b147");
		//     let collator = collator_result.unwrap();
		//     let collsign = CollatorSignature::from(sr25519::Signature(
		// [198, 124, 185, 59, 240, 163, 111, 206, 227, 210, 157, 232,
		//  166, 166, 154, 117, 150, 89, 104, 10, 207, 72, 100, 117, 224, 162, 85, 42, 95, 190, 216, 126, 69, 173,
		//  206, 95, 41, 6, 152, 216, 89, 96, 149, 114, 43, 51, 89, 146, 39, 247, 70, 31, 81, 175, 134, 23, 200,
		//  190, 116, 184, 148, 207, 27, 134]));
		//     let candidate_descriptor = CandidateDescriptor{
		//         para_id: 1.into(),
		//         relay_parent: hash1,
		//         collator: CollatorId::from(collator),
		//         persisted_validation_data_hash: hash1,
		//         pov_hash: hash1,
		//         erasure_root: hash1,
		//         signature: collsign,
		//         para_head: hash1,
		//         validation_code_hash:  ValidationCodeHash::from(hash1)
		//     };
		//     let commitments_new = CandidateCommitments{
		//         upward_messages: vec![vec![1, 2, 3]].try_into().expect("error - upward_messages"),
		//         horizontal_messages: vec![].try_into().expect("error - horizontal_messages"),
		//         head_data: HeadData(vec![1, 2, 3]),
		//         hrmp_watermark: 0_u32,
		//         new_validation_code: ValidationCode(vec![1, 2, 3]).try_into().expect("error - new_validation_code"),
		//         processed_downward_messages: 5
		//     };
		//     let committed_candidate_receipt = CommittedCandidateReceipt{
		//         descriptor: candidate_descriptor,
		//         commitments : commitments_new
		//     };
		//     let statement_second = Statement::Seconded(committed_candidate_receipt);
		//     let unchecked_signed_full_statement_second = UncheckedSignedFullStatement::new(
		// statement_second, ValidatorIndex(5), val_sign.clone());
		//     let sdm_statement_second = StatementDistributionMessage::Statement(
		// hash1, unchecked_signed_full_statement_second);
		//     println!("encode Statement with Seconded statementVDT => {:?}\n\n", sdm_statement_second.encode());

		//     let sdm_large_statement = StatementDistributionMessage::LargeStatement(StatementMetadata{
		//         relay_parent: hash1,
		//         candidate_hash: CandidateHash(hash1),
		//         signed_by: ValidatorIndex(5_u32),
		//         signature: val_sign.clone(),
		//     });
		//     println!("encode largePayload => {:?}\n\n", sdm_large_statement.encode());
		// }

		{
			name:          "Statement with valid statementVDT",
			enumValue:     signedFullStatementWithValid,
			encodingValue: common.MustHexToBytes(testDataStatement["statementValid"]),
		},
		{
			name:          "Statement with Seconded statementVDT",
			enumValue:     signedFullStatementWithSeconded,
			encodingValue: common.MustHexToBytes(testDataStatement["statementSeconded"]),
		},
		{
			name:          "Seconded Statement With LargePayload",
			enumValue:     largePayload,
			encodingValue: common.MustHexToBytes(testDataStatement["statementWithLargePayload"]),
		},
		{
			name:        "invalid struct",
			enumValue:   invalidVayingDataTypeValue{},
			expectedErr: ErrInvalidVayingDataTypeValue,
		},
	}

	for _, c := range testCases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			t.Run("marshal", func(t *testing.T) {
				t.Parallel()

				vdt := NewStatementDistributionMessage()
				err := vdt.SetValue(c.enumValue)

				if c.expectedErr != nil {
					require.EqualError(t, err, c.expectedErr.Error())
					return
				}

				require.NoError(t, err)
				bytes, err := scale.Marshal(vdt)
				require.NoError(t, err)

				require.Equal(t, c.encodingValue, bytes)
			})

			t.Run("unmarshal", func(t *testing.T) {
				t.Parallel()
				if c.expectedErr != nil {
					return
				}

				vdt := NewStatementDistributionMessage()
				err := scale.Unmarshal(c.encodingValue, &vdt)
				require.NoError(t, err)

				actualData, err := vdt.Value()
				require.NoError(t, err)

				require.EqualValues(t, c.enumValue, actualData)
			})
		})
	}
}
