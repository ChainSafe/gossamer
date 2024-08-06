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
	work       parachainruntime.ValidationParameters
	maxPoVSize uint32
	ResultCh   chan<- *ValidationTaskResult
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

func (w *worker) run(queue chan *workerTask, wg *sync.WaitGroup) {
	defer func() {
		logger.Debugf("[STOPPED] worker %x", w.workerID)
		wg.Done()
	}()

	for task := range queue {
		w.executeRequest(task)
	}
}

func (w *worker) executeRequest(task *workerTask) {
	// WIP: This is a dummy implementation of the worker execution for the validation task.  The logic for
	//  validating the parachain block request should be implemented here.
	logger.Debugf("[EXECUTING] worker %x task %v", w.workerID, task.work)

	// todo do basic checks

	validationResult, err := w.instance.ValidateBlock(task.work)

	///////////////////////////////
	//if err != nil {
	//	return nil, fmt.Errorf("executing validate_block: %w", err)
	//}

	//headDataHash, err := validationResult.HeadData.Hash()
	//if err != nil {
	//	return nil, fmt.Errorf("hashing head data: %w", err)
	//}
	//
	//if headDataHash != candidateReceipt.Descriptor.ParaHead {
	//	ci := pvf.ParaHeadHashMismatch
	//	return &pvf.ValidationResult{InvalidResult: &ci}, nil
	//}
	candidateCommitments := parachaintypes.CandidateCommitments{
		UpwardMessages:            validationResult.UpwardMessages,
		HorizontalMessages:        validationResult.HorizontalMessages,
		NewValidationCode:         validationResult.NewValidationCode,
		HeadData:                  validationResult.HeadData,
		ProcessedDownwardMessages: validationResult.ProcessedDownwardMessages,
		HrmpWatermark:             validationResult.HrmpWatermark,
	}

	// if validation produced a new set of commitments, we treat the candidate as invalid
	//if candidateReceipt.CommitmentsHash != candidateCommitments.Hash() {
	//	ci := CommitmentsHashMismatch
	//	return &ValidationResult{InvalidResult: &ci}, nil
	//}
	pvd := parachaintypes.PersistedValidationData{
		ParentHead:             task.work.ParentHeadData,
		RelayParentNumber:      task.work.RelayParentNumber,
		RelayParentStorageRoot: task.work.RelayParentStorageRoot,
		MaxPovSize:             task.maxPoVSize,
	}
	dummyResilt := &ValidationResult{
		ValidResult: &ValidValidationResult{
			CandidateCommitments:    candidateCommitments,
			PersistedValidationData: pvd,
		},
	}
	//////////////////////////

	logger.Debugf("[RESULT] worker %x, result: %v, error: %s", w.workerID, dummyResilt, err)

	task.ResultCh <- &ValidationTaskResult{
		who:    w.workerID,
		Result: dummyResilt,
	}

	//logger.Debugf("[FINISHED] worker %v, error: %s", validationResult, err)
}
