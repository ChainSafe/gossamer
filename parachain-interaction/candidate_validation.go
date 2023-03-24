package parachaininteraction

import (
	"fmt"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

// pub enum ValidationProtocol {
// 	/// Bitfield distribution messages
// 	#[codec(index = 1)]
// 	#[from]
// 	BitfieldDistribution(BitfieldDistributionMessage),
// 	/// Statement distribution messages
// 	#[codec(index = 3)]
// 	#[from]
// 	StatementDistribution(StatementDistributionMessage),
// 	/// Approval distribution messages
// 	#[codec(index = 4)]
// 	#[from]
// 	ApprovalDistribution(ApprovalDistributionMessage),
// }

// This subsystem groups the requests it handles in two categories: candidate validation and PVF pre-checking.

// The first category can be further subdivided in two request types: one which draws out validation data from the state, and another which accepts all validation data exhaustively. Validation returns three possible outcomes on the response channel: the candidate is valid, the candidate is invalid, or an internal error occurred.

// Parachain candidates are validated against their validation function: A piece of Wasm code that describes the state-transition of the parachain. Validation function execution is not metered. This means that an execution which is an infinite loop or simply takes too long must be forcibly exited by some other means. For this reason, we recommend dispatching candidate validation to be done on subprocesses which can be killed if they time-out.

// Received candidates submitted by collators and must have its validity verified by the assigned Polkadot validators. For each candidate to be valid, the validator must successfully verify the following conditions in the following order:

//     The candidate does not exceed any parameters in the persisted validation data (Definition 227).

//     The signature of the collator is valid.

//     Validate the candidate by executing the parachain Runtime (Section 8.3.1).

// If all steps are valid, the Polkadot validator must create the necessary candidate commitments (Definition 107) and submit the appropriate statement for each candidate (Section 8.2.1).

// pub enum ValidationResult {
// 	/// Candidate is valid. The validation process yields these outputs and the persisted validation
// 	/// data used to form inputs.
// 	Valid(CandidateCommitments, PersistedValidationData),
// 	/// Candidate is invalid.
// 	Invalid(InvalidCandidate),
// }

type ValidationResult scale.VaryingDataType

// Set will set a VaryingDataTypeValue using the underlying VaryingDataType
func (vr *ValidationResult) Set(val scale.VaryingDataTypeValue) (err error) {
	// cast to VaryingDataType to use VaryingDataType.Set method
	vdt := scale.VaryingDataType(*vr)
	err = vdt.Set(val)
	if err != nil {
		return fmt.Errorf("setting value to varying data type: %w", err)
	}
	// store original ParentVDT with VaryingDataType that has been set
	*vr = ValidationResult(vdt)
	return nil
}

// Value will return value from underying VaryingDataType
func (vr *ValidationResult) Value() (scale.VaryingDataTypeValue, error) {
	vdt := scale.VaryingDataType(*vr)
	return vdt.Value()
}

// /// Candidate invalidity details
// #[derive(Debug)]
// pub enum InvalidCandidate {
// 	/// Failed to execute `validate_block`. This includes function panicking.
// 	ExecutionError(String),
// 	/// Validation outputs check doesn't pass.
// 	InvalidOutputs,
// 	/// Execution timeout.
// 	Timeout,
// 	/// Validation input is over the limit.
// 	ParamsTooLarge(u64),
// 	/// Code size is over the limit.
// 	CodeTooLarge(u64),
// 	/// Code does not decompress correctly.
// 	CodeDecompressionFailure,
// 	/// PoV does not decompress correctly.
// 	PoVDecompressionFailure,
// 	/// Validation function returned invalid data.
// 	BadReturn,
// 	/// Invalid relay chain parent.
// 	BadParent,
// 	/// POV hash does not match.
// 	PoVHashMismatch,
// 	/// Bad collator signature.
// 	BadSignature,
// 	/// Para head hash does not match.
// 	ParaHeadHashMismatch,
// 	/// Validation code hash does not match.
// 	CodeHashMismatch,
// 	/// Validation has generated different candidate commitments.
// 	CommitmentsHashMismatch,
// }

// / The validation data provides information about how to create the inputs for validation of a candidate.
// / This information is derived from the chain state and will vary from para to para, although some
// / fields may be the same for every para.
// /
// / Since this data is used to form inputs to the validation function, it needs to be persisted by the
// / availability system to avoid dependence on availability of the relay-chain state.
// /
// / Furthermore, the validation data acts as a way to authorize the additional data the collator needs
// / to pass to the validation function. For example, the validation function can check whether the incoming
// / messages (e.g. downward messages) were actually sent by using the data provided in the validation data
// / using so called MQC heads.
// /
// / Since the commitments of the validation function are checked by the relay-chain, secondary checkers
// / can rely on the invariant that the relay-chain only includes para-blocks for which these checks have
// / already been done. As such, there is no need for the validation data used to inform validators and
// / collators about the checks the relay-chain will perform to be persisted by the availability system.
// /
// / The `PersistedValidationData` should be relatively lightweight primarily because it is constructed
// / during inclusion for each candidate and therefore lies on the critical path of inclusion.
type PersistedValidationData struct {
	ParentHead             headData
	RelayParentNumber      uint32
	RelayParentStorageRoot types.Header
	MaxPovSize             uint32
}

func Validate(c CandidateReceipt) (candidateCommitments, PersistedValidationData, error) {
	var candidateCommitments candidateCommitments
	var PersistedValidationData PersistedValidationData
	//  CandidateValidationMessage::ValidateFromChainState(

	// 	CandidateValidationMessage::ValidateFromExhaustive(

	// 	CandidateValidationMessage::PreCheck(

	return candidateCommitments, PersistedValidationData, nil
}

// look at node/core/candidate-validation/src/lib.rs
