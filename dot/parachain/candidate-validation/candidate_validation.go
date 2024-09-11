// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package candidatevalidation

import (
	"context"
	"errors"
	"fmt"

	parachainruntime "github.com/ChainSafe/gossamer/dot/parachain/runtime"
	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/runtime"
)

// CandidateValidation is a parachain subsystem that validates candidate parachain blocks
type CandidateValidation struct {
	SubsystemToOverseer chan<- any
	BlockState          BlockState
	pvfHost             *host // pvfHost is the host for the parachain validation function
}

type BlockState interface {
	GetRuntime(blockHash common.Hash) (instance runtime.Instance, err error)
}

// NewCandidateValidation creates a new CandidateValidation subsystem
func NewCandidateValidation(overseerChan chan<- any, blockState BlockState) *CandidateValidation {
	candidateValidation := CandidateValidation{
		SubsystemToOverseer: overseerChan,
		pvfHost:             newValidationHost(),
		BlockState:          blockState,
	}
	return &candidateValidation
}

// Run starts the CandidateValidation subsystem
func (cv *CandidateValidation) Run(ctx context.Context, overseerToSubsystem <-chan any) {
	for {
		select {
		case msg := <-overseerToSubsystem:
			cv.processMessage(msg)
		case <-ctx.Done():
			if err := ctx.Err(); err != nil {
				logger.Errorf("ctx error: %s\n", err)
			}
			return
		}
	}
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
func (*CandidateValidation) Stop() {
}

// processMessage processes messages sent to the CandidateValidation subsystem
func (cv *CandidateValidation) processMessage(msg any) {
	switch msg := msg.(type) {
	case ValidateFromChainState:
		cv.validateFromChainState(msg)
	case ValidateFromExhaustive:
		validationTask := &ValidationTask{
			PersistedValidationData: msg.PersistedValidationData,
			ValidationCode:          &msg.ValidationCode,
			CandidateReceipt:        &msg.CandidateReceipt,
			PoV:                     msg.PoV,
			ExecutorParams:          msg.ExecutorParams,
			PvfExecTimeoutKind:      msg.PvfExecTimeoutKind,
		}

		result, err := cv.pvfHost.validate(validationTask)

		if err != nil {
			logger.Errorf("failed to validate from exhaustive: %w", err)
			msg.Ch <- parachaintypes.OverseerFuncRes[ValidationResult]{
				Err: err,
			}
		} else {
			msg.Ch <- parachaintypes.OverseerFuncRes[ValidationResult]{
				Data: *result,
			}
		}

	case PreCheck:
		panic("TODO: implement functionality to handle PreCheck, see issue #3921")

	case parachaintypes.ActiveLeavesUpdateSignal:
		_ = cv.ProcessActiveLeavesUpdateSignal(msg)

	case parachaintypes.BlockFinalizedSignal:
		_ = cv.ProcessBlockFinalizedSignal(msg)

	default:
		logger.Errorf("%w: %T", parachaintypes.ErrUnknownOverseerMessage, msg)
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

// validateFromChainState validates a parachain block from chain state message
func (cv *CandidateValidation) validateFromChainState(msg ValidateFromChainState) {
	runtimeInstance, err := cv.BlockState.GetRuntime(msg.CandidateReceipt.Descriptor.RelayParent)
	if err != nil {
		logger.Errorf("getting runtime instance: %w", err)
		msg.Ch <- parachaintypes.OverseerFuncRes[ValidationResult]{
			Err: fmt.Errorf("getting runtime instance: %w", err),
		}
		return
	}

	persistedValidationData, validationCode, err := getValidationData(runtimeInstance,
		msg.CandidateReceipt.Descriptor.ParaID)
	if err != nil {
		logger.Errorf("getting validation data: %w", err)
		msg.Ch <- parachaintypes.OverseerFuncRes[ValidationResult]{
			Err: fmt.Errorf("getting validation data: %w", err),
		}
		return
	}

	validationTask := &ValidationTask{
		PersistedValidationData: *persistedValidationData,
		ValidationCode:          validationCode,
		CandidateReceipt:        &msg.CandidateReceipt,
		PoV:                     msg.Pov,
		ExecutorParams:          msg.ExecutorParams,
		// todo: implement PvfExecTimeoutKind, so that validate can be called with a timeout see issue: #3429
		PvfExecTimeoutKind: parachaintypes.PvfExecTimeoutKind{},
	}

	result, err := cv.pvfHost.validate(validationTask)
	if err != nil {
		msg.Ch <- parachaintypes.OverseerFuncRes[ValidationResult]{
			Err: err,
		}
	} else {
		msg.Ch <- parachaintypes.OverseerFuncRes[ValidationResult]{
			Data: *result,
		}
	}
}
