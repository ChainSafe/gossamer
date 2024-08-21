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

type Host struct {
	wg     sync.WaitGroup
	stopCh chan struct{}

	workerPool *workerPool
}

func (v *Host) Start() {
	v.wg.Add(1)
	logger.Debug("Starting validation host")
	go func() {
		defer v.wg.Done()
	}()
}

func (v *Host) Stop() {
	close(v.stopCh)
	v.wg.Wait()
}

func NewValidationHost() *Host {
	return &Host{
		stopCh:     make(chan struct{}),
		workerPool: newValidationWorkerPool(),
	}
}

func (v *Host) Validate(msg *ValidationTask) {
	logger.Debugf("Validating worker %x", msg.WorkerID)

	validationCodeHash := msg.ValidationCode.Hash()
	// basic checks
	validationErr, internalErr := performBasicChecks(&msg.CandidateReceipt.Descriptor,
		msg.PersistedValidationData.MaxPovSize,
		msg.PoV,
		validationCodeHash)

	if internalErr != nil {
		logger.Errorf("performing basic checks: %w", internalErr)
		intErr := &ValidationTaskResult{
			who:           validationCodeHash,
			InternalError: internalErr,
		}
		msg.ResultCh <- intErr
		return
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

	workerID, err := v.poolContainsWorker(msg)
	if err != nil {
		logger.Errorf("pool contains worker: %w", err)
		intErr := &ValidationTaskResult{
			who:           validationCodeHash,
			InternalError: err,
		}
		msg.ResultCh <- intErr
		return
	}
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
		ResultCh:         msg.ResultCh,
	}
	v.workerPool.submitRequest(*workerID, workTask)
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
