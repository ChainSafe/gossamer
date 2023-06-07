package parachain

import (
	"fmt"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/keystore"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/gtank/merlin"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestEncodeApprovalDistributionMessage(t *testing.T) {
	approvalDistributionMessage, err := scale.NewVaryingDataType(Assignments{}, Approvals{})
	require.NoError(t, err)
	hash := common.Hash{0xAA, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA,
		0xAA, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA}

	expectedEncoding := []byte{0, 4, 170, 170, 170, 170, 170, 170, 170, 170, 170, 170, 170, 170, 170, 170, 170, 170, 170, 170, 170, 170, 170, 170, 170, 170, 170, 170, 170, 170, 170, 170, 170, 170, 0, 0, 0, 0, 0, 1, 0, 0, 0, 46, 144, 255, 239, 40, 55, 228, 78, 222, 92, 94, 244, 137, 127, 156, 251, 80, 27, 143, 38, 117, 132, 171, 66, 234, 113, 41, 83, 39, 175, 191, 85, 195, 149, 8, 236, 143, 153, 239, 9, 171, 76, 214, 28, 120, 70, 136, 160, 199, 132, 159, 244, 32, 224, 186, 80, 27, 142, 161, 118, 188, 133, 51, 8, 229, 197, 156, 193, 28, 201, 15, 144, 143, 147, 107, 212, 52, 152, 1, 64, 108, 217, 44, 155, 243, 128, 215, 226, 46, 64, 175, 18, 193, 38, 156, 2, 0, 0, 0, 0}

	approvalDistributionMessage.Set(Assignments{
		Assignments: []Assignment{{
			IndirectAssignmentCert: fakeAssignmentCert(hash, ValidatorIndex(0)),
			CandidateIndex:         0,
		}},
	})
	fmt.Printf("MSG: %#v\n", approvalDistributionMessage)
	encodedMessage, err := scale.Marshal(approvalDistributionMessage)
	require.NoError(t, err)
	fmt.Printf("bytes: %v\n", encodedMessage)
	require.Equal(t, expectedEncoding, encodedMessage)
}

func fakeAssignmentCert(blockHash common.Hash, validator ValidatorIndex) IndirectAssignmentCert {
	msg := []byte(`WhenParachains?`)
	fmt.Printf("msg: %v\n", msg)
	keyring, err := keystore.NewSr25519Keyring()
	fmt.Printf("keystore err %v\n", err)

	transcript := merlin.NewTranscript("A&V MOD")
	transcript.AppendMessage(msg, []byte{})

	output, proof, err := keyring.KeyAlice.VrfSign(transcript)
	fmt.Printf("output %v\n", output)
	fmt.Printf("proof  %v\n", proof)

	assignmentCertKind, err := scale.NewVaryingDataType(RelayVRFModulo{}, RelayVRFDelay{})
	assignmentCertKind.Set(RelayVRFModulo{sample: 1})
	return IndirectAssignmentCert{
		BlockHash: blockHash,
		Validator: validator,
		Cert: AssignmentCert{
			Kind: AssignmentCertKind(assignmentCertKind),
			Vrf: VrfSignature{
				Output: output,
				Proof:  proof,
			},
		},
	}
}
