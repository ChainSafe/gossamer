// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"sync"
	"time"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/libp2p/go-libp2p/core/peer"
)

// tracker keeps track of messages that have been received, but have failed to
// validate with ErrBlockDoesNotExist. These messages may be needed again in the
// case that we are slightly out of sync with the rest of the network.
type tracker struct {
	blockState BlockState
	handler    *MessageHandler
	votes      votesTracker
	commits    commitsTracker
	in         chan *types.Block // receive imported block from BlockState
	stopped    chan struct{}

	catchUpResponseMessageMutex sync.Mutex
	// round(uint64) is used as key and *CatchUpResponse as value
	catchUpResponseMessages map[uint64]*CatchUpResponse
}

func newTracker(bs BlockState, handler *MessageHandler) *tracker {
	const (
		votesCapacity   = 1000
		commitsCapacity = 1000
	)
	return &tracker{
		blockState:              bs,
		handler:                 handler,
		votes:                   newVotesTracker(votesCapacity),
		commits:                 newCommitsTracker(commitsCapacity),
		in:                      bs.GetImportedBlockNotifierChannel(),
		stopped:                 make(chan struct{}),
		catchUpResponseMessages: make(map[uint64]*CatchUpResponse),
	}
}

func (t *tracker) start() {
	go t.handleBlocks()
}

func (t *tracker) stop() {
	close(t.stopped)
	t.blockState.FreeImportedBlockNotifierChannel(t.in)
}

func (t *tracker) addVote(peerID peer.ID, message *VoteMessage) {
	if message == nil {
		return
	}

	t.votes.add(peerID, message)
}

func (t *tracker) addCommit(cm *CommitMessage) {
	t.commits.add(cm)
}

func (t *tracker) addCatchUpResponse(_ *CatchUpResponse) {
	t.catchUpResponseMessageMutex.Lock()
	defer t.catchUpResponseMessageMutex.Unlock()
	// uncomment when usage is setup properly, see #1531
	// t.catchUpResponseMessages[cr.Round] = cr
}

func (t *tracker) handleBlocks() {
	const timeout = time.Second
	ticker := time.NewTicker(timeout)
	defer ticker.Stop()

	for {
		select {
		case b := <-t.in:
			if b == nil {
				continue
			}

			t.handleBlock(b)
		case <-ticker.C:
			t.handleTick()
		case <-t.stopped:
			return
		}
	}
}

func (t *tracker) handleBlock(b *types.Block) {
	h := b.Header.Hash()
	vms := t.votes.messages(h)
	for _, v := range vms {
		// handleMessage would never error for vote message
		_, err := t.handler.handleMessage(v.from, v.msg)
		if err != nil {
			logger.Warnf("failed to handle vote message %v: %s", v, err)
		}
	}

	// delete block hash that may or may not be in the tracker.
	t.votes.delete(h)

	cm := t.commits.message(h)
	if cm != nil {
		_, err := t.handler.handleMessage("", cm)
		if err != nil {
			logger.Warnf("failed to handle commit message %v: %s", cm, err)
		}

		t.commits.delete(h)
	}
}

func (t *tracker) handleTick() {
	for _, networkVoteMessage := range t.votes.networkVoteMessages() {
		peerID := networkVoteMessage.from
		message := networkVoteMessage.msg
		_, err := t.handler.handleMessage(peerID, message)
		if err != nil {
			// handleMessage would never error for vote message
			logger.Debugf("failed to handle vote message %v from peer id %s: %s", message, peerID, err)
		}

		if message.Round < t.handler.grandpa.state.round && message.SetID == t.handler.grandpa.state.setID {
			t.votes.delete(message.Message.BlockHash)
		}
	}

	t.commits.forEach(func(cm *CommitMessage) {
		_, err := t.handler.handleMessage("", cm)
		if err != nil {
			logger.Debugf("failed to handle commit message %v: %s", cm, err)
			return
		}

		// deleting while iterating is safe to do since
		// each block hash has at most 1 commit message we
		// just handled above.
		t.commits.delete(cm.Vote.Hash)
	})
}
