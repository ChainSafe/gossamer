package types

import (
	"fmt"

	"github.com/ChainSafe/gossamer/dot/parachain"
	parachainTypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/lib/babe/inherents"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

// SecondedCompactStatement is the proposal of a parachain candidate.
type SecondedCompactStatement struct {
	CandidateHash common.Hash
}

// Index returns the index of the type SecondedCompactStatement.
func (SecondedCompactStatement) Index() uint {
	return 0
}

// ValidCompactStatement represents a valid candidate.
type ValidCompactStatement struct {
	CandidateHash common.Hash
}

// Index returns the index of the type ValidCompactStatement.
func (ValidCompactStatement) Index() uint {
	return 1
}

// CompactStatementVDT is the statement that can be made about parachain candidates
// These are the actual values that are signed.
type CompactStatementVDT scale.VaryingDataType

// Set will set a VaryingDataTypeValue using the underlying VaryingDataType
func (cs *CompactStatementVDT) Set(val scale.VaryingDataTypeValue) (err error) {
	vdt := scale.VaryingDataType(*cs)
	err = vdt.Set(val)
	if err != nil {
		return fmt.Errorf("setting value to varying data type: %w", err)
	}
	*cs = CompactStatementVDT(vdt)
	return nil
}

// Value returns the value from the underlying VaryingDataType
func (cs *CompactStatementVDT) Value() (val scale.VaryingDataTypeValue, err error) {
	vdt := scale.VaryingDataType(*cs)
	val, err = vdt.Value()
	if err != nil {
		return nil, fmt.Errorf("getting value from varying data type: %w", err)
	}
	return val, nil
}

// NewCompactStatement creates a new CompactStatementVDT.
func NewCompactStatement() CompactStatementVDT {
	vdt := scale.MustNewVaryingDataType(ValidCompactStatement{}, SecondedCompactStatement{})
	return CompactStatementVDT(vdt)
}

// ExplicitDisputeStatement An explicit statement on a candidate issued as part of a dispute.
type ExplicitDisputeStatement struct {
	Valid         bool
	CandidateHash parachain.CandidateHash
	Session       parachainTypes.SessionIndex
}

// ApprovalVote A vote of approval on a candidate.
type ApprovalVote parachain.CandidateHash

// SignedDisputeStatement A checked dispute statement from an associated validator.
type SignedDisputeStatement struct {
	DisputeStatement   inherents.DisputeStatement
	CandidateHash      common.Hash
	ValidatorPublic    parachainTypes.ValidatorID
	ValidatorSignature parachain.ValidatorSignature
	SessionIndex       parachainTypes.SessionIndex
}

// Statement is the statement that can be made about parachain candidates.
type Statement struct {
	SignedDisputeStatement SignedDisputeStatement
	ValidatorIndex         parachainTypes.ValidatorIndex
}
