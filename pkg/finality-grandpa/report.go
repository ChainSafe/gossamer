package grandpa

// Basic data struct for the state of a round.
type RoundStateReport[ID comparable] struct {
	// Total weight of all votes.
	TotalWeight VoterWeight
	// The threshold voter weight.
	ThresholdWeight VoterWeight

	// Current weight of the prevotes.
	PrevoteCurrentWeight VoteWeight
	// The identities of nodes that have cast prevotes so far.
	PrevoteIDs []ID

	// Current weight of the precommits.
	PrecommitCurrentWeight VoteWeight
	// The identities of nodes that have cast precommits so far.
	PrecommitIDs []ID
}

// Basic data struct for the current state of the voter in a form suitable
// for passing on to other systems.
type VoterStateReport[ID comparable] struct {
	// Voting rounds running in the background.
	BackgroundRounds map[uint64]RoundStateReport[ID]
	// The current best voting round.
	BestRound struct {
		Number     uint64
		RoundState RoundStateReport[ID]
	}
}
