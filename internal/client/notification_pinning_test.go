// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package client

import (
	"testing"

	"github.com/ChainSafe/gossamer/internal/client/api"
	"github.com/ChainSafe/gossamer/internal/primitives/core/hash"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime"
	"github.com/stretchr/testify/require"
)

type TestUnpinBlock[H runtime.Hash] struct {
	counts map[H]uint32
}

func (tub *TestUnpinBlock[H]) PinBlock(hash H) error {
	_, ok := tub.counts[hash]
	if !ok {
		tub.counts[hash] = 0
	}
	tub.counts[hash]++
	return nil
}

func (tub *TestUnpinBlock[H]) UnpinBlock(hash H) {
	if tub.counts[hash] != 0 {
		tub.counts[hash]--
	}
}

func Test_notificationPinningWorker(t *testing.T) {
	t.Run("pin_unpin", func(t *testing.T) {
		unpinMsgChan := make(chan api.UnpinWorkerMessage[hash.H256])
		backend := TestUnpinBlock[hash.H256]{counts: make(map[hash.H256]uint32)}
		worker := newNotificationPinningWorker(unpinMsgChan, &backend)
		require.NotNil(t, worker)

		h := hash.NewRandomH256()

		err := backend.PinBlock(h)
		require.NoError(t, err)
		require.Equal(t, uint32(1), backend.counts[h])

		worker.handleAnnounceMessage(h)
		require.Equal(t, 1, worker.pinnedBlocks.Len())

		worker.handleUnpinMessage(h)
		require.Equal(t, uint32(0), backend.counts[h])
		require.Equal(t, 0, worker.pinnedBlocks.Len())

		close(unpinMsgChan)
	})

	t.Run("multiple_pins", func(t *testing.T) {
		unpinMsgChan := make(chan api.UnpinWorkerMessage[hash.H256])
		backend := TestUnpinBlock[hash.H256]{counts: make(map[hash.H256]uint32)}
		worker := newNotificationPinningWorker(unpinMsgChan, &backend)
		require.NotNil(t, worker)

		h := hash.NewRandomH256()

		// Block got pinned multiple times.
		err := backend.PinBlock(h)
		require.NoError(t, err)
		err = backend.PinBlock(h)
		require.NoError(t, err)
		err = backend.PinBlock(h)
		require.NoError(t, err)
		require.Equal(t, uint32(3), backend.counts[h])

		worker.handleAnnounceMessage(h)
		worker.handleAnnounceMessage(h)
		worker.handleAnnounceMessage(h)
		require.Equal(t, 1, worker.pinnedBlocks.Len())

		worker.handleUnpinMessage(h)
		require.Equal(t, uint32(2), backend.counts[h])
		worker.handleUnpinMessage(h)
		require.Equal(t, uint32(1), backend.counts[h])
		worker.handleUnpinMessage(h)
		require.Equal(t, uint32(0), backend.counts[h])
		require.Equal(t, 0, worker.pinnedBlocks.Len())

		close(unpinMsgChan)
	})

	t.Run("too_many_unpins", func(t *testing.T) {
		unpinMsgChan := make(chan api.UnpinWorkerMessage[hash.H256])
		backend := TestUnpinBlock[hash.H256]{counts: make(map[hash.H256]uint32)}
		worker := newNotificationPinningWorker(unpinMsgChan, &backend)
		require.NotNil(t, worker)

		h := hash.NewRandomH256()
		h2 := hash.NewRandomH256()

		// Block was announced once but unpinned multiple times. The worker should ignore the
		// additional unpins.
		err := backend.PinBlock(h)
		require.NoError(t, err)
		err = backend.PinBlock(h)
		require.NoError(t, err)
		err = backend.PinBlock(h)
		require.NoError(t, err)
		require.Equal(t, uint32(3), backend.counts[h])

		worker.handleAnnounceMessage(h)
		require.Equal(t, 1, worker.pinnedBlocks.Len())

		worker.handleUnpinMessage(h)
		require.Equal(t, uint32(2), backend.counts[h])
		worker.handleUnpinMessage(h)
		require.Equal(t, uint32(2), backend.counts[h])
		require.Equal(t, 0, worker.pinnedBlocks.Len())

		worker.handleUnpinMessage(h2)
		require.Equal(t, 0, worker.pinnedBlocks.Len())
		require.NotContains(t, h2, backend.counts)

		close(unpinMsgChan)
	})

	t.Run("should_evict_when_limit_reached", func(t *testing.T) {
		unpinMsgChan := make(chan api.UnpinWorkerMessage[hash.H256])
		backend := TestUnpinBlock[hash.H256]{counts: make(map[hash.H256]uint32)}
		worker := newNotificationPinningWorkerWithLimit(unpinMsgChan, &backend, 2)
		require.NotNil(t, worker)

		h := hash.NewRandomH256()
		h2 := hash.NewRandomH256()
		h3 := hash.NewRandomH256()
		h4 := hash.NewRandomH256()

		// Multiple blocks are announced but the cache size is too small. We expect that blocks
		// are evicted by the cache and unpinned in the backend.
		err := backend.PinBlock(h)
		require.NoError(t, err)
		err = backend.PinBlock(h2)
		require.NoError(t, err)
		err = backend.PinBlock(h3)
		require.NoError(t, err)
		require.Equal(t, uint32(1), backend.counts[h])
		require.Equal(t, uint32(1), backend.counts[h2])
		require.Equal(t, uint32(1), backend.counts[h3])

		worker.handleAnnounceMessage(h)
		_, ok := worker.pinnedBlocks.Peek(h)
		require.True(t, ok)
		worker.handleAnnounceMessage(h2)
		_, ok = worker.pinnedBlocks.Peek(h2)
		require.True(t, ok)
		worker.handleAnnounceMessage(h3)
		_, ok = worker.pinnedBlocks.Peek(h2)
		require.True(t, ok)
		_, ok = worker.pinnedBlocks.Peek(h3)
		require.True(t, ok)
		require.Equal(t, 2, worker.pinnedBlocks.Len())

		// Hash 1 should have gotten unpinned, since its oldest.
		require.Equal(t, uint32(0), backend.counts[h])
		require.Equal(t, uint32(1), backend.counts[h2])
		require.Equal(t, uint32(1), backend.counts[h3])

		// Hash 2 is getting bumped.
		worker.handleAnnounceMessage(h2)
		_, ok = worker.pinnedBlocks.Peek(h2)
		require.True(t, ok)

		// Since hash 2 was accessed, evict hash 3.
		worker.handleAnnounceMessage(h4)
		_, ok = worker.pinnedBlocks.Peek(h4)
		require.True(t, ok)
		_, ok = worker.pinnedBlocks.Peek(h2)
		require.True(t, ok)
		_, ok = worker.pinnedBlocks.Peek(h3)
		require.False(t, ok)

		close(unpinMsgChan)
	})
}
