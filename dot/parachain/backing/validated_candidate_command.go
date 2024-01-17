// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package backing

import (
	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/lib/common"
)

// processValidatedCandidateCommand notes the result of a background validation of a candidate and reacts accordingly..
func (cb *CandidateBacking) processValidatedCandidateCommand(rpAndCmd relayParentAndCommand) error {
	// TODO: Implement this #3571
	return nil
}

type backgroundValidationResult struct {
	candidateReceipt        *parachaintypes.CandidateReceipt
	candidateCommitments    *parachaintypes.CandidateCommitments
	persistedValidationData *parachaintypes.PersistedValidationData
	err                     error
}

// relayParentAndCommand contains the relay parent and the command to be executed on validated candidate,
// along with the result of the background validation.
type relayParentAndCommand struct {
	relayParent   common.Hash
	command       validatedCandidateCommand
	validationRes backgroundValidationResult
	candidateHash parachaintypes.CandidateHash
}

// validatedCandidateCommand represents commands for handling validated candidates.
// This is not a command to validate a candidate, but to react to a validation result.
type validatedCandidateCommand byte

const (
	// We were instructed to second the candidate that has been already validated.
	second = validatedCandidateCommand(iota) //nolint:unused
	// We were instructed to validate the candidate.
	attest
	// We were not able to `Attest` because backing validator did not send us the PoV.
	attestNoPoV //nolint:unused
)
