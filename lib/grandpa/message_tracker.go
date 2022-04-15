// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"sync"
	"time"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/libp2p/go-libp2p-core/peer"
)

// tracker keeps track of messages that have been received, but have failed to
// validate with ErrBlockDoesNotExist. These messages may be needed again in the
// case that we are slightly out of sync with the rest of the network.
type tracker struct {
	blockState BlockState
	handler    *MessageHandler
	votes      votesTracker

	// map of commit block hash to commit message
	commitMessages map[common.Hash]*CommitMessage
	mapLock        sync.Mutex
	in             chan *types.Block // receive imported block from BlockState
	stopped        chan struct{}

	catchUpResponseMessageMutex sync.Mutex
	// round(uint64) is used as key and *CatchUpResponse as value
	catchUpResponseMessages map[uint64]*CatchUpResponse
}

func newTracker(bs BlockState, handler *MessageHandler) *tracker {
	const votesCapacity = 1000
	return &tracker{
		blockState:              bs,
		handler:                 handler,
		votes:                   newVotesTracker(votesCapacity),
		commitMessages:          make(map[common.Hash]*CommitMessage),
		mapLock:                 sync.Mutex{},
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

	t.mapLock.Lock()
	defer t.mapLock.Unlock()

	t.votes.add(peerID, message)
}

func (t *tracker) addCommit(cm *CommitMessage) {
	t.mapLock.Lock()
	defer t.mapLock.Unlock()
	t.commitMessages[cm.Vote.Hash] = cm
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
	t.mapLock.Lock()
	defer t.mapLock.Unlock()

	h := b.Header.Hash()
	vms := t.votes.getMessagesForBlockHash(h)
	for _, v := range vms {
		// handleMessage would never error for vote message
		_, err := t.handler.handleMessage(v.from, v.msg)
		if err != nil {
			logger.Warnf("failed to handle vote message %v: %s", v, err)
		}
	}

	t.votes.delete(h)

	if cm, has := t.commitMessages[h]; has {
		_, err := t.handler.handleMessage("", cm)
		if err != nil {
			logger.Warnf("failed to handle commit message %v: %s", cm, err)
		}

		delete(t.commitMessages, h)
	}
}

func (t *tracker) handleTick() {
	t.mapLock.Lock()
	defer t.mapLock.Unlock()

	var blockHashesDone []common.Hash
	t.votes.forEach(func(peerID peer.ID, message *VoteMessage) {
		_, err := t.handler.handleMessage(peerID, message)
		if err != nil {
			// handleMessage would never error for vote message
			logger.Debugf("failed to handle vote message %v from peer id %s: %s", message, peerID, err)
		}

		if message.Round < t.handler.grandpa.state.round && message.SetID == t.handler.grandpa.state.setID {
			blockHashesDone = append(blockHashesDone, message.Message.BlockHash)
		}
	})
	for _, blockHashDone := range blockHashesDone {
		t.votes.delete(blockHashDone)
	}

	for _, cm := range t.commitMessages {
		_, err := t.handler.handleMessage("", cm)
		if err != nil {
			logger.Debugf("failed to handle commit message %v: %s", cm, err)
			continue
		}

		delete(t.commitMessages, cm.Vote.Hash)
	}
}
