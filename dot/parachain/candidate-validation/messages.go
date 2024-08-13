// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package candidatevalidation

import (
	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/lib/common"
)

// ValidateFromChainState performs validation of a candidate with provided parameters,
type ValidateFromChainState struct {
	CandidateReceipt parachaintypes.CandidateReceipt
	Pov              parachaintypes.PoV
	ExecutorParams   parachaintypes.ExecutorParams
	ExecKind         parachaintypes.PvfExecTimeoutKind
	Ch               chan parachaintypes.OverseerFuncRes[ValidationResult]
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
	Ch                      chan parachaintypes.OverseerFuncRes[ValidationResult]
}

// ValidationResult represents the result coming from the candidate validation subsystem.
// Validation results can be either a ValidValidationResult or InvalidValidationResult.
//
// If the result is invalid,
// store the reason for invalidity in the InvalidResult field of ValidationResult.
//
// If the result is valid,
// set the values of the ValidResult field of ValidValidationResult.
type ValidationResult struct {
	ValidResult   *ValidValidationResult
	InvalidResult *ReasonForInvalidity
}

func (vr ValidationResult) IsValid() bool {
	return vr.ValidResult != nil
}

type ValidValidationResult struct {
	CandidateCommitments    parachaintypes.CandidateCommitments
	PersistedValidationData parachaintypes.PersistedValidationData
}

type ReasonForInvalidity byte

const (
	// ExecutionError Failed to execute `validate_block`. This includes function panicking.
	ExecutionError ReasonForInvalidity = iota
	// InvalidOutputs Validation outputs check doesn't pass.
	InvalidOutputs
	// Timeout Execution timeout.
	Timeout
	// ParamsTooLarge Validation input is over the limit.
	ParamsTooLarge
	// CodeTooLarge Code size is over the limit.
	CodeTooLarge
	// PoVDecompressionFailure PoV does not decompress correctly.
	PoVDecompressionFailure
	// BadReturn Validation function returned invalid data.
	BadReturn
	// BadParent Invalid relay chain parent.
	BadParent
	// PoVHashMismatch POV hash does not match.
	PoVHashMismatch
	// BadSignature Bad collator signature.
	BadSignature
	// ParaHeadHashMismatch Para head hash does not match.
	ParaHeadHashMismatch
	// CodeHashMismatch Validation code hash does not match.
	CodeHashMismatch
	// CommitmentsHashMismatch Validation has generated different candidate commitments.
	CommitmentsHashMismatch
)

func (ci ReasonForInvalidity) Error() string {
	switch ci {
	case ExecutionError:
		return "failed to execute `validate_block`"
	case InvalidOutputs:
		return "validation outputs check doesn't pass"
	case Timeout:
		return "execution timeout"
	case ParamsTooLarge:
		return "validation input is over the limit"
	case CodeTooLarge:
		return "code size is over the limit"
	case PoVDecompressionFailure:
		return "PoV does not decompress correctly"
	case BadReturn:
		return "validation function returned invalid data"
	case BadParent:
		return "invalid relay chain parent"
	case PoVHashMismatch:
		return "PoV hash does not match"
	case BadSignature:
		return "bad collator signature"
	case ParaHeadHashMismatch:
		return "para head hash does not match"
	case CodeHashMismatch:
		return "validation code hash does not match"
	case CommitmentsHashMismatch:
		return "validation has generated different candidate commitments"
	default:
		return "unknown invalidity reason"
	}
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
