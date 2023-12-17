// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package parachaintypes

import (
	"fmt"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

type SubSystemName string

const (
	CandidateBacking  SubSystemName = "CandidateBacking"
	CollationProtocol SubSystemName = "CollationProtocol"
	AvailabilityStore SubSystemName = "AvailabilityStore"
)

var (
	_ ProvisionerMessage           = (*PMProvisionableData)(nil)
	_ ProvisionableData            = (*PDBackedCandidate)(nil)
	_ ProvisionableData            = (*PDMisbehaviorReport)(nil)
	_ StatementDistributionMessage = (*SDMBacked)(nil)
	_ CollatorProtocolMessage      = (*CPMBacked)(nil)
	_ ProspectiveParachainsMessage = (*PPMCandidateBacked)(nil)
	_ ProspectiveParachainsMessage = (*PPMIntroduceCandidate)(nil)
	_ ProspectiveParachainsMessage = (*PPMCandidateSeconded)(nil)
	_ RuntimeApiMessage            = (*RAMRequest)(nil)
	_ RuntimeApiRequest            = (*RARValidationCodeByHash)(nil)
	_ CandidateValidationMessage   = (*CVMValidateFromExhaustive)(nil)
	_ AvailabilityStoreMessage     = (*ASMStoreAvailableData)(nil)
	_ Misbehaviour                 = (*IssuedAndValidity)(nil)
	_ ValidityDoubleVote           = (*IssuedAndValidity)(nil)
	_ Misbehaviour                 = (*MultipleCandidates)(nil)
	_ Misbehaviour                 = (*UnauthorizedStatement)(nil)
	_ Misbehaviour                 = (*OnSeconded)(nil)
	_ DoubleSign                   = (*OnSeconded)(nil)
	_ Misbehaviour                 = (*OnValidity)(nil)
	_ DoubleSign                   = (*OnValidity)(nil)
)

// OverseerFuncRes is a result of an overseer function
type OverseerFuncRes[T any] struct {
	Err  error
	Data T
}

// ProvisionerMessage is a message to the Provisioner.
type ProvisionerMessage interface {
	IsProvisionerMessage()
}

// PMProvisionableData is a provisioner message.
// This data should become part of a relay chain block.
type PMProvisionableData struct {
	RelayParent       common.Hash
	ProvisionableData ProvisionableData
}

func (PMProvisionableData) IsProvisionerMessage() {}

// ProvisionableData becomes intrinsics or extrinsics which should be included in a future relay chain block.
type ProvisionableData interface {
	IsProvisionableData()
}

// PDBackedCandidate is a provisionable data.
// The Candidate Backing subsystem believes that this candidate is valid, pending availability.
type PDBackedCandidate CandidateReceipt

func (PDBackedCandidate) IsProvisionableData() {}

// PDMisbehaviorReport represents self-contained proofs of validator misbehaviour.
type PDMisbehaviorReport struct {
	ValidatorIndex ValidatorIndex
	Misbehaviour   Misbehaviour
}

func (PDMisbehaviorReport) IsProvisionableData() {}

// Misbehaviour is intended to represent different kinds of misbehaviour along with supporting proofs.
type Misbehaviour interface {
	IsMisbehaviour()
}

// ValidityDoubleVote misbehaviour: voting more than one way on candidate validity.
// Since there are three possible ways to vote, a double vote is possible in
// three possible combinations (unordered)
type ValidityDoubleVote interface {
	Misbehaviour
	IsValidityDoubleVote()
}

// IssuedAndValidity represents an implicit vote by issuing and explicit voting for validity.
type IssuedAndValidity struct {
	CommittedCandidateReceiptAndSign CommittedCandidateReceiptAndSign
	CandidateHashAndSign             struct {
		CandidateHash CandidateHash
		Signature     ValidatorSignature
	}
}

func (IssuedAndValidity) IsMisbehaviour()       {}
func (IssuedAndValidity) IsValidityDoubleVote() {}

// CommittedCandidateReceiptAndSign combines a committed candidate receipt and its associated signature.
type CommittedCandidateReceiptAndSign struct {
	CommittedCandidateReceipt CommittedCandidateReceipt
	Signature                 ValidatorSignature
}

// MultipleCandidates misbehaviour: declaring multiple candidates.
type MultipleCandidates struct {
	First  CommittedCandidateReceiptAndSign
	Second CommittedCandidateReceiptAndSign
}

func (MultipleCandidates) IsMisbehaviour() {}

// SignedStatement represents signed statements about candidates.
type SignedStatement struct {
	Statement StatementVDT       `scale:"1"`
	Signature ValidatorSignature `scale:"2"`
	Sender    ValidatorIndex     `scale:"3"`
}

// UnauthorizedStatement misbehaviour: submitted statement for wrong group.
type UnauthorizedStatement struct {
	// A signed statement which was submitted without proper authority.
	Statement SignedStatement
}

func (UnauthorizedStatement) IsMisbehaviour() {}

// DoubleSign misbehaviour: multiple signatures on same statement.
type DoubleSign interface {
	Misbehaviour
	IsDoubleSign()
}

// OnSeconded represents a double sign on a candidate.
type OnSeconded struct {
	Candidate CommittedCandidateReceipt
	Sign1     ValidatorSignature
	Sign2     ValidatorSignature
}

func (OnSeconded) IsMisbehaviour() {}
func (OnSeconded) IsDoubleSign()   {}

// OnValidity represents a double sign on validity.
type OnValidity struct {
	CandidateHash CandidateHash
	Sign1         ValidatorSignature
	Sign2         ValidatorSignature
}

func (OnValidity) IsMisbehaviour() {}
func (OnValidity) IsDoubleSign()   {}

// StatementDistributionMessage is a message to the Statement Distribution subsystem.
type StatementDistributionMessage interface {
	IsStatementDistributionMessage()
}

// SDMBacked is a statement distribution message.
// it represents a message indicating that a candidate has received sufficient
// validity votes from the backing group. If backed as a result of a local statement,
// it must be preceded by a `Share` message for that statement to ensure awareness of
// full candidates before the `Backed` notification, even in groups of size 1.
type SDMBacked CandidateHash

func (SDMBacked) IsStatementDistributionMessage() {}

// CollatorProtocolMessage represents messages that are received by the Collator Protocol subsystem.
type CollatorProtocolMessage interface {
	IsCollatorProtocolMessage()
}

// CPMBacked is a collator protocol message.
// The candidate received enough validity votes from the backing group.
type CPMBacked struct {
	// Candidate's para id.
	ParaID ParaID
	// Hash of the para head generated by candidate.
	ParaHead common.Hash
}

func (CPMBacked) IsCollatorProtocolMessage() {}

// ProspectiveParachainsMessage represents messages that are sent to the Prospective Parachains subsystem.
type ProspectiveParachainsMessage interface {
	IsProspectiveParachainsMessage()
}

// PPMCandidateBacked is a prospective parachains message.
// it informs the Prospective Parachains Subsystem that
// a previously introduced candidate has been successfully backed.
type PPMCandidateBacked struct {
	ParaID        ParaID
	CandidateHash CandidateHash
}

func (PPMCandidateBacked) IsProspectiveParachainsMessage() {}

// PPMIntroduceCandidate is a prospective parachains message.
// it inform the Prospective Parachains Subsystem about a new candidate.
type PPMIntroduceCandidate struct {
	IntroduceCandidateRequest IntroduceCandidateRequest
	Ch                        chan error
}

func (PPMIntroduceCandidate) IsProspectiveParachainsMessage() {}

// PPMCandidateSeconded is a prospective parachains message.
// it informs the Prospective Parachains Subsystem that a previously introduced candidate
// has been seconded. This requires that the candidate was successfully introduced in
// the past.
type PPMCandidateSeconded struct {
	ParaID        ParaID
	CandidateHash CandidateHash
}

func (PPMCandidateSeconded) IsProspectiveParachainsMessage() {}

// IntroduceCandidateRequest is a request to introduce a candidate into the Prospective Parachains Subsystem.
type IntroduceCandidateRequest struct {
	// The para-id of the candidate.
	CandidateParaID ParaID
	// The candidate receipt itself.
	CommittedCandidateReceipt CommittedCandidateReceipt
	// The persisted validation data of the candidate.
	PersistedValidationData PersistedValidationData
}

// RuntimeApiMessage is a message to the Runtime API subsystem.
type RuntimeApiMessage interface {
	IsRuntimeApiMessage()
}

type RAMRequest struct {
	RelayParent common.Hash
	// Make a request of the runtime API against the post-state of the given relay-parent.
	RuntimeApiRequest RuntimeApiRequest
}

func (RAMRequest) IsRuntimeApiMessage() {}

type RuntimeApiRequest interface {
	IsRuntimeApiRequest()
}

// RARValidationCodeByHash retrieves validation code by its hash. It can return
// past, current, or future code as long as state is available.
type RARValidationCodeByHash struct {
	ValidationCodeHash ValidationCodeHash
	Ch                 chan OverseerFuncRes[ValidationCode]
}

func (RARValidationCodeByHash) IsRuntimeApiRequest() {}

// CandidateValidationMessage represents messages received by the Validation subsystem.
// Validation requests should return an error only in case of internal errors.
type CandidateValidationMessage interface {
	IsCandidateValidationMessage()
}

// CVMValidateFromExhaustive performs full validation of a candidate with provided parameters,
// including `PersistedValidationData` and `ValidationCode`. It doesn't involve acceptance
// criteria checking and is typically used when the candidate's validity is established
// through prior relay-chain checks.
type CVMValidateFromExhaustive struct {
	PersistedValidationData PersistedValidationData
	ValidationCode          ValidationCode
	CandidateReceipt        CandidateReceipt
	PoV                     PoV
	ExecutorParams          ExecutorParams
	PvfExecTimeoutKind      PvfExecTimeoutKind
	Ch                      chan OverseerFuncRes[ValidationResult]
}

func (CVMValidateFromExhaustive) IsCandidateValidationMessage() {}

// ValidationResult represents the result coming from the candidate validation subsystem.
type ValidationResult struct {
	IsValid                 bool
	CandidateCommitments    CandidateCommitments
	PersistedValidationData PersistedValidationData
	Err                     error
}

// AvailabilityStoreMessage represents messages received by the Availability Store subsystem.
type AvailabilityStoreMessage interface {
	IsAvailabilityStoreMessage()
}

// ASMStoreAvailableData computes and checks the erasure root of `AvailableData`
// before storing its chunks in the AV store.
type ASMStoreAvailableData struct {
	// A hash of the candidate this `ASMStoreAvailableData` belongs to.
	CandidateHash CandidateHash
	// The number of validators in the session.
	NumValidators uint32
	// The `AvailableData` itself.
	AvailableData AvailableData
	// Erasure root we expect to get after chunking.
	ExpectedErasureRoot common.Hash
	// channel to send result to.
	Ch chan error
}

func (ASMStoreAvailableData) IsAvailabilityStoreMessage() {}

// AvailableData represents the data that is kept available for each candidate included in the relay chain.
type AvailableData struct {
	// The Proof-of-Validation (PoV) of the candidate
	PoV PoV `scale:"1"`

	// The persisted validation data needed for approval checks
	ValidationData PersistedValidationData `scale:"2"`
}

// ExecutorParams represents the abstract semantics of an execution environment and should remain
// as abstract as possible. There are no mandatory parameters defined at the moment, and if any
// are introduced in the future, they must be clearly documented as mandatory.
type ExecutorParams scale.VaryingDataTypeSlice

// NewExecutorParams returns a new ExecutorParams varying data type slice
func NewExecutorParams() ExecutorParams {
	vdt := NewExecutorParam()
	vdts := scale.NewVaryingDataTypeSlice(scale.VaryingDataType(vdt))
	return ExecutorParams(vdts)
}

// Add takes variadic parameter values to add VaryingDataTypeValue
func (e *ExecutorParams) Add(val scale.VaryingDataTypeValue) (err error) {
	slice := scale.VaryingDataTypeSlice(*e)
	err = slice.Add(val)
	if err != nil {
		return fmt.Errorf("adding value to varying data type slice: %w", err)
	}

	*e = ExecutorParams(slice)
	return nil
}

// ExecutorParam represents the various parameters for modifying the semantics of the execution environment.
type ExecutorParam scale.VaryingDataType

// NewExecutorParam returns a new ExecutorParam varying data type
func NewExecutorParam() ExecutorParam {
	vdt := scale.MustNewVaryingDataType(
		MaxMemoryPages(0),
		StackLogicalMax(0),
		StackNativeMax(0),
		PrecheckingMaxMemory(0),
		PvfPrepTimeout{},
		PvfExecTimeout{},
		WasmExtBulkMemory{},
	)
	return ExecutorParam(vdt)
}

// New will enable scale to create new instance when needed
func (ExecutorParam) New() ExecutorParam {
	return NewExecutorParam()
}

// Set will set a value using the underlying  varying data type
func (s *ExecutorParam) Set(val scale.VaryingDataTypeValue) (err error) {
	vdt := scale.VaryingDataType(*s)
	err = vdt.Set(val)
	if err != nil {
		return fmt.Errorf("setting value to varying data type: %w", err)
	}

	*s = ExecutorParam(vdt)
	return nil
}

// Value returns the value from the underlying varying data type
func (s *ExecutorParam) Value() (scale.VaryingDataTypeValue, error) {
	vdt := scale.VaryingDataType(*s)
	return vdt.Value()
}

// MaxMemoryPages represents the maximum number of memory pages (64KiB bytes per page) that the executor can allocate.
type MaxMemoryPages uint32

// Index returns the index of varying data type
func (MaxMemoryPages) Index() uint {
	return 1
}

// StackLogicalMax defines the limit for the logical stack size in Wasm (maximum number of Wasm values on the stack).
type StackLogicalMax uint32

// Index returns the index of varying data type
func (StackLogicalMax) Index() uint {
	return 2
}

// StackNativeMax represents the limit of the executor's machine stack size in bytes.
type StackNativeMax uint32

// Index returns the index of varying data type
func (StackNativeMax) Index() uint {
	return 3
}

// PrecheckingMaxMemory represents the maximum memory allowance for the preparation worker during pre-checking,
// measured in bytes.
type PrecheckingMaxMemory uint64

// Index returns the index of varying data type
func (PrecheckingMaxMemory) Index() uint {
	return 4
}

// PvfPrepTimeout defines the timeouts for PVF preparation in milliseconds.
type PvfPrepTimeout struct {
	PvfPrepTimeoutKind PvfPrepTimeoutKind `scale:"1"`
	Millisec           uint64             `scale:"2"`
}

// Index returns the index of varying data type
func (PvfPrepTimeout) Index() uint {
	return 5
}

// PvfExecTimeout represents the timeouts for PVF execution in milliseconds.
type PvfExecTimeout struct {
	PvfExecTimeoutKind PvfExecTimeoutKind `scale:"1"`
	Millisec           uint64             `scale:"2"`
}

// Index returns the index of varying data type
func (PvfExecTimeout) Index() uint {
	return 6
}

// WasmExtBulkMemory enables the WASM bulk memory proposal.
type WasmExtBulkMemory struct{}

// Index returns the index of varying data type
func (WasmExtBulkMemory) Index() uint {
	return 7
}

// PvfPrepTimeoutKind is an enumeration representing the type discriminator for PVF preparation timeouts
type PvfPrepTimeoutKind scale.VaryingDataType

// NewPvfPrepTimeoutKind returns a new PvfPrepTimeoutKind varying data type
func NewPvfPrepTimeoutKind() PvfPrepTimeoutKind {
	vdt := scale.MustNewVaryingDataType(Precheck{}, Lenient{})
	return PvfPrepTimeoutKind(vdt)
}

// New will enable scale to create new instance when needed
func (PvfPrepTimeoutKind) New() PvfPrepTimeoutKind {
	return NewPvfPrepTimeoutKind()
}

// Set will set a value using the underlying  varying data type
func (p *PvfPrepTimeoutKind) Set(val scale.VaryingDataTypeValue) (err error) {
	vdt := scale.VaryingDataType(*p)
	err = vdt.Set(val)
	if err != nil {
		return fmt.Errorf("setting value to varying data type: %w", err)
	}

	*p = PvfPrepTimeoutKind(vdt)
	return nil
}

// Value returns the value from the underlying varying data type
func (s *PvfPrepTimeoutKind) Value() (scale.VaryingDataTypeValue, error) {
	vdt := scale.VaryingDataType(*s)
	return vdt.Value()
}

// Precheck defines the time period for prechecking requests. After this duration,
// an unresponsive preparation worker is considered and will be terminated.
type Precheck struct{}

// Index returns the index of varying data type
func (Precheck) Index() uint {
	return 0
}

// Lenient refers to the time period for execution and heads-up requests. It is the duration
// after which the preparation worker is deemed unresponsive and terminated. This timeout
// is more forgiving than the prechecking timeout to avoid honest validators timing out on valid PVFs.
type Lenient struct{}

// Index returns the index of varying data type
func (Lenient) Index() uint {
	return 1
}

// PvfExecTimeoutKind is an enumeration representing the type discriminator for PVF execution timeouts
type PvfExecTimeoutKind scale.VaryingDataType

// NewPvfExecTimeoutKind returns a new PvfExecTimeoutKind varying data type
func NewPvfExecTimeoutKind() PvfExecTimeoutKind {
	vdt := scale.MustNewVaryingDataType(Backing{}, Approval{})
	return PvfExecTimeoutKind(vdt)
}

// New will enable scale to create new instance when needed
func (PvfExecTimeoutKind) New() PvfExecTimeoutKind {
	return NewPvfExecTimeoutKind()
}

// Set will set a value using the underlying  varying data type
func (s *PvfExecTimeoutKind) Set(val scale.VaryingDataTypeValue) (err error) {
	vdt := scale.VaryingDataType(*s)
	err = vdt.Set(val)
	if err != nil {
		return fmt.Errorf("setting value to varying data type: %w", err)
	}

	*s = PvfExecTimeoutKind(vdt)
	return nil
}

// Value returns the value from the underlying varying data type
func (s *PvfExecTimeoutKind) Value() (scale.VaryingDataTypeValue, error) {
	vdt := scale.VaryingDataType(*s)
	return vdt.Value()
}

// Backing represents the amount of time to spend on execution during backing.
type Backing struct{}

// Index returns the index of varying data type
func (Backing) Index() uint {
	return 0
}

// Approval represents the amount of time to spend on execution during approval or disputes.
// This timeout should be much longer than the backing execution timeout to ensure that,
// in the absence of extremely large disparities between hardware, blocks that pass
// backing are considered executable by approval checkers or dispute participants.
type Approval struct{}

// Index returns the index of varying data type
func (Approval) Index() uint {
	return 1
}
