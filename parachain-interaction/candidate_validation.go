package parachaininteraction

import (
	"bytes"
	"errors"
	"fmt"
	"sync"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto/ed25519"
	"github.com/ChainSafe/gossamer/lib/keystore"
	"github.com/ChainSafe/gossamer/lib/runtime"
	"github.com/ChainSafe/gossamer/lib/transaction"
	parachaintypes "github.com/ChainSafe/gossamer/parachain-interaction/types"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

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

// TODO: Finish ValidationResult by adding valid and invalid runtimes

// Candidate is valid. The validation process yields these outputs and the persisted validation
// data used to form inputs.
type Valid struct {
	candidateCommitments    candidateCommitments
	PersistedValidationData *parachaintypes.PersistedValidationData
}

// Index returns VDT index
func (Valid) Index() uint {
	return 1
}

type Invalid InvalidCandidate

// Index returns VDT index
func (Invalid) Index() uint {
	return 2
}

// Candidate invalidity details
type InvalidCandidate scale.VaryingDataType

// Set will set a VaryingDataTypeValue using the underlying VaryingDataType
func (ic *InvalidCandidate) Set(val scale.VaryingDataTypeValue) (err error) {
	// cast to VaryingDataType to use VaryingDataType.Set method
	vdt := scale.VaryingDataType(*ic)
	err = vdt.Set(val)
	if err != nil {
		return fmt.Errorf("setting value to varying data type: %w", err)
	}
	// store original ParentVDT with VaryingDataType that has been set
	*ic = InvalidCandidate(vdt)
	return nil
}

// Value will return value from underying VaryingDataType
func (ic *InvalidCandidate) Value() (scale.VaryingDataTypeValue, error) {
	vdt := scale.VaryingDataType(*ic)
	return vdt.Value()
}

// Failed to execute `validate_block`. This includes function panicking.
type ExecutionError string

// Index returns VDT index
func (ExecutionError) Index() uint {
	return 1
}

// Validation outputs check doesn't pass.
type InvalidOutputs struct{}

// Index returns VDT index
func (InvalidOutputs) Index() uint {
	return 2
}

// Execution timeout
type Timeout struct{}

// Index returns VDT index
func (Timeout) Index() uint {
	return 3
}

// Validation input is over the limit.
type ParamsTooLarge uint64

// Index returns VDT index
func (ParamsTooLarge) Index() uint {
	return 4
}

// Code size is over the limit.
type CodeTooLarge uint64

// Index returns VDT index
func (CodeTooLarge) Index() uint {
	return 5
}

// Code does not decompress correctly.
type CodeDecompressionFailure struct{}

// Index returns VDT index
func (CodeDecompressionFailure) Index() uint {
	return 6
}

// PoV does not decompress correctly.
type PoVDecompressionFailure struct{}

// Index returns VDT index
func (PoVDecompressionFailure) Index() uint {
	return 7
}

// Validation function returned invalid data.
type BadReturn struct{}

// Index returns VDT index
func (BadReturn) Index() uint {
	return 8
}

// Invalid relay chain parent.
type BadParent struct{}

// Index returns VDT index
func (BadParent) Index() uint {
	return 9
}

// Para head hash does not match.
type PoVHashMismatch struct{}

// Index returns VDT index
func (PoVHashMismatch) Index() uint {
	return 10
}

// Validation code hash does not match.
type CodeHashMismatch struct{}

// Index returns VDT index
func (CodeHashMismatch) Index() uint {
	return 11
}

// Validation has generated different candidate commitments.
type CommitmentsHashMismatch struct{}

// Index returns VDT index
func (CommitmentsHashMismatch) Index() uint {
	return 12
}

func ValidateFromChainState(runtimeInstance RuntimeInstance, c CandidateReceipt) (*candidateCommitments, *parachaintypes.PersistedValidationData, error) {
	var PersistedValidationData *parachaintypes.PersistedValidationData

	// TODO: There are three validation functions that gets used alternatively.
	// Figure out which one to use when.

	// get persisted validation data
	assumption := parachaintypes.OccupiedCoreAssumption{}
	// TODO: What value should I choose here?
	assumption.Set(Included{})
	// what's the difference between this and last PersistedValidationData?
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
	validationCode, err := runtimeInstance.ParachainHostValidationCode(c.descriptor.ParaID, assumption)
	if err != nil {
		return nil, nil, fmt.Errorf("getting validation code: %w", err)
	}

	validationCodeHash, err := common.Blake2bHash([]byte(*validationCode))
	if err != nil {
		return nil, nil, fmt.Errorf("hashing validation code: %w", err)
	}

	if validationCodeHash != common.Hash(c.descriptor.ValidationCodeHash) {
		return nil, nil, errors.New("validation code hash does not match")
	}

	// check candidate signature
	err = c.descriptor.CheckCollatorSignature()
	if err != nil {
		return nil, nil, fmt.Errorf("verifying collator signature: %w", err)
	}

	// TODO: check if we can decompress validation code and Pov.BlockData

	// TODO:
	// validation_backend
	// implement validate_candidate_with_retry
	// construct pvf from validation code
	// pvf := Pvf{
	// 	Code:     []byte(*validationCode),
	// 	CodeHash: validationCodeHash,
	// }

	// Instead of looking at the rust code, looks at https://spec.polkadot.network/#sect-parachain-runtime instead
	validationParams := ValidationParameters{
		ParentHeadData: PersistedValidationData.ParentHead,
		// TODO: Fill up block data
		BlockData:              types.BlockData{},
		RelayParentNumber:      PersistedValidationData.RelayParentNumber,
		RelayParentStorageRoot: PersistedValidationData.RelayParentStorageRoot,
	}

	// call validate_block runtime api
	//! Defines primitive types for creating or validating a parachain.
	//!
	//! When compiled with standard library support, this crate exports a `wasm`
	//! module that can be used to validate parachain WASM.
	//!
	//! ## Parachain WASM
	//!
	//! Polkadot parachain WASM is in the form of a module which imports a memory
	//! instance and exports a function `validate_block`.
	//!
	//! `validate` accepts as input two `i32` values, representing a pointer/length pair
	//! respectively, that encodes [`ValidationParams`].
	//!
	//! `validate` returns an `u64` which is a pointer to an `u8` array and its length.
	//! The data in the array is expected to be a SCALE encoded [`ValidationResult`].
	//!
	//! ASCII-diagram demonstrating the return data format:
	//!
	//! ```ignore
	//! [pointer][length]
	//!   32bit   32bit
	//!         ^~~ returned pointer & length
	//! ```
	//!
	//! The wasm-api (enabled only when `std` feature is not enabled and `wasm-api` feature is enabled)
	//! provides utilities for setting up a parachain WASM module in Rust.

	// https://spec.polkadot.network/#sect-code-executor
	// to validate parachain block on parachain runtime.
	// Looks at handle_execute_pvf
	// execute pvf and if we can't, throw an error handle_execute_pvf
	// from output of validation_backend, you can create candidate commitments, which will be the item to return

	wasmInstance, err := setupVM(*validationCode)
	if err != nil {
		return nil, nil, fmt.Errorf("setting up VM: %w", err)
	}

	instance := Instance{
		vm: wasmInstance,
		// Allocator: allocator,
		mutex: sync.Mutex{},
	}

	validationResults, err := instance.ValidateBlock(validationParams)
	if err != nil {
		return nil, nil, fmt.Errorf("executing validate_block: %w", err)
	}

	value, err := validationResults.Value()
	if err != nil {
		return nil, nil, fmt.Errorf("getting value of validation results: %w", err)
	}

	// Invalid
	if value.Index() == 2 {
		// deal with the invalid candidate error
	} else if value.Index() != 1 {
		return nil, nil, errors.New("invalid value")
	}

	// Valid
	validityInfo, ok := value.(Valid)
	if !ok {
		return nil, nil, errors.New("value not of type Valid")
	}

	return &validityInfo.candidateCommitments, validityInfo.PersistedValidationData, nil

	// The candidate does not exceed any parameters in the persisted validation data (Definition 227).

	// The signature of the collator is valid.

	// Validate the candidate by executing the parachain Runtime (Section 8.3.1).

}

type ValidationParameters struct {
	ParentHeadData         headData
	BlockData              types.BlockData
	RelayParentNumber      uint32
	RelayParentStorageRoot common.Hash
}

type Pvf struct {
	Code     []byte
	CodeHash common.Hash
}

// TODO::
// func PreCheck()

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
	ParachainHostPersistedValidationData(parachaidID uint32, assumption parachaintypes.OccupiedCoreAssumption) (*parachaintypes.PersistedValidationData, error)
	ParachainHostValidationCode(parachaidID uint32, assumption parachaintypes.OccupiedCoreAssumption) (*parachaintypes.ValidationCode, error)
}
