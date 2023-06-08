package parachain

import (
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

// ValidatorIndex index of the validator is used as a lightweight replacement of the 'ValidatorId' when appropriate.
type ValidatorIndex uint32

// AssignmentCertKind different kinds of input or criteria that can prove a validator's assignment
// to check a particular parachain.
type AssignmentCertKind scale.VaryingDataType

type RelayVRFModulo struct {
	Sample uint32
}

func (rvm RelayVRFModulo) Index() uint {
	return 0
}

type RelayVRFDelay struct{}

func (rvd RelayVRFDelay) Index() uint {
	return 1
}

type VrfSignature struct {
	Output [32]byte
	Proof  [64]byte
}

// AssignmentCert a certification of assignment
type AssignmentCert struct {
	Kind AssignmentCertKind
	Vrf  VrfSignature
}

// IndirectAssignmentCert an assignment criterion which refers to the candidate under which the assignment is
// relevant by block hash.
type IndirectAssignmentCert struct {
	BlockHash common.Hash
	Validator ValidatorIndex
	Cert      AssignmentCert
}

type CandidateIndex uint32

type Assignment struct {
	IndirectAssignmentCert IndirectAssignmentCert
	CandidateIndex         CandidateIndex
}

type Assignments struct {
	Assignments []Assignment
}

func (a Assignments) Index() uint {
	return 0
}

type ValidatorSignature [64]byte

type IndirectSignedApprovalVote struct {
	BlockHash      common.Hash
	CandidateIndex CandidateIndex
	ValidatorIndex ValidatorIndex
	Signature      ValidatorSignature
}

type Approvals struct {
	Approvals []IndirectSignedApprovalVote
}

func (ms Approvals) Index() uint {
	return 1
}

// ApprovalDistributionMessage network messages used by approval distribution subsystem.
type ApprovalDistributionMessage scale.VaryingDataType
