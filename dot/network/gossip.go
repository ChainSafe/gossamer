// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package network

import (
	"fmt"
	"sync"

	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/lib/common"
)

// gossip submodule
type gossip struct {
	logger    log.LeveledLogger
	seenMap   map[common.Hash]struct{}
	seenMutex sync.RWMutex
}

// newGossip creates a new gossip message tracker
func newGossip() *gossip {
	return &gossip{
		logger:  log.NewFromGlobal(log.AddContext("module", "gossip")),
		seenMap: make(map[common.Hash]struct{}),
	}
}

// hasSeen checks if we have seen given message before.
func (g *gossip) hasSeen(msg NotificationsMessage) (bool, error) {
	msgHash, err := msg.Hash()
	if err != nil {
		return false, fmt.Errorf("could not hash notification message: %w", err)
	}

	g.seenMutex.Lock()
	defer g.seenMutex.Unlock()

	// check if message has not been seen
	_, ok := g.seenMap[msgHash]
	if !ok {
		// set message to has been seen
		g.seenMap[msgHash] = struct{}{}
		return false, nil
	}

	return true, nil
}
