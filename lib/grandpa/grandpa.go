package grandpa

// Grandpa represents the state of the grandpa protocol
type Grandpa struct {
	state *state 	// previous state
	subround 	subround 	// current sub-round
	votes map[voter]*vote 	// votes for next state
	equivocations map[voter]*vote 		// equivocatory votes for this stage
	head common.Hash 	// most recently finalized block hash
}

// NewGrandpa returns a new GRANDPA instance.
// TODO: determine GRANDPA initialization and entrypoint.
func NewGrandpa() *Grandpa {
	return &Grandpa{}
}

// newState returns a new GRANDPA state
func newState(voters []*Voter, counter, round uint64) *state {
	return &state{
		voters: voters,
		counter: counter,
		round: round,
	}
}

func (s *State) validateVote(v *vote) bool {
	// check if v.hash corresponds to a valid block

	// check if the block is an eventual descendant of a previously finalized block

	// check 
}