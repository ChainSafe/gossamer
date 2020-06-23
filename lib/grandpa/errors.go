// Copyright 2020 ChainSafe Systems (ON) Corp.
// This file is part of gossamer.
//
// The gossamer library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The gossamer library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the gossamer library. If not, see <http://www.gnu.org/licenses/>.

package grandpa

import (
	"errors"

	"github.com/ChainSafe/gossamer/lib/blocktree"
)

// ErrNilBlockState is returned when BlockState is nil
var ErrNilBlockState = errors.New("cannot have nil BlockState")

// ErrNilKeypair is returned when the keypair is nil
var ErrNilKeypair = errors.New("cannot have nil keypair")

// ErrBlockDoesNotExist is returned when trying to validate a vote for a block that doesn't exist
var ErrBlockDoesNotExist = errors.New("block does not exist")

// ErrInvalidSignature is returned when trying to validate a vote message with an invalid signature
var ErrInvalidSignature = errors.New("signature is not valid")

// ErrSetIDMismatch is returned when trying to validate a vote message with an invalid voter set ID
var ErrSetIDMismatch = errors.New("set IDs do not match")

// ErrRoundMismatch is returned when trying to validate a vote message that isn't for the current round
var ErrRoundMismatch = errors.New("rounds do not match")

// ErrEquivocation is returned when trying to validate a vote for that is equivocatory
var ErrEquivocation = errors.New("vote is equivocatory")

// ErrVoterNotFound is returned when trying to validate a vote for a voter that isn't in the voter set
var ErrVoterNotFound = errors.New("voter is not in voter set")

// ErrDescendantNotFound is returned when trying to validate a vote for a block that isn't a descendant of the last finalized block
var ErrDescendantNotFound = blocktree.ErrDescendantNotFound

// ErrNoPreVotedBlock is returned when there is no pre-voted block for a round.
// this can only happen in the case of > 1/3 byzantine nodes (ie > 1/3 nodes equivocate or don't submit valid votes)
var ErrNoPreVotedBlock = errors.New("cannot get pre-voted block")

// ErrNoGHOST is returned when there is no GHOST. the only case where this could happen is if there are no votes
// at all, so it shouldn't ever happen.
var ErrNoGHOST = errors.New("cannot determine grandpa-GHOST")

// ErrCannotDecodeSubround is returned when a subround value cannot be decoded
var ErrCannotDecodeSubround = errors.New("cannot decode invalid subround value")

// ErrInvalidMessageType is returned when a network.Message cannot be decoded
var ErrInvalidMessageType = errors.New("cannot decode invalid message type")

// ErrNotFinalizationMessage is returned when calling GetFinalizedHash on a message that isn't a FinalizationMessage
var ErrNotFinalizationMessage = errors.New("cannot get finalized hash from VoteMessage")
