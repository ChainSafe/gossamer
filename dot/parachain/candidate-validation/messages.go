// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package candidatevalidation

import (
	"fmt"

	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

type ValidateFromChainState struct {
	CandidateReceipt parachaintypes.CandidateReceipt
	Pov              parachaintypes.PoV
	ExecutorParams   parachaintypes.ExecutorParams
	ExecKind         parachaintypes.PvfExecTimeoutKind
	Sender           chan CandidateValidationMessageValidateFromExhaustive
}

// CandidateValidationMessageValidateFromExhaustive performs full validation of a candidate with provided parameters,
// including `PersistedValidationData` and `ValidationCode`. It doesn't involve acceptance
// criteria checking and is typically used when the candidate's validity is established
// through prior relay-chain checks.
type CandidateValidationMessageValidateFromExhaustive struct {
	PersistedValidationData parachaintypes.PersistedValidationData
	ValidationCode          parachaintypes.ValidationCode
	CandidateReceipt        parachaintypes.CandidateReceipt
	PoV                     parachaintypes.PoV
	ExecutorParams          parachaintypes.ExecutorParams
	PvfExecTimeoutKind      parachaintypes.PvfExecTimeoutKind
	Ch                      chan parachaintypes.OverseerFuncRes[parachaintypes.ValidationResult]
}

// InvalidCandidate candidate invalidity details
type InvalidCandidate scale.VaryingDataType

// NewInvalidCandidate returns a new InvalidCandidate varying data type
func NewInvalidCandidate() InvalidCandidate {
	vdt := scale.MustNewVaryingDataType(ExecutionError{}, InvalidOutputs{}, Timeout{}, ParamsTooLarge{},
		CodeTooLarge{}, PoVDecompressionFailure{}, BadReturn{}, BadParent{}, PoVHashMismatch{}, BadSignature{},
		ParaHeadHashMismatch{}, CodeHashMismatch{}, CommitmentHashMismatch{})
	return InvalidCandidate(vdt)
}

// New will enable scale to create new instance when needed
func (InvalidCandidate) New() InvalidCandidate {
	return NewInvalidCandidate()
}

// Set will set a value using the underlying  varying data type
func (i *InvalidCandidate) Set(val scale.VaryingDataTypeValue) (err error) {
	vdt := scale.VaryingDataType(*i)
	err = vdt.Set(val)
	if err != nil {
		return fmt.Errorf("setting value to varying data type: %w", err)
	}

	*i = InvalidCandidate(vdt)
	return nil
}

// Value returns the value from the underlying varying data type
func (i *InvalidCandidate) Value() (scale.VaryingDataTypeValue, error) {
	vdt := scale.VaryingDataType(*i)
	return vdt.Value()
}

// ExecutionError failed to execute `valid_black`. This includes function panicking.
type ExecutionError struct {
	Error string
}

// Index returns the index of varying data type
func (ExecutionError) Index() uint {
	return 0
}

// InvalidOutputs validation outputs check doesn't pass
type InvalidOutputs struct{}

// Index returns the index of varying data type
func (InvalidOutputs) Index() uint {
	return 1
}

// Timeout Execution timeout
type Timeout struct{}

// Index returns the index of varying data type
func (Timeout) Index() uint {
	return 2
}

// ParamsTooLarge Validation input is over the limit
type ParamsTooLarge struct {
	ParamsSize uint64
}

// Index returns the index of varying data type
func (ParamsTooLarge) Index() uint {
	return 3
}

// CodeTooLarge code size is over the limit
type CodeTooLarge struct {
	CodeSize uint64
}

// Index returns the index of varying data type
func (CodeTooLarge) Index() uint {
	return 4
}

// PoVDecompressionFailure PoV does not decompress correctly
type PoVDecompressionFailure struct{}

// Index returns the index of varying data type
func (PoVDecompressionFailure) Index() uint {
	return 5
}

// BadReturn Validation function returned invalid data
type BadReturn struct{}

// Index returns the index of varying data type
func (BadReturn) Index() uint {
	return 6
}

// BadParent invalid relay chain parent
type BadParent struct{}

// Index returns the index of varying data type
func (BadParent) Index() uint {
	return 7
}

// PoVHashMismatch PoV hash mismatch
type PoVHashMismatch struct{}

// Index returns the index of varying data type
func (PoVHashMismatch) Index() uint {
	return 8
}

// BadSignature bod collator signature
type BadSignature struct{}

// Index returns the index of varying data type
func (BadSignature) Index() uint {
	return 9
}

// ParaHeadHashMismatch para head hash does not match
type ParaHeadHashMismatch struct{}

// Index returns the index of varying data type
func (ParaHeadHashMismatch) Index() uint {
	return 10
}

// CodeHashMismatch validation code hash does not match
type CodeHashMismatch struct{}

// Index returns the index of varying data type
func (CodeHashMismatch) Index() uint {
	return 11
}

// CommitmentHashMismatch validation has generated different candidate commitments
type CommitmentHashMismatch struct{}

// Index returns the index of varying data type
func (CommitmentHashMismatch) Index() uint {
	return 12
}

// PreCheck try to complie the given validation code and return the result
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
