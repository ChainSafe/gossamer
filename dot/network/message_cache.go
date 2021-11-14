// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package network

import (
	"errors"
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

// put appends peer ID and message data and stores it in cache with TTL.
func (m *messageCache) put(peer peer.ID, msg NotificationsMessage) (bool, error) {
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

// exists checks if <peer ID, message> exist in cache.
func (m *messageCache) exists(peer peer.ID, msg NotificationsMessage) bool {
	key, err := generateCacheKey(peer, msg)
	if err != nil {
		return false
	}

	_, ok := m.cache.Get(key)
	return ok
}

func generateCacheKey(peer peer.ID, msg NotificationsMessage) ([]byte, error) {
	if msg.IsHandshake() {
		return nil, errors.New("cache does not support handshake messages")
	}

	peerMsgHash, err := common.Blake2bHash(append([]byte(peer), msg.Hash().ToBytes()...))
	if err != nil {
		return nil, err
	}

	return peerMsgHash.ToBytes(), nil
}
