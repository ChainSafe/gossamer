// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package ratelimiters

import (
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/stretchr/testify/assert"
)

func TestSpamLimiter_AddRequestAndCheckLimitExceeded(t *testing.T) {
	t.Parallel()

	// Create a SpamLimiter with a limit of 5 requests and a time window of 10 seconds
	limiter := NewSlidingWindowRateLimiter(5, 10*time.Second)

	hash := common.Hash{0x01}

	// Add 5 requests for the same peer and hash
	for i := 0; i < 5; i++ {
		limiter.AddRequest(hash)
	}

	// Limit should not be exceeded after 5 requests
	assert.False(t, limiter.IsLimitExceeded(hash))

	// Add one more request and check that the limit is exceeded
	limiter.AddRequest(hash)
	assert.True(t, limiter.IsLimitExceeded(hash))
}

func TestSpamLimiter_WindowExpiry(t *testing.T) {
	t.Parallel()

	// Create a SpamLimiter with a limit of 3 requests and a time window of 2 seconds
	limiter := NewSlidingWindowRateLimiter(3, 1*time.Second)

	hash := common.Hash{0x02}

	// Add 3 requests
	for i := 0; i < 3; i++ {
		limiter.AddRequest(hash)
	}

	// Limit should not be exceeded
	assert.False(t, limiter.IsLimitExceeded(hash))

	// Wait for the time window to expire
	time.Sleep(2 * time.Second)

	// Add another request, should be considered as the first in a new window
	limiter.AddRequest(hash)
	assert.False(t, limiter.IsLimitExceeded(hash))
}

func TestSpamLimiter_DifferentPeersAndHashes(t *testing.T) {
	t.Parallel()

	// Create a SpamLimiter with a limit of 2 requests and a time window of 5 seconds
	limiter := NewSlidingWindowRateLimiter(2, 5*time.Second)

	hash1 := common.Hash{0x01}
	hash2 := common.Hash{0x02}

	// Add requests for peerID1 and hash1
	limiter.AddRequest(hash1)
	limiter.AddRequest(hash1)

	// Add requests for peerID2 and hash2
	limiter.AddRequest(hash2)
	limiter.AddRequest(hash2)

	// No limit should be exceeded yet
	assert.False(t, limiter.IsLimitExceeded(hash1))
	assert.False(t, limiter.IsLimitExceeded(hash2))

	// Add another request for each and check that the limit is exceeded
	limiter.AddRequest(hash1)
	assert.True(t, limiter.IsLimitExceeded(hash1))

	limiter.AddRequest(hash2)
	assert.True(t, limiter.IsLimitExceeded(hash2))
}
