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
	"bytes"
	"fmt"
	"io"
	"math/big"

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
	Vote  *Vote
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
	return fmt.Sprintf("hash=%s number=%d authorityID=0x%x", m.Hash, m.Number, m.AuthorityID)
}

// Decode SCALE decodes the data into a SignedMessage
func (m *SignedMessage) Decode(r io.Reader) (err error) {
	m.Stage, err = subround(0).Decode(r)
	if err != nil {
		return err
	}

	vote, err := new(Vote).Decode(r)
	if err != nil {
		return err
	}

	m.Hash = vote.hash
	m.Number = vote.number

	sig, err := common.Read64Bytes(r)
	if err != nil {
		return err
	}

	copy(m.Signature[:], sig[:])

	id, err := common.Read32Bytes(r)
	if err != nil {
		return err
	}

	copy(m.AuthorityID[:], id[:])
	return nil
}

// VoteMessage represents a network-level vote message
// https://github.com/paritytech/substrate/blob/master/client/finality-grandpa/src/communication/gossip.rs#L336
type VoteMessage struct {
	Round   uint64
	SetID   uint64
	Message *SignedMessage
}

// Decode SCALE decodes the data into a VoteMessage
func (v *VoteMessage) Decode(r io.Reader) (err error) {
	v.Round, err = common.ReadUint64(r)
	if err != nil {
		return err
	}

	v.SetID, err = common.ReadUint64(r)
	if err != nil {
		return err
	}

	v.Message = new(SignedMessage)
	err = v.Message.Decode(r)
	return err
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

// Encode SCALE encodes the AuthData
func (d *AuthData) Encode() ([]byte, error) {
	return append(d.Signature[:], d.AuthorityID[:]...), nil
}

// Decode SCALE decodes the data into an AuthData
func (d *AuthData) Decode(r io.Reader) error {
	sig, err := common.Read64Bytes(r)
	if err != nil {
		return err
	}

	copy(d.Signature[:], sig[:])

	id, err := common.Read32Bytes(r)
	if err != nil {
		return err
	}

	copy(d.AuthorityID[:], id[:])
	return nil
}

// CommitMessage represents a network finalisation message
type CommitMessage struct {
	Round      uint64
	SetID      uint64
	Vote       *Vote
	Precommits []*Vote
	AuthData   []*AuthData
}

// Decode SCALE decodes the data into a CommitMessage
func (f *CommitMessage) Decode(r io.Reader) (err error) {
	f.Round, err = common.ReadUint64(r)
	if err != nil {
		return err
	}

	f.SetID, err = common.ReadUint64(r)
	if err != nil {
		return err
	}

	f.Vote, err = new(Vote).Decode(r)
	if err != nil {
		return err
	}

	sd := &scale.Decoder{Reader: r}
	numPrecommits, err := sd.Decode(new(big.Int))
	if err != nil {
		return err
	}

	f.Precommits = make([]*Vote, numPrecommits.(*big.Int).Int64())
	for i := range f.Precommits {
		f.Precommits[i], err = new(Vote).Decode(r)
		if err != nil {
			return err
		}
	}

	numAuthData, err := sd.Decode(new(big.Int))
	if err != nil {
		return err
	}

	if numAuthData.(*big.Int).Cmp(numPrecommits.(*big.Int)) != 0 {
		return ErrPrecommitSignatureMismatch
	}

	f.AuthData = make([]*AuthData, numAuthData.(*big.Int).Int64())
	for i := range f.AuthData {
		f.AuthData[i] = new(AuthData)
		err = f.AuthData[i].Decode(r)
		if err != nil {
			return err
		}
	}

	return nil
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

func (s *Service) newCommitMessage(header *types.Header, round uint64) *CommitMessage {
	just := s.justification[round]
	precommits, authData := justificationToCompact(just)
	return &CommitMessage{
		Round:      round,
		Vote:       NewVoteFromHeader(header),
		Precommits: precommits,
		AuthData:   authData,
	}
}

func justificationToCompact(just []*SignedPrecommit) ([]*Vote, []*AuthData) {
	precommits := make([]*Vote, len(just))
	authData := make([]*AuthData, len(just))

	for i, j := range just {
		precommits[i] = j.Vote
		authData[i] = &AuthData{
			Signature:   j.Signature,
			AuthorityID: j.AuthorityID,
		}
	}

	return precommits, authData
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
	PreVoteJustification   []*SignedPrecommit
	PreCommitJustification []*SignedPrecommit
	Hash                   common.Hash
	Number                 uint32
}

func (s *Service) newCatchUpResponse(round, setID uint64) (*catchUpResponse, error) {
	header, err := s.blockState.GetFinalizedHeader(round, setID)
	if err != nil {
		return nil, err
	}

	has, err := s.blockState.HasJustification(header.Hash())
	if err != nil {
		return nil, err
	}

	if !has {
		return nil, ErrNoJustification
	}

	just, err := s.blockState.GetJustification(header.Hash())
	if err != nil {
		return nil, err
	}

	r := &bytes.Buffer{}
	sd := &scale.Decoder{Reader: r}
	_, err = r.Write(just)
	if err != nil {
		return nil, err
	}

	d, err := sd.Decode([]*SignedPrecommit{})
	if err != nil {
		return nil, err
	}
	pvj := d.([]*SignedPrecommit)

	d, err = sd.Decode([]*SignedPrecommit{})
	if err != nil {
		return nil, err
	}
	pcj := d.([]*SignedPrecommit)

	return &catchUpResponse{
		SetID:                  setID,
		Round:                  round,
		PreVoteJustification:   pvj,
		PreCommitJustification: pcj,
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
