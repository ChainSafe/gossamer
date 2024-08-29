package pvf

import (
	"fmt"

	parachainruntime "github.com/ChainSafe/gossamer/dot/parachain/runtime"
	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

var logger = log.NewFromGlobal(log.AddContext("pkg", "pvf"), log.SetLevel(log.Debug))

type Host struct {
	workerPool *workerPool
}

func NewValidationHost() *Host {
	return &Host{
		workerPool: newValidationWorkerPool(),
	}
}

func (v *Host) Validate(msg *ValidationTask) (*ValidationResult, error) {
	logger.Debugf("Start Validating worker %x", msg.WorkerID)
	validationCodeHash := msg.ValidationCode.Hash()
	// performBasicChecks
	validationErr, internalErr := performBasicChecks(&msg.CandidateReceipt.Descriptor,
		msg.PersistedValidationData.MaxPovSize,
		msg.PoV,
		validationCodeHash)

	if internalErr != nil {
		return nil, internalErr
	}
	if validationErr != nil {
		return &ValidationResult{InvalidResult: validationErr}, nil //nolint
	}
	// check if worker is in pool
	workerID, internalErr := v.poolContainsWorker(msg)
	if internalErr != nil {
		return nil, internalErr

	}

	// submit request
	validationParams := parachainruntime.ValidationParameters{
		ParentHeadData:         msg.PersistedValidationData.ParentHead,
		BlockData:              msg.PoV.BlockData,
		RelayParentNumber:      msg.PersistedValidationData.RelayParentNumber,
		RelayParentStorageRoot: msg.PersistedValidationData.RelayParentStorageRoot,
	}
	workTask := &workerTask{
		work:             validationParams,
		maxPoVSize:       msg.PersistedValidationData.MaxPovSize,
		candidateReceipt: msg.CandidateReceipt,
	}
	logger.Debugf("Working Validating worker %x", workerID)
	return v.workerPool.submitRequest(*workerID, workTask)

}

func (v *Host) poolContainsWorker(msg *ValidationTask) (*parachaintypes.ValidationCodeHash, error) {
	if msg.WorkerID != nil {
		return msg.WorkerID, nil
	}
	validationCodeHash := msg.ValidationCode.Hash()
	if v.workerPool.containsWorker(validationCodeHash) {
		return &validationCodeHash, nil
	} else {
		return v.workerPool.newValidationWorker(*msg.ValidationCode)
	}
}

// performBasicChecks Does basic checks of a candidate. Provide the encoded PoV-block.
// Returns ReasonForInvalidity and internal error if any.
func performBasicChecks(candidate *parachaintypes.CandidateDescriptor, maxPoVSize uint32,
	pov parachaintypes.PoV, validationCodeHash parachaintypes.ValidationCodeHash) (
	validationError *ReasonForInvalidity, internalError error) {
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
		ci := ParamsTooLarge
		return &ci, nil
	}

	if povHash != candidate.PovHash {
		ci := PoVHashMismatch
		return &ci, nil
	}

	if validationCodeHash != candidate.ValidationCodeHash {
		ci := CodeHashMismatch
		return &ci, nil
	}

	err = candidate.CheckCollatorSignature()
	if err != nil {
		ci := BadSignature
		return &ci, nil
	}
	return nil, nil
}
