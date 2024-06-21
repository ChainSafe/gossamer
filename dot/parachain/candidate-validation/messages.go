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
	Ch                      chan parachaintypes.OverseerFuncRes[ValidationResult]
}

// ValidationResult represents the result coming from the candidate validation subsystem.
// Validation results can be either Valid or Invalid.
//
// If the result is Invalid,
// set the IsValid field of ValidationResultMessage to false.
// also store the reason for invalidity in the Err field of ValidationResultMessage.
//
// If the result is Valid,
// set the IsValid field of ValidationResultMessage to true.
// set the values of the CandidateCommitments and PersistedValidationData fields of ValidationResultMessage.

type ValidationResult struct {
	IsValid                 bool
	CandidateCommitments    parachaintypes.CandidateCommitments
	PersistedValidationData parachaintypes.PersistedValidationData
	ReasonForInvalidity     error
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
