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
	"sync"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto/ed25519"
)

// tracker keeps track of messages that have been received that have failed to validate with ErrBlockDoesNotExist
// these messages may be needed again in the case that we are slightly out of sync with the rest of the network
type tracker struct {
	blockState     BlockState
	handler        *MessageHandler
	voteMessages   map[common.Hash]map[ed25519.PublicKeyBytes]*networkVoteMessage // map of vote block hash -> array of VoteMessages for that hash
	commitMessages map[common.Hash]*CommitMessage                                 // map of commit block hash to commit message
	mapLock        sync.Mutex
	in             chan *types.Block // receive imported block from BlockState
	chanID         byte              // BlockState channel ID
	//out            chan<- *networkVoteMessage // send a VoteMessage back to grandpa. corresponds to grandpa's in channel
	stopped chan struct{}
}

func newTracker(bs BlockState, handler *MessageHandler) (*tracker, error) {
	in := make(chan *types.Block, 16)
	id, err := bs.RegisterImportedChannel(in)
	if err != nil {
		return nil, err
	}

	return &tracker{
		blockState:     bs,
		handler:        handler,
		voteMessages:   make(map[common.Hash]map[ed25519.PublicKeyBytes]*networkVoteMessage),
		commitMessages: make(map[common.Hash]*CommitMessage),
		mapLock:        sync.Mutex{},
		in:             in,
		chanID:         id,
		//out:            out,
		stopped: make(chan struct{}),
	}, nil
}

func (t *tracker) start() {
	go t.handleBlocks()
}

func (t *tracker) stop() {
	close(t.stopped)
	t.blockState.UnregisterImportedChannel(t.chanID)
	close(t.in)
}

func (t *tracker) addVote(v *networkVoteMessage) {
	if v.msg == nil {
		return
	}

	t.mapLock.Lock()
	defer t.mapLock.Unlock()

	msgs, has := t.voteMessages[v.msg.Message.Hash]
	if !has {
		t.voteMessages[v.msg.Message.Hash] = make(map[ed25519.PublicKeyBytes]*networkVoteMessage)
		msgs = t.voteMessages[v.msg.Message.Hash]
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

			t.mapLock.Lock()
			defer t.mapLock.Unlock()

			h := b.Header.Hash()
			if vms, has := t.voteMessages[h]; has {
				for _, v := range vms {
					_, err := t.handler.handleMessage(v.from, v.msg)
					if err != nil {
						logger.Warn("failed to handle vote message", "message", v, "error", err)
					}
				}
			}

			if cm, has := t.commitMessages[h]; has {
				_, err := t.handler.handleMessage("", cm)
				if err != nil {
					logger.Warn("failed to handle commit message", "message", cm, "error", err)
				}
			}
		case <-t.stopped:
			return
		}
	}
}
