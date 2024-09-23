// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package candidatevalidation

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	parachainruntime "github.com/ChainSafe/gossamer/dot/parachain/runtime"
	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/dot/parachain/util"
	validationprotocol "github.com/ChainSafe/gossamer/dot/parachain/validation-protocol"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/runtime"
	"github.com/klauspost/compress/zstd"
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
		outcome := cv.precheckPvF(msg.RelayParent, msg.ValidationCodeHash)
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

	if persistedValidationData == nil {
		badParent := BadParent
		reason := ValidationResult{
			Invalid: &badParent,
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
	valid, err := runtimeInstance.ParachainHostCheckValidationOutputs(parachaintypes.ParaID(msg.CandidateReceipt.
		Descriptor.ParaID), result.Valid.CandidateCommitments)
	if err != nil {
		msg.Ch <- parachaintypes.OverseerFuncRes[ValidationResult]{
			Err: fmt.Errorf("check validation outputs: Bad request: %w", err),
		}
		return
	}
	if !valid {
		invalidOutput := InvalidOutputs
		reason := &ValidationResult{
			Invalid: &invalidOutput,
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

func (cv *CandidateValidation) precheckPvF(relayParent common.Hash, validationCodeHash parachaintypes.
	ValidationCodeHash) PreCheckOutcome {
	runtimeInstance, err := cv.BlockState.GetRuntime(relayParent)
	if err != nil {
		logger.Errorf("failed to get runtime instance: %w", err)
		return PreCheckOutcomeFailed
	}

	code, err := runtimeInstance.ParachainHostValidationCodeByHash(common.Hash(validationCodeHash))
	if err != nil {
		logger.Errorf("failed to get validation code by hash: %w", err)
		return PreCheckOutcomeFailed
	}

	executorParams, err := util.ExecutorParamsAtRelayParent(runtimeInstance, relayParent)
	if err != nil {
		logger.Errorf("failed to acquire params for the session, thus voting against: %w", err)
		return PreCheckOutcomeInvalid
	}
	fmt.Printf("Executor params: %v\n", executorParams)

	// todo: pvf_prep_timeout

	codeDecompressed, err := maybeCompressedBlobDecompress(*code, validationprotocol.MaxValidationMessageSize)
	if err != nil {
		logger.Errorf("failed to decompress code: %w", err)
		return PreCheckOutcomeInvalid
	}
	fmt.Printf("Decompressed code: %v\n", len(codeDecompressed))

	// todo: call validation_backend.precheck_pvf
	return PreCheckOutcomeValid
}

// An arbitrary prefix, that indicates a blob beginning with should be decompressed with
// Zstd compression.
//
// This differs from the WASM magic bytes, so real WASM blobs will not have this prefix.
var zstdPrefix = []byte{82, 188, 83, 118, 70, 219, 142, 5}

func maybeCompressedBlobDecompress(blob []byte, bombLimit uint64) ([]byte, error) {
	// todo handle check for bombLimit
	if len(blob) < len(zstdPrefix) {
		return nil, fmt.Errorf("blob is too short")
	}
	if bytes.Equal(blob[0:len(zstdPrefix)], zstdPrefix) {
		decoder, err := zstd.NewReader(nil)
		if err != nil {
			return nil, fmt.Errorf("creating zstd decoder: %w", err)
		}
		defer decoder.Close()
		return decoder.DecodeAll(blob[len(zstdPrefix):], nil)
	} else {
		return blob, nil
	}
}
