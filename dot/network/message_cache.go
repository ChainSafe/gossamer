package network

import (
	"time"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/dgraph-io/ristretto"
	"github.com/libp2p/go-libp2p-core/peer"
)

var MsgCacheTTL = 60 * time.Second

type messageCache struct {
	cache *ristretto.Cache
	ttl   time.Duration
}

// NewMessageCache creates a new messageCache which takes config and TTL duration.
func NewMessageCache(config ristretto.Config, ttl time.Duration) (*messageCache, error) {
	cache, err := ristretto.NewCache(&config)
	if err != nil {
		return nil, err
	}

	if ttl == 0 {
		ttl = MsgCacheTTL
	}

	return &messageCache{cache: cache, ttl: ttl}, nil
}

// Put appends peer ID, message data and set it to cache with ttl
func (m *messageCache) Put(peer peer.ID, msg string) (bool, error) {
	key, err := generateCacheKey(peer, msg)
	if err != nil {
		return false, err
	}

	_, ok := m.cache.Get(key)
	if ok {
		return false, nil
	}

	ok = m.cache.SetWithTTL(key, "", 1, m.ttl)
	return ok, nil
}

// Exists checks if peer ID, message data exist in cache
func (m *messageCache) Exists(peer peer.ID, msg string) bool {
	key, err := generateCacheKey(peer, msg)
	if err != nil {
		return false
	}

	_, ok := m.cache.Get(key)
	return ok
}

func generateCacheKey(peer peer.ID, msg string) ([]byte, error) {
	peerBytes, err := peer.Marshal()
	if err != nil {
		return nil, err
	}

	peerMsgHash, err := common.Blake2bHash(append(peerBytes, msg...))
	if err != nil {
		return nil, err
	}

	return peerMsgHash.ToBytes(), nil
}
