package backing

import parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"

type Table interface {
	getCandidate(parachaintypes.CandidateHash) (*parachaintypes.CommittedCandidateReceipt, error)
	importStatement(*TableContext, SignedFullStatementWithPVD) (*Summary, error)
	attestedCandidate(*parachaintypes.CandidateHash, *TableContext) (*AttestedCandidate, error)
}

// A summary of import of a statement.
type Summary struct {
	// The digest of the candidate referenced.
	Candidate parachaintypes.CandidateHash
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
	ValidityVotes interface{} // TODO: figure out actual type
}
