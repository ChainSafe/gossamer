// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package parachaintypes

import "github.com/ChainSafe/gossamer/lib/common"

var (
	_ HypotheticalCandidate = (*HypotheticalCandidateIncomplete)(nil)
	_ HypotheticalCandidate = (*HypotheticalCandidateComplete)(nil)
)

// OverseerFuncRes is a result of an overseer function
type OverseerFuncRes[T any] struct {
	Err  error
	Data T
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
	ParaID() ParaID
	CandidateHash() CandidateHash
	RelayParentHash() common.Hash
}

// HypotheticalCandidateIncomplete represents an incomplete hypothetical candidate.
type HypotheticalCandidateIncomplete struct {
	// ClaimedCandidateHash is the claimed hash of the candidate.
	ClaimedCandidateHash CandidateHash
	// ParaID is the claimed para-ID of the candidate.
	CandidateParaID ParaID
	// ParentHeadDataHash is the claimed head-data hash of the candidate.
	ParentHeadDataHash common.Hash
	// RelayParent is the claimed relay parent of the candidate.
	RelayParent common.Hash
}

func (HypotheticalCandidateIncomplete) isHypotheticalCandidate() {}

func (h HypotheticalCandidateIncomplete) ParaID() ParaID {
	return h.CandidateParaID
}

func (h HypotheticalCandidateIncomplete) CandidateHash() CandidateHash {
	return h.ClaimedCandidateHash
}

func (h HypotheticalCandidateIncomplete) RelayParentHash() common.Hash {
	return h.RelayParent
}

// HypotheticalCandidateComplete represents a complete candidate, including its hash, committed candidate receipt,
// and persisted validation data.
type HypotheticalCandidateComplete struct {
	// The hash of the candidate.
	ClaimedCandidateHash CandidateHash
	// The receipt of the candidate.
	CommittedCandidateReceipt CommittedCandidateReceiptV2
	// The persisted validation data of the candidate.
	PersistedValidationData PersistedValidationData
}

func (HypotheticalCandidateComplete) isHypotheticalCandidate() {}

func (h HypotheticalCandidateComplete) ParaID() ParaID {
	return h.CommittedCandidateReceipt.Descriptor.ParaID
}

func (h HypotheticalCandidateComplete) CandidateHash() CandidateHash {
	return h.ClaimedCandidateHash
}

func (h HypotheticalCandidateComplete) RelayParentHash() common.Hash {
	return h.CommittedCandidateReceipt.Descriptor.RelayParent
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
