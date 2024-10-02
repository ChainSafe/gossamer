package network

import (
	"sync"
	"time"

	"github.com/ChainSafe/gossamer/lib/common"
	lrucache "github.com/ChainSafe/gossamer/lib/utils/lru-cache"
	"github.com/libp2p/go-libp2p/core/peer"
)

const MaxTimeWindow = 10 * time.Second
const MaxCachedPeers = 100
const MaxCachedRequests = 100

type SpamLimiter struct {
	mu         sync.Mutex
	limits     *lrucache.LRUCache[peer.ID, *lrucache.LRUCache[common.Hash, []time.Time]]
	maxReqs    uint32
	windowSize time.Duration
}

// NewSpamLimiter creates a new SpamLimiter with the given maximum number of requests
func NewSpamLimiter(maxReqs uint32, windowSize time.Duration) *SpamLimiter {
	return &SpamLimiter{
		limits:     lrucache.NewLRUCache[peer.ID, *lrucache.LRUCache[common.Hash, []time.Time]](MaxCachedPeers),
		maxReqs:    maxReqs,
		windowSize: windowSize,
	}
}

// AddRequest adds a request to the SpamLimiter
func (rl *SpamLimiter) AddRequest(peer peer.ID, hashedRequest common.Hash) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	// Get or create the internal cache for the peer
	peerCache := rl.limits.Get(peer)
	if peerCache == nil {
		peerCache = lrucache.NewLRUCache[common.Hash, []time.Time](MaxCachedRequests)
		rl.limits.Put(peer, peerCache)
	}

	// Get the timestamps for the hash
	timestamps := peerCache.Get(hashedRequest)
	now := time.Now()

	// Filter requests that are within the time window
	var recentRequests []time.Time
	for _, t := range timestamps {
		if now.Sub(t) <= rl.windowSize {
			recentRequests = append(recentRequests, t)
		}
	}

	// Add the current request and update the cache
	recentRequests = append(recentRequests, now)
	peerCache.Put(hashedRequest, recentRequests)
}

// IsLimitExceeded returns true if the limit is exceeded for the given peer and hash
func (rl *SpamLimiter) IsLimitExceeded(peer peer.ID, hashedRequest common.Hash) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	// Get the internal cache for the peer
	peerCache := rl.limits.Get(peer)
	if peerCache == nil {
		return false
	}

	// Get the timestamps for the hash
	timestamps := peerCache.Get(hashedRequest)
	now := time.Now()

	// Filter requests that are within the time window
	var recentRequests []time.Time
	for _, t := range timestamps {
		if now.Sub(t) <= rl.windowSize {
			recentRequests = append(recentRequests, t)
		}
	}

	// Update the cache with the recent requests
	peerCache.Put(hashedRequest, recentRequests)

	return uint32(len(recentRequests)) > rl.maxReqs
}
