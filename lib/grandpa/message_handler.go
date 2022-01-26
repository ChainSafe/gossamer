// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"bytes"
	"errors"
	"fmt"
	"math/big"
	"reflect"

	"github.com/ChainSafe/chaindb"
	"github.com/ChainSafe/gossamer/dot/telemetry"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/blocktree"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto/ed25519"
	"github.com/ChainSafe/gossamer/pkg/scale"

	"github.com/libp2p/go-libp2p-core/peer"
)

const catchupThreshold = 2

// MessageHandler handles GRANDPA consensus messages
type MessageHandler struct {
	grandpa    *Service
	catchUp    *catchUp
	blockState BlockState
	telemetry  telemetry.Client
}

// NewMessageHandler returns a new MessageHandler
func NewMessageHandler(grandpa *Service, blockState BlockState, telemetryMailer telemetry.Client) *MessageHandler {
	return &MessageHandler{
		grandpa:    grandpa,
		blockState: blockState,
		catchUp:    newCatchUp(grandpa),
		telemetry:  telemetryMailer,
	}
}

//nolint
// TODO: NotificationMessage is used at places. But NotificationMessage we return is always nil.
// HandleMessage handles a GRANDPA consensus message
// if it is a CommitMessage, it updates the BlockState
// if it is a VoteMessage, it sends it to the GRANDPA service
func (h *MessageHandler) handleMessage(from peer.ID, m GrandpaMessage) error {
	logger.Tracef("handling grandpa message: %v", m)

	switch msg := m.(type) {
	case *VoteMessage:
		// send vote message to grandpa service
		h.grandpa.in <- &networkVoteMessage{
			from: from,
			msg:  msg,
		}

		return nil
	case *CommitMessage:
		return h.handleCommitMessage(msg)
	case *NeighbourMessage:
		// we can afford to not retry handling neighbour message, if it errors.
		return h.handleNeighbourMessage(msg, from)
	case *CatchUpRequest:
		return h.handleCatchUpRequest(msg, from)
	case *CatchUpResponse:
		// err := h.handleCatchUpResponse(msg)
		err := h.catchUp.handleCatchUpResponse(msg)
		if errors.Is(err, blocktree.ErrNodeNotFound) || errors.Is(err, chaindb.ErrKeyNotFound) {
			// TODO: revisit if we need to add these message in synchronous manner
			// or not. If not, change catchUpResponseMessages to a normal map.  #1531
			h.grandpa.tracker.addCatchUpResponse(&networkCatchUpResponseMessage{
				from: from,
				msg:  msg,
			})
			h.catchUp.bestResponse = msg
		} else if err != nil {
			logger.Debugf("could not catchup: %s", err)
		}

		return err
	default:
		return ErrInvalidMessageType
	}
}

func (h *MessageHandler) handleNeighbourMessage(msg *NeighbourMessage, from peer.ID) error {
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

	// we shouldn't send a catch up request for blocks we haven't synced yet
	// as we won't be able to process them. We also receive neighbour messages
	// each time a new block is finalized, so we get them very often.
	if int64(msg.Number) > head.Int64() {
		logger.Debug("ignoring neighbour message, because we have not synced to this block number")
		return nil
	}

	logger.Debugf("got neighbour message with number %d, set id %d and round %d, from: %s ",
		msg.Number, msg.SetID, msg.Round, from)

	highestRound, setID, err := h.blockState.GetHighestRoundAndSetID()
	if err != nil {
		return err
	}

	if msg.SetID != setID {
		return ErrSetIDMismatch
	}

	// catch up only if we are behind by more than catchup threshold
	if (int(msg.Round) - int(highestRound)) > catchupThreshold {
		logger.Debugf("lagging behind by %d rounds", int(msg.Round)-int(highestRound))
		return h.catchUp.do(from, msg.Round, msg.SetID)
	} else {
		logger.Debugf("not lagging behind by more than threshold rounds")
	}

	return nil
}

func (h *MessageHandler) handleCommitMessage(msg *CommitMessage) error {
	logger.Debugf("received commit message, msg: %+v", msg)

	containsPrecommitsSignedBy := make([]string, len(msg.AuthData))
	for i, authData := range msg.AuthData {
		containsPrecommitsSignedBy[i] = authData.AuthorityID.String()
	}

	h.telemetry.SendMessage(
		telemetry.NewAfgReceivedCommit(
			msg.Vote.Hash,
			fmt.Sprint(msg.Vote.Number),
			containsPrecommitsSignedBy,
		),
	)

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

func (h *MessageHandler) handleCatchUpRequest(msg *CatchUpRequest, from peer.ID) error {
	if !h.grandpa.authority {
		return nil
	}

	logger.Debugf("received catch up request for round %d and set id %d, from %s",
		msg.Round, msg.SetID, from)

	logger.Debugf("Our latest round is %d", h.grandpa.state.round)

	if msg.SetID != h.grandpa.state.setID {
		return ErrSetIDMismatch
	}

	if msg.Round > h.grandpa.state.round {
		return ErrInvalidCatchUpRound
	}

	// We don't necessarily have to reply with the round asked in the request, we can reply
	// with our latest round.
	resp, err := h.grandpa.newCatchUpResponse(h.grandpa.state.round, h.grandpa.state.setID)
	if err != nil {
		return err
	}

	cm, err := resp.ToConsensusMessage()
	if err != nil {
		return err
	}

	err = h.grandpa.network.SendMessage(from, cm)
	if err != nil {
		return err
	}

	logger.Debugf(
		"successfully sent catch up response with hash %s for round %d and set id %d, to %s",
		resp.Hash, h.grandpa.state.round, h.grandpa.state.setID, from)

	return nil
}

func getEquivocatoryVoters(votes []AuthData) map[ed25519.PublicKeyBytes]struct{} {
	eqvVoters := make(map[ed25519.PublicKeyBytes]struct{})
	voters := make(map[ed25519.PublicKeyBytes]int, len(votes))

	for _, v := range votes {
		voters[v.AuthorityID]++

		if voters[v.AuthorityID] > 1 {
			eqvVoters[v.AuthorityID] = struct{}{}
		}
	}

	return eqvVoters
}

func (h *MessageHandler) verifyCommitMessageJustification(fm *CommitMessage) error {
	if len(fm.Precommits) != len(fm.AuthData) {
		return ErrPrecommitSignatureMismatch
	}

	eqvVoters := getEquivocatoryVoters(fm.AuthData)

	var count int
	for i, pc := range fm.Precommits {
		_, ok := eqvVoters[fm.AuthData[i].AuthorityID]
		if ok {
			continue
		}

		just := &SignedVote{
			Vote:        pc,
			Signature:   fm.AuthData[i].Signature,
			AuthorityID: fm.AuthData[i].AuthorityID,
		}

		err := verifyJustification(h.grandpa, just, fm.Round, h.grandpa.state.setID, precommit)
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
	if uint64(count)+uint64(len(eqvVoters)) < h.grandpa.state.threshold() {
		logger.Debugf(
			"minimum votes not met for finalisation message. Need %d votes and received %d votes.",
			h.grandpa.state.threshold(), count)
		return ErrMinVotesNotMet
	}

	logger.Debugf("validated commit message: %v", fm)
	return nil
}

func verifyJustification(grandpa *Service, just *SignedVote, round, setID uint64, stage Subround) error {
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

	for _, auth := range grandpa.authorities() {
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

	// threshold is two-thirds the number of authorities,
	// uses the current set of authorities to define the threshold
	threshold := (2 * len(auths) / 3)

	if len(fj.Commit.Precommits) < threshold {
		return ErrMinVotesNotMet
	}

	authPubKeys := make([]AuthData, len(fj.Commit.Precommits))
	for i, pcj := range fj.Commit.Precommits {
		authPubKeys[i] = AuthData{AuthorityID: pcj.AuthorityID}
	}

	equivocatoryVoters := getEquivocatoryVoters(authPubKeys)
	var count int

	logger.Debugf(
		"verifying justification: set id %d, round %d, hash %s, number %d, sig count %d",
		setID, fj.Round, fj.Commit.Hash, fj.Commit.Number, len(fj.Commit.Precommits))

	for _, just := range fj.Commit.Precommits {
		if _, ok := equivocatoryVoters[just.AuthorityID]; ok {
			continue
		}

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

		if !isInAuthSet(pk, auths) {
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

		ok, err := pk.Verify(msg, just.Signature[:])
		if err != nil {
			return err
		}

		if !ok {
			return ErrInvalidSignature
		}

		count++
	}

	if count+len(equivocatoryVoters) < threshold {
		return ErrMinVotesNotMet
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
