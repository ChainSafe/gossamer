// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package candidatevalidation

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"sync"

	parachainruntime "github.com/ChainSafe/gossamer/dot/parachain/runtime"
	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

var logger = log.NewFromGlobal(log.AddContext("pkg", "parachain-candidate-validation"))

var (
	ErrValidationCodeMismatch   = errors.New("validation code hash does not match")
	ErrValidationInputOverLimit = errors.New("validation input is over the limit")
)

type CandidateValidation struct {
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	SubsystemToOverseer chan<- any
	OverseerToSubsystem <-chan any
}

func (cv *CandidateValidation) Run(ctx context.Context, OverseerToSubSystem chan any, SubSystemToOverseer chan any) {
	cv.wg.Add(1)
	go cv.processMessages()
}

func (*CandidateValidation) Name() parachaintypes.SubSystemName {
	return parachaintypes.CandidateValidation
}

func (*CandidateValidation) ProcessActiveLeavesUpdateSignal(signal parachaintypes.ActiveLeavesUpdateSignal) error {
	// NOTE: this subsystem does not process active leaves update signal
	return nil
}

func (*CandidateValidation) ProcessBlockFinalizedSignal(signal parachaintypes.BlockFinalizedSignal) error {
	// NOTE: this subsystem does not process block finalized signal
	return nil
}

func (cv *CandidateValidation) Stop() {
	cv.cancel()
	cv.wg.Wait()
}

func (cv *CandidateValidation) processMessages() {
	for {
		select {
		case msg := <-cv.OverseerToSubsystem:
			logger.Debugf("received message %v", msg)
			switch msg := msg.(type) {
			case ValidateFromChainState:
				_, _, _, err := validateFromChainState(msg.RuntimeInstance, msg.PovRequestor, msg.CandidateReceipt)
				if err != nil {
					logger.Errorf("failed to validate candidate from chain state: %w", err)
				}
			case ValidateFromExhaustive:
				// TODO: implement functionality to handle ValidateFromExhaustive, see issue #3920
			case PreCheck:
				// TODO: implement functionality to handle PreCheck, see issue #3921

			case parachaintypes.ActiveLeavesUpdateSignal:
				err := cv.ProcessActiveLeavesUpdateSignal(msg)
				if err != nil {
					logger.Errorf("failed to process active leaves update signal: %w", err)
				}

			case parachaintypes.BlockFinalizedSignal:
				err := cv.ProcessBlockFinalizedSignal(msg)
				if err != nil {
					logger.Errorf("failed to process block finalized signal: %w", err)
				}

			default:
				logger.Error(parachaintypes.ErrUnknownOverseerMessage.Error())
			}

		case <-cv.ctx.Done():
			if err := cv.ctx.Err(); err != nil {
				logger.Errorf("ctx error: %v\n", err)
			}
			cv.wg.Done()
			return
		}
	}
}

// PoVRequestor gets proof of validity by issuing network requests to validators of the current backing group.
// TODO: Implement PoV requestor
type PoVRequestor interface {
	RequestPoV(povHash common.Hash) parachaintypes.PoV
}

func getValidationData(runtimeInstance parachainruntime.RuntimeInstance, paraID uint32,
) (*parachaintypes.PersistedValidationData, *parachaintypes.ValidationCode, error) {

	var mergedError error

	for _, assumptionValue := range []scale.VaryingDataTypeValue{
		parachaintypes.IncludedOccupiedCoreAssumption{},
		parachaintypes.TimedOutOccupiedCoreAssumption{},
		parachaintypes.Free{},
	} {
		assumption := parachaintypes.NewOccupiedCoreAssumption()
		err := assumption.Set(assumptionValue)
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

// validateFromChainState validates a candidate parachain block with provided parameters using relay-chain
// state and using the parachain runtime.
func validateFromChainState(runtimeInstance parachainruntime.RuntimeInstance, povRequestor PoVRequestor,
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
