// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package parachaintypes

import "github.com/ChainSafe/gossamer/lib/common"

var (
	_ ProvisionerMessage = (*ProvisionerMessageProvisionableData)(nil)
	_ ProvisionableData  = (*ProvisionableDataBackedCandidate)(nil)
	_ ProvisionableData  = (*ProvisionableDataMisbehaviorReport)(nil)
	// _ StatementDistributionMessage = (*StatementDistributionMessageBacked)(nil)
	// _ ProspectiveParachainsMessage = (*ProspectiveParachainsMessageCandidateBacked)(nil)
	// _ ProspectiveParachainsMessage = (*ProspectiveParachainsMessageIntroduceCandidate)(nil)
	// _ ProspectiveParachainsMessage = (*ProspectiveParachainsMessageCandidateSeconded)(nil)
	// _ RuntimeApiMessage = (*RuntimeApiMessageRequest)(nil)
	_ RuntimeApiRequest = (*RuntimeApiRequestValidationCodeByHash)(nil)
	// _ CandidateValidationMessage   = (*CandidateValidationMessageValidateFromExhaustive)(nil)
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

// StatementDistributionMessageBacked is a statement distribution message.
// it represents a message indicating that a candidate has received sufficient
// validity votes from the backing group. If backed as a result of a local statement,
// it must be preceded by a `Share` message for that statement to ensure awareness of
// full candidates before the `Backed` notification, even in groups of size 1.
type StatementDistributionMessageBacked CandidateHash

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

// ProspectiveParachainsMessageCandidateSeconded is a prospective parachains message.
// it informs the Prospective Parachains Subsystem that a previously introduced candidate
// has been seconded. This requires that the candidate was successfully introduced in
// the past.
type ProspectiveParachainsMessageCandidateSeconded struct {
	ParaID        ParaID
	CandidateHash CandidateHash
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
