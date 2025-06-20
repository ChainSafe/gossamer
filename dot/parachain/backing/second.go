// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package backing

import (
	"errors"
	"fmt"
	"slices"

	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
)

var (
	errWrongPVDForSecondingCandidate = errors.New(
		"incorrect persisted validation data provided for seconding candidate")
	errUnknownRelayParentForSecondingCandidate = errors.New(
		"attempted to second a candidate with an unknown relay parent")
	errParaOutsideAssignmentForSeconding = errors.New(
		"subsystem requested to second for parachain beyond our assignment scope")
	errAlreadySignedValidStatement = errors.New("already signed a valid statement for this candidate")
)

func (cb *CandidateBacking) handleSecondMessage(
	candidateReceipt parachaintypes.CandidateReceiptV2,
	pvd parachaintypes.PersistedValidationData,
	pov parachaintypes.PoV,
	chRelayParentAndCommand chan relayParentAndCommand,
) error {
	hash, err := candidateReceipt.Hash()
	if err != nil {
		return fmt.Errorf("hashing candidate receipt: %w", err)
	}
	candidateHash := parachaintypes.CandidateHash{Value: hash}

	pvdHash, err := pvd.Hash()
	if err != nil {
		return fmt.Errorf("hashing persisted validation data: %w", err)
	}

	if candidateReceipt.Descriptor.PersistedValidationDataHash != pvdHash {
		return fmt.Errorf("%w; mismatch between persisted validation data hash in candidate descriptor: %s "+
			"and calculated hash of given persisted validation data: %s",
			errWrongPVDForSecondingCandidate,
			candidateReceipt.Descriptor.PersistedValidationDataHash,
			pvdHash,
		)
	}

	rpState, ok := cb.perRelayParent[candidateReceipt.Descriptor.RelayParent]
	if !ok {
		return fmt.Errorf("%w; candidate hash: %s; relay parent: %s",
			errUnknownRelayParentForSecondingCandidate,
			candidateHash,
			candidateReceipt.Descriptor.RelayParent,
		)
	}

	// Return early if the local validator is disabled. At this point in execution,
	// the local node should be a validator, but defensively handle the nil case and
	// continue processing if we unexpectedly don't have validator data, rather than failing.
	validator := rpState.tableContext.validator
	if validator != nil && validator.Disabled {
		return fmt.Errorf("local validator is disabled. Don't validate and second")
	}

	// Return early if we don't have an assigned core.
	if rpState.assignedCore == nil {
		return errNoAssignedCore
	}

	// Sanity check that candidate is from our assignment.
	assignedParas := rpState.claimQueue[*rpState.assignedCore]
	if !slices.Contains(assignedParas, candidateReceipt.Descriptor.ParaID) {
		return fmt.Errorf("%w: candidate hash: %s; candidate paraID: %d; assigned-core: %d",
			errParaOutsideAssignmentForSeconding,
			candidateHash,
			candidateReceipt.Descriptor.ParaID,
			*rpState.assignedCore,
		)
	}

	// If the message is a `CandidateBackingMessage::Second`, sign and dispatch a
	// Seconded statement only if we have not signed a Valid statement for the requested candidate.
	if rpState.issuedStatements[candidateHash] {
		return fmt.Errorf("%w: candidate hash: %s", errAlreadySignedValidStatement, candidateHash)
	}

	// Kick off background validation with intent to second.
	logger.Debugf("validate and second candidate: %s", candidateHash)
	return rpState.validateAndMakeAvailable(
		cb.BlockState,
		cb.SubSystemToOverseer,
		chRelayParentAndCommand,
		candidateReceipt,
		pvd,
		pov,
		uint32(len(rpState.tableContext.validators)),
		second,
		candidateHash,
	)
}
