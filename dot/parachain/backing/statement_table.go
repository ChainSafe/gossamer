// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package backing

import parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"

type Table interface {
	getCandidate(parachaintypes.CandidateHash) (*parachaintypes.CommittedCandidateReceipt, error)
	importStatement(*TableContext, SignedFullStatementWithPVD) (*Summary, error)
	attestedCandidate(parachaintypes.CandidateHash, *TableContext) (*AttestedCandidate, error)
	drainMisbehaviors() []parachaintypes.ProvisionableDataMisbehaviorReport
}

// Summary represents summary of import of a statement.
type Summary struct {
	// The digest of the candidate referenced.
	Candidate parachaintypes.CandidateHash
	// The group that the candidate is in.
	GroupID parachaintypes.ParaID
	// How many validity votes are currently witnessed.
	ValidityVotes uint64
}

// AttestedCandidate represents an attested-to candidate.
type AttestedCandidate struct {
	// The group ID that the candidate is in.
	GroupID parachaintypes.ParaID
	// The candidate data.
	Candidate parachaintypes.CommittedCandidateReceipt
	// Validity attestations.
	ValidityVotes []validityVote
}

// validityVote represents a vote on the validity of a candidate by a validator.
type validityVote struct {
	ValidatorIndex      parachaintypes.ValidatorIndex
	ValidityAttestation parachaintypes.ValidityAttestation
}
