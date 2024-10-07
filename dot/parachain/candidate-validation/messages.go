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

// PreCheck try to compile the given validation code and return the result
// The validation code is specified by the hash and will be queried from the runtime API at
// the given relay-parent.
type PreCheck struct {
	RelayParent        common.Hash
	ValidationCodeHash parachaintypes.ValidationCodeHash
	ResponseSender     chan PreCheckOutcome
}

// PreCheckOutcome represents the outcome of the candidate-validation pre-check request
type PreCheckOutcome byte

const (
	PreCheckOutcomeValid PreCheckOutcome = iota
	PreCheckOutcomeInvalid
	PreCheckOutcomeFailed
)

const (
	// Maximum compressed code size we support right now.
	// At the moment we have runtime upgrade on chain, which restricts scalability severely. If we want
	// to have bigger values, we should fix that first.
	//
	// Used for:
	// * initial genesis for the Parachains configuration
	// * checking updates to this stored runtime configuration do not exceed this limit
	// * when detecting a code decompression bomb in the client
	// NOTE: This value is used in the runtime so be careful when changing it.
	MaxCodeSize             = 3 * 1024 * 1024
	ValidationCodeBobmLimit = MaxCodeSize * 4
)
