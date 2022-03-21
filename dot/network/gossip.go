// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package network

import (
	"fmt"
	"sync"

	"github.com/ChainSafe/gossamer/internal/log"
)

// gossip submodule
type gossip struct {
	logger log.LeveledLogger
	seen   *sync.Map
}

// newGossip creates a new gossip message tracker
func newGossip() *gossip {
	return &gossip{
		logger: log.NewFromGlobal(log.AddContext("module", "gossip")),
		seen:   &sync.Map{},
	}
}

// hasSeen broadcasts messages that have not been seen
func (g *gossip) hasSeen(msg NotificationsMessage) (bool, error) {
	// check if message has not been seen
	msgHash, err := msg.Hash()
	if err != nil {
		return false, fmt.Errorf("could not hash notification message: %w", err)
	}
	if seen, ok := g.seen.Load(msgHash); !ok || !seen.(bool) {
		// set message to has been seen
		g.seen.Store(msgHash, true)
		return false, nil
	}

	return true, nil
}
