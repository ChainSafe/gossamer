// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package parachaintypes

import (
	"fmt"

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
