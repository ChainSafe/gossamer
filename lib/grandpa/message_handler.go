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
	"reflect"
	"sync"

	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto/ed25519"
	"github.com/ChainSafe/gossamer/lib/scale"

	"github.com/libp2p/go-libp2p-core/peer"
)

// MessageHandler handles GRANDPA consensus messages
type MessageHandler struct {
	grandpa         *Service
	blockState      BlockState
	blockNumToSetID *sync.Map // map[uint32]uint64
}

// NewMessageHandler returns a new MessageHandler
func NewMessageHandler(grandpa *Service, blockState BlockState) *MessageHandler {
	return &MessageHandler{
		grandpa:         grandpa,
		blockState:      blockState,
		blockNumToSetID: new(sync.Map),
	}
}

// HandleMessage handles a GRANDPA consensus message
// if it is a FinalizationMessage, it updates the BlockState
// if it is a VoteMessage, it sends it to the GRANDPA service
func (h *MessageHandler) handleMessage(from peer.ID, msg *ConsensusMessage) (network.NotificationsMessage, error) {
	if msg == nil || len(msg.Data) == 0 {
		logger.Trace("received nil message or message with nil data")
		return nil, nil
	}

	m, err := decodeMessage(msg)
	if err != nil {
		return nil, err
	}

	logger.Debug("handling grandpa message", "msg", m)

	switch m.Type() {
	case voteType, precommitType:
		vm, ok := m.(*VoteMessage)
		if h.grandpa != nil && ok {
			// send vote message to grandpa service
			h.grandpa.in <- vm
		}
	case finalizationType:
		if fm, ok := m.(*FinalizationMessage); ok {
			return h.handleFinalizationMessage(fm)
		}
	case neighbourType:
		nm, ok := m.(*NeighbourMessage)
		if !ok {
			return nil, nil
		}

		return nil, h.handleNeighbourMessage(from, nm)
	case catchUpRequestType:
		if r, ok := m.(*catchUpRequest); ok {
			return h.handleCatchUpRequest(r)
		}
	case catchUpResponseType:
		if r, ok := m.(*catchUpResponse); ok {
			return nil, h.handleCatchUpResponse(r)
		}
	default:
		return nil, ErrInvalidMessageType
	}

	return nil, nil
}

func (h *MessageHandler) handleNeighbourMessage(from peer.ID, msg *NeighbourMessage) error {
	currFinalized, err := h.grandpa.blockState.GetFinalizedHeader(0, 0)
	if err != nil {
		return err
	}

	if uint32(currFinalized.Number.Int64()) >= msg.Number {
		return nil
	}

	// TODO: make linter british
	logger.Debug("got neighbor message", "number", msg.Number, "set id", msg.SetID, "round", msg.Round)
	h.blockNumToSetID.Store(msg.Number, msg.SetID)
	h.grandpa.network.SendJustificationRequest(from, msg.Number)
	return nil
}

func (h *MessageHandler) handleFinalizationMessage(msg *FinalizationMessage) (*ConsensusMessage, error) {
	logger.Debug("received finalization message", "round", msg.Round, "hash", msg.Vote.hash)

	if has, _ := h.blockState.HasFinalizedBlock(msg.Round, h.grandpa.state.setID); has {
		return nil, nil
	}

	// check justification here
	err := h.verifyFinalizationMessageJustification(msg)
	if err != nil {
		return nil, err
	}

	// set finalized head for round in db
	err = h.blockState.SetFinalizedHash(msg.Vote.hash, msg.Round, h.grandpa.state.setID)
	if err != nil {
		return nil, err
	}

	// set latest finalized head in db
	err = h.blockState.SetFinalizedHash(msg.Vote.hash, 0, 0)
	if err != nil {
		return nil, err
	}

	// check if msg has same setID but is 2 or more rounds ahead of us, if so, return catch-up request to send
	if msg.Round > h.grandpa.state.round+1 && !h.grandpa.paused.Load().(bool) { // TODO: FinalizationMessage does not have setID, confirm this is correct
		h.grandpa.paused.Store(true)
		h.grandpa.state.round = msg.Round + 1
		req := newCatchUpRequest(msg.Round, h.grandpa.state.setID)
		logger.Debug("sending catch-up request; paused service", "round", msg.Round)
		return req.ToConsensusMessage()
	}

	return nil, nil
}

func (h *MessageHandler) handleCatchUpRequest(msg *catchUpRequest) (*ConsensusMessage, error) {
	logger.Debug("received catch up request", "round", msg.Round, "setID", msg.SetID)
	if msg.SetID != h.grandpa.state.setID {
		return nil, ErrSetIDMismatch
	}

	if msg.Round >= h.grandpa.state.round {
		return nil, ErrInvalidCatchUpRound
	}

	resp, err := h.grandpa.newCatchUpResponse(msg.Round, msg.SetID)
	if err != nil {
		return nil, err
	}

	logger.Debug("sending catch up response", "round", msg.Round, "setID", msg.SetID, "hash", resp.Hash)
	return resp.ToConsensusMessage()
}

func (h *MessageHandler) handleCatchUpResponse(msg *catchUpResponse) error {
	logger.Debug("received catch up response", "round", msg.Round, "setID", msg.SetID, "hash", msg.Hash)

	// if we aren't currently expecting a catch up response, return
	if !h.grandpa.paused.Load().(bool) {
		logger.Debug("not currently paused, ignoring catch up response")
		return nil
	}

	if msg.SetID != h.grandpa.state.setID {
		return ErrSetIDMismatch
	}

	if msg.Round != h.grandpa.state.round-1 {
		return ErrInvalidCatchUpResponseRound
	}

	prevote, err := h.verifyPreVoteJustification(msg)
	if err != nil {
		return err
	}

	if err = h.verifyPreCommitJustification(msg); err != nil {
		return err
	}

	if (msg.Hash == common.Hash{}) || msg.Number == 0 {
		return ErrGHOSTlessCatchUp
	}

	if err = h.verifyCatchUpResponseCompletability(prevote, msg.Hash); err != nil {
		return err
	}

	// update state and signal to grandpa we are ready to initiate
	head, err := h.grandpa.blockState.GetHeader(msg.Hash)
	if err != nil {
		return err
	}

	h.grandpa.head = head
	h.grandpa.state.round = msg.Round
	close(h.grandpa.resumed)
	h.grandpa.resumed = make(chan struct{})
	h.grandpa.paused.Store(false)
	logger.Debug("caught up to round; unpaused service", "round", h.grandpa.state.round)
	return nil
}

// verifyCatchUpResponseCompletability verifies that the pre-commit block is a descendant of, or is, the pre-voted block
func (h *MessageHandler) verifyCatchUpResponseCompletability(prevote, precommit common.Hash) error {
	if prevote == precommit {
		return nil
	}

	// check if the current block is a descendant of prevoted block
	isDescendant, err := h.grandpa.blockState.IsDescendantOf(prevote, precommit)
	if err != nil {
		return err
	}

	if !isDescendant {
		return ErrCatchUpResponseNotCompletable
	}

	return nil
}

// decodeMessage decodes a network-level consensus message into a GRANDPA VoteMessage or FinalizationMessage
func decodeMessage(msg *ConsensusMessage) (m GrandpaMessage, err error) {
	var (
		mi interface{}
		ok bool
	)

	switch msg.Data[0] {
	case voteType, precommitType:
		mi, err = scale.Decode(msg.Data[1:], &VoteMessage{Message: new(SignedMessage)})
		if m, ok = mi.(*VoteMessage); !ok {
			return nil, ErrInvalidMessageType
		}
	case finalizationType:
		mi, err = scale.Decode(msg.Data[1:], &FinalizationMessage{})
		if m, ok = mi.(*FinalizationMessage); !ok {
			return nil, ErrInvalidMessageType
		}
	case neighbourType:
		mi, err = scale.Decode(msg.Data[1:], &NeighbourMessage{})
		if m, ok = mi.(*NeighbourMessage); !ok {
			return nil, ErrInvalidMessageType
		}
	case catchUpRequestType:
		mi, err = scale.Decode(msg.Data[1:], &catchUpRequest{})
		if m, ok = mi.(*catchUpRequest); !ok {
			return nil, ErrInvalidMessageType
		}
	case catchUpResponseType:
		mi, err = scale.Decode(msg.Data[1:], &catchUpResponse{})
		if m, ok = mi.(*catchUpResponse); !ok {
			return nil, ErrInvalidMessageType
		}
	default:
		return nil, ErrInvalidMessageType
	}

	if err != nil {
		return nil, err
	}

	return m, nil
}

func (h *MessageHandler) verifyFinalizationMessageJustification(fm *FinalizationMessage) error {
	// verify justifications
	count := 0
	for _, just := range fm.Justification {
		err := h.verifyJustification(just, fm.Round, h.grandpa.state.setID, precommit)
		if err != nil {
			continue
		}

		if just.Vote.hash == fm.Vote.hash && just.Vote.number == fm.Vote.number {
			count++
		}
	}

	// confirm total # signatures >= grandpa threshold
	if uint64(count) < h.grandpa.state.threshold() {
		logger.Error("minimum votes not met for finalization message", "votes needed", h.grandpa.state.threshold(),
			"votes", fm.Justification)
		return ErrMinVotesNotMet
	}
	return nil
}

func (h *MessageHandler) verifyPreVoteJustification(msg *catchUpResponse) (common.Hash, error) {
	// verify pre-vote justification, returning the pre-voted block if there is one
	votes := make(map[common.Hash]uint64)

	for _, just := range msg.PreVoteJustification {
		err := h.verifyJustification(just, msg.Round, msg.SetID, prevote)
		if err != nil {
			continue
		}

		votes[just.Vote.hash]++
	}

	var prevote common.Hash
	for hash, count := range votes {
		if count >= h.grandpa.state.threshold() {
			prevote = hash
			break
		}
	}

	if (prevote == common.Hash{}) {
		return prevote, ErrMinVotesNotMet
	}

	return prevote, nil
}

func (h *MessageHandler) verifyPreCommitJustification(msg *catchUpResponse) error {
	// verify pre-commit justification
	count := 0
	for _, just := range msg.PreCommitJustification {
		err := h.verifyJustification(just, msg.Round, msg.SetID, precommit)
		if err != nil {
			continue
		}

		if just.Vote.hash == msg.Hash && just.Vote.number == msg.Number {
			count++
		}
	}

	if uint64(count) < h.grandpa.state.threshold() {
		return ErrMinVotesNotMet
	}

	return nil
}

func (h *MessageHandler) verifyJustification(just *Justification, round, setID uint64, stage subround) error {
	// verify signature
	msg, err := scale.Encode(&FullVote{
		Stage: stage,
		Vote:  just.Vote,
		Round: round,
		SetID: setID,
	})
	if err != nil {
		return err
	}

	pk, err := ed25519.NewPublicKey(just.AuthorityID[:])
	if err != nil {
		return err
	}

	ok, err := pk.Verify(msg, just.Signature[:])
	if err != nil {
		return err
	}

	if !ok {
		return ErrInvalidSignature
	}

	// verify authority in justification set
	authFound := false
	for _, auth := range h.grandpa.Authorities() {
		justKey, err := just.AuthorityID.Encode()
		if err != nil {
			return err
		}
		if reflect.DeepEqual(auth.Key.Encode(), justKey) {
			authFound = true
			break
		}
	}
	if !authFound {
		return ErrVoterNotFound
	}
	return nil
}

// VerifyBlockJustification verifies the finality justification for a block
func (s *Service) VerifyBlockJustification(justification []byte) error {
	r := &bytes.Buffer{}
	_, _ = r.Write(justification)
	fj := new(FullJustification)
	err := fj.Decode(r)
	if err != nil {
		return err
	}

	logger.Debug("verifiying justification", "round", fj.Round, "hash", fj.Commit.Hash, "number", fj.Commit.Number, "sig count", len(fj.Commit.Precommits))

	for _, just := range fj.Commit.Precommits {
		// TODO: when catch up is done, we should know all the setIDs
		s, has := s.messageHandler.blockNumToSetID.Load(fj.Commit.Number)
		if !has {
			continue
		}

		setID := s.(uint64)

		// verify signature for each precommit
		// TODO: verify authority is in set; this requires updating catch-up to get the right set
		msg, err := scale.Encode(&FullVote{
			Stage: precommit,
			Vote:  just.Vote,
			Round: fj.Round,
			SetID: setID,
		})
		if err != nil {
			return err
		}

		pk, err := ed25519.NewPublicKey(just.AuthorityID[:])
		if err != nil {
			return err
		}

		ok, err := pk.Verify(msg, just.Signature[:])
		if err != nil {
			return err
		}

		if !ok {
			return ErrInvalidSignature
		}
	}

	return nil
}
