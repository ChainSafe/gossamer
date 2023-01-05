// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"sync"
	"time"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
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
	// // round(uint64) is used as key and *CatchUpResponse as value
	// catchUpResponseMessages map[uint64]*CatchUpResponse

	// block hash is used as key and *CatchUpResponse as value
	catchUpResponseMessages map[common.Hash]*networkCatchUpResponseMessage
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
		catchUpResponseMessages: make(map[common.Hash]*networkCatchUpResponseMessage),
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

func (t *tracker) addCatchUpResponse(cr *networkCatchUpResponseMessage) {
	t.catchUpResponseMessageMutex.Lock()
	defer t.catchUpResponseMessageMutex.Unlock()

	t.catchUpResponseMessages[cr.msg.Hash] = cr
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
		err := t.handler.handleMessage(v.from, v.msg)
		if err != nil {
			logger.Warnf("failed to handle vote message %v: %s", v, err)
		}
	}

	// delete block hash that may or may not be in the tracker.
	// TODO: Check if we should delete all vote messages for h,
	// if fail to process a few of the vote messages.
	t.votes.delete(h)

	cm := t.commits.message(h)
	if cm != nil {
		err := t.handler.handleMessage("", cm)
		if err != nil {
			logger.Warnf("failed to handle commit message %v: %s", cm, err)
		} else {
			t.commits.delete(h)
		}
	}

	if cr, has := t.catchUpResponseMessages[h]; has {
		err := t.handler.handleMessage(cr.from, cr.msg)
		if err != nil {
			logger.Warnf("failed to handle catch up response message %v: %s", cr, err)
		} else {
			delete(t.catchUpResponseMessages, h)
		}
	}

}

func (t *tracker) handleTick() {
	for _, networkVoteMessage := range t.votes.networkVoteMessages() {
		peerID := networkVoteMessage.from
		message := networkVoteMessage.msg
		err := t.handler.handleMessage(peerID, message)
		if err != nil {
			// handleMessage would never error for vote message
			logger.Debugf("failed to handle vote message %v from peer id %s: %s", message, peerID, err)
		}

		if message.Round < t.handler.grandpa.state.round && message.SetID == t.handler.grandpa.state.setID {
			t.votes.delete(message.Message.BlockHash)
		}
	}

	t.commits.forEach(func(cm *CommitMessage) {
		err := t.handler.handleMessage("", cm)
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

type networkCatchUpResponseMessage struct {
	from peer.ID
	msg  *CatchUpResponse
}
