// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package parachain

import (
	"fmt"

	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
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

// CandidateHash returns candidate hash referenced by this statement
func (s *StatementVDT) CandidateHash() (*CandidateHash, error) {
	valVDT, err := s.Value()
	if err != nil {
		return nil, fmt.Errorf("getting value from varying data type: %w", err)
	}

	if valVDT.Index() == 1 {
		val, ok := valVDT.(Seconded)
		if !ok {
			return nil, fmt.Errorf("getting seconded statement: %w", err)
		}

		hash, err := (*parachaintypes.CommittedCandidateReceipt)(&val).Hash()
		if err != nil {
			return nil, fmt.Errorf("computing candidate hash: %w", err)
		}
		return &CandidateHash{Value: hash}, nil
	}

	val, ok := valVDT.(Valid)
	if !ok {
		return nil, fmt.Errorf("getting valid statement: %w", err)
	}
	return (*CandidateHash)(&val), nil
}

// Seconded represents a statement that a validator seconds a candidate.
type Seconded parachaintypes.CommittedCandidateReceipt

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
