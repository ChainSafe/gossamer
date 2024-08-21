package pvf

import (
	"sync"

	parachainruntime "github.com/ChainSafe/gossamer/dot/parachain/runtime"
	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
)

type worker struct {
	workerID parachaintypes.ValidationCodeHash
	instance *parachainruntime.Instance
	queue    chan *workerTask
}

type workerTask struct {
	work             parachainruntime.ValidationParameters
	maxPoVSize       uint32
	candidateReceipt *parachaintypes.CandidateReceipt
	ResultCh         chan<- *ValidationTaskResult
}

func newWorker(validationCode parachaintypes.ValidationCode, queue chan *workerTask) (*worker, error) {
	validationRuntime, err := parachainruntime.SetupVM(validationCode)

	if err != nil {
		return nil, err
	}
	return &worker{
		workerID: validationCode.Hash(),
		instance: validationRuntime,
		queue:    queue,
	}, nil
}

func (w *worker) run(wg *sync.WaitGroup) {
	defer func() {
		logger.Debugf("[STOPPED] worker %x", w.workerID)
		wg.Done()
	}()

	for task := range w.queue {
		w.executeRequest(task)
	}
}

func (w *worker) executeRequest(task *workerTask) {
	logger.Debugf("[EXECUTING] worker %x task %v", w.workerID, task.work)

	validationResult, err := w.instance.ValidateBlock(task.work)

	if err != nil {
		logger.Errorf("executing validate_block: %w", err)
		reasonForInvalidity := ExecutionError
		errorResult := &ValidationResult{
			InvalidResult: &reasonForInvalidity,
		}
		task.ResultCh <- &ValidationTaskResult{
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
		task.ResultCh <- &ValidationTaskResult{
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
		task.ResultCh <- &ValidationTaskResult{
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
		task.ResultCh <- &ValidationTaskResult{
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

	task.ResultCh <- &ValidationTaskResult{
		who:    w.workerID,
		Result: validResult,
	}
}
