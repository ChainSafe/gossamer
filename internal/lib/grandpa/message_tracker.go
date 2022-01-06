// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"sync"

	"github.com/ChainSafe/gossamer/internal/dot/types"
	"github.com/ChainSafe/gossamer/internal/lib/common"
	"github.com/ChainSafe/gossamer/internal/lib/crypto/ed25519"
)

// tracker keeps track of messages that have been received that have failed to validate with ErrBlockDoesNotExist
// these messages may be needed again in the case that we are slightly out of sync with the rest of the network
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
}

func newTracker(bs BlockState, handler *MessageHandler) *tracker {
	in := bs.GetImportedBlockNotifierChannel()

	return &tracker{
		blockState:     bs,
		handler:        handler,
		voteMessages:   make(map[common.Hash]map[ed25519.PublicKeyBytes]*networkVoteMessage),
		commitMessages: make(map[common.Hash]*CommitMessage),
		mapLock:        sync.Mutex{},
		in:             in,
		stopped:        make(chan struct{}),
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

func (t *tracker) handleBlocks() {
	for {
		select {
		case b := <-t.in:
			if b == nil {
				continue
			}

			t.handleBlock(b)
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
