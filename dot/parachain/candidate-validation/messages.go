// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package candidatevalidation

import (
	"fmt"

	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

// ValidateFromChainState performs validation of a candidate with provided parameters,
type ValidateFromChainState struct {
	CandidateReceipt parachaintypes.CandidateReceipt
	Pov              parachaintypes.PoV
	ExecutorParams   parachaintypes.ExecutorParams
	ExecKind         parachaintypes.PvfExecTimeoutKind
	Sender           chan ValidateFromExhaustive
}

// ValidateFromExhaustive performs full validation of a candidate with provided parameters,
// including `PersistedValidationData` and `ValidationCode`. It doesn't involve acceptance
// criteria checking and is typically used when the candidate's validity is established
// through prior relay-chain checks.
type ValidateFromExhaustive struct {
	PersistedValidationData parachaintypes.PersistedValidationData
	ValidationCode          parachaintypes.ValidationCode
	CandidateReceipt        parachaintypes.CandidateReceipt
	PoV                     parachaintypes.PoV
	ExecutorParams          parachaintypes.ExecutorParams
	PvfExecTimeoutKind      parachaintypes.PvfExecTimeoutKind
	Ch                      chan parachaintypes.OverseerFuncRes[ValidationResultMessage]
}

// ValidationResultMessage represents the result coming from the candidate validation subsystem.
type ValidationResultMessage struct {
	IsValid             bool
	ValidationResultVDT ValidationResultVDT
}

type ValidationResultVDT scale.VaryingDataType

func NewValidationResultVDT() ValidationResultVDT {
	vdt, err := scale.NewVaryingDataType(Valid{}, Invalid{})
	if err != nil {
		panic(err)
	}
	return ValidationResultVDT(vdt)
}

// New returns new ValidationResult VDT
func (ValidationResultVDT) New() ValidationResultVDT {
	return NewValidationResultVDT()
}

// Value returns the value from the underlying VaryingDataType
func (vr *ValidationResultVDT) Value() (scale.VaryingDataTypeValue, error) {
	vdt := scale.VaryingDataType(*vr)
	return vdt.Value()
}

// Set will set a VaryingDataTypeValue using the underlying VaryingDataType
func (vr *ValidationResultVDT) Set(val scale.VaryingDataTypeValue) (err error) {
	vdt := scale.VaryingDataType(*vr)
	err = vdt.Set(val)
	if err != nil {
		return fmt.Errorf("setting value to varying data type: %w", err)
	}

	*vr = ValidationResultVDT(vdt)
	return nil
}

type Valid struct {
	CandidateCommitments    parachaintypes.CandidateCommitments
	PersistedValidationData parachaintypes.PersistedValidationData
}

func (Valid) Index() uint {
	return 1
}

type Invalid struct {
	Err error
}

func (Invalid) Index() uint {
	return 2
}

// PreCheck try to compile the given validation code and return the result
// The validation code is specified by the hash and will be queried from the runtime API at
// the given relay-parent.
type PreCheck struct {
	RelayParent        common.Hash
	ValidationCodeHash parachaintypes.ValidationCodeHash
	PreCheckOutcome    chan PreCheckOutcome
}

// PreCheckOutcome represents the outcome of the candidate-validation pre-check request
type PreCheckOutcome byte

const (
	PreCheckOutcomeValid PreCheckOutcome = iota
	PreCheckOutcomeInvalid
	PreCheckOutcomeFailed
)
