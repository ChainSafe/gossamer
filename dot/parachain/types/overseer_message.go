// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package parachaintypes

import "github.com/ChainSafe/gossamer/lib/common"

var (
	_ ProvisionableData     = (*ProvisionableDataBackedCandidate)(nil)
	_ ProvisionableData     = (*ProvisionableDataMisbehaviorReport)(nil)
	_ RuntimeApiRequest     = (*RuntimeApiRequestValidationCodeByHash)(nil)
	_ HypotheticalCandidate = (*HypotheticalCandidateIncomplete)(nil)
	_ HypotheticalCandidate = (*HypotheticalCandidateComplete)(nil)
)

// OverseerFuncRes is a result of an overseer function
type OverseerFuncRes[T any] struct {
	Err  error
	Data T
}

// ProvisionerMessageProvisionableData is a provisioner message.
// This data should become part of a relay chain block.
type ProvisionerMessageProvisionableData struct {
	RelayParent       common.Hash
	ProvisionableData ProvisionableData
}

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

// StatementDistributionMessageBacked is a statement distribution message.
// it represents a message indicating that a candidate has received sufficient
// validity votes from the backing group. If backed as a result of a local statement,
// it must be preceded by a `Share` message for that statement to ensure awareness of
// full candidates before the `Backed` notification, even in groups of size 1.
type StatementDistributionMessageBacked CandidateHash

// StatementDistributionMessageShare is a statement distribution message.
// It is a signed statement in the context of
// given relay-parent hash and it should be distributed to other validators.
type StatementDistributionMessageShare struct {
	RelayParent                common.Hash
	SignedFullStatementWithPVD SignedFullStatementWithPVD
}

// ProspectiveParachainsMessageGetTreeMembership is a prospective parachains message.
// It is intended for retrieving the membership of a candidate in all fragment trees
type ProspectiveParachainsMessageGetTreeMembership struct {
	ParaID        ParaID
	CandidateHash CandidateHash
	ResponseCh    chan []FragmentTreeMembership
}

// ProspectiveParachainsMessageCandidateBacked is a prospective parachains message.
// it informs the Prospective Parachains Subsystem that
// a previously introduced candidate has been successfully backed.
type ProspectiveParachainsMessageCandidateBacked struct {
	ParaID        ParaID
	CandidateHash CandidateHash
}

// ProspectiveParachainsMessageIntroduceCandidate is a prospective parachains message.
// it inform the Prospective Parachains Subsystem about a new candidate.
type ProspectiveParachainsMessageIntroduceCandidate struct {
	IntroduceCandidateRequest IntroduceCandidateRequest
	Ch                        chan error
}

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

// ProspectiveParachainsMessageGetHypotheticalFrontier is a prospective parachains message.
// Get the hypothetical frontier membership of candidates with the given properties
// under the specified active leaves fragment trees.
//
// For any candidate which is already known, this returns the depths the candidate
// occupies.
type ProspectiveParachainsMessageGetHypotheticalFrontier struct {
	HypotheticalFrontierRequest HypotheticalFrontierRequest
	ResponseCh                  chan HypotheticalFrontierResponses
}

// HypotheticalFrontierRequest specifies which candidates are either already included
// or might be included in the hypothetical frontier of fragment trees
// under a given active leaf.
type HypotheticalFrontierRequest struct {
	// Candidates, in arbitrary order, which should be checked for possible membership in fragment trees
	Candidates []HypotheticalCandidate
	// Either a specific fragment tree to check, otherwise all.
	FragmentTreeRelayParent *common.Hash
	// Only return membership if all candidates in the path from the root are backed.
	BackedInPathOnly bool
}

// HypotheticalFrontierResponses contains information about the hypothetical frontier
// membership of multiple candidates under active leaf fragment trees.
type HypotheticalFrontierResponses []HypotheticalFrontierResponse

// HypotheticalFrontierResponse contains information about the hypothetical frontier
// membership of a specific candidate under active leaf fragment trees.
type HypotheticalFrontierResponse struct {
	HypotheticalCandidate HypotheticalCandidate
	Memberships           []FragmentTreeMembership
}

// FragmentTreeMembership indicates the relay-parents whose fragment tree a candidate
// is present in, along with the depths of that tree the candidate is present in.
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

type RuntimeApiMessageRequest struct {
	RelayParent common.Hash
	// Make a request of the runtime API against the post-state of the given relay-parent.
	RuntimeApiRequest RuntimeApiRequest
}

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

// ValidationResult represents the result coming from the candidate validation subsystem.
type ValidationResult struct {
	IsValid                 bool
	CandidateCommitments    CandidateCommitments
	PersistedValidationData PersistedValidationData
	Err                     error
}

// AvailabilityDistributionMessageFetchPoV represents a message instructing
// availability distribution to fetch a remote Proof of Validity (PoV).
type AvailabilityDistributionMessageFetchPoV struct {
	RelayParent common.Hash
	// FromValidator is the validator to fetch the PoV from.
	FromValidator ValidatorIndex
	// ParaID is the ID of the parachain that produced this PoV.
	// This field is only used to provide more context when logging errors
	// from the AvailabilityDistribution subsystem.
	ParaID ParaID
	// CandidateHash is the candidate hash to fetch the PoV for.
	CandidateHash CandidateHash
	// PovHash is the expected hash of the PoV; a PoV not matching this hash will be rejected.
	PovHash common.Hash
	// PovCh is the channel for receiving the result of this fetch.
	// The channel will be closed if the fetching fails for some reason.
	PovCh chan OverseerFuncRes[PoV]
}
