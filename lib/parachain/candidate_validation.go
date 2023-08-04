// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package parachain

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/ChainSafe/gossamer/lib/common"
	parachainruntime "github.com/ChainSafe/gossamer/lib/parachain/runtime"
	parachaintypes "github.com/ChainSafe/gossamer/lib/parachain/types"

	"github.com/ChainSafe/gossamer/pkg/scale"
)

var (
	ErrValidationCodeMismatch   = errors.New("validation code hash does not match")
	ErrValidationInputOverLimit = errors.New("validation input is over the limit")
)

// PoVRequestor gets proof of validity by issuing network requests to validators of the current backing group.
// TODO: Implement PoV requestor
type PoVRequestor interface {
	RequestPoV(povHash common.Hash) PoV
}

func getValidationData(runtimeInstance parachainruntime.RuntimeInstance, paraID uint32,
) (*parachaintypes.PersistedValidationData, *parachaintypes.ValidationCode, error) {

	var mergedError error

	for _, assumptionValue := range []any{
		parachaintypes.IncludedOccupiedCoreAssumption{},
		parachaintypes.TimedOutOccupiedCoreAssumption{},
		parachaintypes.Free{},
	} {
		assumption := parachaintypes.NewOccupiedCoreAssumption()
		err := assumption.SetValue(assumptionValue)
		if err != nil {
			return nil, nil, fmt.Errorf("getting assumption: %w", err)
		}
		persistedValidationData, err := runtimeInstance.ParachainHostPersistedValidationData(paraID, assumption)
		if err != nil {
			mergedError = errors.Join(mergedError, err)
			continue
		}

		validationCode, err := runtimeInstance.ParachainHostValidationCode(paraID, assumption)
		if err != nil {
			return nil, nil, fmt.Errorf("getting validation code: %w", err)
		}

		return persistedValidationData, validationCode, nil
	}

	return nil, nil, fmt.Errorf("getting persisted validation data: %w", mergedError)
}

// ValidateFromChainState validates a candidate parachain block with provided parameters using relay-chain
// state and using the parachain runtime.
func ValidateFromChainState(runtimeInstance parachainruntime.RuntimeInstance, povRequestor PoVRequestor,
	candidateReceipt parachaintypes.CandidateReceipt) (
	*parachaintypes.CandidateCommitments, *parachaintypes.PersistedValidationData, bool, error) {

	persistedValidationData, validationCode, err := getValidationData(runtimeInstance, candidateReceipt.Descriptor.ParaID)
	if err != nil {
		return nil, nil, false, fmt.Errorf("getting validation data: %w", err)
	}

	// check that the candidate does not exceed any parameters in the persisted validation data
	pov := povRequestor.RequestPoV(candidateReceipt.Descriptor.PovHash)

	// basic checks

	// check if encoded size of pov is less than max pov size
	buffer := bytes.NewBuffer(nil)
	encoder := scale.NewEncoder(buffer)
	err = encoder.Encode(pov)
	if err != nil {
		return nil, nil, false, fmt.Errorf("encoding pov: %w", err)
	}
	encodedPoVSize := buffer.Len()
	if encodedPoVSize > int(persistedValidationData.MaxPovSize) {
		return nil, nil, false, fmt.Errorf("%w, limit: %d, got: %d", ErrValidationInputOverLimit,
			persistedValidationData.MaxPovSize, encodedPoVSize)
	}

	validationCodeHash, err := common.Blake2bHash([]byte(*validationCode))
	if err != nil {
		return nil, nil, false, fmt.Errorf("hashing validation code: %w", err)
	}

	if validationCodeHash != common.Hash(candidateReceipt.Descriptor.ValidationCodeHash) {
		return nil, nil, false, fmt.Errorf("%w, expected: %s, got %s", ErrValidationCodeMismatch,
			candidateReceipt.Descriptor.ValidationCodeHash, validationCodeHash)
	}

	// check candidate signature
	err = candidateReceipt.Descriptor.CheckCollatorSignature()
	if err != nil {
		return nil, nil, false, fmt.Errorf("verifying collator signature: %w", err)
	}

	validationParams := parachainruntime.ValidationParameters{
		ParentHeadData:         persistedValidationData.ParentHead,
		BlockData:              pov.BlockData,
		RelayParentNumber:      persistedValidationData.RelayParentNumber,
		RelayParentStorageRoot: persistedValidationData.RelayParentStorageRoot,
	}

	parachainRuntimeInstance, err := parachainruntime.SetupVM(*validationCode)
	if err != nil {
		return nil, nil, false, fmt.Errorf("setting up VM: %w", err)
	}

	validationResults, err := parachainRuntimeInstance.ValidateBlock(validationParams)
	if err != nil {
		return nil, nil, false, fmt.Errorf("executing validate_block: %w", err)
	}

	candidateCommitments := parachaintypes.CandidateCommitments{
		UpwardMessages:            validationResults.UpwardMessages,
		HorizontalMessages:        validationResults.HorizontalMessages,
		NewValidationCode:         validationResults.NewValidationCode,
		HeadData:                  validationResults.HeadData,
		ProcessedDownwardMessages: validationResults.ProcessedDownwardMessages,
		HrmpWatermark:             validationResults.HrmpWatermark,
	}

	isValid, err := runtimeInstance.ParachainHostCheckValidationOutputs(
		candidateReceipt.Descriptor.ParaID, candidateCommitments)
	if err != nil {
		return nil, nil, false, fmt.Errorf("executing validate_block: %w", err)
	}

	return &candidateCommitments, persistedValidationData, isValid, nil
}
