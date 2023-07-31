package scraping

import (
	"github.com/ChainSafe/gossamer/dot/parachain/dispute/types"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestLRUCache(t *testing.T) {
	cache := NewLRUCache(2)

	// with
	hash1 := types.GetRandomHash()
	hash2 := types.GetRandomHash()
	hash3 := types.GetRandomHash()

	cache.Put(hash1, 1)
	cache.Put(hash2, 2)

	v := cache.Get(hash1)
	require.Equal(t, 1, v)

	v = cache.Get(hash2)
	require.Equal(t, 2, v)

	// Test updating the value of an existing key.
	cache.Put(hash1, 10)
	cache.Get(hash1)
	v = cache.Get(hash1)
	require.Equal(t, 10, v)

	// Test cache eviction when capacity is reached.
	cache.Put(hash3, 3) // This will evict key 2 (least recently used).
	v = cache.Get(hash2)
	require.Equal(t, nil, v)

	// Test retrieving non-existing keys.
	v = cache.Get(types.GetRandomHash())
	require.Equal(t, nil, v)
}
