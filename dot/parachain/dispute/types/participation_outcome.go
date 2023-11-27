package types

import (
	"fmt"

	"github.com/ChainSafe/gossamer/pkg/scale"
)

// ValidOutcome is the outcome when the candidate is valid.
type ValidOutcome struct{}

// Index returns the index of the type.
func (ValidOutcome) Index() uint {
	return 0
}

// InvalidOutcome is the outcome when the candidate is invalid.
type InvalidOutcome struct{}

// Index returns the index of the type.
func (InvalidOutcome) Index() uint {
	return 1
}

// UnAvailableOutcome is the outcome when the candidate is unavailable.
type UnAvailableOutcome struct{}

// Index returns the index of the type.
func (UnAvailableOutcome) Index() uint {
	return 2
}

// ErrorOutcome is the outcome when the candidate has an error.
type ErrorOutcome struct{}

// Index returns the index of the type.
func (ErrorOutcome) Index() uint {
	return 3
}

// ParticipationOutcomeVDT is the outcome of the validation process.
type ParticipationOutcomeVDT scale.VaryingDataType

// Set will set a VaryingDataTypeValue using the underlying VaryingDataType
func (po *ParticipationOutcomeVDT) Set(val scale.VaryingDataTypeValue) (err error) {
	vdt := scale.VaryingDataType(*po)
	err = vdt.Set(val)
	if err != nil {
		return fmt.Errorf("setting value to varying data type: %w", err)
	}
	*po = ParticipationOutcomeVDT(vdt)
	return nil
}

// Value returns the value from the underlying VaryingDataType
func (po *ParticipationOutcomeVDT) Value() (scale.VaryingDataTypeValue, error) {
	vdt := scale.VaryingDataType(*po)
	return vdt.Value()
}

// Validity returns true if the outcome is valid.
func (po *ParticipationOutcomeVDT) Validity() (bool, error) {
	val, err := po.Value()
	if err != nil {
		return false, fmt.Errorf("getting value from varying data type: %w", err)
	}

	_, ok := val.(ValidOutcome)
	return ok, nil
}

// NewParticipationOutcomeVDT returns a new ParticipationOutcomeVDT.
func NewParticipationOutcomeVDT() (ParticipationOutcomeVDT, error) {
	outcome, err := scale.NewVaryingDataType(ValidOutcome{}, InvalidOutcome{}, UnAvailableOutcome{}, ErrorOutcome{})
	return ParticipationOutcomeVDT(outcome), err
}

// ParticipationOutcomeType is the type of the participation outcome.
type ParticipationOutcomeType uint

const (
	ParticipationOutcomeValid ParticipationOutcomeType = iota
	ParticipationOutcomeInvalid
	ParticipationOutcomeUnAvailable
	ParticipationOutcomeError
)

// NewCustomParticipationOutcomeVDT returns a new ParticipationOutcomeVDT vdt by setting the outcome to the given type
func NewCustomParticipationOutcomeVDT(outcome ParticipationOutcomeType) (ParticipationOutcomeVDT, error) {
	participationOutcome, err := NewParticipationOutcomeVDT()
	if err != nil {
		return ParticipationOutcomeVDT{}, fmt.Errorf("creating new participation outcome: %w", err)
	}

	switch outcome {
	case ParticipationOutcomeValid:
		err = participationOutcome.Set(ValidOutcome{})
	case ParticipationOutcomeInvalid:
		err = participationOutcome.Set(InvalidOutcome{})
	case ParticipationOutcomeUnAvailable:
		err = participationOutcome.Set(UnAvailableOutcome{})
	case ParticipationOutcomeError:
		err = participationOutcome.Set(ErrorOutcome{})
	default:
		return ParticipationOutcomeVDT{}, fmt.Errorf("invalid participation outcome type: %d", outcome)
	}

	return participationOutcome, err
}
