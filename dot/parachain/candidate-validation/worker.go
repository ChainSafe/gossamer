package candidatevalidation

import (
	"errors"
	"fmt"
	"time"

	parachainruntime "github.com/ChainSafe/gossamer/dot/parachain/runtime"
	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
)

// worker is the thing that can execute a validation request
type worker struct {
	workerID    parachaintypes.ValidationCodeHash
	instance    parachainruntime.ValidatorInstance
	isProcessed map[parachaintypes.CandidateHash]*ValidationResult
}

type workerTask struct {
	work             parachainruntime.ValidationParameters
	maxPoVSize       uint32
	candidateReceipt *parachaintypes.CandidateReceipt
	timeoutKind      parachaintypes.PvfExecTimeoutKind
}

var ErrorPreCheckTimeout = errors.New("precheck timed out")

func newWorker(validationCode parachaintypes.ValidationCode, setupTimeout time.Duration) (*worker, error) {
	runtimeSetupResultCh := make(chan *resultWithError)
	var parachainRuntime *parachainruntime.Instance
	go func() {
		rt, err := parachainruntime.SetupVM(validationCode)
		runtimeSetupResultCh <- &resultWithError{result: rt, err: err}
	}()

	select {
	case validationResultWErr := <-runtimeSetupResultCh:
		if validationResultWErr.err != nil {
			logger.Errorf("setting up runtime instance: %w", validationResultWErr.err)
			return nil, validationResultWErr.err
		}
		parachainRuntime = validationResultWErr.result.(*parachainruntime.Instance)

	case <-time.After(setupTimeout):
		return nil, ErrorPreCheckTimeout
	}

	return &worker{
		workerID:    validationCode.Hash(),
		instance:    parachainRuntime,
		isProcessed: make(map[parachaintypes.CandidateHash]*ValidationResult),
	}, nil
}

type resultWithError struct {
	result any
	err    error
}

func determineTimeout(timeoutKind parachaintypes.PvfExecTimeoutKind) time.Duration {
	value, err := timeoutKind.Value()
	if err != nil {
		return 2 * time.Second
	}
	switch value.(type) {
	case parachaintypes.Approval:
		return 12 * time.Second
	default:
		return 2 * time.Second
	}
}

func (w *worker) executeRequest(task *workerTask) (*ValidationResult, error) {
	if task == nil {
		return nil, fmt.Errorf("task is nil")
	}
	logger.Debugf("[EXECUTING] worker %x task %v", w.workerID, task.work)
	candidateHash, err := parachaintypes.GetCandidateHash(task.candidateReceipt)
	if err != nil {
		return nil, err
	}

	if processed, ok := w.isProcessed[candidateHash]; ok {
		logger.Debugf("candidate %x already processed", candidateHash)
		return processed, nil
	}

	var validationResult *parachainruntime.ValidationResult
	validationResultCh := make(chan *resultWithError)
	timeoutDuration := determineTimeout(task.timeoutKind)

	go func() {
		result, err := w.instance.ValidateBlock(task.work)
		if err != nil {
			validationResultCh <- &resultWithError{result: nil, err: err}
		} else {
			validationResultCh <- &resultWithError{
				result: result,
			}
		}
	}()

	select {
	case validationResultWErr := <-validationResultCh:
		if validationResultWErr.err != nil {
			logger.Errorf("executing validate_block: %w", err)
			reasonForInvalidity := ExecutionError
			return &ValidationResult{Invalid: &reasonForInvalidity}, nil //nolint
		}
		validationResult = validationResultWErr.result.(*parachainruntime.ValidationResult)

	case <-time.After(timeoutDuration):
		logger.Errorf("validation timed out")
		reasonForInvalidity := Timeout
		return &ValidationResult{Invalid: &reasonForInvalidity}, nil
	}

	headDataHash, err := validationResult.HeadData.Hash()
	if err != nil {
		logger.Errorf("hashing head data: %w", err)
		reasonForInvalidity := ExecutionError
		return &ValidationResult{Invalid: &reasonForInvalidity}, nil
	}

	if headDataHash != task.candidateReceipt.Descriptor.ParaHead {
		reasonForInvalidity := ParaHeadHashMismatch
		return &ValidationResult{Invalid: &reasonForInvalidity}, nil
	}
	candidateCommitments := parachaintypes.CandidateCommitments{
		UpwardMessages:            validationResult.UpwardMessages,
		HorizontalMessages:        validationResult.HorizontalMessages,
		NewValidationCode:         validationResult.NewValidationCode,
		HeadData:                  validationResult.HeadData,
		ProcessedDownwardMessages: validationResult.ProcessedDownwardMessages,
		HrmpWatermark:             validationResult.HrmpWatermark,
	}

	// if validation produced a new set of commitments, we treat the candidate as invalid
	if task.candidateReceipt.CommitmentsHash != candidateCommitments.Hash() {
		reasonForInvalidity := CommitmentsHashMismatch
		return &ValidationResult{Invalid: &reasonForInvalidity}, nil
	}
	pvd := parachaintypes.PersistedValidationData{
		ParentHead:             task.work.ParentHeadData,
		RelayParentNumber:      task.work.RelayParentNumber,
		RelayParentStorageRoot: task.work.RelayParentStorageRoot,
		MaxPovSize:             task.maxPoVSize,
	}
	result := &ValidationResult{
		Valid: &Valid{
			CandidateCommitments:    candidateCommitments,
			PersistedValidationData: pvd,
		},
	}
	w.isProcessed[candidateHash] = result
	return result, nil
}
