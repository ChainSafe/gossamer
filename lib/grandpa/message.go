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
	"fmt"
	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto/ed25519"
	"github.com/ChainSafe/gossamer/lib/scale"
)

// GrandpaMessage is implemented by all GRANDPA network messages
// TODO: the fields can be un-exported, as can all the message implementations
type GrandpaMessage interface { //nolint
	ToConsensusMessage() (*network.ConsensusMessage, error)
	Type() byte
}

// TODO Make these a VDT
var (
	voteType            byte
	commitType          byte = 1
	neighbourType       byte = 2
	catchUpRequestType  byte = 3
	catchUpResponseType byte = 4
)

// FullVote represents a vote with additional information about the state
// this is encoded and signed and the signature is included in SignedMessage
type FullVote struct {
	Stage subround
	Vote  Vote
	Round uint64
	SetID uint64
}

// FullVote represents a vote with additional information about the state
// this is encoded and signed and the signature is included in SignedMessage
type FullVoteNew struct {
	Stage subround
	Vote  Vote
	Round uint64
	SetID uint64
}

// SignedMessage represents a block hash and number signed by an authority
type SignedMessage struct {
	Stage       subround // 0 for pre-vote, 1 for pre-commit, 2 for primary proposal
	Hash        common.Hash
	Number      uint32
	Signature   [64]byte // ed25519.SignatureLength
	AuthorityID ed25519.PublicKeyBytes
}

// String returns the SignedMessage as a string
func (m *SignedMessage) String() string {
	return fmt.Sprintf("stage=%s hash=%s number=%d authorityID=%s", m.Stage, m.Hash, m.Number, m.AuthorityID)
}

// VoteMessage represents a network-level vote message
// https://github.com/paritytech/substrate/blob/master/client/finality-grandpa/src/communication/gossip.rs#L336
type VoteMessage struct {
	Round   uint64
	SetID   uint64
	Message SignedMessage
}

// Type returns voteType
func (v *VoteMessage) Type() byte {
	return voteType
}

// ToConsensusMessage converts the VoteMessage into a network-level consensus message
func (v *VoteMessage) ToConsensusMessage() (*ConsensusMessage, error) {
	enc, err := scale.Encode(v)
	if err != nil {
		return nil, err
	}

	return &ConsensusMessage{
		Data: append([]byte{voteType}, enc...),
	}, nil
}

// NeighbourMessage represents a network-level neighbour message
type NeighbourMessage struct {
	Version byte
	Round   uint64
	SetID   uint64
	Number  uint32
}

// ToConsensusMessage converts the NeighbourMessage into a network-level consensus message
func (m *NeighbourMessage) ToConsensusMessage() (*network.ConsensusMessage, error) {
	enc, err := scale.Encode(m)
	if err != nil {
		return nil, err
	}

	return &ConsensusMessage{
		Data: append([]byte{neighbourType}, enc...),
	}, nil
}

// Type returns neighbourType
func (m *NeighbourMessage) Type() byte {
	return neighbourType
}

// AuthData represents signature data within a CommitMessage to be paired with a Precommit
type AuthData struct {
	Signature   [64]byte
	AuthorityID ed25519.PublicKeyBytes
}

// CommitMessage represents a network finalisation message
type CommitMessage struct {
	Round      uint64
	SetID      uint64
	Vote       Vote
	Precommits []Vote
	AuthData   []AuthData
}

// Type returns commitType
func (f *CommitMessage) Type() byte {
	return commitType
}

// ToConsensusMessage converts the CommitMessage into a network-level consensus message
func (f *CommitMessage) ToConsensusMessage() (*ConsensusMessage, error) {
	enc, err := scale.Encode(f)
	if err != nil {
		return nil, err
	}

	return &ConsensusMessage{
		Data: append([]byte{commitType}, enc...),
	}, nil
}

func (s *Service) newCommitMessage(header *types.Header, round uint64) (*CommitMessage, error) {
	pcs, err := s.grandpaState.GetPrecommits(round, s.state.setID)
	if err != nil {
		return nil, err
	}

	precommits, authData := justificationToCompact(pcs)
	return &CommitMessage{
		Round:      round,
		Vote:       *NewVoteFromHeader(header),
		Precommits: precommits,
		AuthData:   authData,
	}, nil
}

func justificationToCompact(just []SignedVote) ([]Vote, []AuthData) {
	precommits := make([]Vote, len(just))
	authData := make([]AuthData, len(just))

	for i, j := range just {
		precommits[i] = j.Vote
		authData[i] = AuthData{
			Signature:   j.Signature,
			AuthorityID: j.AuthorityID,
		}
	}

	return precommits, authData
}

func compactToJustification(vs []Vote, auths []AuthData) ([]SignedVote, error) {
	if len(vs) != len(auths) {
		return nil, errVoteToSignatureMismatch
	}

	just := make([]SignedVote, len(vs))
	for i, v := range vs {
		just[i] = SignedVote{
			Vote:        v,
			Signature:   auths[i].Signature,
			AuthorityID: auths[i].AuthorityID,
		}
	}

	return just, nil
}

type catchUpRequest struct {
	Round uint64
	SetID uint64
}

func newCatchUpRequest(round, setID uint64) *catchUpRequest {
	return &catchUpRequest{
		Round: round,
		SetID: setID,
	}
}

// Type returns catchUpRequestType
func (r *catchUpRequest) Type() byte {
	return catchUpRequestType
}

// ToConsensusMessage converts the catchUpRequest into a network-level consensus message
func (r *catchUpRequest) ToConsensusMessage() (*ConsensusMessage, error) {
	enc, err := scale.Encode(r)
	if err != nil {
		return nil, err
	}

	return &ConsensusMessage{
		Data: append([]byte{catchUpRequestType}, enc...),
	}, nil
}

type catchUpResponse struct {
	SetID                  uint64
	Round                  uint64
	PreVoteJustification   []SignedVote
	PreCommitJustification []SignedVote
	Hash                   common.Hash
	Number                 uint32
}

func (s *Service) newCatchUpResponse(round, setID uint64) (*catchUpResponse, error) {
	header, err := s.blockState.GetFinalisedHeader(round, setID)
	if err != nil {
		return nil, err
	}

	pvs, err := s.grandpaState.GetPrevotes(round, setID)
	if err != nil {
		return nil, err
	}

	pcs, err := s.grandpaState.GetPrecommits(round, setID)
	if err != nil {
		return nil, err
	}

	return &catchUpResponse{
		SetID:                  setID,
		Round:                  round,
		PreVoteJustification:   pvs,
		PreCommitJustification: pcs,
		Hash:                   header.Hash(),
		Number:                 uint32(header.Number.Uint64()),
	}, nil
}

// Type returns catchUpResponseType
func (r *catchUpResponse) Type() byte {
	return catchUpResponseType
}

// ToConsensusMessage converts the catchUpResponse into a network-level consensus message
func (r *catchUpResponse) ToConsensusMessage() (*ConsensusMessage, error) {
	enc, err := scale.Encode(r)
	if err != nil {
		return nil, err
	}

	return &ConsensusMessage{
		Data: append([]byte{catchUpResponseType}, enc...),
	}, nil
}
