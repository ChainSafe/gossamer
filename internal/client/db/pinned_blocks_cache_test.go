// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package db

import (
	"testing"

	"github.com/ChainSafe/gossamer/internal/primitives/runtime"
	"github.com/stretchr/testify/assert"
)

func TestPinnedBlocksCache(t *testing.T) {
	cache := newPinnedBlocksCache[uint, noopExtrinsic]()
	cache.Pin(1)
	value, ok := cache.cache.Peek(1)
	assert.True(t, ok)
	assert.Equal(t, pinnedBlocksCacheEntry[noopExtrinsic]{refCount: 1}, *value)

	assert.True(t, cache.Contains(1))

	cache.Pin(1)
	value, ok = cache.cache.Peek(1)
	assert.True(t, ok)
	assert.Equal(t, pinnedBlocksCacheEntry[noopExtrinsic]{refCount: 2}, *value)

	cache.Pin(1)
	value, ok = cache.cache.Peek(1)
	assert.True(t, ok)
	assert.Equal(t, pinnedBlocksCacheEntry[noopExtrinsic]{refCount: 3}, *value)

	cache.InsertBody(1, []noopExtrinsic{})
	value, ok = cache.cache.Peek(1)
	assert.True(t, ok)
	body := []noopExtrinsic{}
	assert.Equal(t, pinnedBlocksCacheEntry[noopExtrinsic]{refCount: 3, Body: &body}, *value)

	cache.InsertJustifications(1, runtime.Justifications{{
		ConsensusEngineID: runtime.ConsensusEngineID{1, 1, 1, 1},
	}})
	value, ok = cache.cache.Peek(1)
	assert.True(t, ok)
	assert.Equal(t, pinnedBlocksCacheEntry[noopExtrinsic]{
		refCount: 3,
		Body:     &body,
		Justifications: &runtime.Justifications{{
			ConsensusEngineID: runtime.ConsensusEngineID{1, 1, 1, 1},
		}},
	}, *value)

	cache.Unpin(1)
	value, ok = cache.cache.Peek(1)
	assert.True(t, ok)
	assert.Equal(t, pinnedBlocksCacheEntry[noopExtrinsic]{
		refCount: 2,
		Body:     &body,
		Justifications: &runtime.Justifications{{
			ConsensusEngineID: runtime.ConsensusEngineID{1, 1, 1, 1},
		}},
	}, *value)

	cache.Unpin(1)
	value, ok = cache.cache.Peek(1)
	assert.True(t, ok)
	assert.Equal(t, pinnedBlocksCacheEntry[noopExtrinsic]{
		refCount: 1,
		Body:     &body,
		Justifications: &runtime.Justifications{{
			ConsensusEngineID: runtime.ConsensusEngineID{1, 1, 1, 1},
		}},
	}, *value)

	cache.Unpin(1)
	value, ok = cache.cache.Peek(1)
	assert.False(t, ok)
	assert.Nil(t, value)
}
