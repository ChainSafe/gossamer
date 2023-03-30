package parachaininteraction

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto/ed25519"
	"github.com/ChainSafe/gossamer/lib/keystore"
	"github.com/ChainSafe/gossamer/lib/runtime"
	"github.com/ChainSafe/gossamer/lib/transaction"
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

func Validate(runtimeInstance RuntimeInstance, c CandidateReceipt) (*candidateCommitments, *PersistedValidationData, error) {
	var candidateCommitments candidateCommitments
	var PersistedValidationData *PersistedValidationData

	// get persisted validation data
	assumption := OccupiedCoreAssumption{}
	// TODO: What value should I choose here?
	assumption.Set(Included{})
	PersistedValidationData, err := runtimeInstance.ParachainHostPersistedValidationData(c.descriptor.ParaID, assumption)
	if err != nil {
		return nil, nil, fmt.Errorf("getting persisted validation data: %w", err)
	}

	// check that the candidate does not exceed any parameters in the persisted validation data

	// TODO: Get PoV from Candidate
	var pov PoV

	// basic checks

	// check if encoded size of pov is less than max pov size
	buffer := bytes.NewBuffer(nil)
	encoder := scale.NewEncoder(buffer)
	err = encoder.Encode(pov)
	if err != nil {
		return nil, nil, fmt.Errorf("encoding pov: %w", err)
	}
	encoded_pov_size := buffer.Len()
	if encoded_pov_size > int(PersistedValidationData.MaxPovSize) {
		return nil, nil, errors.New("validation input is over the limit")
	}

	// TODO: Implement runtime call to get validation code
	validation_code, err := runtimeInstance.ParachainHostValidationCode(c.descriptor.ParaID, assumption)
	if err != nil {
		return nil, nil, fmt.Errorf("getting validation code: %w", err)
	}

	validationCodeHash := common.NewHash([]byte(*validation_code))

	if validationCodeHash != common.Hash(c.descriptor.ValidationCodeHash) {
		return nil, nil, errors.New("validation code hash does not match")
	}

	// check candidate signature
	err = c.descriptor.CheckCollatorSignature()
	if err != nil {
		return nil, nil, fmt.Errorf("verifying collator signature: %w", err)
	}

	// TODO: check if we can decompress validation code and Pov.BlockData

	// implement validate_candidate_with_retry

	//  CandidateValidationMessage::ValidateFromChainState(
	// - validate_candidate_exhaustive
	//	- implement ParachainHost_persisted_validation_data
	// 		- perform_basic_checks

	// 	CandidateValidationMessage::ValidateFromExhaustive(

	// 	CandidateValidationMessage::PreCheck(

	return &candidateCommitments, PersistedValidationData, nil
	// The candidate does not exceed any parameters in the persisted validation data (Definition 227).

	// The signature of the collator is valid.

	// Validate the candidate by executing the parachain Runtime (Section 8.3.1).

}

// look at node/core/candidate-validation/src/lib.rs

// RuntimeInstance for runtime methods
type RuntimeInstance interface {
	UpdateRuntimeCode([]byte) error
	Stop()
	NodeStorage() runtime.NodeStorage
	NetworkService() runtime.BasicNetwork
	Keystore() *keystore.GlobalKeystore
	Validator() bool
	Exec(function string, data []byte) ([]byte, error)
	SetContextStorage(s runtime.Storage)
	GetCodeHash() common.Hash
	Version() (version runtime.Version)
	Metadata() ([]byte, error)
	BabeConfiguration() (*types.BabeConfiguration, error)
	GrandpaAuthorities() ([]types.Authority, error)
	ValidateTransaction(e types.Extrinsic) (*transaction.Validity, error)
	InitializeBlock(header *types.Header) error
	InherentExtrinsics(data []byte) ([]byte, error)
	ApplyExtrinsic(data types.Extrinsic) ([]byte, error)
	FinalizeBlock() (*types.Header, error)
	ExecuteBlock(block *types.Block) ([]byte, error)
	DecodeSessionKeys(enc []byte) ([]byte, error)
	PaymentQueryInfo(ext []byte) (*types.RuntimeDispatchInfo, error)
	CheckInherents()
	BabeGenerateKeyOwnershipProof(slot uint64, authorityID [32]byte) (
		types.OpaqueKeyOwnershipProof, error)
	BabeSubmitReportEquivocationUnsignedExtrinsic(
		equivocationProof types.BabeEquivocationProof,
		keyOwnershipProof types.OpaqueKeyOwnershipProof,
	) error
	RandomSeed()
	OffchainWorker()
	GenerateSessionKeys()
	GrandpaGenerateKeyOwnershipProof(authSetID uint64, authorityID ed25519.PublicKeyBytes) (
		types.GrandpaOpaqueKeyOwnershipProof, error)
	GrandpaSubmitReportEquivocationUnsignedExtrinsic(
		equivocationProof types.GrandpaEquivocationProof, keyOwnershipProof types.GrandpaOpaqueKeyOwnershipProof,
	) error
	ParachainHostPersistedValidationData(parachaidID uint32, assumption OccupiedCoreAssumption) (*PersistedValidationData, error)
	ParachainHostValidationCode(parachaidID uint32, assumption OccupiedCoreAssumption) (*ValidationCode, error)
}
