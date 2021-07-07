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
	"context"
	"errors"
	"time"

	"github.com/ChainSafe/gossamer/lib/blocktree"
	"github.com/ChainSafe/gossamer/lib/crypto/ed25519"
	"github.com/ChainSafe/gossamer/lib/scale"
)

// receiveMessages receives messages from the in channel until the specified condition is met
func (s *Service) receiveMessages(cond func() bool) {
	ctx, cancel := context.WithCancel(s.ctx)

	go func() {
		for {
			select {
			case msg := <-s.in:
				if msg == nil {
					continue
				}

				logger.Trace("received vote message", "msg", msg)
				vm, ok := msg.(*VoteMessage)
				if !ok {
					continue
				}

				v, err := s.validateMessage(vm)
				if err != nil {
					logger.Debug("failed to validate vote message", "message", vm, "error", err)
					continue
				}

				logger.Debug("validated vote message",
					"vote", v,
					"round", vm.Round,
					"subround", vm.Message.Stage,
					"prevote count", s.lenVotes(prevote),
					"precommit count", s.lenVotes(precommit),
					"votes needed", s.state.threshold(),
				)
			case <-ctx.Done():
				logger.Trace("returning from receiveMessages")
				return
			}
		}
	}()

	for {
		if cond() {
			cancel()
			return
		}
		time.Sleep(time.Millisecond * 10)
	}
}

func (s *Service) createSignedVoteAndVoteMessage(vote *Vote, stage subround) (*SignedVote, *VoteMessage, error) {
	msg, err := scale.Encode(&FullVote{
		Stage: stage,
		Vote:  vote,
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
		Vote:        vote,
		Signature:   ed25519.NewSignatureBytes(sig),
		AuthorityID: s.keypair.Public().(*ed25519.PublicKey).AsBytes(),
	}

	sm := &SignedMessage{
		Stage:       stage,
		Hash:        pc.Vote.hash,
		Number:      pc.Vote.number,
		Signature:   ed25519.NewSignatureBytes(sig),
		AuthorityID: s.keypair.Public().(*ed25519.PublicKey).AsBytes(),
	}

	vm := &VoteMessage{
		Round:   s.state.round,
		SetID:   s.state.setID,
		Message: sm,
	}

	return pc, vm, nil
}

// validateMessage validates a VoteMessage and adds it to the current votes
// it returns the resulting vote if validated, error otherwise
func (s *Service) validateMessage(m *VoteMessage) (*Vote, error) {
	// make sure round does not increment while VoteMessage is being validated
	s.roundLock.Lock()
	defer s.roundLock.Unlock()

	if m.Message == nil {
		return nil, errors.New("invalid VoteMessage; missing Message field")
	}

	// check for message signature
	pk, err := ed25519.NewPublicKey(m.Message.AuthorityID[:])
	if err != nil {
		return nil, err
	}

	switch m.Message.Stage {
	case prevote, primaryProposal:
		pv, has := s.loadVote(pk.AsBytes(), prevote)
		if has && pv.Vote.hash.Equal(m.Message.Hash) {
			return nil, errVoteExists
		}
	case precommit:
		pc, has := s.loadVote(pk.AsBytes(), precommit)
		if has && pc.Vote.hash.Equal(m.Message.Hash) {
			return nil, errVoteExists
		}
	}

	err = validateMessageSignature(pk, m)
	if err != nil {
		return nil, err
	}

	// check that setIDs match
	if m.SetID != s.state.setID {
		return nil, ErrSetIDMismatch
	}

	// check that vote is for current round
	if m.Round != s.state.round {
		if m.Round < s.state.round {
			// peer doesn't know round was finalised, send out another commit message
			header, err := s.blockState.GetFinalizedHeader(m.Round, m.SetID) //nolint
			if err != nil {
				return nil, err
			}

			// send finalised block from previous round to network
			msg, err := s.newCommitMessage(header, m.Round).ToConsensusMessage()
			if err != nil {
				return nil, err
			}

			// TODO: don't broadcast, just send to peer; will address in a follow-up
			s.network.SendMessage(msg)
		}

		// TODO: get justification if your round is lower, or just do catch-up?
		return nil, ErrRoundMismatch
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
	if errors.Is(err, ErrBlockDoesNotExist) || errors.Is(err, blocktree.ErrEndNodeNotFound) {
		// TODO: cancel if block is imported; if we refactor the syncing this will likely become cleaner
		// as we can have an API to synchronously sync and import a block
		go s.network.SendBlockReqestByHash(vote.hash)
		s.tracker.add(m)
	}
	if err != nil {
		return nil, err
	}

	just := &SignedVote{
		Vote:        vote,
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
func (s *Service) checkForEquivocation(voter *Voter, vote *SignedVote, stage subround) bool {
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

	if has && existingVote.Vote.hash != vote.Vote.hash {
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
	has, err := s.blockState.HasHeader(v.hash)
	if err != nil {
		return err
	}

	if !has {
		return ErrBlockDoesNotExist
	}

	// check if the block is an eventual descendant of a previously finalised block
	isDescendant, err := s.blockState.IsDescendantOf(s.head.Hash(), v.hash)
	if err != nil {
		return err
	}

	if !isDescendant {
		return ErrDescendantNotFound
	}

	return nil
}

func validateMessageSignature(pk *ed25519.PublicKey, m *VoteMessage) error {
	msg, err := scale.Encode(&FullVote{
		Stage: m.Message.Stage,
		Vote:  NewVote(m.Message.Hash, m.Message.Number),
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
