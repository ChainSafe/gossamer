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
	"math/big"
	"reflect"

	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/crypto/ed25519"
	"github.com/ChainSafe/gossamer/lib/scale"

	"github.com/libp2p/go-libp2p-core/peer"
)

// MessageHandler handles GRANDPA consensus messages
type MessageHandler struct {
	grandpa    *Service
	blockState BlockState
	catchUp    *catchUp
	responseCh chan<- *catchUpResponse
}

// NewMessageHandler returns a new MessageHandler
func NewMessageHandler(grandpa *Service, blockState BlockState) *MessageHandler {
	responseCh := make(chan *catchUpResponse, 128)

	return &MessageHandler{
		grandpa:    grandpa,
		blockState: blockState,
		catchUp:    newCatchUp(grandpa.authority, grandpa, grandpa.network, responseCh),
		responseCh: responseCh,
	}
}

// HandleMessage handles a GRANDPA consensus message
// if it is a CommitMessage, it updates the BlockState
// if it is a VoteMessage, it sends it to the GRANDPA service
func (h *MessageHandler) handleMessage(from peer.ID, m GrandpaMessage) (network.NotificationsMessage, error) {
	logger.Trace("handling grandpa message", "msg", m)

	switch m.Type() {
	case voteType:
		vm, ok := m.(*VoteMessage)
		if h.grandpa != nil && ok {
			// send vote message to grandpa service
			h.grandpa.in <- &networkVoteMessage{
				from: from,
				msg:  vm,
			}
		}
		return nil, nil
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
			h.catchUp.handleCatchUpResponse(r)
			return nil, nil
		}
	default:
		return nil, ErrInvalidMessageType
	}

	return nil, nil
}

func (h *MessageHandler) handleNeighbourMessage(from peer.ID, msg *NeighbourMessage) error {
	currFinalized, err := h.blockState.GetFinalisedHeader(0, 0)
	if err != nil {
		return err
	}

	// ignore neighbour messages where our best finalised number is greater than theirs
	if uint32(currFinalized.Number.Int64()) >= msg.Number {
		return nil
	}

	// TODO; determine if there is some reason we don't receive justifications in responses near the head (usually),
	// and remove the following code if it's fixed.
	head, err := h.blockState.BestBlockNumber()
	if err != nil {
		return err
	}

	// ignore neighbour messages that are above our head
	if int64(msg.Number) > head.Int64() {
		return nil
	}

	logger.Debug("got neighbour message", "number", msg.Number, "set id", msg.SetID, "round", msg.Round)
	h.catchUp.addNeighbourMessage(from, msg)
	h.grandpa.network.SendJustificationRequest(from, msg.Number)

	// if the peer reports a higher set ID, or the same set ID but a higher round,
	// we have fallen behind and need to initiate catch-up.
	if msg.SetID == h.grandpa.state.setID && msg.Round >= h.grandpa.state.round+2 {
		err = h.catchUp.doCatchUp(from, msg.SetID, msg.Round)
		if err != nil {
			logger.Debug("failed to do catch up", "error", err)
		}
	}

	return nil
}

func (h *MessageHandler) handleCommitMessage(msg *CommitMessage) (*ConsensusMessage, error) {
	logger.Debug("received finalisation message", "msg", msg)

	if has, _ := h.blockState.HasFinalisedBlock(msg.Round, h.grandpa.state.setID); has {
		return nil, nil
	}

	// check justification here
	if err := h.verifyCommitMessageJustification(msg); err != nil {
		return nil, err
	}

	// set finalised head for round in db
	if err := h.blockState.SetFinalisedHash(msg.Vote.Hash, msg.Round, h.grandpa.state.setID); err != nil {
		return nil, err
	}

	pcs, err := compactToJustification(msg.Precommits, msg.AuthData)
	if err != nil {
		return nil, err
	}

	if err = h.grandpa.grandpaState.SetPrecommits(msg.Round, msg.SetID, pcs); err != nil {
		return nil, err
	}

	if msg.Round >= h.grandpa.state.round {
		// set latest finalised head in db
		err = h.blockState.SetFinalisedHash(msg.Vote.Hash, 0, 0)
		if err != nil {
			return nil, err
		}
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
	if !h.grandpa.authority {
		return nil, nil
	}

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

func (h *MessageHandler) handleCatchUpResponse(msg *catchUpResponse) {
	if !h.grandpa.authority {
		return
	}

	logger.Debug("received catch up response", "round", msg.Round, "setID", msg.SetID, "hash", msg.Hash)
	h.responseCh <- msg
}

func (h *MessageHandler) verifyCommitMessageJustification(fm *CommitMessage) error {
	if len(fm.Precommits) != len(fm.AuthData) {
		return ErrPrecommitSignatureMismatch
	}

	count := 0
	for i, pc := range fm.Precommits {
		just := &SignedVote{
			Vote:        pc,
			Signature:   fm.AuthData[i].Signature,
			AuthorityID: fm.AuthData[i].AuthorityID,
		}

		err := verifyJustification(h.grandpa.authorities(), just, fm.Round, h.grandpa.state.setID, precommit)
		if err != nil {
			continue
		}

		isDescendant, err := h.blockState.IsDescendantOf(fm.Vote.Hash, just.Vote.Hash)
		if err != nil {
			logger.Warn("verifyCommitMessageJustification", "error", err)
		}

		if isDescendant {
			count++
		}
	}

	// confirm total # signatures >= grandpa threshold
	if uint64(count) < h.grandpa.state.threshold() {
		logger.Debug("minimum votes not met for finalisation message", "votes needed", h.grandpa.state.threshold(),
			"votes received", count)
		return ErrMinVotesNotMet
	}

	logger.Debug("validated commit message", "msg", fm)
	return nil
}

func (h *MessageHandler) verifyJustification(just *SignedVote, round, setID uint64, stage subround) error {
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

	setID, err := s.grandpaState.GetSetIDByBlockNumber(big.NewInt(int64(fj.Commit.Number)))
	if err != nil {
		return fmt.Errorf("cannot get set ID from block number: %w", err)
	}

	auths, err := s.grandpaState.GetAuthorities(setID)
	if err != nil {
		return fmt.Errorf("cannot get authorities for set ID: %w", err)
	}

	logger.Debug("verifying justification",
		"setID", setID,
		"round", fj.Round,
		"hash", fj.Commit.Hash,
		"number", fj.Commit.Number,
		"sig count", len(fj.Commit.Precommits),
	)

	if len(fj.Commit.Precommits) < (2 * len(auths) / 3) {
		return ErrMinVotesNotMet
	}

	for _, just := range fj.Commit.Precommits {
		if just.Vote.Hash != fj.Commit.Hash {
			return ErrJustificationHashMismatch
		}

		if just.Vote.Number != fj.Commit.Number {
			return ErrJustificationNumberMismatch
		}

		pk, err := ed25519.NewPublicKey(just.AuthorityID[:])
		if err != nil {
			return err
		}

		ok := isInAuthSet(pk, auths)
		if !ok {
			return ErrAuthorityNotInSet
		}

		// verify signature for each precommit
		msg, err := scale.Encode(&FullVote{
			Stage: precommit,
			Vote:  just.Vote,
			Round: fj.Round,
			SetID: setID,
		})
		if err != nil {
			return err
		}

		ok, err = pk.Verify(msg, just.Signature[:])
		if err != nil {
			return err
		}

		if !ok {
			return ErrInvalidSignature
		}
	}

	return nil
}

func isInAuthSet(auth *ed25519.PublicKey, set []*types.GrandpaVoter) bool {
	for _, a := range set {
		if bytes.Equal(a.Key.Encode(), auth.Encode()) {
			return true
		}
	}

	return false
}
