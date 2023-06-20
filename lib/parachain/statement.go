package parachain

import (
	"fmt"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

// Statement is a result of candidate validation. It could be either `Valid` or `Seconded`.
type Statement scale.VaryingDataType

// NewStatement returns a new Statement VaryingDataType
func NewStatement() Statement {
	vdt := scale.MustNewVaryingDataType(Seconded{}, Valid{})
	return Statement(vdt)
}

// Set will set a VaryingDataTypeValue using the underlying VaryingDataType
func (s *Statement) Set(val scale.VaryingDataTypeValue) (err error) {
	vdt := scale.VaryingDataType(*s)
	err = vdt.Set(val)
	if err != nil {
		return fmt.Errorf("setting value to varying data type: %w", err)
	}

	*s = Statement(vdt)
	return nil
}

// Value returns the value from the underlying VaryingDataType
func (s *Statement) Value() (scale.VaryingDataTypeValue, error) {
	vdt := scale.VaryingDataType(*s)
	return vdt.Value()
}

// Seconded represents a statement that a validator seconds a candidate.
type Seconded CommittedCandidateReceipt

// Index returns the VaryingDataType Index
func (s Seconded) Index() uint {
	return 1
}

// Valid represents a statement that a validator has deemed a candidate valid.
type Valid CandidateHash

// Index returns the VaryingDataType Index
func (v Valid) Index() uint {
	return 2
}

// CandidateHash makes it easy to enforce that a hash is a candidate hash on the type level.
type CandidateHash struct {
	Value common.Hash `scale:"1"`
}
