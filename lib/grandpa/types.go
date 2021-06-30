// Copyright 2019 ChainSafe Systems (ON) Corp.
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
	"bytes"
	"encoding/binary"
	"fmt"
	"io"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto/ed25519"
	"github.com/ChainSafe/gossamer/lib/scale"
)

//nolint
type (
	Voter  = types.GrandpaVoter
	Voters = types.GrandpaVoters
)

type subround byte

var (
	prevote         subround
	precommit       subround = 1
	primaryProposal subround = 2
)

func (s subround) Encode() ([]byte, error) {
	return []byte{byte(s)}, nil
}

func (s subround) Decode(r io.Reader) (subround, error) {
	b, err := common.ReadByte(r)
	if err != nil {
		return 255, nil
	}

	if b == 0 {
		return prevote, nil
	} else if b == 1 {
		return precommit, nil
	} else {
		return 255, ErrCannotDecodeSubround
	}
}

func (s subround) String() string {
	if s == prevote {
		return "prevote"
	} else if s == precommit {
		return "precommit"
	}

	return "unknown"
}

// State represents a GRANDPA state
type State struct {
	voters []*Voter // set of voters
	setID  uint64   // authority set ID
	round  uint64   // voting round number
}

// NewState returns a new GRANDPA state
func NewState(voters []*Voter, setID, round uint64) *State {
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
		Key: pk,
		ID:  id,
	}, nil
}

// threshold returns the 2/3 |voters| threshold value
// TODO: determine rounding, is currently set to floor
func (s *State) threshold() uint64 {
	return uint64(2 * len(s.voters) / 3)
}

// Vote represents a vote for a block with the given hash and number
type Vote struct {
	hash   common.Hash
	number uint32
}

// NewVote returns a new Vote given a block hash and number
func NewVote(hash common.Hash, number uint32) *Vote {
	return &Vote{
		hash:   hash,
		number: number,
	}
}

// NewVoteFromHeader returns a new Vote for a given header
func NewVoteFromHeader(h *types.Header) *Vote {
	return &Vote{
		hash:   h.Hash(),
		number: uint32(h.Number.Int64()),
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

// Encode returns the SCALE encoding of a Vote
func (v *Vote) Encode() ([]byte, error) {
	buf := make([]byte, 4)
	binary.LittleEndian.PutUint32(buf, v.number)
	return append(v.hash[:], buf...), nil
}

// Decode returns the SCALE decoded Vote
func (v *Vote) Decode(r io.Reader) (*Vote, error) {
	if v == nil {
		v = new(Vote)
	}

	var err error
	v.hash, err = common.ReadHash(r)
	if err != nil {
		return nil, err
	}

	v.number, err = common.ReadUint32(r)
	if err != nil {
		return nil, err
	}

	return v, nil
}

// String returns the Vote as a string
func (v *Vote) String() string {
	return fmt.Sprintf("hash=%s number=%d", v.hash, v.number)
}

// SignedPrecommit represents a signed precommit message for a finalised block
type SignedPrecommit struct {
	Vote        *Vote
	Signature   [64]byte
	AuthorityID ed25519.PublicKeyBytes
}

// Encode returns the SCALE encoded Justification
func (j *SignedPrecommit) Encode() ([]byte, error) {
	enc, err := j.Vote.Encode()
	if err != nil {
		return nil, err
	}

	enc = append(enc, j.Signature[:]...)
	enc = append(enc, j.AuthorityID[:]...)
	return enc, nil
}

// Decode returns the SCALE decoded Justification
func (j *SignedPrecommit) Decode(r io.Reader) (*SignedPrecommit, error) {
	sd := &scale.Decoder{Reader: r}
	i, err := sd.Decode(j)
	if err != nil {
		return nil, err
	}

	d := i.(*SignedPrecommit)
	j.Vote = d.Vote
	j.Signature = d.Signature
	j.AuthorityID = d.AuthorityID
	return j, nil
}

// Commit contains all the signed precommits for a given block
type Commit struct {
	Hash       common.Hash
	Number     uint32
	Precommits []*SignedPrecommit
}

// Justification represents a finality justification for a block
type Justification struct {
	Round  uint64
	Commit *Commit
}

func newJustification(round uint64, hash common.Hash, number uint32, j []*SignedPrecommit) *Justification {
	return &Justification{
		Round: round,
		Commit: &Commit{
			Hash:       hash,
			Number:     number,
			Precommits: j,
		},
	}
}

// Encode returns the SCALE encoding of a FullJustification
func (j *Justification) Encode() ([]byte, error) {
	return scale.Encode(j)
}

// Decode returns a SCALE decoded FullJustification
func (j *Justification) Decode(r io.Reader) error {
	sd := &scale.Decoder{Reader: r}
	i, err := sd.Decode(&Justification{Commit: &Commit{}})
	if err != nil {
		return err
	}

	dec := i.(*Justification)
	j.Round = dec.Round
	j.Commit = dec.Commit
	return nil
}
