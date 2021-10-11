package network

import (
	"math/big"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/dgraph-io/ristretto"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/stretchr/testify/require"
)

func TestMessageCache(t *testing.T) {
	cacheSize := 64 << 20 // 64 MB
	msgCache, err := newMessageCache(ristretto.Config{
		NumCounters: int64(float64(cacheSize) * 0.05 * 2),
		MaxCost:     int64(float64(cacheSize) * 0.95),
		BufferItems: 64,
		Cost: func(value interface{}) int64 {
			return int64(1)
		},
	}, 800*time.Millisecond)
	require.NoError(t, err)

	peerID := peer.ID("gossamer")
	msg := &BlockAnnounceMessage{
		ParentHash:     common.Hash{1},
		Number:         big.NewInt(77),
		StateRoot:      common.Hash{2},
		ExtrinsicsRoot: common.Hash{3},
		Digest:         types.NewDigest(),
	}

	ok, err := msgCache.put(peerID, msg)
	require.NoError(t, err)
	require.True(t, ok)

	time.Sleep(time.Millisecond * 500)

	ok = msgCache.exists(peerID, msg)
	require.True(t, ok)

	// TODO: Cache has issues with timeout. https://discuss.dgraph.io/t/setwithttl-doesnt-work/14192
	time.Sleep(3 * time.Second)

	ok = msgCache.exists(peerID, msg)
	require.False(t, ok)
}

func TestMessageCacheError(t *testing.T) {
	cacheSize := 64 << 20 // 64 MB
	msgCache, err := newMessageCache(ristretto.Config{
		NumCounters: int64(float64(cacheSize) * 0.05 * 2),
		MaxCost:     int64(float64(cacheSize) * 0.95),
		BufferItems: 64,
		Cost: func(value interface{}) int64 {
			return int64(1)
		},
	}, 800*time.Millisecond)
	require.NoError(t, err)

	peerID := peer.ID("gossamer")
	msg := &BlockAnnounceHandshake{
		Roles:           4,
		BestBlockNumber: 77,
		BestBlockHash:   common.Hash{1},
		GenesisHash:     common.Hash{2},
	}

	ok, err := msgCache.put(peerID, msg)
	require.Error(t, err, "cache does not support handshake messages")
	require.False(t, ok)
}
