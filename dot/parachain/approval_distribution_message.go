// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package parachain

import (
	"fmt"

	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

// AssignmentCertKind different kinds of input or criteria that can prove a validator's assignment
// to check a particular parachain.
type AssignmentCertKind scale.VaryingDataType

// New will enable scale to create new instance when needed
func (AssignmentCertKind) New() AssignmentCertKind {
	return NewAssignmentCertKindVDT()
}

// Set will set VaryingDataTypeValue using undurlying VaryingDataType
func (ack *AssignmentCertKind) Set(val scale.VaryingDataTypeValue) (err error) {
	vdt := scale.VaryingDataType(*ack)
	err = vdt.Set(val)
	if err != nil {
		return fmt.Errorf("setting value te varying data type: %w", err)
	}
	*ack = AssignmentCertKind(vdt)
	return nil
}

// Value returns the value from the underlying VaryingDataType
func (ack *AssignmentCertKind) Value() (scale.VaryingDataTypeValue, error) {
	vdt := scale.VaryingDataType(*ack)
	return vdt.Value()
}

// NewAssignmentCertKindVDT constructor for AssignmentCertKind
func NewAssignmentCertKindVDT() AssignmentCertKind {
	vdt, err := scale.NewVaryingDataType(NewRelayVRFModulo(), NewVRFDelay())
	if err != nil {
		panic(err)
	}
	return AssignmentCertKind(vdt)
}

// RelayVRFModulo an assignment story based on the VRF that authorized the relay-chain block where the
// candidate was included combined with a sample number.
type RelayVRFModulo struct {
	// Sample the sample number used in this cert.
	Sample uint32
}

// NewRelayVRFModulo constructor for RelayVRFModulo
func NewRelayVRFModulo() RelayVRFModulo {
	return RelayVRFModulo{}
}

// Index returns varying data type index
func (rvm RelayVRFModulo) Index() uint {
	return 0
}

// RelayVRFDelay an assignment story based on the VRF that authorized the relay-chain block where the
// candidate was included combined with the index of a particular core.
type RelayVRFDelay struct {
	// CoreIndex the unique (during session) index of a core.
	CoreIndex uint32
}

// NewVRFDelay constructor for RelayVRFDelay
func NewVRFDelay() RelayVRFDelay {
	return RelayVRFDelay{}
}

// Index returns varying data type index
func (RelayVRFDelay) Index() uint {
	return 1
}

// VrfSignature represents VRF signature, which itself consists of a VRF pre-output and DLEQ proof
type VrfSignature struct {
	// Output VRF output
	Output [sr25519.VRFOutputLength]byte `scale:"1"`
	// Proof VRF proof
	Proof [sr25519.VRFProofLength]byte `scale:"2"`
}

// AssignmentCert is a certification of assignment
type AssignmentCert struct {
	// Kind the criterion which is claimed to be met by this cert.
	Kind AssignmentCertKind `scale:"1"`
	// Vrf the VRF signature showing the criterion is met.
	Vrf VrfSignature `scale:"2"`
}

// IndirectAssignmentCert is an assignment criterion which refers to the candidate under which the assignment is
// relevant by block hash.
type IndirectAssignmentCert struct {
	// BlockHash a block hash where the canidate appears.
	BlockHash common.Hash `scale:"1"`
	// Validator the validator index.
	Validator parachaintypes.ValidatorIndex `scale:"2"`
	// Cert the cert itself.
	Cert AssignmentCert `scale:"3"`
}

// CandidateIndex represents the index of the candidate in the list of candidates fully included as-of the block.
type CandidateIndex uint32

// Assignment holds indirect assignment cert and candidate index
type Assignment struct {
	IndirectAssignmentCert IndirectAssignmentCert `scale:"1"`
	CandidateIndex         CandidateIndex         `scale:"2"`
}

// Assignments for candidates in recent, unfinalized blocks.
type Assignments []Assignment

// Index returns varying data type index
func (Assignments) Index() uint {
	return 0
}

// IndirectSignedApprovalVote represents a signed approval vote which references the candidate indirectly via the block.
type IndirectSignedApprovalVote struct {
	// BlockHash a block hash where the candidate appears.
	BlockHash common.Hash `scale:"1"`
	// CandidateIndex the index of the candidate in the list of candidates fully included as-of the block.
	CandidateIndex CandidateIndex `scale:"2"`
	// ValidatorIndex the validator index.
	ValidatorIndex parachaintypes.ValidatorIndex `scale:"3"`
	// Signature the signature of the validator.
	Signature parachaintypes.ValidatorSignature `scale:"4"`
}

// Approvals for candidates in some recent, unfinalized block.
type Approvals []IndirectSignedApprovalVote

// Index returns varying data type index
func (Approvals) Index() uint {
	return 1
}

// ApprovalDistributionMessage network messages used by approval distribution subsystem.
type ApprovalDistributionMessage scale.VaryingDataType

// Set will set a VoryingDataTypeValue using the underlying VaryingDataType
func (adm *ApprovalDistributionMessage) Set(val scale.VaryingDataTypeValue) (err error) {
	vdt := scale.VaryingDataType(*adm)
	err = vdt.Set(val)
	if err != nil {
		return fmt.Errorf("setting value to varying data type: %w", err)
	}
	*adm = ApprovalDistributionMessage(vdt)
	return nil
}

// Value returns the value from the underlying VaryingDataType
func (adm *ApprovalDistributionMessage) Value() (scale.VaryingDataTypeValue, error) {
	vdt := scale.VaryingDataType(*adm)
	return vdt.Value()
}

// New returns new ApprovalDistributionMessage VDT
func (ApprovalDistributionMessage) New() ApprovalDistributionMessage {
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
