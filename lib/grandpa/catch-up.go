// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto/ed25519"
	"github.com/libp2p/go-libp2p-core/peer"
)

const catchUpResponseTimeout = 5 * time.Second

type catchUp struct {
	lock sync.Mutex // applied on requestsSent

	requestsSent map[peer.ID]CatchUpRequest
	bestResponse *atomic.Value // *CatchUpResponse

	catchUpResponseCh chan *CatchUpResponse
	waitingOnResponse *atomic.Value
	grandpa           *Service
}

func newCatchUp(grandpa *Service) *catchUp {
	return &catchUp{
		requestsSent:      make(map[peer.ID]CatchUpRequest),
		grandpa:           grandpa,
		catchUpResponseCh: make(chan *CatchUpResponse),
	}
}

func (c *catchUp) do(to peer.ID, round uint64, setID uint64) error {
	defer c.waitingOnResponse.Store(false)

	if err := c.sendCatchUpRequest(
		to, newCatchUpRequest(round, setID),
	); err != nil {
		logger.Debugf("failed to send catch up request: %s", err.Error())
		return err
	}

	logger.Debugf("successfully sent a catch up request to node %s, for round number %d and set ID %d",
		to, round, setID)

	c.grandpa.paused.Store(true)

	timer := time.NewTimer(catchUpResponseTimeout)
	defer timer.Stop()

	select {
	case <-c.catchUpResponseCh:
		return nil
	case <-timer.C:
		return errors.New("timeout")
	}
}

func (c *catchUp) sendCatchUpRequest(to peer.ID, req *CatchUpRequest) error {
	if c.bestResponse.Load() != nil {
		logger.Debug("ignoring neighbour message since we are already processing a catch-up response")
		return nil
	}

	// TODO: Clean up all request sent before 5 min / (neighbour message interval)
	c.lock.Lock()
	_, ok := c.requestsSent[to]
	c.lock.Unlock()
	if ok {
		logger.Debugf("ignoring neighbour message since we already sent a catch-up request to this peer: %s", to)
		return nil
	}

	c.waitingOnResponse.Store(true)

	cm, err := req.ToConsensusMessage()
	if err != nil {
		return fmt.Errorf("cannot convert request to consensus message: %w", err)
	}

	err = c.grandpa.network.SendMessage(to, cm)
	if err != nil {
		return fmt.Errorf("cannot send grandpa message: %w", err)
	}

	c.lock.Lock()
	c.requestsSent[to] = *req
	c.lock.Unlock()
	c.grandpa.paused.Store(true)

	return nil
}

func (c *catchUp) handleCatchUpResponse(msg *CatchUpResponse) error {
	if !c.grandpa.authority {
		return nil
	}

	logger.Debugf(
		"processing catch up response with hash %s for round %d and set id %d",
		msg.Hash, msg.Round, msg.SetID)

	// if we aren't currently expecting a catch up response, return
	if !c.grandpa.paused.Load().(bool) {
		logger.Debug("not currently paused, ignoring catch up response")
		return nil
	}

	if msg.SetID != c.grandpa.state.setID {
		return fmt.Errorf("%w: received set id %d but have set id %d in state", ErrSetIDMismatch, msg.SetID, c.grandpa.state.setID)
	}

	if msg.Round <= c.grandpa.state.round {
		return fmt.Errorf("%w: received round %d but grandpa round in state is %d", ErrInvalidCatchUpResponseRound, msg.Round, c.grandpa.state.round)
	}

	if c.bestResponse.Load().(*CatchUpResponse).Round >= msg.Round {
		logger.Debug("ignoring catch up response, since we are already processing one with a higher round")
	}

	prevote, err := c.verifyPreVoteJustification(msg)
	if err != nil {
		return fmt.Errorf("cannot verify pre vote justification: %w", err)
	}

	if err = c.verifyPreCommitJustification(msg); err != nil {
		return fmt.Errorf("cannot verify pre commit justification: %w", err)
	}

	if msg.Hash.IsEmpty() || msg.Number == 0 {
		return ErrGHOSTlessCatchUp
	}

	if err = c.verifyCatchUpResponseCompletability(prevote, msg.Hash); err != nil {
		return fmt.Errorf("cannot verify catch-up response completability: %w", err)
	}

	// set prevotes and precommits in db
	if err = c.grandpa.grandpaState.SetPrevotes(msg.Round, msg.SetID, msg.PreVoteJustification); err != nil {
		return fmt.Errorf("cannot set pre votes in grandpa state: %w", err)
	}

	if err = c.grandpa.grandpaState.SetPrecommits(msg.Round, msg.SetID, msg.PreCommitJustification); err != nil {
		return err
	}

	// update state and signal to grandpa we are ready to initiate
	head, err := c.grandpa.blockState.GetHeader(msg.Hash)
	if err != nil {
		logger.Debugf("failed to process catch up response for round %d, storing the catch up response to retry", msg.Round)
		return return fmt.Errorf("cannot get header from grandpa block state: %w", err)
	}

	c.grandpa.head = head
	c.grandpa.state.round = msg.Round
	close(c.grandpa.resumed)
	c.grandpa.resumed = make(chan struct{})
	if c.waitingOnResponse.Load().(bool) {
		c.catchUpResponseCh <- msg
	}

	c.grandpa.paused.Store(false)

	// resetting both response and requests
	c.bestResponse.Store(nil)
	c.lock.Lock()
	c.requestsSent = make(map[peer.ID]CatchUpRequest)
	c.lock.Unlock()

	logger.Debugf("caught up to round; unpaused service and grandpa state round is %d", c.grandpa.state.round)
	return nil
}

// verifyCatchUpResponseCompletability verifies that the pre-commit block is a descendant of, or is, the pre-voted block
func (c *catchUp) verifyCatchUpResponseCompletability(prevote, precommit common.Hash) error {
	if prevote == precommit {
		return nil
	}

	// check if the current block is a descendant of prevoted block
	isDescendant, err := c.grandpa.blockState.IsDescendantOf(prevote, precommit)
	if err != nil {
		return err
	}

	if !isDescendant {
		return ErrCatchUpResponseNotCompletable
	}

	return nil
}

func (c *catchUp) verifyPreCommitJustification(msg *CatchUpResponse) error {
	auths := make([]AuthData, len(msg.PreCommitJustification))
	for i, pcj := range msg.PreCommitJustification {
		auths[i] = AuthData{AuthorityID: pcj.AuthorityID}
	}

	eqvVoters := getEquivocatoryVoters(auths)

	// verify pre-commit justification
	var count uint64
	for idx := range msg.PreCommitJustification {
		just := &msg.PreCommitJustification[idx]

		if _, ok := eqvVoters[just.AuthorityID]; ok {
			continue
		}

		err := verifyJustification(c.grandpa, just, msg.Round, msg.SetID, precommit)
		if err != nil {
			continue
		}

		if just.Vote.Hash == msg.Hash && just.Vote.Number == msg.Number {
			count++
		}
	}

	if count+uint64(len(eqvVoters)) < c.grandpa.state.threshold() {
		return ErrMinVotesNotMet
	}

	return nil
}

func (c *catchUp) verifyPreVoteJustification(msg *CatchUpResponse) (common.Hash, error) {
	voters := make(map[ed25519.PublicKeyBytes]map[common.Hash]int, len(msg.PreVoteJustification))
	eqVotesByHash := make(map[common.Hash]map[ed25519.PublicKeyBytes]struct{})

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

		err := verifyJustification(c.grandpa, just, msg.Round, msg.SetID, prevote)
		if err != nil {
			continue
		}

		votes[just.Vote.Hash]++
	}

	var prevote common.Hash
	for hash, count := range votes {
		equivocatoryVotes := eqVotesByHash[hash]
		if count+uint64(len(equivocatoryVotes)) >= c.grandpa.state.threshold() {
			prevote = hash
			break
		}
	}

	if prevote.IsEmpty() {
		return prevote, ErrMinVotesNotMet
	}

	return prevote, nil
}
