// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"bytes"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto/ed25519"
)

//nolint:revive
type (
	Voter      = types.GrandpaVoter
	Voters     = types.GrandpaVoters
	Vote       = types.GrandpaVote
	SignedVote = types.GrandpaSignedVote
)

// Subround subrounds in a grandpa round
type Subround byte

var (
	prevote         Subround
	precommit       Subround = 1
	primaryProposal Subround = 2
)

func (s Subround) String() string {
	switch s {
	case prevote:
		return "prevote"
	case precommit:
		return "precommit"
	case primaryProposal:
		return "primaryProposal"
	}

	return "unknown"
}

// State represents a GRANDPA state
type State struct {
	voters []Voter // set of voters
	setID  uint64  // authority set ID
	round  uint64  // voting round number
}

// NewState returns a new GRANDPA state
func NewState(voters []Voter, setID, round uint64) *State {
	return &State{
		voters: voters,
		setID:  setID,
		round:  round,
	}
}

// pubkeyToVoter returns a Voter given a public key
func (s *State) pubkeyToVoter(pk *ed25519.PublicKey) (*Voter, error) {
	max := uint64(2^64) - 1
	id := max

	for i, v := range s.voters {
		if bytes.Equal(pk.Encode(), v.Key.Encode()) {
			id = uint64(i)
			break
		}
	}

	if id == max {
		return nil, ErrVoterNotFound
	}

	return &Voter{
		Key: *pk,
		ID:  id,
	}, nil
}

// threshold returns the 2/3 |voters| threshold value
// rounding is currently set to floor, which is ok since we check for strictly greater than the threshold
func (s *State) threshold() uint64 {
	return uint64(2 * len(s.voters) / 3)
}

// NewVote returns a new Vote given a block hash and number
func NewVote(hash common.Hash, number uint32) *Vote {
	return &Vote{
		Hash:   hash,
		Number: number,
	}
}

// NewVoteFromHeader returns a new Vote for a given header
func NewVoteFromHeader(h *types.Header) *Vote {
	return &Vote{
		Hash:   h.Hash(),
		Number: uint32(h.Number.Int64()),
	}
}

// NewVoteFromHash returns a new Vote given a hash and a blockState
func NewVoteFromHash(hash common.Hash, blockState BlockState) (*Vote, error) {
	has, err := blockState.HasHeader(hash)
	if err != nil {
		return nil, err
	}

	if !has {
		return nil, ErrBlockDoesNotExist
	}

	h, err := blockState.GetHeader(hash)
	if err != nil {
		return nil, err
	}

	return NewVoteFromHeader(h), nil
}

// Commit contains all the signed precommits for a given block
type Commit struct {
	Hash       common.Hash
	Number     uint32
	Precommits []SignedVote
}

// Justification represents a finality justification for a block
type Justification struct {
	Round  uint64
	Commit Commit
}

func newJustification(round uint64, hash common.Hash, number uint32, j []SignedVote) *Justification {
	return &Justification{
		Round: round,
		Commit: Commit{
			Hash:       hash,
			Number:     number,
			Precommits: j,
		},
	}
}
