package parachain

import (
	"fmt"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

// Statement is a result of candidate validation. It could be either `Valid` or `Seconded`.
type StatementValues interface {
	Valid | Seconded
}

type Statement struct {
	inner any
}

func setStatement[Value StatementValues](mvdt *Statement, value Value) {
	mvdt.inner = value
}

func (mvdt *Statement) SetValue(value any) (err error) {
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

func (mvdt Statement) IndexValue() (index uint, value any, err error) {
	switch mvdt.inner.(type) {
	case Valid:
		return 2, mvdt.inner, nil

	case Seconded:
		return 1, mvdt.inner, nil

	}
	return 0, nil, scale.ErrUnsupportedVaryingDataTypeValue
}

func (mvdt Statement) Value() (value any, err error) {
	_, value, err = mvdt.IndexValue()
	return
}

func (mvdt Statement) ValueAt(index uint) (value any, err error) {
	switch index {
	case 2:
		return *new(Valid), nil

	case 1:
		return *new(Seconded), nil

	}
	return nil, scale.ErrUnknownVaryingDataTypeValue
}

// NewStatement returns a new Statement VaryingDataType
func NewStatement() Statement {
	return Statement{}
}

// Seconded represents a statement that a validator seconds a candidate.
type Seconded CommittedCandidateReceipt

// Valid represents a statement that a validator has deemed a candidate valid.
type Valid CandidateHash

// CandidateHash makes it easy to enforce that a hash is a candidate hash on the type level.
type CandidateHash struct {
	Value common.Hash `scale:"1"`
}
