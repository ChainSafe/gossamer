package pvf

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"sync"
	"time"

	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"golang.org/x/exp/maps"
)

const (
	maxRequestsAllowed uint = 60
)

type validationWorkerPool struct {
	mtx sync.RWMutex
	wg  sync.WaitGroup

	workers map[parachaintypes.ValidationCodeHash]*validationWorker
}

type ValidationTask struct {
	PersistedValidationData parachaintypes.PersistedValidationData
	WorkerID                *parachaintypes.ValidationCodeHash
	CandidateReceipt        *parachaintypes.CandidateReceipt
	PoV                     parachaintypes.PoV
	ExecutorParams          parachaintypes.ExecutorParams
	PvfExecTimeoutKind      parachaintypes.PvfExecTimeoutKind
	ResultCh                chan<- *ValidationTaskResult
}

type ValidationTaskResult struct {
	who    parachaintypes.ValidationCodeHash
	result *ValidationResult
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

type validationWorker struct {
	worker *worker
	queue  chan *ValidationTask
}

func newValidationWorkerPool() *validationWorkerPool {
	return &validationWorkerPool{
		workers: make(map[parachaintypes.ValidationCodeHash]*validationWorker),
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

func (v *validationWorkerPool) newValidationWorker(who parachaintypes.ValidationCodeHash) {

	worker := newWorker(who)
	workerQueue := make(chan *ValidationTask, maxRequestsAllowed)

	v.wg.Add(1)
	go worker.run(workerQueue, &v.wg)

	v.workers[who] = &validationWorker{
		worker: worker,
		queue:  workerQueue,
	}
	logger.Tracef("potential worker added, total in the pool %d", len(v.workers))
}

// submitRequest given a request, the worker pool will get the peer given the peer.ID
// parameter or if nil the very first available worker or
// to perform the request, the response will be dispatch in the resultCh.
func (v *validationWorkerPool) submitRequest(request *ValidationTask) {

	//task := &validationTask{
	//	request:  request,
	//	resultCh: resultCh,
	//}

	// if the request is bounded to a specific peer then just
	// request it and sent through its queue otherwise send
	// the request in the general queue where all worker are
	// listening on
	v.mtx.RLock()
	defer v.mtx.RUnlock()

	if request.WorkerID != nil {
		syncWorker, inMap := v.workers[*request.WorkerID]
		if inMap {
			if syncWorker == nil {
				panic("sync worker should not be nil")
			}
			syncWorker.queue <- request
			return
		}
	}

	// if the exact peer is not specified then
	// randomly select a worker and assign the
	// task to it, if the amount of workers is
	var selectedWorkerIdx int
	workers := maps.Values(v.workers)
	nBig, err := rand.Int(rand.Reader, big.NewInt(int64(len(workers))))
	if err != nil {
		panic(fmt.Errorf("fail to get a random number: %w", err))
	}
	selectedWorkerIdx = int(nBig.Int64())
	selectedWorker := workers[selectedWorkerIdx]
	selectedWorker.queue <- request
}

func (v *validationWorkerPool) containsWorker(workerID parachaintypes.ValidationCodeHash) bool {
	v.mtx.RLock()
	defer v.mtx.RUnlock()

	_, inMap := v.workers[workerID]
	return inMap
}
