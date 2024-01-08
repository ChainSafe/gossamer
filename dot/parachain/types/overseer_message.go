// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package parachaintypes

import "github.com/ChainSafe/gossamer/lib/common"

var (
	_ ProvisionerMessage           = (*ProvisionerMessageProvisionableData)(nil)
	_ ProvisionableData            = (*ProvisionableDataBackedCandidate)(nil)
	_ ProvisionableData            = (*ProvisionableDataMisbehaviorReport)(nil)
	_ Misbehaviour                 = (*MultipleCandidates)(nil)
	_ Misbehaviour                 = (*UnauthorizedStatement)(nil)
	_ Misbehaviour                 = (*IssuedAndValidity)(nil)
	_ Misbehaviour                 = (*OnSeconded)(nil)
	_ Misbehaviour                 = (*OnValidity)(nil)
	_ DoubleSign                   = (*OnSeconded)(nil)
	_ DoubleSign                   = (*OnValidity)(nil)
	_ StatementDistributionMessage = (*StatementDistributionMessageBacked)(nil)
	_ CollatorProtocolMessage      = (*CollatorProtocolMessageBacked)(nil)
	_ ProspectiveParachainsMessage = (*ProspectiveParachainsMessageCandidateBacked)(nil)
	_ ProspectiveParachainsMessage = (*ProspectiveParachainsMessageIntroduceCandidate)(nil)
	_ ProspectiveParachainsMessage = (*ProspectiveParachainsMessageCandidateSeconded)(nil)
	_ ProspectiveParachainsMessage = (*ProspectiveParachainsMessageGetHypotheticalFrontier)(nil)
	_ HypotheticalCandidate        = (*HypotheticalCandidateIncomplete)(nil)
	_ HypotheticalCandidate        = (*HypotheticalCandidateComplete)(nil)
	_ RuntimeApiMessage            = (*RuntimeApiMessageRequest)(nil)
	_ RuntimeApiRequest            = (*RuntimeApiRequestValidationCodeByHash)(nil)
	_ CandidateValidationMessage   = (*CandidateValidationMessageValidateFromExhaustive)(nil)
	_ AvailabilityStoreMessage     = (*AvailabilityStoreMessageStoreAvailableData)(nil)
	_ ValidityDoubleVote           = (*IssuedAndValidity)(nil)
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

// ProvisionerMessageProvisionableData is a provisioner message.
// This data should become part of a relay chain block.
type ProvisionerMessageProvisionableData struct {
	RelayParent       common.Hash
	ProvisionableData ProvisionableData
}

func (ProvisionerMessageProvisionableData) IsProvisionerMessage() {}

// ProvisionableData becomes intrinsics or extrinsics which should be included in a future relay chain block.
type ProvisionableData interface {
	IsProvisionableData()
}

// ProvisionableDataBackedCandidate is a provisionable data.
// The Candidate Backing subsystem believes that this candidate is valid, pending availability.
type ProvisionableDataBackedCandidate CandidateReceipt

func (ProvisionableDataBackedCandidate) IsProvisionableData() {}

// ProvisionableDataMisbehaviorReport represents self-contained proofs of validator misbehaviour.
type ProvisionableDataMisbehaviorReport struct {
	ValidatorIndex ValidatorIndex
	Misbehaviour   Misbehaviour
}

func (ProvisionableDataMisbehaviorReport) IsProvisionableData() {}

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

// StatementDistributionMessageBacked is a statement distribution message.
// it represents a message indicating that a candidate has received sufficient
// validity votes from the backing group. If backed as a result of a local statement,
// it must be preceded by a `Share` message for that statement to ensure awareness of
// full candidates before the `Backed` notification, even in groups of size 1.
type StatementDistributionMessageBacked CandidateHash

func (StatementDistributionMessageBacked) IsStatementDistributionMessage() {}

// CollatorProtocolMessage represents messages that are received by the Collator Protocol subsystem.
type CollatorProtocolMessage interface {
	IsCollatorProtocolMessage()
}

// CollatorProtocolMessageBacked is a collator protocol message.
// The candidate received enough validity votes from the backing group.
type CollatorProtocolMessageBacked struct {
	// Candidate's para id.
	ParaID ParaID
	// Hash of the para head generated by candidate.
	ParaHead common.Hash
}

func (CollatorProtocolMessageBacked) IsCollatorProtocolMessage() {}

// ProspectiveParachainsMessage represents messages that are sent to the Prospective Parachains subsystem.
type ProspectiveParachainsMessage interface {
	IsProspectiveParachainsMessage()
}

// ProspectiveParachainsMessageCandidateBacked is a prospective parachains message.
// it informs the Prospective Parachains Subsystem that
// a previously introduced candidate has been successfully backed.
type ProspectiveParachainsMessageCandidateBacked struct {
	ParaID        ParaID
	CandidateHash CandidateHash
}

func (ProspectiveParachainsMessageCandidateBacked) IsProspectiveParachainsMessage() {}

// ProspectiveParachainsMessageIntroduceCandidate is a prospective parachains message.
// it inform the Prospective Parachains Subsystem about a new candidate.
type ProspectiveParachainsMessageIntroduceCandidate struct {
	IntroduceCandidateRequest IntroduceCandidateRequest
	Ch                        chan error
}

func (ProspectiveParachainsMessageIntroduceCandidate) IsProspectiveParachainsMessage() {}

// IntroduceCandidateRequest is a request to introduce a candidate into the Prospective Parachains Subsystem.
type IntroduceCandidateRequest struct {
	// The para-id of the candidate.
	CandidateParaID ParaID
	// The candidate receipt itself.
	CommittedCandidateReceipt CommittedCandidateReceipt
	// The persisted validation data of the candidate.
	PersistedValidationData PersistedValidationData
}

// ProspectiveParachainsMessageCandidateSeconded is a prospective parachains message.
// it informs the Prospective Parachains Subsystem that a previously introduced candidate
// has been seconded. This requires that the candidate was successfully introduced in
// the past.
type ProspectiveParachainsMessageCandidateSeconded struct {
	ParaID        ParaID
	CandidateHash CandidateHash
}

func (ProspectiveParachainsMessageCandidateSeconded) IsProspectiveParachainsMessage() {}

// Get the hypothetical frontier membership of candidates with the given properties
// under the specified active leaves' fragment trees.
//
// For any candidate which is already known, this returns the depths the candidate
// occupies.
type ProspectiveParachainsMessageGetHypotheticalFrontier struct {
	HypotheticalFrontierRequest HypotheticalFrontierRequest
	Ch                          chan HypotheticalFrontierResponse
}

func (ProspectiveParachainsMessageGetHypotheticalFrontier) IsProspectiveParachainsMessage() {}

// Request specifying which candidates are either already included
// or might be included in the hypothetical frontier of fragment trees
// under a given active leaf
type HypotheticalFrontierRequest struct {
	// Candidates, in arbitrary order, which should be checked for possible membership in fragment trees
	Candidates []HypotheticalCandidate
	// Either a specific fragment tree to check, otherwise all.
	FragmentTreeRelayParent *common.Hash
	// Only return membership if all candidates in the path from the root are backed.
	BackedInPathOnly bool
}

type HypotheticalFrontierResponse []struct {
	HypotheticalCandidate  HypotheticalCandidate
	FragmentTreeMembership []FragmentTreeMembership
}

// Indicates the relay-parents whose fragment tree a candidate
// is present in and the depths of that tree the candidate is present in.
type FragmentTreeMembership struct {
	RelayParent common.Hash
	Depths      []uint
}

// HypotheticalCandidate represents a candidate to be evaluated for membership
// in the prospective parachains subsystem.
//
// Hypothetical candidates can be categorised into two types: complete and incomplete.
//
//   - Complete candidates have already had their potentially heavy candidate receipt
//     fetched, making them suitable for stricter evaluation.
//
//   - Incomplete candidates are simply claims about properties that a fetched candidate
//     would have and are evaluated less strictly.
type HypotheticalCandidate interface {
	isHypotheticalCandidate()
}

// HypotheticalCandidateIncomplete represents an incomplete hypothetical candidate.
// this
type HypotheticalCandidateIncomplete struct {
	// CandidateHash is the claimed hash of the candidate.
	CandidateHash CandidateHash
	// ParaID is the claimed para-ID of the candidate.
	CandidateParaID ParaID
	// ParentHeadDataHash is the claimed head-data hash of the candidate.
	ParentHeadDataHash common.Hash
	// RelayParent is the claimed relay parent of the candidate.
	RelayParent common.Hash
}

func (HypotheticalCandidateIncomplete) isHypotheticalCandidate() {}

// HypotheticalCandidateComplete represents a complete candidate, including its hash, committed candidate receipt,
// and persisted validation data.
type HypotheticalCandidateComplete struct {
	CandidateHash             CandidateHash
	CommittedCandidateReceipt CommittedCandidateReceipt
	PersistedValidationData   PersistedValidationData
}

func (HypotheticalCandidateComplete) isHypotheticalCandidate() {}

// RuntimeApiMessage is a message to the Runtime API subsystem.
type RuntimeApiMessage interface {
	IsRuntimeApiMessage()
}

type RuntimeApiMessageRequest struct {
	RelayParent common.Hash
	// Make a request of the runtime API against the post-state of the given relay-parent.
	RuntimeApiRequest RuntimeApiRequest
}

func (RuntimeApiMessageRequest) IsRuntimeApiMessage() {}

type RuntimeApiRequest interface {
	IsRuntimeApiRequest()
}

// RuntimeApiRequestValidationCodeByHash retrieves validation code by its hash. It can return
// past, current, or future code as long as state is available.
type RuntimeApiRequestValidationCodeByHash struct {
	ValidationCodeHash ValidationCodeHash
	Ch                 chan OverseerFuncRes[ValidationCode]
}

func (RuntimeApiRequestValidationCodeByHash) IsRuntimeApiRequest() {}

// CandidateValidationMessage represents messages received by the Validation subsystem.
// Validation requests should return an error only in case of internal errors.
type CandidateValidationMessage interface {
	IsCandidateValidationMessage()
}

// CandidateValidationMessageValidateFromExhaustive performs full validation of a candidate with provided parameters,
// including `PersistedValidationData` and `ValidationCode`. It doesn't involve acceptance
// criteria checking and is typically used when the candidate's validity is established
// through prior relay-chain checks.
type CandidateValidationMessageValidateFromExhaustive struct {
	PersistedValidationData PersistedValidationData
	ValidationCode          ValidationCode
	CandidateReceipt        CandidateReceipt
	PoV                     PoV
	ExecutorParams          ExecutorParams
	PvfExecTimeoutKind      PvfExecTimeoutKind
	Ch                      chan OverseerFuncRes[ValidationResult]
}

func (CandidateValidationMessageValidateFromExhaustive) IsCandidateValidationMessage() {}

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

// AvailabilityStoreMessageStoreAvailableData computes and checks the erasure root of `AvailableData`
// before storing its chunks in the AV store.
type AvailabilityStoreMessageStoreAvailableData struct {
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

func (AvailabilityStoreMessageStoreAvailableData) IsAvailabilityStoreMessage() {}

// AvailableData represents the data that is kept available for each candidate included in the relay chain.
type AvailableData struct {
	// The Proof-of-Validation (PoV) of the candidate
	PoV PoV `scale:"1"`

	// The persisted validation data needed for approval checks
	ValidationData PersistedValidationData `scale:"2"`
}
