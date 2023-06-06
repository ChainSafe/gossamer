package parachain

import (
	"fmt"

	"github.com/ChainSafe/gossamer/pkg/scale"
)

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
func (cs *CompactStatement) Value() (scale.VaryingDataTypeValue, error) {
	vdt := scale.VaryingDataType(*cs)
	return vdt.Value()
}

func NewCompactStatement() CompactStatement {
	vdt := scale.MustNewVaryingDataType(ParachainCandidateProposal{}, Valid{})
	return CompactStatement(vdt)
}

// Proposal of a parachain candidate.
type ParachainCandidateProposal CandidateHash

func (p ParachainCandidateProposal) Index() uint {
	return 1
}
