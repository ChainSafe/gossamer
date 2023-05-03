package parachaintypes

import (
	"fmt"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

// ValidationCode is parachain validation code.
type ValidationCode []byte

// PersistedValidationData should be relatively lightweight primarily because it is constructed
// during inclusion for each candidate and therefore lies on the critical path of inclusion.
type PersistedValidationData struct {
	ParentHead             []byte      `scale:"1"`
	RelayParentNumber      uint32      `scale:"2"`
	RelayParentStorageRoot common.Hash `scale:"3"`
	MaxPovSize             uint32      `scale:"4"`
}

// OccupiedCoreAssumption is an assumption being made about the state of an occupied core.
type OccupiedCoreAssumption scale.VaryingDataType

// Set will set a VaryingDataTypeValue using the underlying VaryingDataType
func (o *OccupiedCoreAssumption) Set(val scale.VaryingDataTypeValue) (err error) {
	// cast to VaryingDataType to use VaryingDataType.Set method
	vdt := scale.VaryingDataType(*o)
	err = vdt.Set(val)
	if err != nil {
		return fmt.Errorf("setting value to varying data type: %w", err)
	}
	// store original ParentVDT with VaryingDataType that has been set
	*o = OccupiedCoreAssumption(vdt)
	return nil
}

// Value will return value from underying VaryingDataType
func (o *OccupiedCoreAssumption) Value() (scale.VaryingDataTypeValue, error) {
	vdt := scale.VaryingDataType(*o)
	return vdt.Value()
}

// Included means the candidate occupying the core was made available and included to free the core.
type Included struct{}

// Index returns VDT index
func (Included) Index() uint {
	return 0
}

func (Included) String() string {
	return "Included"
}

// TimedOut means the candidate occupying the core timed out and freed the core without advancing the para.
type TimedOut struct{}

// Index returns VDT index
func (TimedOut) Index() uint {
	return 1
}

func (TimedOut) String() string {
	return "TimedOut"
}

// Free means the core was not occupied to begin with.
type Free struct{}

// Index returns VDT index
func (Free) Index() uint {
	return 2
}

func (Free) String() string {
	return "Free"
}

// NewOccupiedCoreAssumption creates a OccupiedCoreAssumption varying data type.
func NewOccupiedCoreAssumption() OccupiedCoreAssumption {
	vdt, err := scale.NewVaryingDataType(Included{}, Free{}, TimedOut{})
	if err != nil {
		panic(err)
	}

	return OccupiedCoreAssumption(vdt)
}
