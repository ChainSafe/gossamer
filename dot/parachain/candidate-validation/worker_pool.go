package candidatevalidation

import (
	"fmt"

	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
)

type workerPool struct {

	// todo, make sure other functions work with paraID
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

func newValidationWorkerPool() *workerPool {
	return &workerPool{
		workers: make(map[parachaintypes.ValidationCodeHash]*worker),
	}
}

func (v *workerPool) newValidationWorker(validationCode parachaintypes.ValidationCode) (*worker, error) {

	worker, err := newWorker(validationCode)
	if err != nil {
		return nil, fmt.Errorf("failed to create a new worker: %w", err)
	}

	v.workers[worker.workerID] = worker

	return worker, nil
}

// submitRequest given a request, the worker pool will get the worker for a given workerID
// a channel in returned that the response will be dispatch on
func (v *workerPool) submitRequest(workerID parachaintypes.ValidationCodeHash,
	request *workerTask) (*ValidationResult, error) {
	logger.Debugf("pool submit request workerID %x", workerID)

	syncWorker, inMap := v.workers[workerID]
	if inMap {
		if syncWorker == nil {
			panic("sync worker should not be nil")
		}
		logger.Debugf("sending request", workerID)
		return syncWorker.executeRequest(request)
	}
	return nil, fmt.Errorf("worker not found")
}

func (v *workerPool) containsWorker(workerID parachaintypes.ValidationCodeHash) bool {
	_, inMap := v.workers[workerID]
	return inMap
}
