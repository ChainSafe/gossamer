package triedb

import "github.com/ChainSafe/gossamer/pkg/trie/triedb/hash"

// The values cached by [TrieCache].
type CachedValues[H any] interface {
	NonExistingCachedValue[H] | ExistingHashCachedValue[H] | ExistingCachedValue[H]
	CachedValue[H]
}

// A value cached by [TrieCache].
type CachedValue[H any] interface {
	data() []byte
	hash() *H
}

// Constructor for [CachedValue]
func NewCachedValue[H any, CV CachedValues[H]](cv CV) CachedValue[H] {
	return cv
}

// The value doesn't exist in the trie.
type NonExistingCachedValue[H any] struct{}

func (NonExistingCachedValue[H]) data() []byte { return nil } //nolint:unused
func (NonExistingCachedValue[H]) hash() *H     { return nil } //nolint:unused

// The hash is cached and not the data because it was not accessed.
type ExistingHashCachedValue[H any] struct {
	Hash H
}

func (ExistingHashCachedValue[H]) data() []byte  { return nil }        //nolint:unused
func (ehcv ExistingHashCachedValue[H]) hash() *H { return &ehcv.Hash } //nolint:unused

// The value exists in the trie.
type ExistingCachedValue[H any] struct {
	// The hash of the value.
	Hash H
	// The actual data of the value.
	Data []byte
}

func (ecv ExistingCachedValue[H]) data() []byte { return ecv.Data }  //nolint:unused
func (ecv ExistingCachedValue[H]) hash() *H     { return &ecv.Hash } //nolint:unused

// A cache that can be used to speed-up certain operations when accessing [TrieDB].
//
// For every lookup in the trie, every node is always fetched and decoded on the fly. Fetching and
// decoding a node always takes some time and can kill the performance of any application that is
// doing quite a lot of trie lookups. To circumvent this performance degradation, a cache can be
// used when looking up something in the trie. Any cache that should be used with the [TrieDB]
// needs to implement this interface.
//
// The interface consists of two cache levels, first the trie nodes cache and then the value cache.
// The trie nodes cache, as the name indicates, is for caching trie nodes as [NodeOwned]. These
// trie nodes are referenced by their hash. The value cache is caching [CachedValue]'s and these
// are referenced by the key to look them up in the trie. As multiple different tries can have
// different values under the same key, it up to the cache implementation to ensure that the
// correct value is returned. As each trie has a different root, this root can be used to
// differentiate values under the same key.
type TrieCache[H hash.Hash] interface {
	// Lookup value for the given key.
	// Returns the nil if the key is unknown or otherwise the value is returned
	// [TrieCache.SetValue] is used to make the cache aware of data that is associated
	// to a key.
	//
	// NOTE: The cache can be used for different tries, aka with different roots. This means
	// that the cache implementation needs to take care of always returning the correct value
	// for the current trie root.
	GetValue(key []byte) CachedValue[H]
	// Cache the given value for the given key.
	//
	// NOTE: The cache can be used for different tries, aka with different roots. This means
	// that the cache implementation needs to take care of caching values for the current
	// trie root.
	SetValue(key []byte, value CachedValue[H])

	// Get or insert a [NodeOwned].
	// The cache implementation should look up based on the given hash if the node is already
	// known. If the node is not yet known, the given fetchNode function can be used to fetch
	// the particular node.
	// Returns the [NodeOwned] or an error that happened on fetching the node.
	GetOrInsertNode(hash H, fetchNode func() (NodeOwned[H], error)) (NodeOwned[H], error)

	// Get the [NodeOwned] that corresponds to the given hash.
	GetNode(hash H) NodeOwned[H]
}
