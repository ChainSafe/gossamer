package network

import (
	"testing"
	"time"

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

	peerID := peer.ID("gossamer")
	msgData := []byte("testData")
	require.NoError(t, err)

	ok, err := msgCache.Put(peerID, msgData)
	require.NoError(t, err)
	require.True(t, ok)

	time.Sleep(750 * time.Millisecond)

	ok = msgCache.Exists(peerID, msgData)
	require.True(t, ok)

	time.Sleep(50 * time.Millisecond)

	ok = msgCache.Exists(peerID, msgData)
	require.False(t, ok)
}
