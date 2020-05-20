package grandpa

import (
	"errors"

	"github.com/ChainSafe/gossamer/lib/blocktree"
)

// ErrBlockDoesNotExist is returned when trying to validate a vote for a block that doesn't exist
var ErrBlockDoesNotExist = errors.New("block does not exist")

// ErrInvalidSignature is returned when trying to validate a vote message with an invalid signature
var ErrInvalidSignature = errors.New("signature is not valid")

// ErrSetIDMismatch is returned when trying to validate a vote message with an invalid voter set ID
var ErrSetIDMismatch = errors.New("set IDs do not match")

// ErrEquivocation is returned when trying to validate a vote for that is equivocatory
var ErrEquivocation = errors.New("vote is equivocatory")

// ErrVoterNotFound is returned when trying to validate a vote for a voter that isn't in the voter set
var ErrVoterNotFound = errors.New("voter is not in voter set")

// ErrDescendantNotFound is returned when trying to validate a vote for a block that isn't a descendant of the last finalized block
var ErrDescendantNotFound = blocktree.ErrDescendantNotFound

// ErrNoPreVotedBlock is returned when there is no pre-voted block for a round.
// this can only happen in the case of > 1/3 byzantine nodes (ie > 1/3 nodes equivocate or don't submit valid votes)
var ErrNoPreVotedBlock = errors.New("cannot get pre-voted block")
