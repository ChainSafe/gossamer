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
	ErrValidationCodeMismatch    = errors.New("validation code hash does not match")
	ErrValidationInputOverLimit  = errors.New("validation input is over the limit")
	ErrValidationParamsTooLarge  = errors.New("validation parameters are too large")
	ErrValidationPoVHashMismatch = errors.New("PoV hash does not match candidate PoV hash")
	ErrValidationBadSignature    = errors.New("bad signature")
)

type CandidateValidation struct {
	wg       sync.WaitGroup
	stopChan chan struct{}

	SubsystemToOverseer chan<- any
	OverseerToSubsystem <-chan any
}

func NewCandidateValidation(overseerChan chan<- any) *CandidateValidation {
	candidateValidation := CandidateValidation{
		SubsystemToOverseer: overseerChan,
	}

	return &candidateValidation
}

func (cv *CandidateValidation) Run(context.Context, chan any, chan any) {
	cv.wg.Add(1)
	go cv.processMessages(&cv.wg)
}

func (*CandidateValidation) Name() parachaintypes.SubSystemName {
	return parachaintypes.CandidateValidation
}

func (*CandidateValidation) ProcessActiveLeavesUpdateSignal(parachaintypes.ActiveLeavesUpdateSignal) error {
	// NOTE: this subsystem does not process active leaves update signal
	return nil
}

func (*CandidateValidation) ProcessBlockFinalizedSignal(parachaintypes.BlockFinalizedSignal) error {
	// NOTE: this subsystem does not process block finalized signal
	return nil
}

func (cv *CandidateValidation) Stop() {
	close(cv.stopChan)
	cv.wg.Wait()
}

func (cv *CandidateValidation) processMessages(wg *sync.WaitGroup) {
	defer wg.Done()
	for {
		select {
		case msg := <-cv.OverseerToSubsystem:
			logger.Debugf("received message %v", msg)
			switch msg := msg.(type) {
			case ValidateFromChainState:
				// TODO: implement functionality to handle ValidateFromChainState, see issue #3919
			case ValidateFromExhaustive:
				result, err := validateFromExhaustive(msg.PersistedValidationData, msg.ValidationCode,
					msg.CandidateReceipt, msg.PoV)
				if err != nil {
					logger.Errorf("failed to validate from exhaustive: %w", err)
					msg.Ch <- parachaintypes.OverseerFuncRes[ValidationResultMessage]{Err: err}
					continue
				}
				msg.Ch <- parachaintypes.OverseerFuncRes[ValidationResultMessage]{
					Data: ValidationResultMessage{
						ValidationResult: *result,
					},
				}
			case PreCheck:
				// TODO: implement functionality to handle PreCheck, see issue #3921

			case parachaintypes.ActiveLeavesUpdateSignal:
				_ = cv.ProcessActiveLeavesUpdateSignal(msg)

			case parachaintypes.BlockFinalizedSignal:
				_ = cv.ProcessBlockFinalizedSignal(msg)

			default:
				logger.Errorf("%w: %T", parachaintypes.ErrUnknownOverseerMessage, msg)
			}

		case <-cv.stopChan:
			return
		}
	}
}

// PoVRequestor gets proof of validity by issuing network requests to validators of the current backing group.
// TODO: Implement PoV requestor, issue #3919
type PoVRequestor interface {
	RequestPoV(povHash common.Hash) parachaintypes.PoV
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

// validateFromExhaustive validates a candidate parachain block with provided parameters
func validateFromExhaustive(persistedValidationData parachaintypes.PersistedValidationData,
	validationCode parachaintypes.ValidationCode,
	candidateReceipt parachaintypes.CandidateReceipt, pov parachaintypes.PoV) (
	*parachainruntime.ValidationResult, error) {

	validationCodeHash := validationCode.Hash()
	// basic checks
	err := performBasicChecks(&candidateReceipt.Descriptor, persistedValidationData.MaxPovSize, pov,
		validationCodeHash)
	if err != nil {
		return nil, err
	}

	parachainRuntimeInstance, err := parachainruntime.SetupVM(validationCode)
	if err != nil {
		return nil, fmt.Errorf("setting up VM: %w", err)
	}

	validationParams := parachainruntime.ValidationParameters{
		ParentHeadData:         persistedValidationData.ParentHead,
		BlockData:              pov.BlockData,
		RelayParentNumber:      persistedValidationData.RelayParentNumber,
		RelayParentStorageRoot: persistedValidationData.RelayParentStorageRoot,
	}

	validationResults, err := parachainRuntimeInstance.ValidateBlock(validationParams)
	if err != nil {
		return nil, fmt.Errorf("executing validate_block: %w", err)
	}

	return validationResults, nil
}

// performBasicChecks Does basic checks of a candidate. Provide the encoded PoV-block. Returns nil if basic checks
// are passed, `Err` otherwise.
func performBasicChecks(candidate *parachaintypes.CandidateDescriptor, maxPoVSize uint32,
	pov parachaintypes.PoV, validationCodeHash parachaintypes.ValidationCodeHash) error {
	povHash, err := pov.Hash()
	if err != nil {
		return fmt.Errorf("hashing PoV: %w", err)
	}

	encodedPoV, err := pov.Encode()
	if err != nil {
		return fmt.Errorf("encoding PoV: %w", err)
	}
	encodedPoVSize := uint32(len(encodedPoV))

	if encodedPoVSize > maxPoVSize {
		return fmt.Errorf("%w, limit: %d, got: %d", ErrValidationParamsTooLarge, maxPoVSize, encodedPoVSize)
	}

	if povHash != candidate.PovHash {
		return ErrValidationPoVHashMismatch
	}

	if validationCodeHash != candidate.ValidationCodeHash {
		return ErrValidationCodeMismatch
	}

	err = candidate.CheckCollatorSignature()
	if err != nil {
		return ErrValidationBadSignature
	}
	return nil
}
