// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"fmt"

	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto/ed25519"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

// GrandpaMessage is implemented by all GRANDPA network messages
type GrandpaMessage interface { //nolint:revive
	ToConsensusMessage() (*network.ConsensusMessage, error)
}

// NewGrandpaMessage returns a new VaryingDataType to represent a GrandpaMessage
func newGrandpaMessage() scale.VaryingDataType {
	return scale.MustNewVaryingDataType(
		VoteMessage{}, CommitMessage{}, NeighbourMessage{},
		CatchUpRequest{}, CatchUpResponse{})
}

// FullVote represents a vote with additional information about the state
// this is encoded and signed and the signature is included in SignedMessage
type FullVote struct {
	Stage Subround
	Vote  Vote
	Round uint64
	SetID uint64
}

// SignedMessage represents a block hash and number signed by an authority
type SignedMessage struct {
	Stage       Subround // 0 for pre-vote, 1 for pre-commit, 2 for primary proposal
	Hash        common.Hash
	Number      uint32
	Signature   [64]byte // ed25519.SignatureLength
	AuthorityID ed25519.PublicKeyBytes
}

// String returns the SignedMessage as a string
func (m SignedMessage) String() string {
	return fmt.Sprintf("stage=%s hash=%s number=%d authorityID=%s", m.Stage, m.Hash, m.Number, m.AuthorityID)
}

// VoteMessage represents a network-level vote message
// https://github.com/paritytech/substrate/blob/master/client/finality-grandpa/src/communication/gossip.rs#L336
type VoteMessage struct {
	Round   uint64
	SetID   uint64
	Message SignedMessage
}

// Index Returns VDT index
func (v VoteMessage) Index() uint { return 0 }

// ToConsensusMessage converts the VoteMessage into a network-level consensus message
func (v *VoteMessage) ToConsensusMessage() (*ConsensusMessage, error) {
	msg := newGrandpaMessage()
	err := msg.Set(*v)
	if err != nil {
		return nil, err
	}

	enc, err := scale.Marshal(msg)
	if err != nil {
		return nil, err
	}

	return &ConsensusMessage{
		Data: enc,
	}, nil
}

// NeighbourMessage represents a network-level neighbour message
type NeighbourMessage struct {
	Version byte
	Round   uint64
	SetID   uint64
	Number  uint32
}

// Index Returns VDT index
func (m NeighbourMessage) Index() uint { return 2 }

// ToConsensusMessage converts the NeighbourMessage into a network-level consensus message
func (m *NeighbourMessage) ToConsensusMessage() (*network.ConsensusMessage, error) {
	msg := newGrandpaMessage()
	err := msg.Set(*m)
	if err != nil {
		return nil, err
	}

	enc, err := scale.Marshal(msg)
	if err != nil {
		return nil, err
	}

	return &ConsensusMessage{
		Data: enc,
	}, nil
}

// AuthData represents signature data within a CommitMessage to be paired with a precommit
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

// Index Returns VDT index
func (f CommitMessage) Index() uint { return 1 }

// ToConsensusMessage converts the CommitMessage into a network-level consensus message
func (f *CommitMessage) ToConsensusMessage() (*ConsensusMessage, error) {
	msg := newGrandpaMessage()
	err := msg.Set(*f)
	if err != nil {
		return nil, err
	}

	enc, err := scale.Marshal(msg)
	if err != nil {
		return nil, err
	}

	return &ConsensusMessage{
		Data: enc,
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

// CatchUpRequest struct to represent a CatchUpRequest message
type CatchUpRequest struct {
	Round uint64
	SetID uint64
}

func newCatchUpRequest(round, setID uint64) *CatchUpRequest {
	return &CatchUpRequest{
		Round: round,
		SetID: setID,
	}
}

// Index Returns VDT index
func (r CatchUpRequest) Index() uint { return 3 }

// ToConsensusMessage converts the catchUpRequest into a network-level consensus message
func (r *CatchUpRequest) ToConsensusMessage() (*ConsensusMessage, error) {
	msg := newGrandpaMessage()
	err := msg.Set(*r)
	if err != nil {
		return nil, err
	}

	enc, err := scale.Marshal(msg)
	if err != nil {
		return nil, err
	}

	return &ConsensusMessage{
		Data: enc,
	}, nil
}

// CatchUpResponse struct to represent a CatchUpResponse message
type CatchUpResponse struct {
	SetID                  uint64
	Round                  uint64
	PreVoteJustification   []SignedVote
	PreCommitJustification []SignedVote
	Hash                   common.Hash
	Number                 uint32
}

func (s *Service) newCatchUpResponse(round, setID uint64) (*CatchUpResponse, error) {
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

	return &CatchUpResponse{
		SetID:                  setID,
		Round:                  round,
		PreVoteJustification:   pvs,
		PreCommitJustification: pcs,
		Hash:                   header.Hash(),
		Number:                 uint32(header.Number.Uint64()),
	}, nil
}

// Index Returns VDT index
func (r CatchUpResponse) Index() uint { return 4 }

// ToConsensusMessage converts the catchUpResponse into a network-level consensus message
func (r *CatchUpResponse) ToConsensusMessage() (*ConsensusMessage, error) {
	msg := newGrandpaMessage()
	err := msg.Set(*r)
	if err != nil {
		return nil, err
	}

	enc, err := scale.Marshal(msg)
	if err != nil {
		return nil, err
	}

	return &ConsensusMessage{
		Data: enc,
	}, nil
}
