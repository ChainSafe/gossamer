package network

import (
	"time"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/dgraph-io/ristretto"
	"github.com/libp2p/go-libp2p-core/peer"
)

// MsgCacheTTL is default duration a key-value will be stored in MessageCache.
var MsgCacheTTL = 5 * time.Minute

// MessageCache is used to detect duplicated messages per peer.
type MessageCache struct {
	cache *ristretto.Cache
	ttl   time.Duration
}

// NewMessageCache creates a new MessageCache which takes config and TTL duration.
func NewMessageCache(config ristretto.Config, ttl time.Duration) (*MessageCache, error) {
	cache, err := ristretto.NewCache(&config)
	if err != nil {
		return nil, err
	}

	if ttl == 0 {
		ttl = MsgCacheTTL
	}

	return &MessageCache{cache: cache, ttl: ttl}, nil
}

// Put appends peer ID and message data and stores it in cache with TTL.
func (m *MessageCache) Put(peer peer.ID, msg string) (bool, error) {
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
func (m *MessageCache) Exists(peer peer.ID, msg string) bool {
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
