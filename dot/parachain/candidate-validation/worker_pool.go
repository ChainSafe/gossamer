package candidatevalidation

import (
	"fmt"
	"time"

	parachainruntime "github.com/ChainSafe/gossamer/dot/parachain/runtime"
	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
)

type workerPool struct {
	workers map[parachaintypes.ValidationCodeHash]*worker
}

type PvFPrepData struct {
	code           parachaintypes.ValidationCode
	codeHash       parachaintypes.ValidationCodeHash
	executorParams parachaintypes.ExecutorParams
	prepTimeout    time.Duration
	prepKind       parachaintypes.PvfPrepTimeoutKind
}

type ValidationTask struct {
	PersistedValidationData parachaintypes.PersistedValidationData
	WorkerID                *parachaintypes.ValidationCodeHash
	CandidateReceipt        *parachaintypes.CandidateReceipt
	PoV                     parachaintypes.PoV
	ExecutorParams          parachaintypes.ExecutorParams
	PvfExecTimeoutKind      parachaintypes.PvfExecTimeoutKind
	ValidationCode          *parachaintypes.ValidationCode
}

// ValidationResult represents the result coming from the candidate validation subsystem.
// Validation results can be either valid or invalid.
//
// If the result is invalid, store the reason for invalidity.
//
// If the result is valid, store persisted validation data and candidate commitments.
type ValidationResult struct {
	Valid   *Valid
	Invalid *ReasonForInvalidity
}

func (vr ValidationResult) IsValid() bool {
	return vr.Valid != nil
}

type Valid struct {
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

func newWorkerPool() *workerPool {
	return &workerPool{
		workers: make(map[parachaintypes.ValidationCodeHash]*worker),
	}
}

func (v *workerPool) addNewWorker(validationCode parachaintypes.ValidationCode, setupTimeout time.Duration) error {
	workerID := validationCode.Hash()
	if !v.containsWorker(workerID) {
		worker, err := newWorker(validationCode, setupTimeout)
		if err != nil {
			return fmt.Errorf("failed to create a new worker: %w", err)
		}
		v.workers[workerID] = worker
	}
	return nil
}

// handlePrecheckPvF handles the precheck of the parachain validation function. It checks if the worker for the given
// code hash exists. If not, it creates a new worker.
func (v *workerPool) handlePrecheckPvF(data PvFPrepData) error {
	if !v.containsWorker(data.codeHash) {
		err := v.addNewWorker(data.code, data.prepTimeout)
		if err != nil {
			return err
		}
	}
	return nil
}

// executeRequest given a request, the worker pool will get the worker for a given task and submit the request
// to the worker. The worker will execute the request and return the result. If the worker does not exist, a new worker
// will be created and the request will be submitted to the worker.
func (v *workerPool) executeRequest(msg *ValidationTask) (*ValidationResult, error) {
	validationCodeHash := msg.ValidationCode.Hash()

	// create worker if not in pool
	if !v.containsWorker(validationCodeHash) {
		err := v.addNewWorker(*msg.ValidationCode, determineTimeout(msg.PvfExecTimeoutKind))
		if err != nil {
			return nil, err
		}
	}
	worker := v.workers[validationCodeHash]

	logger.Debugf("sending request", validationCodeHash)

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
		timeoutKind:      msg.PvfExecTimeoutKind,
	}
	return worker.executeRequest(workTask)

}

func (v *workerPool) containsWorker(workerID parachaintypes.ValidationCodeHash) bool {
	_, inMap := v.workers[workerID]
	return inMap
}
