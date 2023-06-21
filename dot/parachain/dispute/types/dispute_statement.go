package types

import (
	"fmt"
	"github.com/ChainSafe/gossamer/lib/babe/inherents"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/parachain"
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

// CompactStatement is the statement that can be made about parachain candidates
// These are the actual values that are signed.
type CompactStatement scale.VaryingDataType

// Set will set a VaryingDataTypeValue using the underlying VaryingDataType
func (cs *CompactStatement) Set(val scale.VaryingDataTypeValue) (err error) {
	vdt := scale.VaryingDataType(*cs)
	err = vdt.Set(val)
	if err != nil {
		return fmt.Errorf("setting value to varying data type: %w", err)
	}
	*cs = CompactStatement(vdt)
	return nil
}

// Value returns the value from the underlying VaryingDataType
func (cs *CompactStatement) Value() (val scale.VaryingDataTypeValue, err error) {
	vdt := scale.VaryingDataType(*cs)
	val, err = vdt.Value()
	if err != nil {
		return nil, fmt.Errorf("getting value from varying data type: %w", err)
	}
	return val, nil
}

// NewCompactStatement creates a new CompactStatement.
func NewCompactStatement() (CompactStatement, error) {
	vdt, err := scale.NewVaryingDataType(ValidCompactStatement{}, SecondedCompactStatement{})
	if err != nil {
		return CompactStatement{}, fmt.Errorf("failed to create varying data type: %w", err)
	}

	return CompactStatement(vdt), nil
}

// ExplicitDisputeStatement An explicit statement on a candidate issued as part of a dispute.
type ExplicitDisputeStatement struct {
	Valid         bool
	CandidateHash parachain.CandidateHash
	Session       parachain.SessionIndex
}

// ApprovalVote A vote of approval on a candidate.
type ApprovalVote parachain.CandidateHash

// SignedDisputeStatement A checked dispute statement from an associated validator.
type SignedDisputeStatement struct {
	DisputeStatement   inherents.DisputeStatement
	CandidateHash      common.Hash
	ValidatorPublic    parachain.ValidatorID
	ValidatorSignature parachain.ValidatorSignature
	SessionIndex       parachain.SessionIndex
}

// Statement is the statement that can be made about parachain candidates.
type Statement struct {
	SignedDisputeStatement SignedDisputeStatement
	ValidatorIndex         parachain.ValidatorIndex
}
