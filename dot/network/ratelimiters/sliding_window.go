// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package ratelimiters

import (
	"sync"
	"time"

	"github.com/ChainSafe/gossamer/lib/common"
	lrucache "github.com/ChainSafe/gossamer/lib/utils/lru-cache"
)

const DefaultMaxSlidingWindowTime = 1 * time.Minute
const DefaultMaxCachedRequestSize = 500

// SlidingWindowRateLimiter is a rate limiter implementation designed to prevent
// more than `maxReqs` requests from being processed within a `windowSize` time window
type SlidingWindowRateLimiter struct {
	mu         sync.Mutex
	limits     *lrucache.LRUCache[common.Hash, []time.Time]
	maxReqs    uint32
	windowSize time.Duration
}

// NewSlidingWindowRateLimiter creates a new SpamLimiter with the given maximum number of requests
func NewSlidingWindowRateLimiter(maxReqs uint32, windowSize time.Duration) *SlidingWindowRateLimiter {
	return &SlidingWindowRateLimiter{
		limits:     lrucache.NewLRUCache[common.Hash, []time.Time](DefaultMaxCachedRequestSize),
		maxReqs:    maxReqs,
		windowSize: windowSize,
	}
}

// AddRequest adds a request to the SpamLimiter
func (rl *SlidingWindowRateLimiter) AddRequest(id common.Hash) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	recentRequests := rl.recentRequests(id)

	// Add the current request and update the cache
	recentRequests = append(recentRequests, time.Now())
	rl.limits.Put(id, recentRequests)
}

// IsLimitExceeded returns true if the limit is exceeded for the given peer and hash
func (rl *SlidingWindowRateLimiter) IsLimitExceeded(id common.Hash) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	recentRequests := rl.recentRequests(id)
	rl.limits.Put(id, recentRequests)

	return uint32(len(recentRequests)) > rl.maxReqs
}

func (rl *SlidingWindowRateLimiter) recentRequests(id common.Hash) []time.Time {
	// Get the timestamps for the hash
	timestamps := rl.limits.Get(id)
	if timestamps == nil {
		return []time.Time{}
	}

	now := time.Now()

	// Filter requests that are within the time window
	var recentRequests []time.Time
	for _, t := range timestamps {
		if now.Sub(t) <= rl.windowSize {
			recentRequests = append(recentRequests, t)
		}
	}

	return recentRequests
}
