// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package validationprotocol

import (
	"bytes"
	_ "embed"
	"fmt"
	"testing"

	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/lib/common"
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
	approvalDistribution := ApprovalDistribution{NewApprovalDistributionMessageVDT()}
	approvalDistribution.SetValue(Assignments{
		Assignment{
			IndirectAssignmentCert: fakeAssignmentCert(hashA, parachaintypes.ValidatorIndex(1), false),
			CandidateIndex:         4,
		},
	})
	vpApprovalDistributionAssignments := NewValidationProtocolVDT()
	vpApprovalDistributionAssignments.SetValue(approvalDistribution)
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
	var validatorSignature parachaintypes.ValidatorSignature
	tempSignature := common.MustHexToBytes(testValidationProtocolHex["validatorSignature"])
	copy(validatorSignature[:], tempSignature)

	approvalDistributionApprovals := ApprovalDistribution{NewApprovalDistributionMessageVDT()}
	approvalDistributionApprovals.SetValue(Approvals{
		IndirectSignedApprovalVote{
			BlockHash:      hashA,
			CandidateIndex: 10,
			ValidatorIndex: 11,
			Signature:      validatorSignature,
		},
	})

	vpApprovalDistributionApprovals := NewValidationProtocolVDT()
	vpApprovalDistributionApprovals.SetValue(approvalDistributionApprovals)
	vpApprovalDistributionApprovalsValue, err := vpApprovalDistributionApprovals.Value()
	require.NoError(t, err)

	/* ValidationProtocol with StatementDistribution with Statement rust code:
	fn try_validation_protocol_statement_distribution_full_statement() {
		let data : &[u8; 64] = &[
			198, 124, 185, 59, 240, 163, 111, 206, 227, 210, 157, 232, 166, 166, 154, 117, 150, 89,
			104, 10, 207, 72, 100, 117, 224, 162, 85, 42, 95, 190, 216, 126, 69, 173, 206, 95, 41,
			6, 152, 216, 89, 96, 149, 114, 43, 51, 89, 146, 39, 247, 70, 31, 81, 175, 134, 23, 200,
			190, 116, 184, 148, 207, 27, 134,
		];
		let val_sign = ValidatorSignature::try_from(&data[..]).unwrap();

		let hash1 = Hash::repeat_byte(10);
		let hash2 = Hash::repeat_byte(15);
		let compact = UncheckedSignedStatement::new(
			CompactStatement::Seconded(polkadot_primitives::CandidateHash(hash2)),
			ValidatorIndex(5),
			val_sign,
		);

		let msg = v3::StatementDistributionMessage::Statement(hash1, compact);

		let validation_msg = v3::ValidationProtocol::StatementDistribution(msg);

		println!("encode validation SecondedStatement => {:?}\n\n", validation_msg.encode());
	}
	*/
	stmtRelayParentHash := common.Hash(bytes.Repeat([]byte{10}, 32))
	compact := parachaintypes.NewCompactSeconded(
		parachaintypes.CandidateHash{Value: common.Hash(bytes.Repeat([]byte{15}, 32))})

	sdm := NewStatementDistributionMessage()
	err = sdm.SetValue(Statement{
		RelayParent: stmtRelayParentHash,
		Compact: parachaintypes.UncheckedSignedCompactStatement{
			Payload:        *compact.ToEncodable(),
			ValidatorIndex: 5,
			Signature:      validatorSignature,
		},
	})
	require.NoError(t, err)

	vpStatementDistributionStatement := NewValidationProtocolVDT()
	err = vpStatementDistributionStatement.SetValue(StatementDistribution{StatementDistributionMessage: sdm})
	require.NoError(t, err)

	vpStatementDistributionStatementValue, err := vpStatementDistributionStatement.Value()
	require.NoError(t, err)

	/* ValidationProtocol with StatementDistribution with BackedCandidateManifest rust code
	fn try_validation_protocol_statement_distribution() {
		let hash1 = Hash::repeat_byte(10);
		let hash2 = Hash::repeat_byte(15);
		let data = v3::BackedCandidateManifest{
			relay_parent: hash1,
				candidate_hash: polkadot_primitives::CandidateHash(hash2) ,
				group_index: GroupIndex(0),
				para_id: polkadot_primitives::Id::from(0),
				parent_head_data_hash: hash1,
				statement_knowledge: v3::StatementFilter::blank(0),
		};

		let msg = v3::StatementDistributionMessage::BackedCandidateManifest(data);
		let validation_msg = v3::ValidationProtocol::StatementDistribution(msg);
		println!("encode validation SecondedStatement => {:?}\n\n", validation_msg.encode());
	}
	*/

	stmtFilter, err := parachaintypes.NewStatementFilter(0, false)
	require.NoError(t, err)

	backedCandidate := BackedCandidateManifest{
		RelayParent:        common.Hash(bytes.Repeat([]byte{10}, 32)),
		CandidateHash:      parachaintypes.CandidateHash{Value: common.Hash(bytes.Repeat([]byte{15}, 32))},
		GroupIndex:         parachaintypes.GroupIndex(0),
		ParaID:             parachaintypes.ParaID(0),
		ParentHeadDataHash: common.Hash(bytes.Repeat([]byte{10}, 32)),
		StatementKnwoledge: *stmtFilter,
	}

	sdm = NewStatementDistributionMessage()
	err = sdm.SetValue(backedCandidate)
	require.NoError(t, err)

	vp := NewValidationProtocolVDT()
	err = vp.SetValue(StatementDistribution{StatementDistributionMessage: sdm})
	require.NoError(t, err)

	backedCandidateManifestSDM, err := vp.Value()
	require.NoError(t, err)

	/* ValidationProtocol with StatementDistribution with BackedCandidateKnown rust code
	fn try_validation_protocol_statement_distribution() {
		let hash1 = Hash::repeat_byte(10);
		let hash2 = Hash::repeat_byte(15);
		let data = v3::BackedCandidateAcknowledgement{
			candidate_hash: polkadot_primitives::CandidateHash(hash2),
		 	statement_knowledge: v3::StatementFilter::blank(0),
		};

		let msg = v3::StatementDistributionMessage::BackedCandidateKnown(data);

		let validation_msg = v3::ValidationProtocol::StatementDistribution(msg);

		println!("encode validation SecondedStatement => {:?}\n\n", validation_msg.encode());
	}
	*/

	candidateKnown := BackedCandidateKnown{
		CandidateHash:      parachaintypes.CandidateHash{Value: common.Hash(bytes.Repeat([]byte{15}, 32))},
		StatementKnwoledge: *stmtFilter,
	}

	sdm = NewStatementDistributionMessage()
	err = sdm.SetValue(candidateKnown)
	require.NoError(t, err)

	candidateKnownVP := NewValidationProtocolVDT()
	err = candidateKnownVP.SetValue(StatementDistribution{StatementDistributionMessage: sdm})
	require.NoError(t, err)

	backedCandidateKnownSDM, err := candidateKnownVP.Value()
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

	bitfieldDistribution := BitfieldDistribution{NewBitfieldDistributionMessageVDT()}
	bitfieldDistribution.SetValue(UncheckedBitfield{
		Hash: hashA,
		UncheckedSignedAvailabilityBitfield: parachaintypes.UncheckedSignedAvailabilityBitfield{
			Payload: func() parachaintypes.BitVec {
				bv, err := parachaintypes.NewBitVec([]bool{true, true, true, true, true, true, true, true, true, true, true,
					true, true, true, true, true, true, true, true, true, true, true, true, true, true, true, true, true,
					true, true, true, true})
				require.NoError(t, err)

				return bv
			}(),
			ValidatorIndex: 0,
			Signature:      validatorSignature,
		},
	})

	vpBitfieldDistribution := NewValidationProtocolVDT()
	vpBitfieldDistribution.SetValue(bitfieldDistribution)
	vpBitfieldDistributionVal, err := vpBitfieldDistribution.Value()
	require.NoError(t, err)

	testCases := map[string]struct {
		enumValue     any
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
			enumValue: vpStatementDistributionStatementValue,
			encodingValue: common.MustHexToBytes(
				testValidationProtocolHex["statementDistributionMessageStatement"]),
		},
		"ValidationProtocol_with_StatementDistribution_with_BackedCandidate": {
			enumValue: backedCandidateManifestSDM,
			encodingValue: common.MustHexToBytes(
				testValidationProtocolHex["statementDistributionMessageBackedCandidateManifest"]),
		},
		"ValidationProtocol_with_StatementDistribution_with_BackedCandidateKnown": {
			enumValue: backedCandidateKnownSDM,
			encodingValue: common.MustHexToBytes(
				testValidationProtocolHex["statementDistributionMessageBackedCandidateAcknowledgement"]),
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
			err := validationProtocol.SetValue(c.enumValue)
			require.NoError(t, err)

			encoded, err := scale.Marshal(validationProtocol)
			require.NoError(t, err)
			require.Equal(t, c.encodingValue, encoded)
		})
	}
}
