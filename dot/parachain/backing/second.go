// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package backing

import (
	"errors"
	"fmt"

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
	candidateReceipt parachaintypes.CandidateReceipt,
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

	// Sanity check that candidate is from our assignment.
	if candidateReceipt.Descriptor.ParaID != uint32(*rpState.assignment) {
		return fmt.Errorf("%w: candidate hash: %s; candidate paraID: %d; assignment: %d",
			errParaOutsideAssignmentForSeconding,
			candidateHash,
			candidateReceipt.Descriptor.ParaID,
			*rpState.assignment,
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
		rpState.relayParent,
		pvd,
		pov,
		uint32(len(rpState.tableContext.validators)),
		second,
		candidateHash,
	)
}
