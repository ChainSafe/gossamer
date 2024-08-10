package pvf

import (
	"fmt"
	"sync"
	"time"

	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
)

const (
	maxRequestsAllowed uint = 60
)

type validationWorkerPool struct {
	mtx sync.RWMutex
	wg  sync.WaitGroup

	workers map[parachaintypes.ValidationCodeHash]*worker
}

type ValidationTask struct {
	PersistedValidationData parachaintypes.PersistedValidationData
	WorkerID                *parachaintypes.ValidationCodeHash
	CandidateReceipt        *parachaintypes.CandidateReceipt
	PoV                     parachaintypes.PoV
	ExecutorParams          parachaintypes.ExecutorParams
	PvfExecTimeoutKind      parachaintypes.PvfExecTimeoutKind
	ValidationCode          *parachaintypes.ValidationCode
	ResultCh                chan<- *ValidationTaskResult
}

type ValidationTaskResult struct {
	who           parachaintypes.ValidationCodeHash
	Result        *ValidationResult
	InternalError error
}

// ValidationResult represents the result coming from the candidate validation subsystem.
// Validation results can be either a ValidValidationResult or InvalidValidationResult.
//
// If the result is invalid,
// store the reason for invalidity in the InvalidResult field of ValidationResult.
//
// If the result is valid,
// set the values of the ValidResult field of ValidValidationResult.
type ValidationResult struct {
	ValidResult   *ValidValidationResult
	InvalidResult *ReasonForInvalidity
}

func (vr ValidationResult) IsValid() bool {
	return vr.ValidResult != nil
}

type ValidValidationResult struct {
	CandidateCommitments    parachaintypes.CandidateCommitments
	PersistedValidationData parachaintypes.PersistedValidationData
}

type ReasonForInvalidity byte

const (
	// ExecutionError Failed to execute `validate_block`. This includes function panicking.
	ExecutionError ReasonForInvalidity = iota
	// InvalidOutputs Validation outputs check doesn't pass.
	InvalidOutputs
	// Timeout Execution timeout.
	Timeout
	// ParamsTooLarge Validation input is over the limit.
	ParamsTooLarge
	// CodeTooLarge Code size is over the limit.
	CodeTooLarge
	// PoVDecompressionFailure PoV does not decompress correctly.
	PoVDecompressionFailure
	// BadReturn Validation function returned invalid data.
	BadReturn
	// BadParent Invalid relay chain parent.
	BadParent
	// PoVHashMismatch POV hash does not match.
	PoVHashMismatch
	// BadSignature Bad collator signature.
	BadSignature
	// ParaHeadHashMismatch Para head hash does not match.
	ParaHeadHashMismatch
	// CodeHashMismatch Validation code hash does not match.
	CodeHashMismatch
	// CommitmentsHashMismatch Validation has generated different candidate commitments.
	CommitmentsHashMismatch
)

func (ci ReasonForInvalidity) Error() string {
	switch ci {
	case ExecutionError:
		return "failed to execute `validate_block`"
	case InvalidOutputs:
		return "validation outputs check doesn't pass"
	case Timeout:
		return "execution timeout"
	case ParamsTooLarge:
		return "validation input is over the limit"
	case CodeTooLarge:
		return "code size is over the limit"
	case PoVDecompressionFailure:
		return "PoV does not decompress correctly"
	case BadReturn:
		return "validation function returned invalid data"
	case BadParent:
		return "invalid relay chain parent"
	case PoVHashMismatch:
		return "PoV hash does not match"
	case BadSignature:
		return "bad collator signature"
	case ParaHeadHashMismatch:
		return "para head hash does not match"
	case CodeHashMismatch:
		return "validation code hash does not match"
	case CommitmentsHashMismatch:
		return "validation has generated different candidate commitments"
	default:
		return "unknown invalidity reason"
	}
}

//type validationWorker struct {
//	worker *worker
//	queue  chan *workerTask
//}

func newValidationWorkerPool() *validationWorkerPool {
	return &validationWorkerPool{
		workers: make(map[parachaintypes.ValidationCodeHash]*worker),
	}
}

// stop will shutdown all the available workers goroutines
func (v *validationWorkerPool) stop() error {
	v.mtx.RLock()
	defer v.mtx.RUnlock()

	for _, sw := range v.workers {
		close(sw.queue)
	}

	allWorkersDoneCh := make(chan struct{})
	go func() {
		defer close(allWorkersDoneCh)
		v.wg.Wait()
	}()

	timeoutTimer := time.NewTimer(30 * time.Second)
	select {
	case <-timeoutTimer.C:
		return fmt.Errorf("timeout reached while finishing workers")
	case <-allWorkersDoneCh:
		if !timeoutTimer.Stop() {
			<-timeoutTimer.C
		}

		return nil
	}
}

func (v *validationWorkerPool) newValidationWorker(validationCode parachaintypes.ValidationCode) (*parachaintypes.
	ValidationCodeHash, error) {

	workerQueue := make(chan *workerTask, maxRequestsAllowed)
	worker, err := newWorker(validationCode, workerQueue)
	if err != nil {
		logger.Errorf("failed to create a new worker: %w", err)
		return nil, err
	}
	v.wg.Add(1)
	go worker.run(workerQueue, &v.wg)

	v.workers[worker.workerID] = worker

	return &worker.workerID, nil
}

// submitRequest given a request, the worker pool will get the peer given the peer.ID
// parameter or if nil the very first available worker or
// to perform the request, the response will be dispatch in the resultCh.
func (v *validationWorkerPool) submitRequest(workerID parachaintypes.ValidationCodeHash, request *workerTask) {
	v.mtx.RLock()
	defer v.mtx.RUnlock()
	logger.Debugf("pool submit request workerID %x", workerID)

	syncWorker, inMap := v.workers[workerID]
	if inMap {
		if syncWorker == nil {
			panic("sync worker should not be nil")
		}
		logger.Debugf("sending request", workerID)
		syncWorker.queue <- request
		return
	}

	logger.Errorf("workerID %x not found in the pool", workerID)
	request.ResultCh <- &ValidationTaskResult{
		who:           workerID,
		InternalError: fmt.Errorf("workerID %x not found in the pool", workerID),
	}
}

func (v *validationWorkerPool) containsWorker(workerID parachaintypes.ValidationCodeHash) bool {
	v.mtx.RLock()
	defer v.mtx.RUnlock()

	_, inMap := v.workers[workerID]
	return inMap
}
