//go:build integration

// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package network

import (
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/dgraph-io/ristretto/v2"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/require"
)

func TestMessageCache(t *testing.T) {
	t.Parallel()

	cacheSize := 64 << 20 // 64 MB
	msgCache, err := newMessageCache(ristretto.Config[[]byte, string]{
		NumCounters: int64(float64(cacheSize) * 0.05 * 2),
		MaxCost:     int64(float64(cacheSize) * 0.95),
		BufferItems: 64,
		Cost: func(value string) int64 {
			return int64(1)
		},
	}, 800*time.Millisecond)
	require.NoError(t, err)

	peerID := peer.ID("gossamer")
	msg := &BlockAnnounceMessage{
		ParentHash:     common.Hash{1},
		Number:         77,
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

	time.Sleep(3 * time.Second)

	ok = msgCache.exists(peerID, msg)
	require.False(t, ok)
}
