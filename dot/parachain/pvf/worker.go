package pvf

import (
	parachainruntime "github.com/ChainSafe/gossamer/dot/parachain/runtime"
	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
)

// TODO(ed): figure out a better name for this that describes what it does
type worker struct {
	workerID parachaintypes.ValidationCodeHash
	instance *parachainruntime.Instance
	// TODO(ed): determine if wasProcessed is stored here or in host
	isProcessed map[parachaintypes.CandidateHash]struct{}
	// TODO make this a buffered channel, and determine the buffer size
	workerTasksChan chan *workerTask
}

type workerTask struct {
	work             parachainruntime.ValidationParameters
	maxPoVSize       uint32
	candidateReceipt *parachaintypes.CandidateReceipt
}

func newWorker(validationCode parachaintypes.ValidationCode, queue chan *workerTask) (*worker, error) {
	validationRuntime, err := parachainruntime.SetupVM(validationCode)

	if err != nil {
		return nil, err
	}
	return &worker{
		workerID:        validationCode.Hash(),
		instance:        validationRuntime,
		workerTasksChan: queue,
	}, nil
}

func (w *worker) executeRequest(task *workerTask) chan *ValidationTaskResult {
	logger.Debugf("[EXECUTING] worker %x task %v", w.workerID, task.work)
	resultCh := make(chan *ValidationTaskResult)

	go func() {
		defer close(resultCh)
		candidateHash, err := parachaintypes.GetCandidateHash(task.candidateReceipt)
		if err != nil {
			// TODO: handle error
			logger.Errorf("getting candidate hash: %w", err)
		}

		// do isProcessed check here
		if _, ok := w.isProcessed[candidateHash]; ok {
			// TODO: determine what the isPreccessed check should return, and if re-trying is allowed
			//  get a better understanding of what the isProcessed check should be checking for
			logger.Debugf("candidate %x already processed", candidateHash)
		}
		validationResult, err := w.instance.ValidateBlock(task.work)

		if err != nil {
			logger.Errorf("executing validate_block: %w", err)
			reasonForInvalidity := ExecutionError
			errorResult := &ValidationResult{
				InvalidResult: &reasonForInvalidity,
			}
			resultCh <- &ValidationTaskResult{
				who:    w.workerID,
				Result: errorResult,
			}
			return
		}

		headDataHash, err := validationResult.HeadData.Hash()
		if err != nil {
			logger.Errorf("hashing head data: %w", err)
			reasonForInvalidity := ExecutionError
			errorResult := &ValidationResult{
				InvalidResult: &reasonForInvalidity,
			}
			resultCh <- &ValidationTaskResult{
				who:    w.workerID,
				Result: errorResult,
			}
			return
		}

		if headDataHash != task.candidateReceipt.Descriptor.ParaHead {
			reasonForInvalidity := ParaHeadHashMismatch
			errorResult := &ValidationResult{
				InvalidResult: &reasonForInvalidity,
			}
			resultCh <- &ValidationTaskResult{
				who:    w.workerID,
				Result: errorResult,
			}
			return
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
			errorResult := &ValidationResult{
				InvalidResult: &reasonForInvalidity,
			}
			resultCh <- &ValidationTaskResult{
				who:    w.workerID,
				Result: errorResult,
			}
			return
		}
		pvd := parachaintypes.PersistedValidationData{
			ParentHead:             task.work.ParentHeadData,
			RelayParentNumber:      task.work.RelayParentNumber,
			RelayParentStorageRoot: task.work.RelayParentStorageRoot,
			MaxPovSize:             task.maxPoVSize,
		}
		validResult := &ValidationResult{
			ValidResult: &ValidValidationResult{
				CandidateCommitments:    candidateCommitments,
				PersistedValidationData: pvd,
			},
		}

		logger.Debugf("[RESULT] worker %x, result: %v, error: %s", w.workerID, validResult, err)

		resultCh <- &ValidationTaskResult{
			who:    w.workerID,
			Result: validResult,
		}
	}()
	return resultCh
}
