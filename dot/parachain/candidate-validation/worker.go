package candidatevalidation

import (
	"fmt"
	"time"

	parachainruntime "github.com/ChainSafe/gossamer/dot/parachain/runtime"
	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
)

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

func newWorker(validationCode parachaintypes.ValidationCode) (*worker, error) {
	validationRuntime, err := parachainruntime.SetupVM(validationCode)

	if err != nil {
		return nil, err
	}
	return &worker{
		workerID:    validationCode.Hash(),
		instance:    validationRuntime,
		isProcessed: make(map[parachaintypes.CandidateHash]*ValidationResult),
	}, nil
}

type resultWithError struct {
	result *parachainruntime.ValidationResult
	err    error
}

func determineTimeout(timeoutKind parachaintypes.PvfExecTimeoutKind) time.Duration {
	value, err := timeoutKind.Value()
	if err != nil {
		return 2 * time.Second
	}
	switch value.(type) {
	case parachaintypes.Backing:
		return 2 * time.Second
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

	// do isProcessed check here
	if processed, ok := w.isProcessed[candidateHash]; ok {
		logger.Debugf("candidate %x already processed", candidateHash)
		return processed, nil
	}

	var validationResult *parachainruntime.ValidationResult
	validationResultCh := make(chan (*resultWithError))
	timeoutDuration := determineTimeout(task.timeoutKind)

	go func() {
		result, err := w.instance.ValidateBlock(task.work)
		if err != nil {
			validationResultCh <- &resultWithError{result: nil, err: err}
		}
		validationResultCh <- &resultWithError{
			result: result,
		}
	}()

	select {
	case validationResultWErr := <-validationResultCh:
		if validationResultWErr.err != nil {
			logger.Errorf("executing validate_block: %w", err)
			reasonForInvalidity := ExecutionError
			return &ValidationResult{InvalidResult: &reasonForInvalidity}, nil //nolint
		}
		validationResult = validationResultWErr.result

	case <-time.After(timeoutDuration):
		logger.Errorf("validation timed out")
		reasonForInvalidity := Timeout
		return &ValidationResult{InvalidResult: &reasonForInvalidity}, nil
	}

	headDataHash, err := validationResult.HeadData.Hash()
	if err != nil {
		logger.Errorf("hashing head data: %w", err)
		reasonForInvalidity := ExecutionError
		return &ValidationResult{InvalidResult: &reasonForInvalidity}, nil
	}

	if headDataHash != task.candidateReceipt.Descriptor.ParaHead {
		reasonForInvalidity := ParaHeadHashMismatch
		return &ValidationResult{InvalidResult: &reasonForInvalidity}, nil
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
		return &ValidationResult{InvalidResult: &reasonForInvalidity}, nil
	}
	pvd := parachaintypes.PersistedValidationData{
		ParentHead:             task.work.ParentHeadData,
		RelayParentNumber:      task.work.RelayParentNumber,
		RelayParentStorageRoot: task.work.RelayParentStorageRoot,
		MaxPovSize:             task.maxPoVSize,
	}
	result := &ValidationResult{
		ValidResult: &Valid{
			CandidateCommitments:    candidateCommitments,
			PersistedValidationData: pvd,
		},
	}
	w.isProcessed[candidateHash] = result
	return result, nil
}
