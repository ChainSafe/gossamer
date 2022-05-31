// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	"github.com/ChainSafe/gossamer/dot/telemetry"
	"github.com/ChainSafe/gossamer/lib/blocktree"
	"github.com/ChainSafe/gossamer/lib/crypto/ed25519"
	"github.com/ChainSafe/gossamer/lib/grandpa/models"
	"github.com/ChainSafe/gossamer/pkg/scale"

	"github.com/libp2p/go-libp2p-core/peer"
)

// receiveVoteMessages receives messages from the in channel until a grandpa round finishes.
func (s *Service) receiveVoteMessages(ctx context.Context) {
	for {
		select {
		case msg, ok := <-s.in:
			if !ok {
				return
			}

			if msg == nil || msg.Msg == nil {
				continue
			}

			logger.Debugf("received vote message %v from %s", msg.Msg, msg.From)
			vm := msg.Msg

			switch vm.Message.Stage {
			case models.Prevote, models.PrimaryProposal:
				s.telemetry.SendMessage(
					telemetry.NewAfgReceivedPrevote(
						vm.Message.BlockHash,
						fmt.Sprint(vm.Message.Number),
						vm.Message.AuthorityID.String(),
					),
				)
			case models.Precommit:
				s.telemetry.SendMessage(
					telemetry.NewAfgReceivedPrecommit(
						vm.Message.BlockHash,
						fmt.Sprint(vm.Message.Number),
						vm.Message.AuthorityID.String(),
					),
				)
			default:
				logger.Warnf("unsupported stage %s", vm.Message.Stage.String())
			}

			v, err := s.validateVoteMessage(msg.From, vm)
			if err != nil {
				logger.Debugf("failed to validate vote message %v: %s", vm, err)
				continue
			}

			logger.Debugf(
				"validated vote message %v from %s, round %d, subround %d, "+
					"prevote count %d, precommit count %d, votes needed %d",
				v, vm.Message.AuthorityID, vm.Round, vm.Message.Stage,
				s.lenVotes(models.Prevote), s.lenVotes(models.Precommit), s.state.Threshold()+1)
		case <-ctx.Done():
			logger.Trace("returning from receiveMessages")
			return
		}
	}
}

func (s *Service) createSignedVoteAndVoteMessage(vote *models.Vote, stage models.Subround) (
	signedVote *models.SignedVote, voteMessage *models.VoteMessage, err error) {
	msg, err := scale.Marshal(models.FullVote{
		Stage: stage,
		Vote:  *vote,
		Round: s.state.Round,
		SetID: s.state.SetID,
	})
	if err != nil {
		return nil, nil, err
	}

	sig, err := s.keypair.Sign(msg)
	if err != nil {
		return nil, nil, err
	}

	pc := &models.SignedVote{
		Vote:        *vote,
		Signature:   ed25519.NewSignatureBytes(sig),
		AuthorityID: s.keypair.Public().(*ed25519.PublicKey).AsBytes(),
	}

	sm := &models.SignedMessage{
		Stage:       stage,
		BlockHash:   pc.Vote.Hash,
		Number:      pc.Vote.Number,
		Signature:   ed25519.NewSignatureBytes(sig),
		AuthorityID: s.keypair.Public().(*ed25519.PublicKey).AsBytes(),
	}

	vm := &models.VoteMessage{
		Round:   s.state.Round,
		SetID:   s.state.SetID,
		Message: *sm,
	}

	return pc, vm, nil
}

// validateVoteMessage validates a VoteMessage and adds it to the current votes
// it returns the resulting vote if validated, error otherwise
func (s *Service) validateVoteMessage(from peer.ID, m *models.VoteMessage) (*models.Vote, error) {
	// make sure round does not increment while VoteMessage is being validated
	s.roundLock.Lock()
	defer s.roundLock.Unlock()

	// check for message signature
	pk, err := ed25519.NewPublicKey(m.Message.AuthorityID[:])
	if err != nil {
		// TODO Affect peer reputation
		// https://github.com/ChainSafe/gossamer/issues/2505
		return nil, err
	}

	err = validateMessageSignature(pk, m)
	if err != nil {
		// TODO Affect peer reputation
		// https://github.com/ChainSafe/gossamer/issues/2505
		return nil, err
	}

	if m.SetID != s.state.SetID {
		return nil, ErrSetIDMismatch
	}

	const maxRoundsLag = 1
	minRoundAccepted := s.state.Round - maxRoundsLag
	if minRoundAccepted > s.state.Round {
		// we overflowed below 0 so set the minimum to 0.
		minRoundAccepted = 0
	}

	const maxRoundsAhead = 1
	maxRoundAccepted := s.state.Round + maxRoundsAhead

	if m.Round < minRoundAccepted || m.Round > maxRoundAccepted {
		// Discard message
		// TODO: affect peer reputation, this is shameful impolite behaviour
		// https://github.com/ChainSafe/gossamer/issues/2505
		return nil, nil //nolint:nilnil
	}

	if m.Round < s.state.Round {
		// message round is lagging by 1
		// peer doesn't know round was finalised, send out another commit message
		header, err := s.blockState.GetFinalisedHeader(m.Round, m.SetID)
		if err != nil {
			return nil, err
		}

		cm, err := s.newCommitMessage(header, m.Round)
		if err != nil {
			return nil, err
		}

		// send finalised block from previous round to network
		msg, err := cm.ToConsensusMessage()
		if err != nil {
			return nil, err
		}

		if err = s.network.SendMessage(from, msg); err != nil {
			logger.Warnf("failed to send CommitMessage: %s", err)
		}

		// TODO: get justification if your round is lower, or just do catch-up? (#1815)
		return nil, errRoundMismatch(m.Round, s.state.Round)
	} else if m.Round > s.state.Round {
		// Message round is higher by 1 than the round of our state,
		// we may be lagging behind, so store the message in the tracker
		// for processing later in the coming few milliseconds.
		s.tracker.addVote(from, m)
		return nil, errRoundMismatch(m.Round, s.state.Round)
	}

	// check for equivocation ie. multiple votes within one subround
	voter, err := s.state.PubkeyToVoter(pk)
	if err != nil {
		return nil, err
	}

	vote := models.NewVote(m.Message.BlockHash, m.Message.Number)

	// if the vote is from ourselves, return an error
	kb := [32]byte(s.publicKeyBytes())
	if bytes.Equal(m.Message.AuthorityID[:], kb[:]) {
		return nil, errVoteFromSelf
	}

	err = s.validateVote(vote)
	if errors.Is(err, ErrBlockDoesNotExist) ||
		errors.Is(err, blocktree.ErrDescendantNotFound) ||
		errors.Is(err, blocktree.ErrEndNodeNotFound) ||
		errors.Is(err, blocktree.ErrStartNodeNotFound) {
		s.tracker.addVote(from, m)
	}
	if err != nil {
		return nil, err
	}

	just := &models.SignedVote{
		Vote:        *vote,
		Signature:   m.Message.Signature,
		AuthorityID: pk.AsBytes(),
	}

	equivocated := s.checkForEquivocation(voter, just, m.Message.Stage)
	if equivocated {
		return nil, ErrEquivocation
	}

	switch m.Message.Stage {
	case models.Prevote, models.PrimaryProposal:
		s.prevotes.Store(pk.AsBytes(), just)
	case models.Precommit:
		s.precommits.Store(pk.AsBytes(), just)
	}

	return vote, nil
}

// checkForEquivocation checks if the vote is an equivocatory vote.
// it returns true if so, false otherwise.
// additionally, if the vote is equivocatory, it updates the service's votes and equivocations.
func (s *Service) checkForEquivocation(voter *models.Voter, vote *models.SignedVote, stage models.Subround) bool {
	v := voter.Key.AsBytes()

	// save justification, since equivocatory vote may still be used in justification
	var eq map[ed25519.PublicKeyBytes][]*models.SignedVote

	switch stage {
	case models.Prevote, models.PrimaryProposal:
		eq = s.pvEquivocations
	case models.Precommit:
		eq = s.pcEquivocations
	}

	s.mapLock.Lock()
	defer s.mapLock.Unlock()

	_, has := eq[v]
	if has {
		// if the voter has already equivocated, every vote in that round is an equivocatory vote
		eq[v] = append(eq[v], vote)
		return true
	}

	existingVote, has := s.loadVote(v, stage)
	if !has {
		return false
	}

	if has && existingVote.Vote.Hash != vote.Vote.Hash {
		// the voter has already voted, all their votes are now equivocatory
		eq[v] = []*models.SignedVote{existingVote, vote}
		s.deleteVote(v, stage)
		return true
	}

	return false
}

// validateVote checks if the block that is being voted for exists, and that it is a descendant of a
// previously finalised block.
func (s *Service) validateVote(v *models.Vote) error {
	// check if v.hash corresponds to a valid block
	has, err := s.blockState.HasHeader(v.Hash)
	if err != nil {
		return err
	}

	if !has {
		return ErrBlockDoesNotExist
	}

	// check if the block is an eventual descendant of a previously finalised block
	isDescendant, err := s.blockState.IsDescendantOf(s.head.Hash(), v.Hash)
	if err != nil {
		return err
	}

	if !isDescendant {
		return errVoteBlockMismatch
	}

	return nil
}

func validateMessageSignature(pk *ed25519.PublicKey, m *models.VoteMessage) error {
	msg, err := scale.Marshal(models.FullVote{
		Stage: m.Message.Stage,
		Vote:  *models.NewVote(m.Message.BlockHash, m.Message.Number),
		Round: m.Round,
		SetID: m.SetID,
	})
	if err != nil {
		return err
	}

	ok, err := pk.Verify(msg, m.Message.Signature[:])
	if err != nil {
		return err
	}

	if !ok {
		return ErrInvalidSignature
	}

	return nil
}
