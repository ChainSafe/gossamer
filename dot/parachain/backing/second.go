// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package backing

import (
	"errors"
	"fmt"

	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

var (
	errWrongPVDForSecondingCandidate = errors.New(
		"candidate backing was asked to second candidate with wrong persisted validation data")
	errUnknownRelayParentForSecondingCandidate = errors.New(
		"we were asked to second a candidate outside of our view")
	errParaOutsideAssignmentForSeconding = errors.New(
		"subsystem asked to second for parachain outside of our assignment")
	errAlreadySignedValidStatement = errors.New("we have already signed a valid statement for this candidate")
)

func (cb *CandidateBacking) handleSecondMessage(
	candidateReceipt parachaintypes.CandidateReceipt,
	pvd parachaintypes.PersistedValidationData,
	pov parachaintypes.PoV,
	chRelayParentAndCommand chan RelayParentAndCommand,
) error {
	pvdBytes, err := scale.Marshal(pvd)
	if err != nil {
		return fmt.Errorf("marshalling persisted validation data: %w", err)
	}

	pvdHash, err := common.Blake2bHash(pvdBytes)
	if err != nil {
		return fmt.Errorf("hashing persisted validation data: %w", err)
	}

	if candidateReceipt.Descriptor.PersistedValidationDataHash != pvdHash {
		return errWrongPVDForSecondingCandidate
	}

	rpState, ok := cb.perRelayParent[candidateReceipt.Descriptor.RelayParent]
	if !ok {
		return errUnknownRelayParentForSecondingCandidate
	}

	// Sanity check that candidate is from our assignment.
	if candidateReceipt.Descriptor.ParaID != uint32(rpState.Assignment) {
		return errParaOutsideAssignmentForSeconding
	}

	hash, err := candidateReceipt.Hash()
	if err != nil {
		return fmt.Errorf("hashing candidate receipt: %w", err)
	}
	candidateHash := parachaintypes.CandidateHash{Value: hash}

	// If the message is a `CandidateBackingMessage::Second`, sign and dispatch a
	// Seconded statement only if we have not signed a Valid statement for the requested candidate.
	if rpState.issuedStatements[candidateHash] {
		return errAlreadySignedValidStatement
	}

	// Kick off background validation with intent to second.
	logger.Debugf("validate and second candidate: %s", candidateHash)
	return rpState.validateAndMakeAvailable(
		executorParamsAtRelayParent,
		cb.SubSystemToOverseer,
		chRelayParentAndCommand,
		candidateReceipt,
		rpState.RelayParent,
		pvd,
		pov,
		uint32(len(rpState.TableContext.validators)),
		Second,
		candidateHash,
	)
}
