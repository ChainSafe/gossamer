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
	"errors"
	"sync"
	"sync/atomic"

	"github.com/ChainSafe/gossamer/lib/common"

	"github.com/libp2p/go-libp2p-core/peer"
)

type catchUp struct {
	isAuthority bool
	isStarted   *atomic.Value
	peers       *sync.Map //map[peer.ID]struct{}
	state       *State
	network     Network
}

func newCatchUp(isAuthority bool, state *State, network Network) *catchUp {
	isStarted := new(atomic.Value)
	isStarted.Store(false)

	return &catchUp{
		isAuthority: isAuthority,
		isStarted:   isStarted,
		state:       state,
		peers:       new(sync.Map),
	}
}

func (c *catchUp) addPeer(id peer.ID) {
	c.peers.Store(id, struct{}{}) // TODO: store neighbour message info
}

func (c *catchUp) doCatchUp(from peer.ID, setID, round uint64) error {
	if !c.isAuthority {
		// only authorities need to participate in catch-up
		return nil
	}

	if c.isStarted.Load().(bool) {
		return errors.New("already started")
	}
	c.isStarted.Store(true)
	msg := newCatchUpRequest(round, setID)
	cm, err := msg.ToConsensusMessage()
	if err != nil {
		return err
	}
	resp, err := c.network.SendCatchUpRequest(from, messageID, cm)
	if err != nil {
		return err
	}

	if resp == nil {
		// TODO: request from other peers
	}

	catchUpResp, err := decodeMessage(resp)
	if err != nil {
		return err
	}

	_ = catchUpResp

	return nil
}

func (c *catchUp) handleCatchUpResponse(msg *catchUpResponse) error {
	logger.Debug("received catch up response", "round", msg.Round, "setID", msg.SetID, "hash", msg.Hash)

	// // if we aren't currently expecting a catch up response, return
	// if !h.grandpa.paused.Load().(bool) {
	// 	logger.Debug("not currently paused, ignoring catch up response")
	// 	return nil
	// }

	if msg.SetID != c.state.setID {
		return ErrSetIDMismatch
	}

	if msg.Round != c.state.round-1 {
		return ErrInvalidCatchUpResponseRound
	}

	prevote, err := c.verifyPreVoteJustification(msg)
	if err != nil {
		return err
	}

	if err = c.verifyPreCommitJustification(msg); err != nil {
		return err
	}

	if (msg.Hash == common.Hash{}) || msg.Number == 0 {
		return ErrGHOSTlessCatchUp
	}

	if err = c.verifyCatchUpResponseCompletability(prevote, msg.Hash); err != nil {
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
func (c *catchUp) verifyCatchUpResponseCompletability(prevote, precommit common.Hash) error {
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

func (c *catchUp) verifyCommitMessageJustification(fm *CommitMessage) error {
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

func (c *catchUp) verifyPreVoteJustification(msg *catchUpResponse) (common.Hash, error) {
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

func (c *catchUp) verifyPreCommitJustification(msg *catchUpResponse) error {
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

func (c *catchUp) verifyJustification(just *SignedPrecommit, round, setID uint64, stage subround) error {
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
