// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package network

import (
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/assert"
)

func TestSpamLimiter_AddRequestAndIsLimitExceeded(t *testing.T) {
	t.Parallel()

	// Create a SpamLimiter with a limit of 5 requests and a time window of 10 seconds
	limiter := NewSpamLimiter(5, 10*time.Second)

	peerID := peer.ID("peer1")
	hash := common.Hash{0x01}

	// Add 5 requests for the same peer and hash
	for i := 0; i < 5; i++ {
		limiter.AddRequest(peerID, hash)
	}

	// Limit should not be exceeded after 5 requests
	assert.False(t, limiter.IsLimitExceeded(peerID, hash))

	// Add one more request and check that the limit is exceeded
	limiter.AddRequest(peerID, hash)
	assert.True(t, limiter.IsLimitExceeded(peerID, hash))
}

func TestSpamLimiter_WindowExpiry(t *testing.T) {
	t.Parallel()

	// Create a SpamLimiter with a limit of 3 requests and a time window of 2 seconds
	limiter := NewSpamLimiter(3, 1*time.Second)

	peerID := peer.ID("peer2")
	hash := common.Hash{0x02}

	// Add 3 requests
	for i := 0; i < 3; i++ {
		limiter.AddRequest(peerID, hash)
	}

	// Limit should not be exceeded
	assert.False(t, limiter.IsLimitExceeded(peerID, hash))

	// Wait for the time window to expire
	time.Sleep(2 * time.Second)

	// Add another request, should be considered as the first in a new window
	limiter.AddRequest(peerID, hash)
	assert.False(t, limiter.IsLimitExceeded(peerID, hash))
}

func TestSpamLimiter_DifferentPeersAndHashes(t *testing.T) {
	// Create a SpamLimiter with a limit of 2 requests and a time window of 5 seconds
	limiter := NewSpamLimiter(2, 5*time.Second)

	peerID1 := peer.ID("peer1")
	peerID2 := peer.ID("peer2")
	hash1 := common.Hash{0x01}
	hash2 := common.Hash{0x02}

	// Add requests for peerID1 and hash1
	limiter.AddRequest(peerID1, hash1)
	limiter.AddRequest(peerID1, hash1)

	// Add requests for peerID2 and hash2
	limiter.AddRequest(peerID2, hash2)
	limiter.AddRequest(peerID2, hash2)

	// No limit should be exceeded yet
	assert.False(t, limiter.IsLimitExceeded(peerID1, hash1))
	assert.False(t, limiter.IsLimitExceeded(peerID2, hash2))

	// Add another request for each and check that the limit is exceeded
	limiter.AddRequest(peerID1, hash1)
	assert.True(t, limiter.IsLimitExceeded(peerID1, hash1))

	limiter.AddRequest(peerID2, hash2)
	assert.True(t, limiter.IsLimitExceeded(peerID2, hash2))
}
