package parachaintypes

import (
	"fmt"

	"github.com/ChainSafe/gossamer/pkg/scale"
)

// ValidityAttestation is an implicit or explicit attestation to the validity of a parachain
// candidate.
type ValidityAttestation scale.VaryingDataType

// Set will set a VaryingDataTypeValue using the underlying VaryingDataType
func (va *ValidityAttestation) Set(val scale.VaryingDataTypeValue) (err error) {
	// cast to VaryingDataType to use VaryingDataType.Set method
	vdt := scale.VaryingDataType(*va)
	err = vdt.Set(val)
	if err != nil {
		return fmt.Errorf("setting value to varying data type: %w", err)
	}
	// store original ParentVDT with VaryingDataType that has been set
	*va = ValidityAttestation(vdt)
	return nil
}

// Value returns the value from the underlying VaryingDataType
func (va *ValidityAttestation) Value() (scale.VaryingDataTypeValue, error) {
	vdt := scale.VaryingDataType(*va)
	return vdt.Value()
}

// Implicit is for Implicit attestation.
type Implicit ValidatorSignature //skipcq

// Index returns VDT index
func (Implicit) Index() uint { //skipcq
	return 1
}

func (i Implicit) String() string { //skipcq:SCC-U1000
	return fmt.Sprintf("implicit(%s)", ValidatorSignature(i))
}

// Explicit is for Explicit attestation.
type Explicit ValidatorSignature //skipcq

// Index returns VDT index
func (Explicit) Index() uint { //skipcq
	return 2
}

func (e Explicit) String() string { //skipcq:SCC-U1000
	return fmt.Sprintf("explicit(%s)", ValidatorSignature(e))
}

// newValidityAttestation creates a ValidityAttestation varying data type.
func NewValidityAttestation() ValidityAttestation { //skipcq
	vdt, err := scale.NewVaryingDataType(Implicit{}, Explicit{})
	if err != nil {
		panic(err)
	}

	return ValidityAttestation(vdt)
}
