package parachain

import (
	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
)

type Table interface {
	importStatement(*TableContext, *CheckedSignedFullStatement) (*Summary, error)
	attestedCandidate(*CandidateHash, *TableContext) (*AttestedCandidate, error)
}

// A summary of import of a statement.
type Summary struct {
	// The digest of the candidate referenced.
	Candidate CandidateHash
	// The group that the candidate is in.
	GroupID uint32
	// How many validity votes are currently witnessed.
	ValidityVotes uint64
}

// An attested-to candidate.
type AttestedCandidate struct {
	// The group ID that the candidate is in.
	GroupID uint32
	// The candidate data.
	Candidate parachaintypes.CommittedCandidateReceipt
	// Validity attestations.
	ValidityVotes interface{}
}
