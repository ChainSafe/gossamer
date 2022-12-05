// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"bytes"
	"errors"
	"fmt"

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

var (
	ErrNeighbourVersionNotSupported = errors.New("neighbour version not supported")
)

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
		telemetry:  telemetryMailer,
	}
}

// HandleMessage handles a GRANDPA consensus message
// if it is a CommitMessage, it updates the BlockState
// if it is a VoteMessage, it sends it to the GRANDPA service
func (h *MessageHandler) handleMessage(from peer.ID, m GrandpaMessage) error {
	logger.Tracef("handling grandpa message: %v", m)

	switch msg := m.(type) {
	case *VoteMessage:
		err := h.grandpa.handleVoteMessage(from, msg)
		if err != nil {
			return fmt.Errorf("handling vote message: %w", err)
		}
		return nil
	case *CommitMessage:
		err := h.grandpa.handleCommitMessage(msg)
		if err != nil {
			return fmt.Errorf("handling commit message: %w", err)
		}

		return nil
	case *NeighbourPacketV1:
		// we can afford to not retry handling neighbour message, if it errors.
		err := h.handleNeighbourMessage(msg, from)
		if err != nil {
			return fmt.Errorf("handling neighbour message: %w", err)
		}

		return nil
	case *CatchUpRequest:
		err := h.handleCatchUpRequest(msg, from)
		if err != nil {
			return fmt.Errorf("handling catch up request message: %w", err)
		}

		return nil
	case *CatchUpResponse:
		err := h.catchUp.handleCatchUpResponse(msg)
		if errors.Is(err, blocktree.ErrNodeNotFound) || errors.Is(err, chaindb.ErrKeyNotFound) {
			h.grandpa.tracker.addCatchUpResponse(&networkCatchUpResponseMessage{
				from: from,
				msg:  msg,
			})
			h.catchUp.bestResponse.Store(msg)
		} else if err != nil {
			logger.Debugf("could not catchup: %s", err)
		}

		return nil
	default:
		return ErrInvalidMessageType
	}
}

func (h *MessageHandler) handleNeighbourMessage(msg *NeighbourPacketV1, from peer.ID) error {
	if h.grandpa.authority {
		// TODO(#2931): this is a simple hack to ensure that the neighbour messages
		// sent by gossamer are being received by substrate nodes
		// not intended to be production code
		h.grandpa.roundLock.Lock()
		neighbourMessage := &NeighbourPacketV1{
			Round:  h.grandpa.state.round,
			SetID:  h.grandpa.state.setID,
			Number: uint32(h.grandpa.head.Number),
		}
		h.grandpa.roundLock.Unlock()

		cm, err := neighbourMessage.ToConsensusMessage()
		if err != nil {
			return fmt.Errorf("converting neighbour message to network message: %w", err)
		}

		logger.Debugf("sending neighbour message: %v", neighbourMessage)
		h.grandpa.network.GossipMessage(cm)
	}

	currFinalized, err := h.blockState.GetFinalisedHeader(0, 0)
	if err != nil {
		return err
	}

	// ignore neighbour messages where our best finalised number is greater than theirs
	if currFinalized.Number >= uint(msg.Number) {
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
	if uint(msg.Number) > head {
		logger.Debug("ignoring neighbour message, because we have not synced to this block number")
		return nil
	}

	logger.Debugf("got neighbour message with number %d, set id %d and round %d", msg.Number, msg.SetID, msg.Round)

	highestRound, setID, err := h.blockState.GetHighestRoundAndSetID()
	if err != nil {
		return fmt.Errorf("cannot get highest round and set id: %w", err)
	}

	if msg.SetID != setID {
		fmt.Println("here")
		return fmt.Errorf("%w: received %d and expected %d", ErrSetIDMismatch, msg.SetID, setID)
	}

	// catch up only if we are behind by more than catchup threshold
	if int(msg.Round-highestRound) > catchupThreshold {
		logger.Debugf("lagging behind by %d rounds", msg.Round-highestRound)
		return h.catchUp.do(from, msg.Round, msg.SetID)
	}

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
		return fmt.Errorf("%w: received %d and grandpa state round is %d",
			ErrInvalidCatchUpRound, msg.Round, h.grandpa.state.round)
	}

	// We don't necessarily have to reply with the round asked in the request, we can reply
	// with our latest round.
	resp, err := h.grandpa.newCatchUpResponse(h.grandpa.state.round, h.grandpa.state.setID)
	if err != nil {
		return fmt.Errorf("creating catch up response: %w", err)
	}

	cm, err := resp.ToConsensusMessage()
	if err != nil {
		return fmt.Errorf("converting to consensus message: %w", err)
	}

	err = h.grandpa.network.SendMessage(from, cm)
	if err != nil {
		return fmt.Errorf("sending message: %w", err)
	}

	logger.Debugf(
		"successfully sent catch up response with hash %s for round %d and set id %d, to %s",
		resp.Hash, h.grandpa.state.round, h.grandpa.state.setID, from)

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

// VerifyBlockJustification verifies the finality justification for a block, returns scale encoded justification with
//
//	any extra bytes removed.
func (s *Service) VerifyBlockJustification(hash common.Hash, justification []byte) ([]byte, error) {
	fj := Justification{}
	err := scale.Unmarshal(justification, &fj)
	if err != nil {
		return nil, err
	}

	if !hash.Equal(fj.Commit.Hash) {
		return nil, fmt.Errorf("%w: justification %s and block hash %s",
			ErrJustificationMismatch, fj.Commit.Hash.Short(), hash.Short())
	}

	setID, err := s.grandpaState.GetSetIDByBlockNumber(uint(fj.Commit.Number))
	if err != nil {
		return nil, fmt.Errorf("cannot get set ID from block number: %w", err)
	}

	has, err := s.blockState.HasFinalisedBlock(fj.Round, setID)
	if err != nil {
		return nil, err
	}

	if has {
		return nil, fmt.Errorf("already have finalised block with setID=%d and round=%d", setID, fj.Round)
	}

	isDescendant, err := isDescendantOfHighestFinalisedBlock(s.blockState, fj.Commit.Hash)
	if err != nil {
		return nil, err
	}

	if !isDescendant {
		return nil, errVoteBlockMismatch
	}

	auths, err := s.grandpaState.GetAuthorities(setID)
	if err != nil {
		return nil, fmt.Errorf("cannot get authorities for set ID: %w", err)
	}

	// threshold is two-thirds the number of authorities,
	// uses the current set of authorities to define the threshold
	threshold := (2 * len(auths) / 3)

	if len(fj.Commit.Precommits) < threshold {
		return nil, ErrMinVotesNotMet
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
		// check if vote was for descendant of committed block
		isDescendant, err := s.blockState.IsDescendantOf(hash, just.Vote.Hash)
		if err != nil {
			return nil, err
		}

		if !isDescendant {
			return nil, ErrPrecommitBlockMismatch
		}

		publicKey, err := ed25519.NewPublicKey(just.AuthorityID[:])
		if err != nil {
			return nil, err
		}

		if !isInAuthSet(publicKey, auths) {
			return nil, ErrAuthorityNotInSet
		}

		// verify signature for each precommit
		msg, err := scale.Marshal(FullVote{
			Stage: precommit,
			Vote:  just.Vote,
			Round: fj.Round,
			SetID: setID,
		})
		if err != nil {
			return nil, err
		}

		ok, err := publicKey.Verify(msg, just.Signature[:])
		if err != nil {
			return nil, err
		}

		if !ok {
			return nil, ErrInvalidSignature
		}

		if _, ok := equivocatoryVoters[just.AuthorityID]; ok {
			continue
		}

		count++
	}

	if count+len(equivocatoryVoters) < threshold {
		return nil, ErrMinVotesNotMet
	}

	err = verifyBlockHashAgainstBlockNumber(s.blockState, fj.Commit.Hash, uint(fj.Commit.Number))
	if err != nil {
		return nil, err
	}

	for _, preCommit := range fj.Commit.Precommits {
		err := verifyBlockHashAgainstBlockNumber(s.blockState, preCommit.Vote.Hash, uint(preCommit.Vote.Number))
		if err != nil {
			return nil, err
		}
	}

	err = s.blockState.SetFinalisedHash(hash, fj.Round, setID)
	if err != nil {
		return nil, err
	}

	logger.Debugf(
		"set finalised block with hash %s, round %d and set id %d",
		hash, fj.Round, setID)
	return scale.Marshal(fj)
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

func isInAuthSet(auth *ed25519.PublicKey, set []types.GrandpaVoter) bool {
	for _, a := range set {
		if bytes.Equal(a.Key.Encode(), auth.Encode()) {
			return true
		}
	}

	return false
}
