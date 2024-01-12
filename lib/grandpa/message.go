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
type GrandpaMessage interface {
	ToConsensusMessage() (*network.ConsensusMessage, error)
}

type grandpaMessages interface {
	VoteMessage | CommitMessage | VersionedNeighbourPacket | CatchUpRequest | CatchUpResponse
}

type grandpaMessage struct {
	inner any
}

func setgrandpaMessage[Value grandpaMessages](mvdt *grandpaMessage, value Value) {
	mvdt.inner = value
}

func (mvdt *grandpaMessage) SetValue(value any) (err error) {
	switch value := value.(type) {
	case VoteMessage:
		setgrandpaMessage(mvdt, value)
		return

	case CommitMessage:
		setgrandpaMessage(mvdt, value)
		return

	case VersionedNeighbourPacket:
		setgrandpaMessage(mvdt, value)
		return

	case CatchUpRequest:
		setgrandpaMessage(mvdt, value)
		return

	case CatchUpResponse:
		setgrandpaMessage(mvdt, value)
		return

	default:
		return fmt.Errorf("unsupported type")
	}
}

func (mvdt *grandpaMessage) IndexValue() (index uint, value any, err error) {
	switch mvdt.inner.(type) {
	case VoteMessage:
		return 0, mvdt.inner, nil

	case CommitMessage:
		return 1, mvdt.inner, nil

	case VersionedNeighbourPacket:
		return 2, mvdt.inner, nil

	case CatchUpRequest:
		return 3, mvdt.inner, nil

	case CatchUpResponse:
		return 4, mvdt.inner, nil

	}
	return 0, nil, scale.ErrUnsupportedVaryingDataTypeValue
}

func (mvdt *grandpaMessage) Value() (value any, err error) {
	_, value, err = mvdt.IndexValue()
	return
}
func (mvdt *grandpaMessage) ValueAt(index uint) (value any, err error) {
	switch index {
	case 0:
		return *new(VoteMessage), nil

	case 1:
		return *new(CommitMessage), nil

	case 2:
		return *new(VersionedNeighbourPacket), nil

	case 3:
		return *new(CatchUpRequest), nil

	case 4:
		return *new(CatchUpResponse), nil

	}
	return nil, scale.ErrUnknownVaryingDataTypeValue
}

// NewGrandpaMessage returns a new VaryingDataType to represent a GrandpaMessage
func newGrandpaMessage() grandpaMessage {
	// return scale.MustNewVaryingDataType(
	// 	VoteMessage{}, CommitMessage{}, newVersionedNeighbourPacket(),
	// 	CatchUpRequest{}, CatchUpResponse{})
	return grandpaMessage{}
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
	BlockHash   common.Hash
	Number      uint32
	Signature   [64]byte // ed25519.SignatureLength
	AuthorityID ed25519.PublicKeyBytes
}

// String returns the SignedMessage as a string
func (m SignedMessage) String() string {
	return fmt.Sprintf("stage=%s hash=%s number=%d authorityID=%s", m.Stage, m.BlockHash, m.Number, m.AuthorityID)
}

// VoteMessage represents a network-level vote message
// https://github.com/paritytech/substrate/blob/master/client/finality-grandpa/src/communication/gossip.rs#L336
type VoteMessage struct {
	Round   uint64
	SetID   uint64
	Message SignedMessage
}

func (v VoteMessage) String() string {
	return fmt.Sprintf("round=%d, setID=%d, message={%s}", v.Round, v.SetID, v.Message)
}

// Index returns VDT index
func (VoteMessage) Index() uint { return 0 }

// ToConsensusMessage converts the VoteMessage into a network-level consensus message
func (v *VoteMessage) ToConsensusMessage() (*ConsensusMessage, error) {
	msg := newGrandpaMessage()
	err := msg.SetValue(*v)
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

// VersionedNeighbourPacket represents the enum of neighbour messages
type VersionedNeighbourPacketValues interface {
	NeighbourPacketV1
}

type VersionedNeighbourPacket struct {
	inner any
}

func setVersionedNeighbourPacket[Value VersionedNeighbourPacketValues](mvdt *VersionedNeighbourPacket, value Value) {
	mvdt.inner = value
}

func (mvdt *VersionedNeighbourPacket) SetValue(value any) (err error) {
	switch value := value.(type) {
	case NeighbourPacketV1:
		setVersionedNeighbourPacket(mvdt, value)
		return
	default:
		return fmt.Errorf("unsupported type")
	}
}

func (mvdt *VersionedNeighbourPacket) IndexValue() (index uint, value any, err error) {
	switch mvdt.inner.(type) {
	case NeighbourPacketV1:
		return 1, mvdt.inner, nil
	}
	return 0, nil, scale.ErrUnsupportedVaryingDataTypeValue
}

func (mvdt *VersionedNeighbourPacket) Value() (value any, err error) {
	_, value, err = mvdt.IndexValue()
	return
}

func (mvdt *VersionedNeighbourPacket) ValueAt(index uint) (value any, err error) {
	switch index {
	case 1:
		return *new(NeighbourPacketV1), nil
	}
	return nil, scale.ErrUnknownVaryingDataTypeValue
}

// // Index returns VDT index
// func (VersionedNeighbourPacket) Index() uint { return 2 }

// func (vnp VersionedNeighbourPacket) String() string {
// 	val, err := vnp.Value()
// 	if err != nil {
// 		return "VersionedNeighbourPacket()"
// 	}

// 	return fmt.Sprintf("VersionedNeighbourPacket(%s)", val)
// }

// func newVersionedNeighbourPacket() VersionedNeighbourPacket {
// 	vdt := scale.MustNewVaryingDataType(NeighbourPacketV1{})

// 	return VersionedNeighbourPacket(vdt)
// }

// // Set updates the current VDT value to be `val`
// func (vnp *VersionedNeighbourPacket) Set(val scale.VaryingDataTypeValue) (err error) {
// 	vdt := scale.VaryingDataType(*vnp)
// 	err = vdt.Set(val)
// 	if err != nil {
// 		return fmt.Errorf("setting varying data type value: %w", err)
// 	}
// 	*vnp = VersionedNeighbourPacket(vdt)
// 	return nil
// }

// // Value returns the current VDT value
// func (vnp *VersionedNeighbourPacket) Value() (val scale.VaryingDataTypeValue, err error) {
// 	vdt := scale.VaryingDataType(*vnp)
// 	return vdt.Value()
// }

// NeighbourPacketV1 represents a network-level neighbour message
// currently, round and setID represents a struct containing an u64
// https://github.com/paritytech/substrate/blob/master/client/finality-grandpa/src/communication/mod.rs#L660
type NeighbourPacketV1 struct {
	Round  uint64
	SetID  uint64
	Number uint32
}

// // Index returns VDT index
// func (NeighbourPacketV1) Index() uint { return 1 }

func (m NeighbourPacketV1) String() string {
	return fmt.Sprintf("NeighbourPacketV1{Round=%d, SetID=%d, Number=%d}", m.Round, m.SetID, m.Number)
}

// ToConsensusMessage converts the NeighbourMessage into a network-level consensus message
func (m *NeighbourPacketV1) ToConsensusMessage() (*network.ConsensusMessage, error) {
	versionedNeighbourPacket := VersionedNeighbourPacket{}
	err := versionedNeighbourPacket.SetValue(*m)
	if err != nil {
		return nil, fmt.Errorf("setting neighbour packet v1: %w", err)
	}

	msg := newGrandpaMessage()
	err = msg.SetValue(versionedNeighbourPacket)
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

func (a *AuthData) String() string {
	return fmt.Sprintf("AuthData{Signature=0x%x, AuthorityID=%s}", a.Signature, a.AuthorityID)
}

// CommitMessage represents a network finalisation message
type CommitMessage struct {
	Round      uint64
	SetID      uint64
	Vote       Vote
	Precommits []Vote
	AuthData   []AuthData
}

func (s *Service) newCommitMessage(header *types.Header, round, setID uint64) (*CommitMessage, error) {
	pcs, err := s.grandpaState.GetPrecommits(round, setID)
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

// // Index returns VDT index
// func (CommitMessage) Index() uint { return 1 }

func (m CommitMessage) String() string {
	return fmt.Sprintf("CommitMessage{Round=%d, SetID=%d, Vote={%s}, Precommits=%v, AuthData=%v}",
		m.Round, m.SetID, m.Vote, m.Precommits, m.AuthData)
}

// ToConsensusMessage converts the CommitMessage into a network-level consensus message
func (m *CommitMessage) ToConsensusMessage() (*ConsensusMessage, error) {
	msg := newGrandpaMessage()
	err := msg.SetValue(*m)
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

// // Index returns VDT index
// func (CatchUpRequest) Index() uint { return 3 }

func (r CatchUpRequest) String() string {
	return fmt.Sprintf("CatchUpRequest{Round=%d, SetID=%d}", r.Round, r.SetID)
}

// ToConsensusMessage converts the catchUpRequest into a network-level consensus message
func (r *CatchUpRequest) ToConsensusMessage() (*ConsensusMessage, error) {
	msg := newGrandpaMessage()
	err := msg.SetValue(*r)
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
		Number:                 uint32(header.Number),
	}, nil
}

// // Index returns VDT index
// func (CatchUpResponse) Index() uint { return 4 }

func (r CatchUpResponse) String() string {
	return fmt.Sprintf("CatchUpResponse{SetID=%d, Round=%d, PreVoteJustification=%v, "+
		"PreCommitJustification=%v, Hash=%s, Number=%d}",
		r.SetID, r.Round, r.PreVoteJustification, r.PreCommitJustification, r.Hash, r.Number)
}

// ToConsensusMessage converts the catchUpResponse into a network-level consensus message
func (r *CatchUpResponse) ToConsensusMessage() (*ConsensusMessage, error) {
	msg := newGrandpaMessage()
	err := msg.SetValue(*r)
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
