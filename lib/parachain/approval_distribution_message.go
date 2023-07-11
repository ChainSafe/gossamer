// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package parachain

import (
	"fmt"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"
	parachaintypes "github.com/ChainSafe/gossamer/lib/parachain/types"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

type AssignmentCertKindValues interface {
	RelayVRFModulo | RelayVRFDelay
}

// AssignmentCertKind different kinds of input or criteria that can prove a validator's assignment
// to check a particular parachain.
type AssignmentCertKind struct {
	inner any
}

func setAssignmentCertKind[Value AssignmentCertKindValues](mvdt *AssignmentCertKind, value Value) {
	mvdt.inner = value
}

func (mvdt *AssignmentCertKind) SetValue(value any) (err error) {
	switch value := value.(type) {
	case RelayVRFModulo:
		setAssignmentCertKind(mvdt, value)
		return

	case RelayVRFDelay:
		setAssignmentCertKind(mvdt, value)
		return

	default:
		return fmt.Errorf("unsupported type")
	}
}

func (mvdt AssignmentCertKind) IndexValue() (index uint, value any, err error) {
	switch mvdt.inner.(type) {
	case RelayVRFModulo:
		return 0, mvdt.inner, nil

	case RelayVRFDelay:
		return 1, mvdt.inner, nil

	}
	return 0, nil, scale.ErrUnsupportedVaryingDataTypeValue
}

func (mvdt AssignmentCertKind) Value() (value any, err error) {
	_, value, err = mvdt.IndexValue()
	return
}

func (mvdt AssignmentCertKind) ValueAt(index uint) (value any, err error) {
	switch index {
	case 0:
		return *new(RelayVRFModulo), nil

	case 1:
		return *new(RelayVRFDelay), nil

	}
	return nil, scale.ErrUnknownVaryingDataTypeValue
}

// NewAssignmentCertKindVDT constructor for AssignmentCertKind
func NewAssignmentCertKindVDT() AssignmentCertKind {
	return AssignmentCertKind{}
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

// IndirectSignedApprovalVote represents a signed approval vote which references the candidate indirectly via the block.
type IndirectSignedApprovalVote struct {
	// BlockHash a block hash where the candidate appears.
	BlockHash common.Hash `scale:"1"`
	// CandidateIndex the index of the candidate in the list of candidates fully included as-of the block.
	CandidateIndex CandidateIndex `scale:"2"`
	// ValidatorIndex the validator index.
	ValidatorIndex parachaintypes.ValidatorIndex `scale:"3"`
	// Signature the signature of the validator.
	Signature ValidatorSignature `scale:"4"`
}

// Approvals for candidates in some recent, unfinalized block.
type Approvals []IndirectSignedApprovalVote

// ApprovalDistributionMessage network messages used by approval distribution subsystem.
type ApprovalDistributionMessageValues interface {
	Assignments | Approvals
}

// ApprovalDistributionMessage network messages used by approval distribution subsystem.
type ApprovalDistributionMessage struct {
	inner any
}

func setApprovalDistributionMessage[Value ApprovalDistributionMessageValues](mvdt *ApprovalDistributionMessage, value Value) {
	mvdt.inner = value
}

func (mvdt *ApprovalDistributionMessage) SetValue(value any) (err error) {
	switch value := value.(type) {
	case Assignments:
		setApprovalDistributionMessage(mvdt, value)
		return

	case Approvals:
		setApprovalDistributionMessage(mvdt, value)
		return

	default:
		return fmt.Errorf("unsupported type")
	}
}

func (mvdt ApprovalDistributionMessage) IndexValue() (index uint, value any, err error) {
	switch mvdt.inner.(type) {
	case Assignments:
		return 0, mvdt.inner, nil

	case Approvals:
		return 1, mvdt.inner, nil

	}
	return 0, nil, scale.ErrUnsupportedVaryingDataTypeValue
}

func (mvdt ApprovalDistributionMessage) Value() (value any, err error) {
	_, value, err = mvdt.IndexValue()
	return
}

func (mvdt ApprovalDistributionMessage) ValueAt(index uint) (value any, err error) {
	switch index {
	case 0:
		return *new(Assignments), nil

	case 1:
		return *new(Approvals), nil

	}
	return nil, scale.ErrUnknownVaryingDataTypeValue
}

// NewApprovalDistributionMessageVDT ruturns a new ApprovalDistributionMessage VaryingDataType
func NewApprovalDistributionMessageVDT() ApprovalDistributionMessage {
	return ApprovalDistributionMessage{}
}
