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

// ParticipationOutcome is the outcome of the validation process.
type ParticipationOutcome scale.VaryingDataType

// Set will set a VaryingDataTypeValue using the underlying VaryingDataType
func (po *ParticipationOutcome) Set(val scale.VaryingDataTypeValue) (err error) {
	vdt := scale.VaryingDataType(*po)
	err = vdt.Set(val)
	if err != nil {
		return fmt.Errorf("setting value to varying data type: %w", err)
	}
	*po = ParticipationOutcome(vdt)
	return nil
}

// Value returns the value from the underlying VaryingDataType
func (po *ParticipationOutcome) Value() (scale.VaryingDataTypeValue, error) {
	vdt := scale.VaryingDataType(*po)
	return vdt.Value()
}

// Validity returns true if the outcome is valid.
func (po *ParticipationOutcome) Validity() (bool, error) {
	val, err := po.Value()
	if err != nil {
		return false, fmt.Errorf("getting value from varying data type: %w", err)
	}

	_, ok := val.(ValidOutcome)
	if !ok {
		return true, nil
	}

	return false, nil
}

// NewParticipationOutcome returns a new ParticipationOutcome.
func NewParticipationOutcome() (ParticipationOutcome, error) {
	outcome, err := scale.NewVaryingDataType(ValidOutcome{}, InvalidOutcome{}, UnAvailableOutcome{}, ErrorOutcome{})
	return ParticipationOutcome(outcome), err
}

// ParticipationOutcomeType is the type of the participation outcome.
type ParticipationOutcomeType uint

const (
	ParticipationOutcomeValid ParticipationOutcomeType = iota
	ParticipationOutcomeInvalid
	ParticipationOutcomeUnAvailable
	ParticipationOutcomeError
)

// NewCustomParticipationOutcome returns a new ParticipationOutcome vdt by setting the outcome to the given type
func NewCustomParticipationOutcome(outcome ParticipationOutcomeType) (ParticipationOutcome, error) {
	participationOutcome, err := NewParticipationOutcome()
	if err != nil {
		return ParticipationOutcome{}, fmt.Errorf("creating new participation outcome: %w", err)
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
		return ParticipationOutcome{}, fmt.Errorf("invalid participation outcome type: %d", outcome)
	}

	return participationOutcome, err
}
