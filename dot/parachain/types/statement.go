// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package parachaintypes

import (
	"fmt"

	"github.com/ChainSafe/gossamer/pkg/scale"
)

// Statement is a result of candidate validation. It could be either `Valid` or `Seconded`.
type StatementVDTValues interface {
	Valid | Seconded
}

type StatementVDT struct {
	inner any
}

func setStatement[Value StatementVDTValues](mvdt *StatementVDT, value Value) {
	mvdt.inner = value
}

func (mvdt *StatementVDT) SetValue(value any) (err error) {
	switch value := value.(type) {
	case Valid:
		setStatement(mvdt, value)
		return

	case Seconded:
		setStatement(mvdt, value)
		return

	default:
		return fmt.Errorf("unsupported type")
	}
}

func (mvdt StatementVDT) IndexValue() (index uint, value any, err error) {
	switch mvdt.inner.(type) {
	case Valid:
		return 2, mvdt.inner, nil

	case Seconded:
		return 1, mvdt.inner, nil

	}
	return 0, nil, scale.ErrUnsupportedVaryingDataTypeValue
}

func (mvdt StatementVDT) Value() (value any, err error) {
	_, value, err = mvdt.IndexValue()
	return
}

func (mvdt StatementVDT) ValueAt(index uint) (value any, err error) {
	switch index {
	case 2:
		return *new(Valid), nil

	case 1:
		return *new(Seconded), nil

	}
	return nil, scale.ErrUnknownVaryingDataTypeValue
}

// NewStatement returns a new statement varying data type
func NewStatementVDT() StatementVDT {
	return StatementVDT{}
}

// Seconded represents a statement that a validator seconds a candidate.
type Seconded CommittedCandidateReceipt

// Valid represents a statement that a validator has deemed a candidate valid.
type Valid CandidateHash
