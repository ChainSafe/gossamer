// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"errors"

	"github.com/ChainSafe/gossamer/lib/blocktree"
)

var (
	// ErrBlockDoesNotExist is returned when trying to validate a vote for a block that doesn't exist
	ErrBlockDoesNotExist = errors.New("block does not exist")

	// ErrInvalidSignature is returned when trying to validate a vote message with an invalid signature
	ErrInvalidSignature = errors.New("signature is not valid")

	// ErrSetIDMismatch is returned when trying to validate a vote message
	// with an invalid voter set ID, or when receiving a catch up message
	// with a different set ID
	ErrSetIDMismatch = errors.New("set IDs do not match")

	// ErrEquivocation is returned when trying to validate a vote for that is equivocatory
	ErrEquivocation = errors.New("vote is equivocatory")

	// ErrVoterNotFound is returned when trying to validate a vote for a voter that isn't in the voter set
	ErrVoterNotFound = errors.New("voter is not in voter set")

	// ErrDescendantNotFound is returned when trying to validate a vote
	// for a block that isn't a descendant of the last finalised block
	ErrDescendantNotFound = blocktree.ErrDescendantNotFound

	// ErrNoPreVotedBlock is returned when there is no pre-voted block for a round.
	// this can only happen in the case of > 1/3 byzantine nodes (ie > 1/3 nodes equivocate or don't submit valid votes)
	ErrNoPreVotedBlock = errors.New("cannot get pre-voted block")

	// ErrNoGHOST is returned when there is no GHOST. the only case where this could happen is if there are no votes
	// at all, so it shouldn't ever happen.
	ErrNoGHOST = errors.New("cannot determine grandpa-GHOST")

	// ErrCannotDecodeSubround is returned when a subround value cannot be decoded
	ErrCannotDecodeSubround = errors.New("cannot decode invalid subround value")

	// ErrInvalidMessageType is returned when a network.Message cannot be decoded
	ErrInvalidMessageType = errors.New("cannot decode invalid message type")

	// ErrNotCommitMessage is returned when calling GetFinalisedHash on a message that isn't a CommitMessage
	ErrNotCommitMessage = errors.New("cannot get finalised hash from VoteMessage")

	// ErrNoJustification is returned when no justification can be found for a block, ie. it has not been finalised
	ErrNoJustification       = errors.New("no justification found for block")
	ErrJustificationMismatch = errors.New("justification does not correspond to given block hash")

	ErrBlockHashMismatch = errors.New("block hash does not correspond to given block number")

	ErrBlockNumbersMismatch = errors.New("block numbers mismatch")

	// ErrMinVotesNotMet is returned when the number of votes is less than the required minimum in a Justification
	ErrMinVotesNotMet = errors.New("minimum number of votes not met in a Justification")

	// ErrInvalidCatchUpRound is returned when a catch-up message is received with an invalid round
	ErrInvalidCatchUpRound = errors.New("catch up request is for future round")

	// ErrInvalidCatchUpResponseRound is returned when a catch-up response is received with an invalid round
	ErrInvalidCatchUpResponseRound = errors.New("catch up response is not for previous round")

	// ErrGHOSTlessCatchUp is returned when a catch up response
	// does not contain a valid grandpa-GHOST (ie. finalised block)
	ErrGHOSTlessCatchUp = errors.New("catch up response does not contain grandpa-GHOST")

	// ErrCatchUpResponseNotCompletable is returned when the round represented by the catch up response is not completable
	ErrCatchUpResponseNotCompletable = errors.New("catch up response is not completable")

	// ErrPrecommitSignatureMismatch is returned when the number of precommits
	// and signatures in a CommitMessage do not match
	ErrPrecommitSignatureMismatch = errors.New("number of precommits does not match number of signatures")

	// ErrPrecommitBlockMismatch is returned when a precommit hash within a
	// justification is not a descendant of the committed block
	ErrPrecommitBlockMismatch = errors.New("precommit block is not descendant of committed block")

	// ErrAuthorityNotInSet is returned when a precommit within a justification is signed by a key not in the authority set
	ErrAuthorityNotInSet = errors.New("authority is not in set")

	errVoteToSignatureMismatch  = errors.New("votes and authority count mismatch")
	errVoteBlockMismatch        = errors.New("block in vote is not descendant of previously finalised block")
	errVoteFromSelf             = errors.New("got vote from ourselves")
	errRoundOutOfBounds         = errors.New("round out of bounds")
	errRoundsMismatch           = errors.New("rounds mismatch")
	errInvalidEquivocationStage = errors.New("invalid stage for equivocating")
)
