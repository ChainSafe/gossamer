// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package backing

import (
	"errors"
	"fmt"

	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/lib/common"
)

var (
	errNilBackgroundValidationResult = errors.New("background validation result is nil")
	errRelayParentNoLongerRelevent   = errors.New("relay parent is no longer relevant")
)

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
	validationRes *backgroundValidationResult
	candidateHash parachaintypes.CandidateHash
}

// validatedCandidateCommand represents commands for handling validated candidates.
// This is not a command to validate a candidate, but to react to a validation result.
type validatedCandidateCommand byte

const (
	// We were instructed to second the candidate that has been already validated.
	second = validatedCandidateCommand(iota)
	// We were instructed to validate the candidate.
	attest
	// We were not able to `Attest` because backing validator did not send us the PoV.
	attestNoPoV
)

// processValidatedCandidateCommand notes the result of a background validation of a candidate and reacts accordingly..
func (cb *CandidateBacking) processValidatedCandidateCommand(rpAndCmd relayParentAndCommand) error {
	rpState, ok := cb.perRelayParent[rpAndCmd.relayParent]
	if !ok {
		return fmt.Errorf("%w: %s", errRelayParentNoLongerRelevent, rpAndCmd.relayParent)
	}
	if rpState == nil {
		return fmt.Errorf("%w; relay parent: %s", errNilRelayParentState, rpAndCmd.relayParent)
	}

	delete(rpState.awaitingValidation, rpAndCmd.candidateHash)

	switch rpAndCmd.command {
	case second:
		if rpAndCmd.validationRes == nil {
			return fmt.Errorf("unable to second the candidate, %w; relay Parent: %s; candidate hash: %s",
				errNilBackgroundValidationResult, rpAndCmd.relayParent, rpAndCmd.candidateHash.Value)
		}
		handleCommandSecond(*rpAndCmd.validationRes)
	case attest:
		if rpAndCmd.validationRes == nil {
			return fmt.Errorf("unable to attest the candidate, %w; relay Parent: %s; candidate hash: %s",
				errNilBackgroundValidationResult, rpAndCmd.relayParent, rpAndCmd.candidateHash.Value)
		}
		handleCommandAttest(*rpAndCmd.validationRes)
	case attestNoPoV:
		handleCommandAttestNoPoV(rpAndCmd.candidateHash)
	}

	return nil
}

func handleCommandSecond(bgValidationResult backgroundValidationResult)   {}
func handleCommandAttest(bgValidationResult backgroundValidationResult)   {}
func handleCommandAttestNoPoV(candidateHash parachaintypes.CandidateHash) {}
