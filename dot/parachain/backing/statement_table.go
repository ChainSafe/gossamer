package backing

import parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"

// A summary of import of a statement.
type Summary struct {
	// The digest of the candidate referenced.
	Candidate parachaintypes.CandidateHash
	// The group that the candidate is in.
	GroupID uint32
	// How many validity votes are currently witnessed.
	ValidityVotes uint64
}
