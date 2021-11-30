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
	"reflect"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto/ed25519"
	"github.com/ChainSafe/gossamer/pkg/scale"

	"github.com/libp2p/go-libp2p-core/peer"
)

type catchUp struct {
	isAuthority           bool
	isStarted             *atomic.Value
	authorityPeers        *sync.Map //map[peer.ID]*NeighbourMessage
	highestFinalizedRound uint64    // TODO: get this from neighbour messages
	responseCh            <-chan *CatchUpResponse

	grandpa *Service
	network Network
}

func newCatchUp(isAuthority bool, grandpa *Service, network Network, responseCh <-chan *CatchUpResponse) *catchUp {
	isStarted := new(atomic.Value)
	isStarted.Store(false)

	return &catchUp{
		isAuthority:    isAuthority,
		isStarted:      isStarted,
		grandpa:        grandpa,
		network:        network,
		authorityPeers: new(sync.Map),
		responseCh:     responseCh,
	}
}

func (c *catchUp) addPeer(id peer.ID) {
	c.authorityPeers.Store(id, nil) // TODO: store neighbour message info
}

func (c *catchUp) doCatchUp(from peer.ID, setID, round uint64) error {
	if !c.isAuthority {
		// only authorities need to participate in catch-up
		return nil
	}

	if c.isStarted.Load().(bool) {
		return errors.New("already started")
	}

	logger.Debugf("beginning catch-up process", "setID", setID, "target round", round)

	currSetID, err := c.grandpa.grandpaState.GetCurrentSetID()
	if err != nil {
		return err
	}

	if setID > currSetID {
		// we aren't ready to catch up yet, wait until we've synced enough of the
		// chain to know the authorities at this set ID
		logger.Debugf("ignoring catch-up, not at target set ID", "current", currSetID, "target", setID)
		return nil
	}

	// pause voting while we do catch-up
	c.grandpa.paused.Store(true)

	c.isStarted.Store(true)
	defer c.isStarted.Store(false)

	resp, err := c.sendCatchUpRequest(from, newCatchUpRequest(round, setID))
	if err != nil {
		return err
	}

	if resp == nil {
		// TODO: request from other peers
		return errors.New("peer did not send catch up response :(")
	}

	logger.Debugf("got catch up response", "resp", resp)

	// make sure grandpa.state.setID and grandpa.state.voters are set correctly before verifying response
	err = c.grandpa.updateAuthorities()
	if err != nil {
		return err
	}

	return c.handleCatchUpResponse(resp)
}

func (c *catchUp) sendCatchUpRequest(to peer.ID, req *CatchUpRequest) (*CatchUpResponse, error) {
	cm, err := req.ToConsensusMessage()
	if err != nil {
		return nil, err
	}

	err = c.network.SendMessage(to, cm)
	if err != nil {
		return nil, err
	}

	timer := time.NewTimer(time.Second * 5)
	defer timer.Stop()

	select {
	case resp := <-c.responseCh:
		return resp, nil
	case <-timer.C:
		return nil, errors.New("timeout")
	}
}

// TODO: track authority peers and only take into account neighbour messages from them for catch-up
func (c *catchUp) addNeighbourMessage(from peer.ID, msg *NeighbourMessage) {
	if msg.SetID == c.grandpa.state.setID && msg.Round > c.highestFinalizedRound {
		c.highestFinalizedRound = msg.Round
	}

	c.authorityPeers.Store(from, msg)
}

func (c *catchUp) handleCatchUpResponse(msg *CatchUpResponse) error {
	logger.Debugf("handling catch up response", "round", msg.Round, "setID", msg.SetID, "hash", msg.Hash)

	// if we aren't currently expecting a catch up response, return
	if !c.grandpa.paused.Load().(bool) {
		logger.Debug("not currently paused, ignoring catch up response")
		return nil
	}

	if msg.SetID != c.grandpa.state.setID {
		return ErrSetIDMismatch
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

	if msg.Round != c.highestFinalizedRound+1 {
		return nil
	}

	// update state and signal to grandpa we are ready to initiate
	head, err := c.grandpa.blockState.GetHeader(msg.Hash)
	if err != nil {
		return err
	}

	c.grandpa.head = head
	c.grandpa.state.round = msg.Round
	close(c.grandpa.resumed)
	c.grandpa.resumed = make(chan struct{})
	c.grandpa.paused.Store(false)
	logger.Debugf("caught up to round; unpaused service", "set ID", c.grandpa.state.setID, "round", c.grandpa.state.round)
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

// func (c *catchUp) verifyPreVoteJustification(msg *CatchUpResponse) (common.Hash, error) {
// 	// verify pre-vote justification, returning the pre-voted block if there is one
// 	votes := make(map[common.Hash]uint64)

// 	for _, just := range msg.PreVoteJustification {
// 		err := verifyJustification(c.grandpa.authorities(), &just, msg.Round, msg.SetID, prevote)
// 		if err != nil {
// 			continue
// 		}

// 		votes[just.Vote.Hash]++
// 	}

// 	var prevote common.Hash
// 	for hash, count := range votes {
// 		if count >= c.grandpa.state.threshold() {
// 			prevote = hash
// 			break
// 		}
// 	}

// 	if (prevote == common.Hash{}) {
// 		return prevote, ErrMinVotesNotMet
// 	}

// 	return prevote, nil
// }

// func (c *catchUp) verifyPreCommitJustification(msg *CatchUpResponse) error {
// 	// verify pre-commit justification
// 	count := 0
// 	for _, just := range msg.PreCommitJustification {
// 		err := verifyJustification(c.grandpa.authorities(), &just, msg.Round, msg.SetID, precommit)
// 		if err != nil {
// 			continue
// 		}

// 		if just.Vote.Hash == msg.Hash && just.Vote.Number == msg.Number {
// 			count++
// 		}
// 	}

// 	if uint64(count) < c.grandpa.state.threshold() {
// 		return ErrMinVotesNotMet
// 	}

// 	return nil
// }

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

		err := verifyJustification(c.grandpa.authorities(), just, msg.Round, msg.SetID, prevote)
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

		err := verifyJustification(c.grandpa.authorities(), just, msg.Round, msg.SetID, precommit)
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

// func (h *MessageHandler) verifyJustification(just *SignedVote, round, setID uint64, stage Subround) error {
// 	// verify signature
// 	msg, err := scale.Marshal(FullVote{
// 		Stage: stage,
// 		Vote:  just.Vote,
// 		Round: round,
// 		SetID: setID,
// 	})
// 	if err != nil {
// 		return err
// 	}

// 	pk, err := ed25519.NewPublicKey(just.AuthorityID[:])
// 	if err != nil {
// 		return err
// 	}

// 	ok, err := pk.Verify(msg, just.Signature[:])
// 	if err != nil {
// 		return err
// 	}

// 	if !ok {
// 		return ErrInvalidSignature
// 	}

// 	// verify authority in justification set
// 	authFound := false

// 	for _, auth := range h.grandpa.authorities() {
// 		justKey, err := just.AuthorityID.Encode()
// 		if err != nil {
// 			return err
// 		}
// 		if reflect.DeepEqual(auth.Key.Encode(), justKey) {
// 			authFound = true
// 			break
// 		}
// 	}
// 	if !authFound {
// 		return ErrVoterNotFound
// 	}
// 	return nil
// }

func verifyJustification(authorities []*types.Authority, just *SignedVote, round, setID uint64, stage Subround) error {
	// verify signature
	msg, err := scale.Marshal(&FullVote{
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
	for _, auth := range authorities {
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
