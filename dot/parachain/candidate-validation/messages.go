// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package candidatevalidation

import (
	"fmt"

	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

type PersistedValidationData struct {
	ParentHead             parachaintypes.HeadData
	RelayParentNumber      parachaintypes.BlockNumber
	RelayParentStorageRoot common.Hash
	MaxPovSize             uint32
}

type ValidateFromChainState struct {
	CandidateReceipt parachaintypes.CandidateReceipt
	Pov              parachaintypes.PoV
	ExecutorParams   parachaintypes.ExecutorParams
	ExecKind         parachaintypes.PvfExecTimeoutKind
	Sender           chan ValidationResultMessage
}

type ValidateFromExhaustive struct {
	PersistedValidationData PersistedValidationData
	ValidationCode          parachaintypes.ValidationCode
	CandidateReceipt        parachaintypes.CandidateReceipt
	Pov                     parachaintypes.PoV
	ExecutorParams          parachaintypes.ExecutorParams
	ExecKind                parachaintypes.PvfExecTimeoutKind
	Sender                  chan ValidationResultMessage
}

type ValidationResultMessage struct {
	ValidationResult ValidationResult
	ValidationFailed string
}

// ValidationResult represents the result of the validation of the candidate
type ValidationResult scale.VaryingDataType

// NewValidationResult returns a new ValidationResult varying data type
func NewValidationResult() ValidationResult {
	vdt := scale.MustNewVaryingDataType(Valid{}, Invalid{})
	return ValidationResult(vdt)
}

// Valid candidate is valid, The validation process yields these outputs and the persisted
// validation data used to form inputs.
type Valid struct {
	CandidateCommitments    parachaintypes.CandidateCommitments
	PersistedValidationData PersistedValidationData
}

// Index returns the index of varying data type
func (Valid) Index() uint {
	return 0
}

// Invalid candidate is invalid.
type Invalid struct {
	InvalidCandidate InvalidCandidate
}

// Index returns the index of varying data type
func (Invalid) Index() uint {
	return 1
}

// InvalidCandidate candidate invalidity details
type InvalidCandidate scale.VaryingDataType

// NewInvalidCandidate returns a new InvalidCandidate varying data type
func NewInvalidCandidate() InvalidCandidate {
	vdt := scale.MustNewVaryingDataType(ExecutionError{}, InvalidOutputs{}, Timeout{}, ParamsTooLarge{},
		CodeTooLarge{}, PoVDecompressionFailure{}, BadReturn{}, BadParent{}, PoVHashMismatch{}, BadSignature{},
		ParaHeadHashMismatch{}, CodeHashMismatch{}, CommitmentHashMismatch{})
	return InvalidCandidate(vdt)
}

// New will enable scale to create new instance when needed
func (InvalidCandidate) New() InvalidCandidate {
	return NewInvalidCandidate()
}

// Set will set a value using the underlying  varying data type
func (i *InvalidCandidate) Set(val scale.VaryingDataTypeValue) (err error) {
	vdt := scale.VaryingDataType(*i)
	err = vdt.Set(val)
	if err != nil {
		return fmt.Errorf("setting value to varying data type: %w", err)
	}

	*i = InvalidCandidate(vdt)
	return nil
}

// Value returns the value from the underlying varying data type
func (i *InvalidCandidate) Value() (scale.VaryingDataTypeValue, error) {
	vdt := scale.VaryingDataType(*i)
	return vdt.Value()
}

// ExecutionError failed to execute `valid_black`. This includes function panicking.
type ExecutionError struct {
	Error string
}

// Index returns the index of varying data type
func (ExecutionError) Index() uint {
	return 0
}

// InvalidOutputs validation outputs check doesn't pass
type InvalidOutputs struct{}

// Index returns the index of varying data type
func (InvalidOutputs) Index() uint {
	return 1
}

// Timeout Execution timeout
type Timeout struct{}

// Index returns the index of varying data type
func (Timeout) Index() uint {
	return 2
}

// ParamsTooLarge Validation input is over the limit
type ParamsTooLarge struct {
	ParamsSize uint64
}

// Index returns the index of varying data type
func (ParamsTooLarge) Index() uint {
	return 3
}

// CodeTooLarge code size is over the limit
type CodeTooLarge struct {
	CodeSize uint64
}

// Index returns the index of varying data type
func (CodeTooLarge) Index() uint {
	return 4
}

// PoVDecompressionFailure PoV does not decompress correctly
type PoVDecompressionFailure struct{}

// Index returns the index of varying data type
func (PoVDecompressionFailure) Index() uint {
	return 5
}

// BadReturn Validation function returned invalid data
type BadReturn struct{}

// Index returns the index of varying data type
func (BadReturn) Index() uint {
	return 6
}

// BadParent invalid relay chain parent
type BadParent struct{}

// Index returns the index of varying data type
func (BadParent) Index() uint {
	return 7
}

// PoVHashMismatch PoV hash mismatch
type PoVHashMismatch struct{}

// Index returns the index of varying data type
func (PoVHashMismatch) Index() uint {
	return 8
}

// BadSignature bod collator signature
type BadSignature struct{}

// Index returns the index of varying data type
func (BadSignature) Index() uint {
	return 9
}

// ParaHeadHashMismatch para head hash does not match
type ParaHeadHashMismatch struct{}

// Index returns the index of varying data type
func (ParaHeadHashMismatch) Index() uint {
	return 10
}

// CodeHashMismatch validation code hash does not match
type CodeHashMismatch struct{}

// Index returns the index of varying data type
func (CodeHashMismatch) Index() uint {
	return 11
}

// CommitmentHashMismatch validation has generated different candidate commitments
type CommitmentHashMismatch struct{}

// Index returns the index of varying data type
func (CommitmentHashMismatch) Index() uint {
	return 12
}

// PreCheck try to complie the given validation code and return the result
// The validation code is specified by the hash and will be queried from the runtime API at
// the given relay-parent.
type PreCheck struct {
	RelayParent        common.Hash
	ValidationCodeHash parachaintypes.ValidationCodeHash
	PreCheckOutcome    chan PreCheckOutcome
}

// PreCheckOutcome represents the outcome of the candidate-validation pre-check request
type PreCheckOutcome scale.VaryingDataType

// NewPreCheckOutcome returns a new PreCheckOutcome varying data type
func NewPreCheckOutcome() PreCheckOutcome {
	vdt := scale.MustNewVaryingDataType(PreCheckValid{}, PreCheckInvalid{}, PreCheckFailed{})
	return PreCheckOutcome(vdt)
}

// New will enable scale to create new instance when needed
func (PreCheckOutcome) New() PreCheckOutcome {
	return NewPreCheckOutcome()
}

// Set will set a value using the underlying  varying data type
func (p *PreCheckOutcome) Set(val scale.VaryingDataTypeValue) (err error) {
	vdt := scale.VaryingDataType(*p)
	err = vdt.Set(val)
	if err != nil {
		return fmt.Errorf("setting value to varying data type: %w", err)
	}

	*p = PreCheckOutcome(vdt)
	return nil
}

// Value returns the value from the underlying varying data type
func (p *PreCheckOutcome) Value() (scale.VaryingDataTypeValue, error) {
	vdt := scale.VaryingDataType(*p)
	return vdt.Value()
}

// PreCheckValid The PVF has been compiled successfully within the given constraints.
type PreCheckValid struct{}

// Index returns the index of varying data type
func (PreCheckValid) Index() uint {
	return 0
}

// PreCheckInvalid  The PVF could not be compiled. This variant is used when the candidate-validation subsystem
// can be sure that the PVF is invalid. To give a couple of examples: a PVF that cannot be
// decompressed or that does not represent a structurally valid WebAssembly file.
type PreCheckInvalid struct{}

// Index returns the index of varying data type
func (PreCheckInvalid) Index() uint {
	return 1
}

// PreCheckFailed This variant is used when the PVF cannot be compiled but for other reasons that are not
// included into [`PreCheckOutcome::Invalid`]. This variant can indicate that the PVF in
// question is invalid, however it is not necessary that PVF that received this judgement
// is invalid.
//
// For example, if during compilation the preparation worker was killed we cannot be sure why
// it happened: because the PVF was malicious made the worker to use too much memory or its
// because the host machine is under severe memory pressure and it decided to kill the worker.
type PreCheckFailed struct{}

// Index returns the index of varying data type
func (PreCheckFailed) Index() uint {
	return 2
}
