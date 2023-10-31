// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package parachaintypes

import (
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

type SubSystemName string

const (
	CandidateBacking  SubSystemName = "CandidateBacking"
	CollationProtocol SubSystemName = "CollationProtocol"
	AvailabilityStore SubSystemName = "AvailabilityStore"
)

// OverseerFuncRes is a result of an overseer function
type OverseerFuncRes[T any] struct {
	Err  error
	Data T
}

// ProvisionerMessage is a message to the Provisioner.
type ProvisionerMessage struct {
	Value any
}

// PMProvisionableData is a provisioner message.
// This data should become part of a relay chain block.
type PMProvisionableData struct {
	RelayParent       common.Hash
	ProvisionableData any
}

// ProvisionableData becomes intrinsics or extrinsics which should be included in a future relay chain block.
type ProvisionableData struct {
	Value any
}

// PDBackedCandidate is a provisionable data.
// The Candidate Backing subsystem believes that this candidate is valid, pending availability.
type PDBackedCandidate CandidateReceipt

// PDMisbehaviorReport represents self-contained proofs of validator misbehaviour.
type PDMisbehaviorReport struct {
	ValidatorIndex ValidatorIndex
	Misbehaviour   Misbehaviour
}

// Misbehaviour is intended to represent different kinds of misbehaviour along with supporting proofs.
//
// TODO: implement this by referring
// https://github.com/paritytech/polkadot-sdk/blob/7ca0d65f19497ac1c3c7ad6315f1a0acb2ca32f8/polkadot/statement-table/src/lib.rs#L53-L60 //nolint:lll
type Misbehaviour byte

// StatementDistributionMessage is a message to the Statement Distribution subsystem.
type StatementDistributionMessage struct {
	Value any
}

// SDMBacked is a statement distribution message.
// it represents a message indicating that a candidate has received sufficient
// validity votes from the backing group. If backed as a result of a local statement,
// it must be preceded by a `Share` message for that statement to ensure awareness of
// full candidates before the `Backed` notification, even in groups of size 1.
type SDMBacked CandidateHash

// CollatorProtocolMessage represents messages that are received by the Collator Protocol subsystem.
type CollatorProtocolMessage struct {
	Value any
}

// CPMBacked is a collator protocol message.
// The candidate received enough validity votes from the backing group.
type CPMBacked struct {
	// Candidate's para id.
	ParaID ParaID
	// Hash of the para head generated by candidate.
	ParaHead common.Hash
}

// ProspectiveParachainsMessage represents messages that are sent to the Prospective Parachains subsystem.
type ProspectiveParachainsMessage struct {
	Value any
}

// PPMCandidateBacked is a prospective parachains message.
// it informs the Prospective Parachains Subsystem that
// a previously introduced candidate has been successfully backed.
type PPMCandidateBacked struct {
	ParaID        ParaID
	CandidateHash CandidateHash
}

// PPMIntroduceCandidate is a prospective parachains message.
// it inform the Prospective Parachains Subsystem about a new candidate.
type PPMIntroduceCandidate struct {
	IntroduceCandidateRequest IntroduceCandidateRequest
	Ch                        chan error
}

// PPMCandidateSeconded is a prospective parachains message.
// it informs the Prospective Parachains Subsystem that a previously introduced candidate
// has been seconded. This requires that the candidate was successfully introduced in
// the past.
type PPMCandidateSeconded struct {
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

// RuntimeApiMessage is a message to the Runtime API subsystem.
type RuntimeApiMessage struct {
	RelayParent common.Hash
	// Make a request of the runtime API against the post-state of the given relay-parent.
	RuntimeApiRequest any
}

// RAMValidationCodeByHash retrieves validation code by its hash. It can return
// past, current, or future code as long as state is available.
type RAMValidationCodeByHash struct {
	ValidationCodeHash ValidationCodeHash
	Ch                 chan OverseerFuncRes[ValidationCode]
}

// CandidateValidationMessage represents messages received by the Validation subsystem.
// Validation requests should return an error only in case of internal errors.
type CandidateValidationMessage struct {
	Value any
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
	PvfPrepTimeoutKind      PvfPrepTimeoutKind
	Ch                      chan OverseerFuncRes[ValidationResult]
}

// ExecutorParams represents the abstract semantics of an execution environment and should remain
// as abstract as possible. There are no mandatory parameters defined at the moment, and if any
// are introduced in the future, they must be clearly documented as mandatory.
//
//	// TODO: Implement this #3544
//	// https://github.com/paritytech/polkadot-sdk/blob/7ca0d65f19497ac1c3c7ad6315f1a0acb2ca32f8/polkadot/primitives/src/v6/executor_params.rs#L97-L98
type ExecutorParams scale.VaryingDataTypeSlice

// PvfPrepTimeoutKind is an enumeration representing the type discriminator for PVF execution timeouts.
type PvfPrepTimeoutKind byte

const (
	// Backing represents the amount of time to spend on execution during backing.
	Backing PvfPrepTimeoutKind = iota

	// Approval represents the amount of time to spend on execution during approval or disputes.
	// This timeout should be much longer than the backing execution timeout to ensure that,
	// in the absence of extremely large disparities between hardware, blocks that pass
	// backing are considered executable by approval checkers or dispute participants.
	Approval
)

// ValidationResult represents the result coming from the candidate validation subsystem.
type ValidationResult struct {
	IsValid                 bool
	CandidateCommitments    CandidateCommitments
	PersistedValidationData PersistedValidationData
	Err                     error
}

// AvailabilityStoreMessage represents messages received by the Availability Store subsystem.
type AvailabilityStoreMessage struct {
	Value any
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

// AvailableData represents the data that is kept available for each candidate included in the relay chain.
type AvailableData struct {
	// The Proof-of-Validation (PoV) of the candidate
	PoV PoV `scale:"1"`

	// The persisted validation data needed for approval checks
	ValidationData PersistedValidationData `scale:"2"`
}
