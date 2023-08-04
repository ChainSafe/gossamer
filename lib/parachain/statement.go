// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package parachain

import (
	"fmt"

	"github.com/ChainSafe/gossamer/lib/common"
	parachaintypes "github.com/ChainSafe/gossamer/lib/parachain/types"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

// Statement is a result of candidate validation. It could be either `Valid` or `Seconded`.
type Statement scale.VaryingDataType

// NewStatement returns a new statement varying data type
func NewStatement() Statement {
	vdt := scale.MustNewVaryingDataType(Seconded{}, Valid{})
	return Statement(vdt)
}

// New will enable scale to create new instance when needed
func (Statement) New() Statement {
	return NewStatement()
}

// Set will set a value using the underlying  varying data type
func (s *Statement) Set(val scale.VaryingDataTypeValue) (err error) {
	vdt := scale.VaryingDataType(*s)
	err = vdt.Set(val)
	if err != nil {
		return fmt.Errorf("setting value to varying data type: %w", err)
	}

	*s = Statement(vdt)
	return nil
}

// Value returns the value from the underlying varying data type
func (s *Statement) Value() (scale.VaryingDataTypeValue, error) {
	vdt := scale.VaryingDataType(*s)
	return vdt.Value()
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
