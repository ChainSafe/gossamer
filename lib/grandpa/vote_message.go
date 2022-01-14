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
	"github.com/ChainSafe/gossamer/pkg/scale"

	"github.com/libp2p/go-libp2p-core/peer"
)

type networkVoteMessage struct {
	from peer.ID
	msg  *VoteMessage
}

// receiveVoteMessages receives messages from the in channel until a grandpa round finishes.
func (s *Service) receiveVoteMessages(ctx context.Context) {
	for {
		select {
		case msg, ok := <-s.in:
			if !ok {
				return
			}

			if msg == nil || msg.msg == nil {
				continue
			}

			logger.Tracef("received vote message %v from %s", msg.msg, msg.from)
			vm := msg.msg

			switch vm.Message.Stage {
			case prevote, primaryProposal:
				s.telemetry.SendMessage(
					telemetry.NewAfgReceivedPrevote(
						&vm.Message.Hash,
						fmt.Sprint(vm.Message.Number),
						vm.Message.AuthorityID.String(),
					),
				)
			case precommit:
				s.telemetry.SendMessage(
					telemetry.NewAfgReceivedPrecommit(
						&vm.Message.Hash,
						fmt.Sprint(vm.Message.Number),
						vm.Message.AuthorityID.String(),
					),
				)
			default:
				logger.Warnf("unsupported stage %s", vm.Message.Stage.String())
			}

			v, err := s.validateVoteMessage(msg.from, vm)
			if err != nil {
				logger.Debugf("failed to validate vote message %v: %s", vm, err)
				continue
			}

			logger.Debugf(
				"validated vote message %v from %s, round %d, subround %d, "+
					"prevote count %d, precommit count %d, votes needed %d",
				v, vm.Message.AuthorityID, vm.Round, vm.Message.Stage,
				s.lenVotes(prevote), s.lenVotes(precommit), s.state.threshold()+1)
		case <-ctx.Done():
			logger.Trace("returning from receiveMessages")
			return
		}
	}
}

func (s *Service) createSignedVoteAndVoteMessage(vote *Vote, stage Subround) (*SignedVote, *VoteMessage, error) {
	msg, err := scale.Marshal(FullVote{
		Stage: stage,
		Vote:  *vote,
		Round: s.state.round,
		SetID: s.state.setID,
	})
	if err != nil {
		return nil, nil, err
	}

	sig, err := s.keypair.Sign(msg)
	if err != nil {
		return nil, nil, err
	}

	pc := &SignedVote{
		Vote:        *vote,
		Signature:   ed25519.NewSignatureBytes(sig),
		AuthorityID: s.keypair.Public().(*ed25519.PublicKey).AsBytes(),
	}

	sm := &SignedMessage{
		Stage:       stage,
		Hash:        pc.Vote.Hash,
		Number:      pc.Vote.Number,
		Signature:   ed25519.NewSignatureBytes(sig),
		AuthorityID: s.keypair.Public().(*ed25519.PublicKey).AsBytes(),
	}

	vm := &VoteMessage{
		Round:   s.state.round,
		SetID:   s.state.setID,
		Message: *sm,
	}

	return pc, vm, nil
}

// validateVoteMessage validates a VoteMessage and adds it to the current votes
// it returns the resulting vote if validated, error otherwise
func (s *Service) validateVoteMessage(from peer.ID, m *VoteMessage) (*Vote, error) {
	// make sure round does not increment while VoteMessage is being validated
	s.roundLock.Lock()
	defer s.roundLock.Unlock()

	// check for message signature
	pk, err := ed25519.NewPublicKey(m.Message.AuthorityID[:])
	if err != nil {
		return nil, err
	}

	switch m.Message.Stage {
	case prevote, primaryProposal:
		pv, has := s.loadVote(pk.AsBytes(), prevote)
		if has && pv.Vote.Hash.Equal(m.Message.Hash) {
			return nil, errVoteExists
		}
	case precommit:
		pc, has := s.loadVote(pk.AsBytes(), precommit)
		if has && pc.Vote.Hash.Equal(m.Message.Hash) {
			return nil, errVoteExists
		}
	}

	err = validateMessageSignature(pk, m)
	if err != nil {
		return nil, err
	}

	if m.SetID != s.state.setID {
		return nil, ErrSetIDMismatch
	}

	// check that vote is for current round
	if m.Round != s.state.round {
		if m.Round < s.state.round {
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
		} else {
			// round is higher than ours, perhaps we are behind. store vote in tracker for now
			s.tracker.addVote(&networkVoteMessage{
				from: from,
				msg:  m,
			})
		}

		// TODO: get justification if your round is lower, or just do catch-up? (#1815)
		return nil, errRoundMismatch(m.Round, s.state.round)
	}

	// check for equivocation ie. multiple votes within one subround
	voter, err := s.state.pubkeyToVoter(pk)
	if err != nil {
		return nil, err
	}

	vote := NewVote(m.Message.Hash, m.Message.Number)

	// if the vote is from ourselves, ignore
	kb := [32]byte(s.publicKeyBytes())
	if bytes.Equal(m.Message.AuthorityID[:], kb[:]) {
		return vote, nil
	}

	err = s.validateVote(vote)
	if errors.Is(err, ErrBlockDoesNotExist) ||
		errors.Is(err, blocktree.ErrDescendantNotFound) ||
		errors.Is(err, blocktree.ErrEndNodeNotFound) ||
		errors.Is(err, blocktree.ErrStartNodeNotFound) {
		s.tracker.addVote(&networkVoteMessage{
			from: from,
			msg:  m,
		})
	}
	if err != nil {
		return nil, err
	}

	just := &SignedVote{
		Vote:        *vote,
		Signature:   m.Message.Signature,
		AuthorityID: pk.AsBytes(),
	}

	equivocated := s.checkForEquivocation(voter, just, m.Message.Stage)
	if equivocated {
		return nil, ErrEquivocation
	}

	switch m.Message.Stage {
	case prevote, primaryProposal:
		s.prevotes.Store(pk.AsBytes(), just)
	case precommit:
		s.precommits.Store(pk.AsBytes(), just)
	}

	return vote, nil
}

// checkForEquivocation checks if the vote is an equivocatory vote.
// it returns true if so, false otherwise.
// additionally, if the vote is equivocatory, it updates the service's votes and equivocations.
func (s *Service) checkForEquivocation(voter *Voter, vote *SignedVote, stage Subround) bool {
	v := voter.Key.AsBytes()

	// save justification, since equivocatory vote may still be used in justification
	var eq map[ed25519.PublicKeyBytes][]*SignedVote

	switch stage {
	case prevote, primaryProposal:
		eq = s.pvEquivocations
	case precommit:
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
		eq[v] = []*SignedVote{existingVote, vote}
		s.deleteVote(v, stage)
		return true
	}

	return false
}

// validateVote checks if the block that is being voted for exists, and that it is a descendant of a
// previously finalised block.
func (s *Service) validateVote(v *Vote) error {
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
		return errInvalidVoteBlock
	}

	return nil
}

func validateMessageSignature(pk *ed25519.PublicKey, m *VoteMessage) error {
	msg, err := scale.Marshal(FullVote{
		Stage: m.Message.Stage,
		Vote:  *NewVote(m.Message.Hash, m.Message.Number),
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
