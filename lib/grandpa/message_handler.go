// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"bytes"
	"errors"
	"fmt"
	"math/big"
	"reflect"

	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/blocktree"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto/ed25519"
	"github.com/ChainSafe/gossamer/pkg/scale"

	"github.com/libp2p/go-libp2p-core/peer"
)

// MessageHandler handles GRANDPA consensus messages
type MessageHandler struct {
	grandpa    *Service
	blockState BlockState
}

// NewMessageHandler returns a new MessageHandler
func NewMessageHandler(grandpa *Service, blockState BlockState) *MessageHandler {
	return &MessageHandler{
		grandpa:    grandpa,
		blockState: blockState,
	}
}

// HandleMessage handles a GRANDPA consensus message
// if it is a CommitMessage, it updates the BlockState
// if it is a VoteMessage, it sends it to the GRANDPA service
func (h *MessageHandler) handleMessage(from peer.ID, m GrandpaMessage) (network.NotificationsMessage, error) {
	logger.Tracef("handling grandpa message: %v", m)

	switch msg := m.(type) {
	case *VoteMessage:
		// send vote message to grandpa service
		h.grandpa.in <- &networkVoteMessage{
			from: from,
			msg:  msg,
		}

		return nil, nil
	case *CommitMessage:
		return nil, h.handleCommitMessage(msg)
	case *NeighbourMessage:
		return nil, h.handleNeighbourMessage(msg)
	case *CatchUpRequest:
		return h.handleCatchUpRequest(msg)
	case *CatchUpResponse:
		return nil, h.handleCatchUpResponse(msg)
	default:
		return nil, ErrInvalidMessageType
	}
}

func (h *MessageHandler) handleNeighbourMessage(msg *NeighbourMessage) error {
	currFinalized, err := h.blockState.GetFinalisedHeader(0, 0)
	if err != nil {
		return err
	}

	// ignore neighbour messages where our best finalised number is greater than theirs
	if uint32(currFinalized.Number.Int64()) >= msg.Number {
		return nil
	}

	// TODO; determine if there is some reason we don't receive justifications in responses near the head (usually),
	// and remove the following code if it's fixed. (#1815)
	head, err := h.blockState.BestBlockNumber()
	if err != nil {
		return err
	}

	// ignore neighbour messages that are above our head
	if int64(msg.Number) > head.Int64() {
		return nil
	}

	logger.Debugf("got neighbour message with number %d, set id %d and round %d", msg.Number, msg.SetID, msg.Round)
	// TODO: should we send a justification request here? potentially re-connect this to sync package? (#1815)
	return nil
}

func (h *MessageHandler) handleCommitMessage(msg *CommitMessage) error {
	logger.Debugf("received commit message %v", msg)
	if has, _ := h.blockState.HasFinalisedBlock(msg.Round, h.grandpa.state.setID); has {
		return nil
	}

	// check justification here
	if err := h.verifyCommitMessageJustification(msg); err != nil {
		if errors.Is(err, blocktree.ErrStartNodeNotFound) {
			// we haven't synced the committed block yet, add this to the tracker for later processing
			h.grandpa.tracker.addCommit(msg)
		}
		return err
	}

	// set finalised head for round in db
	if err := h.blockState.SetFinalisedHash(msg.Vote.Hash, msg.Round, h.grandpa.state.setID); err != nil {
		return err
	}

	pcs, err := compactToJustification(msg.Precommits, msg.AuthData)
	if err != nil {
		return err
	}

	if err = h.grandpa.grandpaState.SetPrecommits(msg.Round, msg.SetID, pcs); err != nil {
		return err
	}

	// TODO: re-add catch-up logic (#1531)
	return nil
}

func (h *MessageHandler) handleCatchUpRequest(msg *CatchUpRequest) (*ConsensusMessage, error) {
	if !h.grandpa.authority {
		return nil, nil
	}

	logger.Debugf("received catch up request for round %d and set id %d",
		msg.Round, msg.SetID)

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

	logger.Debugf(
		"sending catch up response with hash %s for round %d and set id %d",
		resp.Hash, msg.Round, msg.SetID)
	return resp.ToConsensusMessage()
}

func (h *MessageHandler) handleCatchUpResponse(msg *CatchUpResponse) error {
	if !h.grandpa.authority {
		return nil
	}

	logger.Debugf(
		"received catch up response with hash %s for round %d and set id %d",
		msg.Hash, msg.Round, msg.SetID)

	// TODO: re-add catch-up logic (#1531)
	if true {
		return nil
	}

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

	if msg.Hash.IsEmpty() || msg.Number == 0 {
		return ErrGHOSTlessCatchUp
	}

	if err = h.verifyCatchUpResponseCompletability(prevote, msg.Hash); err != nil {
		return err
	}

	// set prevotes and precommits in db
	if err = h.grandpa.grandpaState.SetPrevotes(msg.Round, msg.SetID, msg.PreVoteJustification); err != nil {
		return err
	}

	if err = h.grandpa.grandpaState.SetPrecommits(msg.Round, msg.SetID, msg.PreCommitJustification); err != nil {
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
	logger.Debugf("caught up to round; unpaused service and grandpa state round is %d", h.grandpa.state.round)
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

		err := h.verifyJustification(just, fm.Round, h.grandpa.state.setID, precommit)
		if err != nil {
			continue
		}

		isDescendant, err := h.blockState.IsDescendantOf(fm.Vote.Hash, just.Vote.Hash)
		if err != nil {
			logger.Warnf("verifyCommitMessageJustification: %s", err)
			continue
		}

		if isDescendant {
			count++
		}
	}

	// confirm total # signatures >= grandpa threshold
	if uint64(count) < h.grandpa.state.threshold() {
		logger.Debugf(
			"minimum votes not met for finalisation message. Need %d votes and received %d votes.",
			h.grandpa.state.threshold(), count)
		return ErrMinVotesNotMet
	}

	logger.Debugf("validated commit message: %v", fm)
	return nil
}

func (h *MessageHandler) verifyPreVoteJustification(msg *CatchUpResponse) (common.Hash, error) {
	// verify pre-vote justification, returning the pre-voted block if there is one
	votes := make(map[common.Hash]uint64)

	for i := range msg.PreVoteJustification {
		just := &msg.PreVoteJustification[i]
		err := h.verifyJustification(just, msg.Round, msg.SetID, prevote)
		if err != nil {
			continue
		}

		votes[just.Vote.Hash]++
	}

	var prevote common.Hash
	for hash, count := range votes {
		if count >= h.grandpa.state.threshold() {
			prevote = hash
			break
		}
	}

	if prevote.IsEmpty() {
		return prevote, ErrMinVotesNotMet
	}

	return prevote, nil
}

func (h *MessageHandler) verifyPreCommitJustification(msg *CatchUpResponse) error {
	// verify pre-commit justification
	count := 0
	for i := range msg.PreCommitJustification {
		just := &msg.PreCommitJustification[i]
		err := h.verifyJustification(just, msg.Round, msg.SetID, precommit)
		if err != nil {
			continue
		}

		if just.Vote.Hash == msg.Hash && just.Vote.Number == msg.Number {
			count++
		}
	}

	if uint64(count) < h.grandpa.state.threshold() {
		return ErrMinVotesNotMet
	}

	return nil
}

func (h *MessageHandler) verifyJustification(just *SignedVote, round, setID uint64, stage Subround) error {
	// verify signature
	msg, err := scale.Marshal(FullVote{
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
func (s *Service) VerifyBlockJustification(hash common.Hash, justification []byte) error {
	fj := Justification{}
	err := scale.Unmarshal(justification, &fj)
	if err != nil {
		return err
	}

	setID, err := s.grandpaState.GetSetIDByBlockNumber(big.NewInt(int64(fj.Commit.Number)))
	if err != nil {
		return fmt.Errorf("cannot get set ID from block number: %w", err)
	}

	has, err := s.blockState.HasFinalisedBlock(fj.Round, setID)
	if err != nil {
		return err
	}

	if has {
		return fmt.Errorf("already have finalised block with setID=%d and round=%d", setID, fj.Round)
	}

	auths, err := s.grandpaState.GetAuthorities(setID)
	if err != nil {
		return fmt.Errorf("cannot get authorities for set ID: %w", err)
	}

	logger.Debugf(
		"verifying justification: set id %d, round %d, hash %s, number %d, sig count %d",
		setID, fj.Round, fj.Commit.Hash, fj.Commit.Number, len(fj.Commit.Precommits))

	if len(fj.Commit.Precommits) < (2 * len(auths) / 3) {
		return ErrMinVotesNotMet
	}

	for _, just := range fj.Commit.Precommits {
		// check if vote was for descendant of committed block
		isDescendant, err := s.blockState.IsDescendantOf(hash, just.Vote.Hash)
		if err != nil {
			return err
		}

		if !isDescendant {
			return ErrPrecommitBlockMismatch
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
		msg, err := scale.Marshal(FullVote{
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

	err = s.blockState.SetFinalisedHash(hash, fj.Round, setID)
	if err != nil {
		return err
	}

	logger.Debugf(
		"set finalised block with hash %s, round %d and set id %d",
		hash, fj.Round, setID)
	return nil
}

func isInAuthSet(auth *ed25519.PublicKey, set []types.GrandpaVoter) bool {
	for _, a := range set {
		if bytes.Equal(a.Key.Encode(), auth.Encode()) {
			return true
		}
	}

	return false
}
