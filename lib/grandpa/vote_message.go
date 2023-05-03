// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/ChainSafe/gossamer/dot/telemetry"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/blocktree"
	"github.com/ChainSafe/gossamer/lib/crypto/ed25519"
	"github.com/ChainSafe/gossamer/pkg/scale"

	"github.com/libp2p/go-libp2p/core/peer"
)

var (
	errBeforeFinalizedBlock   = errors.New("before latest finalized block")
	errEmptyKeyOwnershipProof = errors.New("key ownership proof is nil")
)

type networkVoteMessage struct {
	from peer.ID
	msg  *VoteMessage
}

func (s *Service) sendTelemetryVoteMessage(vm *VoteMessage) {
	switch vm.Message.Stage {
	case prevote, primaryProposal:
		s.telemetry.SendMessage(
			telemetry.NewAfgReceivedPrevote(
				vm.Message.BlockHash,
				fmt.Sprint(vm.Message.Number),
				vm.Message.AuthorityID.String(),
			),
		)
	case precommit:
		s.telemetry.SendMessage(
			telemetry.NewAfgReceivedPrecommit(
				vm.Message.BlockHash,
				fmt.Sprint(vm.Message.Number),
				vm.Message.AuthorityID.String(),
			),
		)
	default:
		logger.Warnf("unsupported stage %s", vm.Message.Stage)
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

	publicKeyBytes := s.keypair.Public().(*ed25519.PublicKey).AsBytes()
	pc := &SignedVote{
		Vote:        *vote,
		Signature:   ed25519.NewSignatureBytes(sig),
		AuthorityID: publicKeyBytes,
	}

	sm := &SignedMessage{
		Stage:       stage,
		BlockHash:   pc.Vote.Hash,
		Number:      pc.Vote.Number,
		Signature:   ed25519.NewSignatureBytes(sig),
		AuthorityID: publicKeyBytes,
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
		// TODO Affect peer reputation
		// https://github.com/ChainSafe/gossamer/issues/2505
		return nil, fmt.Errorf("creating public key: %w", err)
	}

	err = validateMessageSignature(pk, m)
	if err != nil {
		// TODO Affect peer reputation
		// https://github.com/ChainSafe/gossamer/issues/2505
		return nil, fmt.Errorf("validating message signature: %w", err)
	}

	if m.SetID != s.state.setID {
		return nil, ErrSetIDMismatch
	}

	const maxRoundsLag = 1
	minRoundAccepted := s.state.round - maxRoundsLag
	if minRoundAccepted > s.state.round {
		// we overflowed below 0 so set the minimum to 0.
		minRoundAccepted = 0
	}

	const maxRoundsAhead = 1
	maxRoundAccepted := s.state.round + maxRoundsAhead

	if m.Round < minRoundAccepted || m.Round > maxRoundAccepted {
		// Discard message
		// TODO: affect peer reputation, this is shameful impolite behaviour
		// https://github.com/ChainSafe/gossamer/issues/2505
		return nil, fmt.Errorf("%w: received round: %d, round should be between: <%d, %d>",
			errRoundOutOfBounds, m.Round, minRoundAccepted, maxRoundAccepted)
	}

	if m.Round < s.state.round {
		// message round is lagging by 1
		// peer doesn't know round was finalised, send out another commit message
		header, err := s.blockState.GetFinalisedHeader(m.Round, m.SetID)
		if err != nil {
			return nil, fmt.Errorf("getting finalised header: %w", err)
		}

		// TODO: should we use `m.SetID` or `s.state.setID`?
		cm, err := s.newCommitMessage(header, m.Round, s.state.setID)
		if err != nil {
			return nil, fmt.Errorf("creating commit message: %w", err)
		}

		// send finalised block from previous round to network
		msg, err := cm.ToConsensusMessage()
		if err != nil {
			return nil, fmt.Errorf("converting commit message to consensus message: %w", err)
		}

		if err = s.network.SendMessage(from, msg); err != nil {
			logger.Warnf("failed to send CommitMessage: %s", err)
		}

		// TODO: get justification if your round is lower, or just do catch-up? (#1815)
		return nil, fmt.Errorf("%w: received round %d but state round is %d",
			errRoundsMismatch, m.Round, s.state.round)
	} else if m.Round > s.state.round {

		// Message round is higher by 1 than the round of our state,
		// we may be lagging behind, so store the message in the tracker
		// for processing later in the coming few milliseconds.
		s.tracker.addVote(from, m)
		return nil, fmt.Errorf("%w: received round %d but state round is %d",
			errRoundsMismatch, m.Round, s.state.round)
	}

	// check for equivocation ie. multiple votes within one subround
	voter, err := s.state.pubkeyToVoter(pk)
	if err != nil {
		return nil, fmt.Errorf("transforming public key into a voter: %w", err)
	}

	vote := NewVote(m.Message.BlockHash, m.Message.Number)

	// if the vote is from ourselves, return an error
	kb := [32]byte(s.publicKeyBytes())
	if bytes.Equal(m.Message.AuthorityID[:], kb[:]) {
		return nil, fmt.Errorf("%w", errVoteFromSelf)
	}

	err = s.validateVote(vote)
	if errors.Is(err, ErrBlockDoesNotExist) ||
		errors.Is(err, blocktree.ErrDescendantNotFound) ||
		errors.Is(err, blocktree.ErrEndNodeNotFound) ||
		errors.Is(err, blocktree.ErrStartNodeNotFound) {
		s.tracker.addVote(from, m)
	}
	if err != nil {
		return nil, fmt.Errorf("validating vote: %w", err)
	}

	just := &SignedVote{
		Vote:        *vote,
		Signature:   m.Message.Signature,
		AuthorityID: pk.AsBytes(),
	}

	err = s.checkAndReportEquivocation(voter, just, m.Message.Stage)
	if err != nil {
		return nil, fmt.Errorf("checking for equivocation: %w", err)
	}

	switch m.Message.Stage {
	case prevote, primaryProposal:
		s.prevotes.Store(pk.AsBytes(), just)
	case precommit:
		s.precommits.Store(pk.AsBytes(), just)
	}

	return vote, nil
}

// checkAndReportEquivocation checks if the vote is an equivocatory vote.
// If it is an equivocatory vote, the error `ErrEquivocation` is returned, the service's votes and
// equivocations are updated and the equivocation is reported to the runtime.
func (s *Service) checkAndReportEquivocation(voter *Voter, vote *SignedVote, stage Subround) error {
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
		return fmt.Errorf("%w: voter %s",
			ErrEquivocation, v)
	}

	existingVote, has := s.loadVote(v, stage)
	if !has {
		return nil
	}

	if has && existingVote.Vote.Hash != vote.Vote.Hash {
		// the voter has already voted, all their votes are now equivocatory
		eq[v] = []*SignedVote{existingVote, vote}
		s.deleteVote(v, stage)

		err := s.reportEquivocation(stage, existingVote, vote)
		if err != nil {
			logger.Errorf("reporting equivocation: %s", err)
		}
		return fmt.Errorf("%w: voter %s has existing vote %s and new vote %s",
			ErrEquivocation, v, existingVote.Vote.Hash, vote.Vote.Hash)
	}

	return nil
}

func (s *Service) reportEquivocation(stage Subround, existingVote *SignedVote, currentVote *SignedVote) error {
	setID, err := s.grandpaState.GetCurrentSetID()
	if err != nil {
		return fmt.Errorf("getting authority set id: %w", err)
	}

	round, err := s.grandpaState.GetLatestRound()
	if err != nil {
		return fmt.Errorf("getting latest round: %w", err)
	}

	pubKey := existingVote.AuthorityID

	bestBlockHash := s.blockState.BestBlockHash()
	runtime, err := s.blockState.GetRuntime(bestBlockHash)
	if err != nil {
		return fmt.Errorf("getting runtime: %w", err)
	}

	opaqueKeyOwnershipProof, err := runtime.GrandpaGenerateKeyOwnershipProof(setID, pubKey)
	if err != nil {
		return fmt.Errorf("getting key ownership proof: %w", err)
	} else if opaqueKeyOwnershipProof == nil {
		return errEmptyKeyOwnershipProof
	}

	grandpaEquivocation := types.GrandpaEquivocation{
		RoundNumber:     round,
		ID:              pubKey,
		FirstVote:       existingVote.Vote,
		FirstSignature:  existingVote.Signature,
		SecondVote:      currentVote.Vote,
		SecondSignature: currentVote.Signature,
	}

	equivocationVote := types.NewGrandpaEquivocation()
	switch stage {
	case prevote:
		err = equivocationVote.Set(types.PreVote(grandpaEquivocation))
		if err != nil {
			return fmt.Errorf("setting grandpa equivocation VDT as prevote equivocation: %w", err)
		}
	case precommit:
		err = equivocationVote.Set(types.PreCommit(grandpaEquivocation))
		if err != nil {
			return fmt.Errorf("setting grandpa equivocation VDT as precommit equivocation: %w", err)
		}
	case primaryProposal:
		return fmt.Errorf("%w: %s (%d)", errInvalidEquivocationStage, stage, stage)
	default:
		panic(fmt.Sprintf("equivocation stage not implemented: %s (%d)", stage, stage))

	}

	equivocationProof := types.GrandpaEquivocationProof{
		SetID:        setID,
		Equivocation: *equivocationVote,
	}

	err = runtime.GrandpaSubmitReportEquivocationUnsignedExtrinsic(equivocationProof, opaqueKeyOwnershipProof)
	if err != nil {
		return fmt.Errorf("submitting grandpa equivocation report to runtime: %w", err)
	}

	return nil
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
		return errVoteBlockMismatch
	}

	return nil
}

func validateMessageSignature(pk *ed25519.PublicKey, m *VoteMessage) error {
	msg, err := scale.Marshal(FullVote{
		Stage: m.Message.Stage,
		Vote:  *NewVote(m.Message.BlockHash, m.Message.Number),
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
