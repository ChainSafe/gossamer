// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"sync"
	"time"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto/ed25519"
)

// tracker keeps track of messages that have been received, but have failed to
// validate with ErrBlockDoesNotExist. These messages may be needed again in the
// case that we are slightly out of sync with the rest of the network.
type tracker struct {
	blockState BlockState
	handler    *MessageHandler
	// map of vote block hash -> array of VoteMessages for that hash
	voteMessages map[common.Hash]map[ed25519.PublicKeyBytes]*networkVoteMessage
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
	return &tracker{
		blockState:              bs,
		handler:                 handler,
		voteMessages:            make(map[common.Hash]map[ed25519.PublicKeyBytes]*networkVoteMessage),
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

func (t *tracker) addVote(v *networkVoteMessage) {
	if v.msg == nil {
		return
	}

	t.mapLock.Lock()
	defer t.mapLock.Unlock()

	msgs, has := t.voteMessages[v.msg.Message.Hash]
	if !has {
		msgs = make(map[ed25519.PublicKeyBytes]*networkVoteMessage)
		t.voteMessages[v.msg.Message.Hash] = msgs
	}

	msgs[v.msg.Message.AuthorityID] = v
}

func (t *tracker) addCommit(cm *CommitMessage) {
	t.mapLock.Lock()
	defer t.mapLock.Unlock()
	t.commitMessages[cm.Vote.Hash] = cm
}

func (t *tracker) addCatchUpResponse(cr *CatchUpResponse) {
	t.catchUpResponseMessageMutex.Lock()
	defer t.catchUpResponseMessageMutex.Unlock()
	t.catchUpResponseMessages[cr.Round] = cr
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
	if vms, has := t.voteMessages[h]; has {
		for _, v := range vms {
			// handleMessage would never error for vote message
			_, err := t.handler.handleMessage(v.from, v.msg)
			if err != nil {
				logger.Warnf("failed to handle vote message %v: %s", v, err)
			}
		}

		delete(t.voteMessages, h)
	}

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

	for _, vms := range t.voteMessages {
		for _, v := range vms {
			// handleMessage would never error for vote message
			_, err := t.handler.handleMessage(v.from, v.msg)
			if err != nil {
				logger.Debugf("failed to handle vote message %v: %s", v, err)
			}

			if v.msg.Round < t.handler.grandpa.state.round && v.msg.SetID == t.handler.grandpa.state.setID {
				delete(t.voteMessages, v.msg.Message.Hash)
			}
		}
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
