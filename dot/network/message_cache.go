package network

import (
	"time"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/dgraph-io/ristretto"
	"github.com/libp2p/go-libp2p-core/peer"
)

// msgCacheTTL is default duration a key-value will be stored in messageCache.
var msgCacheTTL = 5 * time.Minute

// messageCache is used to detect duplicated messages per peer.
type messageCache struct {
	cache *ristretto.Cache
	ttl   time.Duration
}

// newMessageCache creates a new messageCache which takes config and TTL duration.
func newMessageCache(config ristretto.Config, ttl time.Duration) (*messageCache, error) {
	cache, err := ristretto.NewCache(&config)
	if err != nil {
		return nil, err
	}

	if ttl == 0 {
		ttl = msgCacheTTL
	}

	return &messageCache{cache: cache, ttl: ttl}, nil
}

// Put appends peer ID and message data and stores it in cache with TTL.
func (m *messageCache) Put(peer peer.ID, msg []byte) (bool, error) {
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

// Exists checks if <peer ID, message data> exist in cache.
func (m *messageCache) Exists(peer peer.ID, msg []byte) bool {
	key, err := generateCacheKey(peer, msg)
	if err != nil {
		return false
	}

	_, ok := m.cache.Get(key)
	return ok
}

func generateCacheKey(peer peer.ID, msg []byte) ([]byte, error) {
	peerMsgHash, err := common.Blake2bHash(append([]byte(peer), msg...))
	if err != nil {
		return nil, err
	}

	return peerMsgHash.ToBytes(), nil
}
