// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package candidatevalidation

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/ChainSafe/gossamer/dot/parachain/pvf"
	parachainruntime "github.com/ChainSafe/gossamer/dot/parachain/runtime"
	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/runtime"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

var logger = log.NewFromGlobal(log.AddContext("pkg", "parachain-candidate-validation"))

var (
	ErrValidationCodeMismatch   = errors.New("validation code hash does not match")
	ErrValidationInputOverLimit = errors.New("validation input is over the limit")
)

// CandidateValidation is a parachain subsystem that validates candidate parachain blocks
type CandidateValidation struct {
	wg                  sync.WaitGroup
	stopChan            chan struct{}
	SubsystemToOverseer chan<- any
	OverseerToSubsystem <-chan any
	BlockState          BlockState
	pvfHost             *pvf.ValidationHost
}

type BlockState interface {
	GetRuntime(blockHash common.Hash) (instance runtime.Instance, err error)
}

// NewCandidateValidation creates a new CandidateValidation subsystem
func NewCandidateValidation(overseerChan chan<- any, blockState BlockState) *CandidateValidation {
	candidateValidation := CandidateValidation{
		SubsystemToOverseer: overseerChan,
		pvfHost:             pvf.NewValidationHost(),
		BlockState:          blockState,
	}
	return &candidateValidation
}

// Run starts the CandidateValidation subsystem
func (cv *CandidateValidation) Run(context.Context, chan any, chan any) {
	cv.wg.Add(1)
	go cv.pvfHost.Start()
	go cv.processMessages(&cv.wg)
}

// Name returns the name of the subsystem
func (*CandidateValidation) Name() parachaintypes.SubSystemName {
	return parachaintypes.CandidateValidation
}

// ProcessActiveLeavesUpdateSignal processes active leaves update signal
func (*CandidateValidation) ProcessActiveLeavesUpdateSignal(parachaintypes.ActiveLeavesUpdateSignal) error {
	// NOTE: this subsystem does not process active leaves update signal
	return nil
}

// ProcessBlockFinalizedSignal processes block finalized signal
func (*CandidateValidation) ProcessBlockFinalizedSignal(parachaintypes.BlockFinalizedSignal) error {
	// NOTE: this subsystem does not process block finalized signal
	return nil
}

// Stop stops the CandidateValidation subsystem
func (cv *CandidateValidation) Stop() {
	cv.pvfHost.Stop()
	close(cv.stopChan)
	cv.wg.Wait()
}

// processMessages processes messages sent to the CandidateValidation subsystem
func (cv *CandidateValidation) processMessages(wg *sync.WaitGroup) {
	defer wg.Done()
	for {
		select {
		case msg := <-cv.OverseerToSubsystem:
			logger.Debugf("received message %v", msg)
			switch msg := msg.(type) {
			case ValidateFromChainState:
				cv.validateFromChainState(msg)

			case ValidateFromExhaustive:
				taskResult := make(chan *pvf.ValidationTaskResult)
				validationTask := &pvf.ValidationTask{
					PersistedValidationData: msg.PersistedValidationData,
					ValidationCode:          &msg.ValidationCode,
					CandidateReceipt:        &msg.CandidateReceipt,
					PoV:                     msg.PoV,
					ExecutorParams:          msg.ExecutorParams,
					PvfExecTimeoutKind:      msg.PvfExecTimeoutKind,
					ResultCh:                taskResult,
				}
				go cv.pvfHost.Validate(validationTask)

				result := <-taskResult
				if result.InternalError != nil {
					logger.Errorf("failed to validate from exhaustive: %w", result.InternalError)
					msg.Ch <- parachaintypes.OverseerFuncRes[pvf.ValidationResult]{
						Err: result.InternalError,
					}
				} else {
					msg.Ch <- parachaintypes.OverseerFuncRes[pvf.ValidationResult]{
						Data: *result.Result,
					}
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

// getValidationData gets validation data for a parachain block from the runtime instance
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

func (cv *CandidateValidation) validateFromChainState(msg ValidateFromChainState) {
	runtimeInstance, err := cv.BlockState.GetRuntime(msg.CandidateReceipt.Descriptor.RelayParent)
	if err != nil {
		logger.Errorf("getting runtime instance: %w", err)
		msg.Ch <- parachaintypes.OverseerFuncRes[pvf.ValidationResult]{
			Err: err,
		}
		return
	}

	persistedValidationData, validationCode, err := getValidationData(runtimeInstance,
		msg.CandidateReceipt.Descriptor.ParaID)
	if err != nil {
		logger.Errorf("getting validation data: %w", err)
		msg.Ch <- parachaintypes.OverseerFuncRes[pvf.ValidationResult]{
			Err: err,
		}
		return
	}

	taskResult := make(chan *pvf.ValidationTaskResult)
	validationTask := &pvf.ValidationTask{
		PersistedValidationData: *persistedValidationData,
		ValidationCode:          validationCode,
		CandidateReceipt:        &msg.CandidateReceipt,
		PoV:                     msg.Pov,
		ExecutorParams:          msg.ExecutorParams,
		PvfExecTimeoutKind:      parachaintypes.PvfExecTimeoutKind{},
		ResultCh:                taskResult,
	}
	go cv.pvfHost.Validate(validationTask)

	result := <-taskResult
	if result.InternalError != nil {
		logger.Errorf("failed to validate from chain state: %w", result.InternalError)
		msg.Ch <- parachaintypes.OverseerFuncRes[pvf.ValidationResult]{
			Err: result.InternalError,
		}
	} else {
		msg.Ch <- parachaintypes.OverseerFuncRes[pvf.ValidationResult]{
			Data: *result.Result,
		}
	}
}

// performBasicChecks Does basic checks of a candidate. Provide the encoded PoV-block.
// Returns ReasonForInvalidity and internal error if any.
func performBasicChecks(candidate *parachaintypes.CandidateDescriptor, maxPoVSize uint32,
	pov parachaintypes.PoV, validationCodeHash parachaintypes.ValidationCodeHash) (validationError *pvf.
	ReasonForInvalidity, internalError error) {
	povHash, err := pov.Hash()
	if err != nil {
		return nil, fmt.Errorf("hashing PoV: %w", err)
	}

	encodedPoV, err := scale.Marshal(pov)
	if err != nil {
		return nil, fmt.Errorf("encoding PoV: %w", err)
	}
	encodedPoVSize := uint32(len(encodedPoV))

	if encodedPoVSize > maxPoVSize {
		ci := pvf.ParamsTooLarge
		return &ci, nil
	}

	if povHash != candidate.PovHash {
		ci := pvf.PoVHashMismatch
		return &ci, nil
	}

	if validationCodeHash != candidate.ValidationCodeHash {
		ci := pvf.CodeHashMismatch
		return &ci, nil
	}

	err = candidate.CheckCollatorSignature()
	if err != nil {
		ci := pvf.BadSignature
		return &ci, nil
	}
	return nil, nil
}
