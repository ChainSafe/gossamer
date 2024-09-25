// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/internal/database"
	"github.com/ChainSafe/gossamer/internal/primitives/core/hash"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime"
	"github.com/ChainSafe/gossamer/lib/blocktree"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto/ed25519"

	client_grandpa "github.com/ChainSafe/gossamer/internal/client/consensus/grandpa"
	finality_grandpa "github.com/ChainSafe/gossamer/pkg/finality-grandpa"

	"github.com/libp2p/go-libp2p/core/peer"
)

var (
	ErrNeighbourVersionNotSupported = errors.New("neighbour version not supported")
)

// MessageHandler handles GRANDPA consensus messages
type MessageHandler struct {
	grandpa    *Service
	blockState BlockState
	telemetry  Telemetry

	isStart bool // This is a temp hacky way
}

// NewMessageHandler returns a new MessageHandler
func NewMessageHandler(grandpa *Service, blockState BlockState, telemetryMailer Telemetry) *MessageHandler {
	return &MessageHandler{
		grandpa:    grandpa,
		blockState: blockState,
		telemetry:  telemetryMailer,
		isStart:    true,
	}
}

// HandleMessage handles a GRANDPA consensus message
// if it is a CommitMessage, it updates the BlockState
// if it is a VoteMessage, it sends it to the GRANDPA service
func (h *MessageHandler) handleMessage(from peer.ID, m GrandpaMessage) (network.NotificationsMessage, error) {
	logger.Tracef("handling grandpa message: %v", m)

	switch msg := m.(type) {
	case *VoteMessage:
		err := h.grandpa.handleVoteMessage(from, msg)
		if err != nil {
			return nil, fmt.Errorf("handling vote message: %w", err)
		}
		return nil, nil //nolint:nilnil
	case *CommitMessage:
		err := h.grandpa.handleCommitMessage(msg)
		if err != nil {
			return nil, fmt.Errorf("handling commit message: %w", err)
		}

		return nil, nil //nolint:nilnil
	case *NeighbourPacketV1:
		// we can afford to not retry handling neighbour message, if it errors.
		return nil, h.handleNeighbourMessage(msg, from)
	case *CatchUpRequest:
		return h.handleCatchUpRequest(msg)
	case *CatchUpResponse:
		err := h.handleCatchUpResponse(msg)
		if errors.Is(err, blocktree.ErrNodeNotFound) {
			// TODO: we are adding these messages to reprocess them again, but we
			// haven't added code to reprocess them. Do that.
			// Also, revisit if we need to add these message in synchronous manner
			// or not. If not, change catchUpResponseMessages to a normal map.  #1531
			h.grandpa.tracker.addCatchUpResponse(msg)
		}
		return nil, err
	default:
		return nil, ErrInvalidMessageType
	}
}

func (h *MessageHandler) handleNeighbourMessage(packet *NeighbourPacketV1, from peer.ID) error {
	// TODO(#2931)
	// This should be the receiver side of the handling messages, NOT GOSSIP
	if h.grandpa.state.round < packet.Round {
		err := h.grandpa.catchUp.tryCatchUp(packet.Round, packet.SetID, from)
		if err != nil {
			return err
		}
	}

	// TODO handle in normal case?
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

	logger.Warnf(
		"received catch up response with hash %s for round %d and set id %d",
		msg.Hash, msg.Round, msg.SetID)

	err := verifyBlockHashAgainstBlockNumber(h.blockState, msg.Hash, uint(msg.Number))
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			h.grandpa.tracker.addCatchUpResponse(msg)
			logger.Infof("we might not have synced to the given block %s yet: %s", msg.Hash, err)
			return nil
		}
		return err
	}

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

func getEquivocatoryVoters(votes []AuthData) map[ed25519.PublicKeyBytes]struct{} {
	eqvVoters := make(map[ed25519.PublicKeyBytes]struct{})
	voters := make(map[ed25519.PublicKeyBytes][64]byte, len(votes))

	for _, v := range votes {
		signature, present := voters[v.AuthorityID]
		if present && !bytes.Equal(signature[:], v.Signature[:]) {
			eqvVoters[v.AuthorityID] = struct{}{}
		} else {
			voters[v.AuthorityID] = v.Signature
		}
	}

	return eqvVoters
}

func isDescendantOfHighestFinalisedBlock(blockState BlockState, hash common.Hash) (bool, error) {
	highestHeader, err := blockState.GetHighestFinalisedHeader()
	if err != nil {
		return false, fmt.Errorf("could not get highest finalised header: %w", err)
	}

	return blockState.IsDescendantOf(highestHeader.Hash(), hash)
}

func (h *MessageHandler) verifyPreVoteJustification(msg *CatchUpResponse) (common.Hash, error) {
	voters := make(map[ed25519.PublicKeyBytes]map[common.Hash]int, len(msg.PreVoteJustification))
	eqVotesByHash := make(map[common.Hash]map[ed25519.PublicKeyBytes]struct{})

	for _, pvj := range msg.PreVoteJustification {
		err := verifyBlockHashAgainstBlockNumber(h.blockState, pvj.Vote.Hash, uint(pvj.Vote.Number))
		if err != nil {
			if errors.Is(err, database.ErrNotFound) {
				h.grandpa.tracker.addCatchUpResponse(msg)
				logger.Infof("we might not have synced to the given block %s yet: %s", pvj.Vote.Hash, err)
				continue
			}
			return common.Hash{}, err
		}
	}
	// identify equivocatory votes by hash
	for _, justification := range msg.PreVoteJustification {
		hashsToCount, ok := voters[justification.AuthorityID]
		if !ok {
			hashsToCount = make(map[common.Hash]int)
		}

		hashsToCount[justification.Vote.Hash]++
		voters[justification.AuthorityID] = hashsToCount

		if hashsToCount[justification.Vote.Hash] > 1 {
			pubKeysOnHash, ok := eqVotesByHash[justification.Vote.Hash]
			if !ok {
				pubKeysOnHash = make(map[ed25519.PublicKeyBytes]struct{})
			}

			pubKeysOnHash[justification.AuthorityID] = struct{}{}
			eqVotesByHash[justification.Vote.Hash] = pubKeysOnHash
		}
	}

	// verify pre-vote justification, returning the pre-voted block if there is one
	votes := make(map[common.Hash]uint64)
	for idx := range msg.PreVoteJustification {
		just := &msg.PreVoteJustification[idx]

		// if the current voter is on equivocatory map then ignore the vote
		if _, ok := eqVotesByHash[just.Vote.Hash][just.AuthorityID]; ok {
			continue
		}

		err := verifyJustification(just, msg.Round, msg.SetID, prevote, h.grandpa.authorityKeySet())
		if err != nil {
			continue
		}

		votes[just.Vote.Hash]++
	}

	var prevote common.Hash
	for hash, count := range votes {
		equivocatoryVotes := eqVotesByHash[hash]
		if count+uint64(len(equivocatoryVotes)) >= h.grandpa.state.threshold() {
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
	auths := make([]AuthData, len(msg.PreCommitJustification))
	for i, pcj := range msg.PreCommitJustification {
		auths[i] = AuthData{AuthorityID: pcj.AuthorityID}
	}

	isDescendant, err := isDescendantOfHighestFinalisedBlock(h.blockState, msg.Hash)
	if err != nil {
		return fmt.Errorf("checking if descendant of highest block: %w", err)
	}
	if !isDescendant {
		return errVoteBlockMismatch
	}

	eqvVoters := getEquivocatoryVoters(auths)

	// verify pre-commit justification
	var count uint64
	for idx := range msg.PreCommitJustification {
		just := &msg.PreCommitJustification[idx]

		err = verifyBlockHashAgainstBlockNumber(h.blockState, just.Vote.Hash, uint(just.Vote.Number))
		if err != nil {
			if errors.Is(err, database.ErrNotFound) {
				h.grandpa.tracker.addCatchUpResponse(msg)
				logger.Infof("we might not have synced to the given block %s yet: %s", just.Vote.Hash, err)
				continue
			}
			return fmt.Errorf("verifying block hash against block number: %w", err)
		}

		err := verifyJustification(just, msg.Round, msg.SetID, precommit, h.grandpa.authorityKeySet())
		if err != nil {
			logger.Errorf("could not verify precommit justification for block %s from authority %s: %s",
				just.Vote.Hash.String(), just.AuthorityID.String(), err)
			continue
		}

		if _, ok := eqvVoters[just.AuthorityID]; ok {
			continue
		}

		if just.Vote.Hash == msg.Hash && just.Vote.Number == msg.Number {
			count++
		}
	}

	if count+uint64(len(eqvVoters)) < h.grandpa.state.threshold() {
		return ErrMinVotesNotMet
	}

	return nil
}

// VerifyBlockJustification verifies the finality justification for a block, returns scale encoded justification with
// any extra bytes removed.
func (s *Service) VerifyBlockJustification(finalizedHash common.Hash, finalizedNumber uint, encoded []byte) (
	round uint64, setID uint64, err error,
) {
	setID, err = s.grandpaState.GetSetIDByBlockNumber(finalizedNumber)
	if err != nil {
		return 0, 0, fmt.Errorf("cannot get set ID from block number: %w", err)
	}

	auths, err := s.grandpaState.GetAuthorities(setID)
	if err != nil {
		return 0, 0, fmt.Errorf("cannot get authorities for set ID: %w", err)
	}

	logger.Debugf("verifying justification within set id %d and authorities %d", setID, len(auths))

	idsAndWeights := make([]finality_grandpa.IDWeight[string], len(auths))
	for idx, auth := range auths {
		idsAndWeights[idx] = finality_grandpa.IDWeight[string]{
			ID:     string(auth.Key.Encode()),
			Weight: 1,
		}
	}

	voters := finality_grandpa.NewVoterSet(idsAndWeights)
	target := client_grandpa.HashNumber[hash.H256, uint32]{
		Hash:   hash.H256(finalizedHash.ToBytes()),
		Number: uint32(finalizedNumber), //nolint:gosec
	}

	justification, err := client_grandpa.DecodeGrandpaJustificationVerifyFinalizes[hash.H256, uint32, runtime.BlakeTwo256](
		encoded, target, setID, *voters)
	if err != nil {
		return 0, 0, fmt.Errorf("decoding and verifying justification: %w", err)
	}

	return justification.Justification.Round, setID, nil
}

func verifyBlockHashAgainstBlockNumber(bs BlockState, hash common.Hash, number uint) error {
	header, err := bs.GetHeader(hash)
	if err != nil {
		return fmt.Errorf("could not get header from block hash: %w", err)
	}

	if header.Number != number {
		return fmt.Errorf("%w: expected number %d from header but got number %d",
			ErrBlockHashMismatch, header.Number, number)
	}
	return nil
}
