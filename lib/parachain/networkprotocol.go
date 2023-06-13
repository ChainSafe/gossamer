package parachain

import (
	"fmt"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

// A static context used for all relay-vrf-modulo VRFs.
const RELAY_VRF_MODULO_CONTEXT = "A&V MOD"

// / A static context used for all relay-vrf-modulo VRFs.
const RELAY_VRF_DELAY_CONTEXT = "A&V DELAY"

// ValidatorIndex index of the validator is used as a lightweight replacement of the 'ValidatorId' when appropriate.
type ValidatorIndex uint32

// AssignmentCertKind different kinds of input or criteria that can prove a validator's assignment
// to check a particular parachain.
type AssignmentCertKind scale.VaryingDataType

func (ack AssignmentCertKind) New() AssignmentCertKind {
	return NewAssignmentCertKindVDT()
}

// Set will set VaryingDataTypeValue using undurlying VaryingDataType
func (ack *AssignmentCertKind) Set(val scale.VaryingDataTypeValue) (err error) {
	// cast to VaryingDataType to use VaryingDataType.Set method
	vdt := scale.VaryingDataType(*ack)
	err = vdt.Set(val)
	if err != nil {
		return fmt.Errorf("setting value te varying data type: %w", err)
	}
	// store ariginal ParentVDT with VaryingDataType that has been set
	*ack = AssignmentCertKind(vdt)
	return nil
}

// Value returns the value from the underlying VaryingDataType
func (ack *AssignmentCertKind) Value() (scale.VaryingDataTypeValue, error) {
	vdt := scale.VaryingDataType(*ack)
	return vdt.Value()
}

func NewAssignmentCertKindVDT() AssignmentCertKind {
	vdt, err := scale.NewVaryingDataType(NewRelayVRFModulo(), NewVRFDelay())
	if err != nil {
		panic(err)
	}
	return AssignmentCertKind(vdt)
}

// RelayVRFModulo an assignment story based on the VRF that authorized the relay-chain block where the
// candidate was included combined with a sample number.
//
// The context used to produce the bytes is RELAY_VRF_MODULO_CONTEXT
type RelayVRFModulo struct {
	// Sample the sample number used in this cert.
	Sample uint32
}

func NewRelayVRFModulo() RelayVRFModulo {
	return RelayVRFModulo{}
}

// Index returns varying data type index
func (rvm RelayVRFModulo) Index() uint {
	return 0
}

// RelayVRFDelay an assignment story based on the VRF that authorized the relay-chain block where the
// candidate was included combined with the index of a particular core.
//
// The context is RELAY_VRF_DELAY_CONTEXT
type RelayVRFDelay struct {
	// CoreIndex the unique (during session) index of a core.
	CoreIndex uint32
}

func NewVRFDelay() RelayVRFDelay {
	return RelayVRFDelay{}
}

// Index returns varying data type index
func (rvd RelayVRFDelay) Index() uint {
	return 1
}

// VrfSignature VRF signature data
type VrfSignature struct {
	// Output VRF output
	Output [sr25519.VRFOutputLength]byte `scale:"1"`
	// Proof VRF proof
	Proof [sr25519.VRFProofLength]byte `scale:"2"`
}

// AssignmentCert a certification of assignment
type AssignmentCert struct {
	// Kind the criterion which is claimed to be met by this cert.
	Kind AssignmentCertKind `scale:"1"`
	// Vrf the VRF signature showing the criterion is met.
	Vrf VrfSignature `scale:"2"`
}

// IndirectAssignmentCert an assignment criterion which refers to the candidate under which the assignment is
// relevant by block hash.
type IndirectAssignmentCert struct {
	// BlockHash a block hash where the canidate appears.
	BlockHash common.Hash `scale:"1"`
	// Validator the validator index.
	Validator ValidatorIndex `scale:"2"`
	// Cert the cert itself.
	Cert AssignmentCert `scale:"3"`
}

// CandidateIndex the index of the candidate in the list of candidates fully included as-of the block.
type CandidateIndex uint32

// Assignment holds indirect assignment cert and candidate index
type Assignment struct {
	IndirectAssignmentCert IndirectAssignmentCert `scale:"1"`
	CandidateIndex         CandidateIndex         `scale:"2"`
}

//func NewAssignment() Assignment {
//	assignment := Assignment{
//		IndirectAssignmentCert: IndirectAssignmentCert{
//			BlockHash: common.Hash{},
//			Validator: 0,
//			Cert: AssignmentCert{
//				Kind: AssignmentCertKind(NewAssignmentCertKindVDT()),
//				Vrf:  VrfSignature{},
//			},
//		},
//		CandidateIndex: 0,
//	}
//	return assignment
//}

// Assignments for candidates in recent, unfinalized blocks.
type Assignments struct {
	Assignments []Assignment
}

//func NewAssignments() Assignments {
//	assignemns := Assignments{}
//	assignemns.Assignments = append(assignemns.Assignments, NewAssignment())
//	return assignemns
//}

// Index returns varying data type index
func (a Assignments) Index() uint {
	return 0
}

// ValidatorSignature with which parachain validators sign blocks.
type ValidatorSignature [sr25519.SignatureLength]byte

// IndirectSignedApprovalVote A signed approval vote which references the candidate indirectly via the block.
//
// In practice, we have a look-up from block hash and candidate index to candidate hash,
// so this can be transformed into a `SignedApprovalVote`.
type IndirectSignedApprovalVote struct {
	// BlockHash a block hash where the candidate appears.
	BlockHash common.Hash `scale:"1"`
	// CandidateIndex the index of the candidate in the list of candidates fully included as-of the block.
	CandidateIndex CandidateIndex `scale:"2"`
	// ValidatorIndex the validator index.
	ValidatorIndex ValidatorIndex `scale:"3"`
	// Signature the signature of the validator.
	Signature ValidatorSignature `scale:"4"`
}

// Approvals for candidates in some recent, unfinalized block.
type Approvals struct {
	Approvals []IndirectSignedApprovalVote
}

// Index returns varying data type index
func (ms Approvals) Index() uint {
	return 1
}

// ApprovalDistributionMessage network messages used by approval distribution subsystem.
type ApprovalDistributionMessage scale.VaryingDataType

// Set will set a VoryingDataTypeValue using the underlying VaryingDataType
func (adm *ApprovalDistributionMessage) Set(val scale.VaryingDataTypeValue) (err error) {
	// cast to VaryingDataType to use VaryingDataType.Set method
	vdt := scale.VaryingDataType(*adm)
	err = vdt.Set(val)
	if err != nil {
		return fmt.Errorf("setting value to varying data type: %w", err)
	}
	// store original ParentVDT with VaryingDataType that has been set
	*adm = ApprovalDistributionMessage(vdt)
	return nil
}

// Value returns the value from the underlying VaryingDataType
func (adm *ApprovalDistributionMessage) Value() (scale.VaryingDataTypeValue, error) {
	vdt := scale.VaryingDataType(*adm)
	return vdt.Value()
}

func (adm ApprovalDistributionMessage) New() ApprovalDistributionMessage {
	return NewApprovalDistributionMessageVDT()
}

// NewApprovalDistributionMessageVDT ruturns a new ApprovalDistributionMessage VaryingDataType
func NewApprovalDistributionMessageVDT() ApprovalDistributionMessage {
	vdt, err := scale.NewVaryingDataType(Assignments{}, Approvals{})
	if err != nil {
		panic(err)
	}
	return ApprovalDistributionMessage(vdt)
}
