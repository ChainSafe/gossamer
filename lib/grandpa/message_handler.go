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
	"math/big"
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
// if it is a CommitMessage, it updates the BlockState
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
	case voteType:
		vm, ok := m.(*VoteMessage)
		if h.grandpa != nil && ok {
			// send vote message to grandpa service
			h.grandpa.in <- vm
		}
	case commitType:
		if fm, ok := m.(*CommitMessage); ok {
			return h.handleCommitMessage(fm)
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
	logger.Debug("got neighbour message", "number", msg.Number, "set id", msg.SetID, "round", msg.Round)
	h.blockNumToSetID.Store(msg.Number, msg.SetID)
	h.grandpa.network.SendJustificationRequest(from, msg.Number)

	// TODO; determine if there is some reason we don't receive justifications in responses near the head (usually),
	// and remove the following code if it's fixed.
	head, err := h.grandpa.blockState.BestBlockNumber()
	if err != nil {
		return err
	}

	// don't finalise too close to head, until we add justification request + verification functionality.
	// this prevents us from marking the wrong block as final and getting stuck on the wrong chain
	if uint32(head.Int64())-4 < msg.Number {
		return nil
	}

	// TODO: instead of assuming the finalised hash is the one we currently know about,
	// request the justification from the network before setting it as finalised.
	hash, err := h.grandpa.blockState.GetHashByNumber(big.NewInt(int64(msg.Number)))
	if err != nil {
		return err
	}

	if err = h.grandpa.blockState.SetFinalizedHash(hash, msg.Round, msg.SetID); err != nil {
		return err
	}

	if err = h.grandpa.blockState.SetFinalizedHash(hash, 0, 0); err != nil {
		return err
	}

	logger.Info("🔨 finalised block", "number", msg.Number, "hash", hash)
	return nil
}

func (h *MessageHandler) handleCommitMessage(msg *CommitMessage) (*ConsensusMessage, error) {
	logger.Debug("received finalisation message", "round", msg.Round, "hash", msg.Vote.hash)

	if has, _ := h.blockState.HasFinalizedBlock(msg.Round, h.grandpa.state.setID); has {
		return nil, nil
	}

	// check justification here
	err := h.verifyCommitMessageJustification(msg)
	if err != nil {
		return nil, err
	}

	// set finalised head for round in db
	err = h.blockState.SetFinalizedHash(msg.Vote.hash, msg.Round, h.grandpa.state.setID)
	if err != nil {
		return nil, err
	}

	// set latest finalised head in db
	err = h.blockState.SetFinalizedHash(msg.Vote.hash, 0, 0)
	if err != nil {
		return nil, err
	}

	// check if msg has same setID but is 2 or more rounds ahead of us, if so, return catch-up request to send
	if msg.Round > h.grandpa.state.round+1 && !h.grandpa.paused.Load().(bool) { // TODO: CommitMessage does not have setID, confirm this is correct
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

// decodeMessage decodes a network-level consensus message into a GRANDPA VoteMessage or CommitMessage
func decodeMessage(msg *ConsensusMessage) (m GrandpaMessage, err error) {
	var (
		mi interface{}
		ok bool
	)

	switch msg.Data[0] {
	case voteType:
		m = &VoteMessage{}
		_, err = scale.Decode(msg.Data[1:], m)
	case commitType:
		r := &bytes.Buffer{}
		_, _ = r.Write(msg.Data[1:])
		cm := &CommitMessage{}
		err = cm.Decode(r)
		m = cm
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

func (h *MessageHandler) verifyCommitMessageJustification(fm *CommitMessage) error {
	if len(fm.Precommits) != len(fm.AuthData) {
		return ErrPrecommitSignatureMismatch
	}

	count := 0
	for i, pc := range fm.Precommits {
		just := &SignedPrecommit{
			Vote:        pc,
			Signature:   fm.AuthData[i].Signature,
			AuthorityID: fm.AuthData[i].AuthorityID,
		}

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
		logger.Error("minimum votes not met for finalisation message", "votes needed", h.grandpa.state.threshold(),
			"votes received", len(fm.Precommits))
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

func (h *MessageHandler) verifyJustification(just *SignedPrecommit, round, setID uint64, stage subround) error {
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
	for _, auth := range h.grandpa.authorities() {
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
	fj := new(Justification)
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
