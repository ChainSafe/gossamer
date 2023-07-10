package grandpa

import "github.com/tidwall/btree"

type VoterSetState struct{}

type SharedVoterSetState struct{}

// CurrentRounds A map with voter status information for currently live rounds,
// which votes have we cast and what are they.
type CurrentRounds struct {
	*btree.Map[uint64, HasVoted]
}

// TODO this is an enum
type HasVoted struct{}

func NewCurrentRounds() *CurrentRounds {
	// TODO what degree should I use?
	bTree := btree.NewMap[uint64, HasVoted](2)
	return &CurrentRounds{bTree}
}
