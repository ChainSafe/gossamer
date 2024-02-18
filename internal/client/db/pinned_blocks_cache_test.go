package db

import (
	"testing"

	"github.com/ChainSafe/gossamer/internal/primitives/runtime"
	"github.com/stretchr/testify/assert"
)

func TestPinnedBlocksCache(t *testing.T) {
	cache := newPinnedBlocksCache[uint]()
	cache.Pin(1)
	value, ok := cache.cache.Peek(1)
	assert.True(t, ok)
	assert.Equal(t, &pinnedBlocksCacheEntry{refCount: 1}, value)

	cache.Pin(1)
	value, ok = cache.cache.Peek(1)
	assert.True(t, ok)
	assert.Equal(t, &pinnedBlocksCacheEntry{refCount: 2}, value)

	cache.Pin(1)
	value, ok = cache.cache.Peek(1)
	assert.True(t, ok)
	assert.Equal(t, &pinnedBlocksCacheEntry{refCount: 3}, value)

	cache.InsertBody(1, &[]runtime.Extrinsic{})
	value, ok = cache.cache.Peek(1)
	assert.True(t, ok)
	body := []runtime.Extrinsic{}
	assert.Equal(t, &pinnedBlocksCacheEntry{refCount: 3, Body: &body}, value)

	cache.InsertJustifications(1, &runtime.Justifications{{
		ConsensusEngineID: runtime.ConsensusEngineID{1, 1, 1, 1},
	}})
	value, ok = cache.cache.Peek(1)
	assert.True(t, ok)
	assert.Equal(t, &pinnedBlocksCacheEntry{
		refCount: 3,
		Body:     &body,
		Justifications: &runtime.Justifications{{
			ConsensusEngineID: runtime.ConsensusEngineID{1, 1, 1, 1},
		}},
	}, value)

	cache.Unpin(1)
	value, ok = cache.cache.Peek(1)
	assert.True(t, ok)
	assert.Equal(t, &pinnedBlocksCacheEntry{
		refCount: 2,
		Body:     &body,
		Justifications: &runtime.Justifications{{
			ConsensusEngineID: runtime.ConsensusEngineID{1, 1, 1, 1},
		}},
	}, value)

	cache.Unpin(1)
	value, ok = cache.cache.Peek(1)
	assert.True(t, ok)
	assert.Equal(t, &pinnedBlocksCacheEntry{
		refCount: 1,
		Body:     &body,
		Justifications: &runtime.Justifications{{
			ConsensusEngineID: runtime.ConsensusEngineID{1, 1, 1, 1},
		}},
	}, value)

	cache.Unpin(1)
	value, ok = cache.cache.Peek(1)
	assert.False(t, ok)
}
