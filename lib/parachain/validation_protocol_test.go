// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package parachain

import (
	_ "embed"
	"fmt"
	"testing"

	"github.com/ChainSafe/gossamer/lib/common"
	parachaintypes "github.com/ChainSafe/gossamer/lib/parachain/types"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

//go:embed testdata/validation_protocol.yaml

var testValidationProtocolHexRaw string

var testValidationProtocolHex map[string]string

func init() {
	err := yaml.Unmarshal([]byte(testValidationProtocolHexRaw), &testValidationProtocolHex)
	if err != nil {
		fmt.Printf("Error unmarshaling test data: %s\n", err)
		return
	}
}

func TestMarshalUnMarshalValidationProtocol(t *testing.T) {
	t.Parallel()
	/* ValidationProtocol with ApprovalDistribution with Assignments Rust code
	fn try_msg_assignments_encode() {
		let hash = Hash::repeat_byte(0xAA);
		let validator_index = ValidatorIndex(1);
		let cert = fake_assignment_cert(hash, validator_index);
		let assignments = vec![(cert.clone(), 4u32)];
		let msg = protocol_v1::ApprovalDistributionMessage::Assignments(assignments.clone());
		let val_proto = protocol_v1::ValidationProtocol::ApprovalDistribution(msg.clone());
		println!("encode validation proto => {:?}\n\n", val_proto.encode());
	}
	*/
	approvalDistribution := NewApprovalDistributionVDT()
	approvalDistribution.Set(Assignments{
		Assignment{
			IndirectAssignmentCert: fakeAssignmentCert(hash, parachaintypes.ValidatorIndex(1), false),
			CandidateIndex:         4,
		},
	})
	vpApprovalDistributionAssignments := NewValidationProtocolVDT()
	vpApprovalDistributionAssignments.Set(approvalDistribution)
	vpApprovalDistributionAssignmentsValue, err := vpApprovalDistributionAssignments.Value()
	require.NoError(t, err)

	/* ValidationProtocol with ApprovalDistribution with Approvals rust code:
	fn try_msg_approvals_encode() {
		let hash = Hash::repeat_byte(0xAA);
		let candidate_index = 0u32;
		let validator_index = ValidatorIndex(0);
		let approval = IndirectSignedApprovalVote {
			block_hash: hash,
			candidate_index,
			validator: validator_index,
			signature: dummy_signature(),
		};
		let msg = protocol_v1::ApprovalDistributionMessage::Approvals(vec![approval.clone()]);
		let val_proto = protocol_v1::ValidationProtocol::ApprovalDistribution(msg.clone());
		println!("encode validation proto => {:?}\n\n", val_proto.encode());
	}
	*/
	var validatorSignature ValidatorSignature
	tempSignature := common.MustHexToBytes(testValidationProtocolHex["validatorSignature"])
	copy(validatorSignature[:], tempSignature)

	approvalDistributionApprovals := NewApprovalDistributionVDT()
	approvalDistributionApprovals.Set(Approvals{
		IndirectSignedApprovalVote{
			BlockHash:      hash,
			CandidateIndex: 10,
			ValidatorIndex: 11,
			Signature:      validatorSignature,
		},
	})

	vpApprovalDistributionApprovals := NewValidationProtocolVDT()
	vpApprovalDistributionApprovals.Set(approvalDistributionApprovals)
	vpApprovalDistributionApprovalsValue, err := vpApprovalDistributionApprovals.Value()
	require.NoError(t, err)

	/* ValidationProtocol with StatementDistribution with Statement rust code:
	 fn try_validation_protocol_statement_distribution_full_statement() {
		let hash1 = Hash::repeat_byte(170);
		let val_sign = ValidatorSignature::from(
			Signature([198, 124, 185, 59, 240, 163, 111, 206, 227, 210, 157, 232, 166, 166, 154, 117, 150, 89,
				104, 10, 207, 72, 100, 117, 224, 162, 85, 42, 95, 190, 216, 126, 69, 173, 206, 95, 41, 6, 152,
				216, 89, 96, 149, 114, 43, 51, 89, 146, 39, 247, 70, 31, 81, 175, 134, 23, 200, 190, 116, 184,
				148, 207, 27, 134]));
		let keystore: KeystorePtr = Arc::new(LocalKeystore::in_memory());
		let collator_result = Keystore::sr25519_generate_new(
			&*keystore,
			ValidatorId::ID,
			Some(&Sr25519Keyring::Alice.to_seed()),
		);
		let collator = collator_result.unwrap();
		let collsign = CollatorSignature::from(Signature([198, 124, 185, 59, 240, 163, 111, 206, 227, 210, 157, 232,
			166, 166, 154, 117, 150, 89, 104, 10, 207, 72, 100, 117, 224, 162, 85, 42, 95, 190, 216, 126, 69, 173, 206,
			95, 41, 6, 152, 216, 89, 96, 149, 114, 43, 51, 89, 146, 39, 247, 70, 31, 81, 175, 134, 23, 200, 190, 116,
			184, 148, 207, 27, 134]));
		let candidate_descriptor = CandidateDescriptor{
			para_id: 1.into(),
			relay_parent: hash1,
			collator: polkadot_primitives::CollatorId::from(collator),
			persisted_validation_data_hash: hash1,
			pov_hash: hash1,
			erasure_root: hash1,
			signature: collsign,
			para_head: hash1,
			validation_code_hash: ValidationCodeHash::from(hash1)
		};
		let commitments_new = CandidateCommitments{
			upward_messages: vec![vec![1, 2, 3]].try_into().expect("error - upward_messages"),
			horizontal_messages: vec![].try_into().expect("error - horizontal_messages"),
			head_data: HeadData(vec![1, 2, 3]),
			hrmp_watermark: 0_u32,
			new_validation_code: ValidationCode(vec![1, 2, 3]).try_into().expect("error - new_validation_code"),
			processed_downward_messages: 5
		};
		let committed_candidate_receipt = CommittedCandidateReceipt {
			 descriptor: candidate_descriptor, commitments: commitments_new };
		let statement_second = Statement::Seconded(committed_candidate_receipt);
		let unchecked_signed_full_statement_second = UncheckedSignedFullStatement::new(
			statement_second, ValidatorIndex(5), val_sign.clone());
		let sdm_statement_second = protocol_v1::StatementDistributionMessage::Statement(hash1,
			unchecked_signed_full_statement_second);
		let validation_sdm_statement = protocol_v1::ValidationProtocol::StatementDistribution(sdm_statement_second);
		println!("encode validation SecondedStatement => {:?}\n\n", validation_sdm_statement.encode());

	}
	*/
	var collatorID parachaintypes.CollatorID
	tempID := common.MustHexToBytes(testValidationProtocolHex["collatorID"])
	copy(collatorID[:], tempID)
	var collatorSignature parachaintypes.CollatorSignature
	copy(collatorSignature[:], tempSignature)

	statementSecond := Seconded{
		Descriptor: parachaintypes.CandidateDescriptor{
			ParaID:                      1,
			RelayParent:                 hash,
			Collator:                    collatorID,
			PersistedValidationDataHash: hash,
			PovHash:                     hash,
			ErasureRoot:                 hash,
			Signature:                   collatorSignature,
			ParaHead:                    hash,
			ValidationCodeHash:          parachaintypes.ValidationCodeHash(hash),
		},
		Commitments: parachaintypes.CandidateCommitments{
			UpwardMessages:            []parachaintypes.UpwardMessage{[]byte{1, 2, 3}},
			HorizontalMessages:        nil,
			NewValidationCode:         &parachaintypes.ValidationCode{1, 2, 3},
			HeadData:                  []byte{1, 2, 3},
			ProcessedDownwardMessages: 5,
			HrmpWatermark:             0,
		},
	}
	statement := NewStatement()
	statement.Set(statementSecond)

	statementDistributionStatement := NewStatementDistributionVDT()
	statementDistributionStatement.Set(SignedFullStatement{
		Hash: hash,
		UncheckedSignedFullStatement: UncheckedSignedFullStatement{
			Payload:        statement,
			ValidatorIndex: 5,
			Signature:      validatorSignature,
		},
	})

	vpStatementDistributionStatement := NewValidationProtocolVDT()
	vpStatementDistributionStatement.Set(statementDistributionStatement)
	vpStatementDistributionStatementValue, err := vpStatementDistributionStatement.Value()
	require.NoError(t, err)

	/* ValidationProtocol with StatementDistribution with Large Statement rust code
	fn try_validation_protocol_statement_distribution() {
		let hash1 = Hash::repeat_byte(170);
		let val_sign = ValidatorSignature::from(
			Signature([198, 124, 185, 59, 240, 163, 111, 206, 227, 210, 157, 232, 166, 166, 154, 117, 150, 89,
				104, 10, 207, 72, 100, 117, 224, 162, 85, 42, 95, 190, 216, 126, 69, 173, 206, 95, 41, 6, 152,
				216, 89, 96, 149, 114, 43, 51, 89, 146, 39, 247, 70, 31, 81, 175, 134, 23, 200, 190, 116, 184,
				148, 207, 27, 134]));
		let sdm_large_statement = protocol_v1::StatementDistributionMessage::LargeStatement(StatementMetadata{
			relay_parent: hash1,
			candidate_hash: CandidateHash(hash1),
			signed_by: ValidatorIndex(5_u32),
			signature: val_sign.clone(),
		});
		let validation_sdm_large_statement = protocol_v1::ValidationProtocol::StatementDistribution(sdm_large_statement);
		println!("encode validation SecondedStatementWithLargePayload => {:?}\n\n", validation_sdm_large_statement.encode());
	}
	*/
	statementDistributionLargeStatement := NewStatementDistributionVDT()
	statementDistributionLargeStatement.Set(SecondedStatementWithLargePayload{
		RelayParent:   hash,
		CandidateHash: CandidateHash{Value: hash},
		SignedBy:      5,
		Signature:     validatorSignature,
	})

	vpStatementDistributionLargeStatement := NewValidationProtocolVDT()
	vpStatementDistributionLargeStatement.Set(statementDistributionLargeStatement)
	vpStatementDistributionLargeStatementValue, err := vpStatementDistributionLargeStatement.Value()
	require.NoError(t, err)

	/* ValidationProtocol with BitfieldDistribution rust code
	fn try_validation_protocol_bitfield_distribution_a() {
		let hash_a :Hash = [170; 32].into();
		let keystore: KeystorePtr = Arc::new(MemoryKeystore::new());
		let payload = AvailabilityBitfield(bitvec![u8, bitvec::order::Lsb0; 1u8; 32]);
		let signing_context = SigningContext { session_index: 1, parent_hash: hash_a.clone() };
		let validator_0 =
			Keystore::sr25519_generate_new(&*keystore, ValidatorId::ID, None).expect("key created");
		let valid_signed = Signed::<AvailabilityBitfield>::sign(
			&keystore,
			payload,
			&signing_context,
			ValidatorIndex(0),
			&validator_0.into(),
		)
		.ok()
		.flatten()
		.expect("should be signed");
		let bitfield_distribition_message = protocol_v1::BitfieldDistributionMessage::Bitfield(
			hash_a,
			valid_signed.into(),
		);
		let val_proto = ValidationProtocol::BitfieldDistribution(bitfield_distribition_message.clone());
		println!("encode validation proto => {:?}\n\n", val_proto.encode());
	}
	*/
	bitfieldDistribution := NewBitfieldDistributionVDT()
	bitfieldDistribution.Set(Bitfield{
		Hash: hash,
		UncheckedSignedAvailabilityBitfield: UncheckedSignedAvailabilityBitfield{
			Payload: scale.NewBitVec([]bool{true, true, true, true, true, true, true, true, true, true, true,
				true, true, true, true, true, true, true, true, true, true, true, true, true, true, true, true, true,
				true, true, true, true}),
			ValidatorIndex: 0,
			Signature:      validatorSignature,
		},
	})

	vpBitfieldDistribution := NewValidationProtocolVDT()
	vpBitfieldDistribution.Set(bitfieldDistribution)
	vpBitfieldDistributionVal, err := vpBitfieldDistribution.Value()
	require.NoError(t, err)

	testCases := map[string]struct {
		enumValue     scale.VaryingDataTypeValue
		encodingValue []byte
	}{
		"ValidationProtocol_with_ApprovalDistribution_with_Assignments": {
			enumValue:     vpApprovalDistributionAssignmentsValue,
			encodingValue: common.MustHexToBytes(testValidationProtocolHex["approvalDistributionMessageAssignments"]),
		},
		"ValidationProtocol_with_ApprovalDistribution_with_Approvals": {
			enumValue:     vpApprovalDistributionApprovalsValue,
			encodingValue: common.MustHexToBytes(testValidationProtocolHex["approvalDistributionMessageApprovals"]),
		},
		"ValidationProtocol_with_StatementDistribution_with_Statement": {
			enumValue:     vpStatementDistributionStatementValue,
			encodingValue: common.MustHexToBytes(testValidationProtocolHex["statementDistributionMessageStatement"]),
		},
		"ValidationProtocol_with_StatementDistribution_with_Large_Statement": {
			enumValue:     vpStatementDistributionLargeStatementValue,
			encodingValue: common.MustHexToBytes(testValidationProtocolHex["statementDistributionMessageLargeStatement"]),
		},
		"ValidationProtocol_with_BitfieldDistribution": {
			enumValue:     vpBitfieldDistributionVal,
			encodingValue: common.MustHexToBytes(testValidationProtocolHex["bitfieldDistribution"]),
		},
	}

	for name, c := range testCases {
		c := c
		t.Run("unmarshal "+name, func(t *testing.T) {
			t.Parallel()

			validationProtocol := NewValidationProtocolVDT()

			err := scale.Unmarshal(c.encodingValue, &validationProtocol)
			require.NoError(t, err)

			validationProtocolDecoded, err := validationProtocol.Value()
			require.NoError(t, err)
			require.Equal(t, c.enumValue, validationProtocolDecoded)
		})
		t.Run("marshal "+name, func(t *testing.T) {
			t.Parallel()

			validationProtocol := NewValidationProtocolVDT()
			err := validationProtocol.Set(c.enumValue)
			require.NoError(t, err)

			encoded, err := scale.Marshal(validationProtocol)
			require.NoError(t, err)
			require.Equal(t, c.encodingValue, encoded)
		})
	}
}
