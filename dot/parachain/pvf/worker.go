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
}

type workerTask struct {
	work             parachainruntime.ValidationParameters
	maxPoVSize       uint32
	candidateReceipt *parachaintypes.CandidateReceipt
}

func newWorker(validationCode parachaintypes.ValidationCode) (*worker, error) {
	validationRuntime, err := parachainruntime.SetupVM(validationCode)

	if err != nil {
		return nil, err
	}
	return &worker{
		workerID: validationCode.Hash(),
		instance: validationRuntime,
	}, nil
}

func (w *worker) executeRequest(task *workerTask) (*ValidationResult, error) {
	logger.Debugf("[EXECUTING] worker %x task %v", w.workerID, task.work)
	candidateHash, err := parachaintypes.GetCandidateHash(task.candidateReceipt)
	if err != nil {
		return nil, err
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
	return &ValidationResult{
		ValidResult: &ValidValidationResult{
			CandidateCommitments:    candidateCommitments,
			PersistedValidationData: pvd,
		},
	}, nil

}
