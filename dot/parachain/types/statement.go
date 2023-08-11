package parachaintypes

// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

import (
	"fmt"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

// StatementVDT is a result of candidate validation. It could be either `Valid` or `Seconded`.
type StatementVDT scale.VaryingDataType

// NewStatementVDT returns a new statement varying data type
func NewStatementVDT() StatementVDT {
	vdt := scale.MustNewVaryingDataType(Seconded{}, Valid{})
	return StatementVDT(vdt)
}

// New will enable scale to create new instance when needed
func (StatementVDT) New() StatementVDT {
	return NewStatementVDT()
}

// Set will set a value using the underlying  varying data type
func (s *StatementVDT) Set(val scale.VaryingDataTypeValue) (err error) {
	vdt := scale.VaryingDataType(*s)
	err = vdt.Set(val)
	if err != nil {
		return fmt.Errorf("setting value to varying data type: %w", err)
	}

	*s = StatementVDT(vdt)
	return nil
}

// Value returns the value from the underlying varying data type
func (s *StatementVDT) Value() (scale.VaryingDataTypeValue, error) {
	vdt := scale.VaryingDataType(*s)
	return vdt.Value()
}

// Seconded represents a statement that a validator seconds a candidate.
type Seconded CommittedCandidateReceipt

// Index returns the index of varying data type
func (Seconded) Index() uint {
	return 1
}

// Valid represents a statement that a validator has deemed a candidate valid.
type Valid CandidateHash

// Index returns the index of varying data type
func (Valid) Index() uint {
	return 2
}

// CandidateHash makes it easy to enforce that a hash is a candidate hash on the type level.
type CandidateHash struct {
	Value common.Hash `scale:"1"`
}

// UncheckedSignedFullStatement is a Variant of `SignedFullStatement` where the signature has not yet been verified.
type UncheckedSignedFullStatement struct {
	// The payload is part of the signed data. The rest is the signing context,
	// which is known both at signing and at validation.
	Payload StatementVDT `scale:"1"`

	// The index of the validator signing this statement.
	ValidatorIndex ValidatorIndex `scale:"2"`

	// The signature by the validator of the signed payload.
	Signature ValidatorSignature `scale:"3"`
}

// ValidatorSignature represents the signature with which parachain validators sign blocks.
type ValidatorSignature Signature

// Signature represents a cryptographic signature.
type Signature [64]byte
