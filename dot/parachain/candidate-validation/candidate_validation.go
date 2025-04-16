// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package candidatevalidation

import (
	"context"
	"errors"
	"fmt"
	"time"

	parachainruntime "github.com/ChainSafe/gossamer/dot/parachain/runtime"
	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/dot/parachain/util"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/runtime"
)

// CandidateValidation is a parachain subsystem that validates candidate parachain blocks
type CandidateValidation struct {
	SubsystemToOverseer chan<- any
	BlockState          BlockState
	// pvfHost is the host for the parachain validation function
	pvfHost *host
	// Cache of claim queues per relay parent
	claimQueueCache map[common.Hash]parachaintypes.ClaimQueue
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

		// Cache of claim queues per relay parent
		claimQueueCache: make(map[common.Hash]parachaintypes.ClaimQueue),
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
func (cv *CandidateValidation) ProcessActiveLeavesUpdateSignal(signal parachaintypes.ActiveLeavesUpdateSignal) error {
	// Clear cache for deactivated leaves
	cv.clearOldCache(signal)
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
		validationResult, err := cv.validateFromExhaustive(msg)
		if err != nil {
			msg.Ch <- parachaintypes.OverseerFuncRes[ValidationResult]{
				Err: err,
			}
		} else {
			msg.Ch <- parachaintypes.OverseerFuncRes[ValidationResult]{
				Data: *validationResult,
			}
		}

	case PreCheck:
		outcome, err := cv.precheckPvF(msg.RelayParent, msg.ValidationCodeHash)
		if err != nil {
			logger.Errorf("failed to precheck: %w", err)
		}
		logger.Debugf("Precheck outcome: %v", outcome)
		msg.ResponseSender <- outcome

	case parachaintypes.ActiveLeavesUpdateSignal:
		_ = cv.ProcessActiveLeavesUpdateSignal(msg)

	case parachaintypes.BlockFinalizedSignal:
		_ = cv.ProcessBlockFinalizedSignal(msg)

	default:
		logger.Errorf("%w: %T", parachaintypes.ErrUnknownOverseerMessage, msg)
	}
}

func (cv *CandidateValidation) validateFromExhaustive(msg ValidateFromExhaustive) (*ValidationResult, error) {
	runtimeInstance, err := cv.BlockState.GetRuntime(msg.CandidateReceipt.Descriptor.RelayParent)
	if err != nil {
		return nil, fmt.Errorf("getting runtime instance: %w", err)
	}

	expectedSessionIndex, err := runtimeInstance.ParachainHostSessionIndexForChild()
	if err != nil {
		return nil, fmt.Errorf("getting session index: %w", err)
	}

	execKind, err := msg.PvfExecTimeoutKind.Value()
	if err != nil {
		return nil, fmt.Errorf("getting execution kind: %w", err)
	}

	// We only check the session index for backing.
	_, isBackingExecKind := execKind.(parachaintypes.Backing)
	if isBackingExecKind && expectedSessionIndex != msg.CandidateReceipt.Descriptor.SessionIndex {
		return &ValidationResult{Invalid: InvalidSessionIndex.Ptr()}, nil
	}

	// Get claim queue from cache or fetch from runtime
	claimQueue, err := cv.getOrFetchClaimQueue(runtimeInstance, msg.CandidateReceipt.Descriptor.RelayParent)
	if err != nil {
		return nil, fmt.Errorf("getting claim queue: %w", err)
	}

	validationTask := &ValidationTask{
		PersistedValidationData: msg.PersistedValidationData,
		ValidationCode:          &msg.ValidationCode,
		CandidateReceipt:        &msg.CandidateReceipt,
		PoV:                     msg.PoV,
		ExecutorParams:          msg.ExecutorParams,
		PvfExecTimeoutKind:      msg.PvfExecTimeoutKind,
		ClaimQueue:              claimQueue,
	}

	return cv.pvfHost.validate(validationTask)
}

// PoVRequestor gets proof of validity by issuing network requests to validators of the current backing group.
// TODO: Implement PoV requestor, issue #3919
type PoVRequestor interface {
	RequestPoV(povHash common.Hash) parachaintypes.PoV
}

// getValidationData gets validation data for a parachain block from the runtime instance
func getValidationData(runtimeInstance parachainruntime.RuntimeInstance, paraID parachaintypes.ParaID,
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

	if persistedValidationData == nil {
		reason := ValidationResult{
			Invalid: BadParent.Ptr(),
		}
		msg.Ch <- parachaintypes.OverseerFuncRes[ValidationResult]{
			Data: reason,
		}
		return
	}
	validationTask := &ValidationTask{
		PersistedValidationData: *persistedValidationData,
		ValidationCode:          validationCode,
		CandidateReceipt:        &msg.CandidateReceipt,
		PoV:                     msg.Pov,
		ExecutorParams:          msg.ExecutorParams,
		PvfExecTimeoutKind:      msg.ExecKind,
	}

	result, err := cv.pvfHost.validate(validationTask)
	if err != nil {
		msg.Ch <- parachaintypes.OverseerFuncRes[ValidationResult]{
			Err: err,
		}
		return
	}
	if !result.IsValid() {
		msg.Ch <- parachaintypes.OverseerFuncRes[ValidationResult]{
			Data: *result,
		}
		return
	}
	valid, err := runtimeInstance.ParachainHostCheckValidationOutputs(
		msg.CandidateReceipt.Descriptor.ParaID,
		result.Valid.CandidateCommitments)
	if err != nil {
		msg.Ch <- parachaintypes.OverseerFuncRes[ValidationResult]{
			Err: fmt.Errorf("check validation outputs: Bad request: %w", err),
		}
		return
	}
	if !valid {
		reason := &ValidationResult{
			Invalid: InvalidOutputs.Ptr(),
		}
		msg.Ch <- parachaintypes.OverseerFuncRes[ValidationResult]{
			Data: *reason,
		}
		return
	}
	msg.Ch <- parachaintypes.OverseerFuncRes[ValidationResult]{
		Data: *result,
	}
}

// precheckPvF prechecks the parachain validation function by retrieving the validation code from the runtime instance
// and calling the precheck method on the pvf host. It returns the precheck outcome.
func (cv *CandidateValidation) precheckPvF(relayParent common.Hash, validationCodeHash parachaintypes.
	ValidationCodeHash) (PreCheckOutcome, error) {
	runtimeInstance, err := cv.BlockState.GetRuntime(relayParent)
	if err != nil {
		return PreCheckOutcomeFailed, fmt.Errorf("failed to get runtime instance: %w", err)
	}

	code, err := runtimeInstance.ParachainHostValidationCodeByHash(common.Hash(validationCodeHash))
	if err != nil {
		return PreCheckOutcomeFailed, fmt.Errorf("failed to get validation code by hash: %w", err)
	}

	executorParams, err := util.ExecutorParamsAtRelayParent(runtimeInstance, relayParent)
	if err != nil {
		return PreCheckOutcomeInvalid, fmt.Errorf("failed to acquire params for the session, thus voting against: %w", err)
	}

	kind := parachaintypes.NewPvfPrepTimeoutKind()
	err = kind.SetValue(parachaintypes.Precheck{})
	if err != nil {
		return PreCheckOutcomeFailed, fmt.Errorf("failed to set value: %w", err)
	}

	prepTimeout := pvfPrepTimeout(*executorParams, kind)

	pvf := PvFPrepData{
		code:           *code,
		codeHash:       validationCodeHash,
		executorParams: *executorParams,
		prepTimeout:    prepTimeout,
		prepKind:       kind,
	}
	err = cv.pvfHost.precheck(pvf)
	if err != nil {
		return PreCheckOutcomeFailed, fmt.Errorf("failed to precheck: %w", err)
	}
	return PreCheckOutcomeValid, nil
}

// pvfPrepTimeout To determine the amount of timeout time for the pvf execution.
//
//	The time period after which the preparation worker is considered
//
// unresponsive and will be killed.
func pvfPrepTimeout(params parachaintypes.ExecutorParams, kind parachaintypes.PvfPrepTimeoutKind) time.Duration {
	for _, param := range params {
		val, err := param.Value()
		if err != nil {
			logger.Errorf("determining parameter values %w", err)
		}
		switch val := val.(type) {
		case parachaintypes.PvfPrepTimeout:
			// convert milliseconds to nanoseconds and cast to time.Duration.
			return time.Duration(val.Millisec * 1000000)
		}
	}

	timeoutKind, err := kind.Value()
	if err != nil {
		return time.Second * 2
	}
	switch timeoutKind.(type) {
	case parachaintypes.Precheck:
		return time.Second * 2
	case parachaintypes.Lenient:
		return time.Second * 10
	default:
		return time.Second * 2
	}
}

// getOrFetchClaimQueue gets the claim queue from cache or fetches it from runtime
func (cv *CandidateValidation) getOrFetchClaimQueue(
	runtimeInstance runtime.Instance, relayParent common.Hash) (parachaintypes.ClaimQueue, error) {
	// Check cache first
	if claimQueue, ok := cv.claimQueueCache[relayParent]; ok {
		return claimQueue, nil
	}

	// Fetch from runtime if not in cache
	claimQueue, err := runtimeInstance.ParachainHostClaimQueue()
	if err != nil {
		return nil, fmt.Errorf("getting claim queue: %w", err)
	}

	// Store in cache
	cv.claimQueueCache[relayParent] = claimQueue
	return claimQueue, nil
}

// clearOldCache clears claim queues for old relay parents
func (cv *CandidateValidation) clearOldCache(activeLeavesUpdate parachaintypes.ActiveLeavesUpdateSignal) {
	// Remove deactivated leaves from cache
	for _, deactivated := range activeLeavesUpdate.Deactivated {
		delete(cv.claimQueueCache, deactivated)
	}
}
