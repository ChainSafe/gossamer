package pvf

import (
	"fmt"
	"sync"

	parachainruntime "github.com/ChainSafe/gossamer/dot/parachain/runtime"
	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

var logger = log.NewFromGlobal(log.AddContext("pkg", "pvf"), log.SetLevel(log.Debug))

type ValidationHost struct {
	wg     sync.WaitGroup
	stopCh chan struct{}

	workerPool *validationWorkerPool
}

func (v *ValidationHost) Start() {
	fmt.Printf("v.wg %v\n", v)
	v.wg.Add(1)
	logger.Debug("Starting validation host")
	go func() {
		defer v.wg.Done()
	}()
}

func (v *ValidationHost) Stop() {
	close(v.stopCh)
	v.wg.Wait()
}

func NewValidationHost() *ValidationHost {
	return &ValidationHost{
		stopCh:     make(chan struct{}),
		workerPool: newValidationWorkerPool(),
	}
}

func (v *ValidationHost) Validate(msg *ValidationTask) {
	logger.Debugf("Validating worker", "workerID", msg.WorkerID)

	validationCodeHash := msg.ValidationCode.Hash()
	// basic checks
	validationErr, internalErr := performBasicChecks(&msg.CandidateReceipt.Descriptor,
		msg.PersistedValidationData.MaxPovSize,
		msg.PoV,
		validationCodeHash)
	// TODO(ed): confirm how to handle internal errors
	if internalErr != nil {
		logger.Errorf("performing basic checks: %w", internalErr)
	}

	if validationErr != nil {
		valErr := &ValidationTaskResult{
			who: validationCodeHash,
			Result: &ValidationResult{
				InvalidResult: validationErr,
			},
		}
		msg.ResultCh <- valErr
		return
	}

	workerID := v.poolContainsWorker(msg)
	validationParams := parachainruntime.ValidationParameters{
		ParentHeadData:         msg.PersistedValidationData.ParentHead,
		BlockData:              msg.PoV.BlockData,
		RelayParentNumber:      msg.PersistedValidationData.RelayParentNumber,
		RelayParentStorageRoot: msg.PersistedValidationData.RelayParentStorageRoot,
	}
	workTask := &workerTask{
		work:       validationParams,
		maxPoVSize: msg.PersistedValidationData.MaxPovSize,
		ResultCh:   msg.ResultCh,
	}
	v.workerPool.submitRequest(workerID, workTask)
}

func (v *ValidationHost) poolContainsWorker(msg *ValidationTask) parachaintypes.ValidationCodeHash {
	if msg.WorkerID != nil {
		return *msg.WorkerID
	}
	if v.workerPool.containsWorker(msg.ValidationCode.Hash()) {
		return msg.ValidationCode.Hash()
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
