// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only
package triedb

import (
	"testing"

	"github.com/ChainSafe/gossamer/internal/primitives/core/hash"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime"
	"github.com/ChainSafe/gossamer/pkg/trie"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_TrieDBRawIterator(t *testing.T) {
	entries := map[string][]byte{
		"no":           make([]byte, 1),
		"noot":         make([]byte, 2),
		"not":          make([]byte, 3),
		"notable":      make([]byte, 4),
		"notification": make([]byte, 5),
		"test":         make([]byte, 6),
		"dimartiro":    make([]byte, 7),
		"bigvalue":     make([]byte, 33),
		"bigbigvalue":  make([]byte, 66),
	}

	db := NewMemoryDB()
	trieDB := NewEmptyTrieDB[hash.H256, runtime.BlakeTwo256](db)
	trieDB.SetVersion(trie.V1)

	for k, v := range entries {
		err := trieDB.Set([]byte(k), v)
		assert.NoError(t, err)
	}
	assert.NoError(t, trieDB.commit())

	t.Run("iterate_over_all_raw_items", func(t *testing.T) {
		iter, err := NewTrieDBRawIterator(trieDB)
		assert.NoError(t, err)

		i := 0
		for {
			item, err := iter.nextRawItem(true)
			assert.NoError(t, err)
			if item == nil {
				break
			}
			i++
		}
		assert.Equal(t, 13, i)
	})

	t.Run("iterate_over_all_entries", func(t *testing.T) {
		iter, err := NewTrieDBRawIterator(trieDB)
		assert.NoError(t, err)

		i := 0
		for {
			item, err := iter.NextItem()
			assert.NoError(t, err)
			if item == nil {
				break
			}
			assert.Contains(t, entries, string(item.Key))
			assert.Equal(t, item.Value, entries[string(item.Key)])
			i++
		}
		assert.Equal(t, len(entries), i)
	})

	t.Run("seek", func(t *testing.T) {
		iter, err := NewTrieDBRawIterator(trieDB)
		assert.NoError(t, err)

		found, err := iter.seek([]byte("no"), true)
		assert.NoError(t, err)
		assert.True(t, found)

		item, err := iter.NextItem()
		assert.NoError(t, err)
		assert.NotNil(t, item)
		assert.Equal(t, "no", string(item.Key))

		item, err = iter.NextItem()
		assert.NoError(t, err)
		assert.NotNil(t, item)
		assert.Equal(t, "noot", string(item.Key))
	})

	t.Run("seek_leaf", func(t *testing.T) {
		iter, err := NewTrieDBRawIterator(trieDB)
		assert.NoError(t, err)

		found, err := iter.seek([]byte("dimartiro"), true)
		assert.NoError(t, err)
		assert.True(t, found)

		item, err := iter.NextItem()
		assert.NoError(t, err)
		assert.NotNil(t, item)
		assert.Equal(t, "dimartiro", string(item.Key))
	})

	t.Run("seek_leaf_using_prefix", func(t *testing.T) {
		iter, err := NewPrefixedTrieDBRawIterator(trieDB, []byte("dimar"))
		assert.NoError(t, err)

		key, err := iter.NextKey()
		assert.NoError(t, err)
		assert.NotNil(t, key)
		assert.Equal(t, "dimartiro", string(key))

		key, err = iter.NextKey()
		assert.NoError(t, err)
		assert.Nil(t, key)

		key, err = iter.NextKey()
		assert.NoError(t, err)
		assert.Nil(t, key)
	})

	t.Run("iterate_over_all_prefixed_entries", func(t *testing.T) {
		iter, err := NewPrefixedTrieDBRawIterator(trieDB, []byte("no"))
		assert.NoError(t, err)

		i := 0
		for {
			item, err := iter.NextItem()
			assert.NoError(t, err)
			if item == nil {
				break
			}
			assert.Contains(t, entries, string(item.Key))
			assert.Equal(t, item.Value, entries[string(item.Key)])
			i++
		}
		assert.Equal(t, 5, i)
	})

	t.Run("prefixed_raw_iterator", func(t *testing.T) {
		iter, err := NewPrefixedTrieDBRawIterator(trieDB, []byte("noot"))
		assert.NoError(t, err)

		item, err := iter.NextItem()
		assert.NoError(t, err)
		assert.NotNil(t, item)
		assert.Equal(t, "noot", string(item.Key))
	})

	t.Run("iterate_over_all_prefixed_entries_then_seek", func(t *testing.T) {
		iter, err := NewPrefixedTrieDBRawIteratorThenSeek(trieDB, []byte("no"), []byte("noot"))
		assert.NoError(t, err)

		i := 0
		for {
			item, err := iter.NextItem()
			assert.NoError(t, err)
			if item == nil {
				break
			}
			assert.Contains(t, entries, string(item.Key))
			assert.Equal(t, item.Value, entries[string(item.Key)])
			i++
		}
		assert.Equal(t, 4, i)
	})
}

func TestTrieDBIterator(t *testing.T) {
	entries := map[string][]byte{
		"no":           make([]byte, 1),
		"noot":         make([]byte, 2),
		"not":          make([]byte, 3),
		"notable":      make([]byte, 4),
		"notification": make([]byte, 5),
		"test":         make([]byte, 6),
		"dimartiro":    make([]byte, 7),
		"bigvalue":     make([]byte, 33),
		"bigbigvalue":  make([]byte, 66),
	}

	db := NewMemoryDB()
	trieDB := NewEmptyTrieDB[hash.H256, runtime.BlakeTwo256](db)
	trieDB.SetVersion(trie.V1)

	for k, v := range entries {
		err := trieDB.Set([]byte(k), v)
		assert.NoError(t, err)
	}
	assert.NoError(t, trieDB.commit())

	t.Run("iterate_over_all_keys", func(t *testing.T) {
		iter, err := NewTrieDBIterator(trieDB)
		require.NoError(t, err)

		i := 0
		for {
			item, err := iter.Next()
			require.NoError(t, err)
			if item == nil {
				break
			}
			i++
		}
		require.Equal(t, len(entries), i)
	})

	t.Run("seek", func(t *testing.T) {
		iter, err := NewTrieDBIterator(trieDB)
		assert.NoError(t, err)

		err = iter.Seek([]byte("no"))
		assert.NoError(t, err)

		item, err := iter.Next()
		assert.NoError(t, err)
		assert.NotNil(t, item)
		assert.Equal(t, []byte("no"), item.Key)
		assert.Equal(t, make([]byte, 1), item.Value)

		item, err = iter.Next()
		assert.NoError(t, err)
		assert.NotNil(t, item)
		assert.Equal(t, []byte("noot"), item.Key)
		assert.Equal(t, make([]byte, 2), item.Value)
	})

	t.Run("iterate_over_all_keys_using_Seq", func(t *testing.T) {
		iter, err := NewTrieDBIterator(trieDB)
		require.NoError(t, err)

		i := 0
		for _, err := range iter.Items() {
			require.NoError(t, err)
			i++
		}
		require.Equal(t, len(entries), i)
	})

	t.Run("iterate_over_all_prefixed_entries", func(t *testing.T) {
		iter, err := NewPrefixedTrieDBIterator(trieDB, []byte("no"))
		require.NoError(t, err)

		i := 0
		for item, err := range iter.Items() {
			require.NoError(t, err)
			require.Equal(t, entries[string(item.Key)], item.Value)
			i++
		}
		require.Equal(t, 5, i)
	})

	t.Run("iterate_over_all_prefixed_entries_then_seek", func(t *testing.T) {
		iter, err := NewPrefixedTrieDBIteratorThenSeek(trieDB, []byte("no"), []byte("noot"))
		assert.NoError(t, err)

		i := 0
		for item, err := range iter.Items() {
			assert.NoError(t, err)
			require.Equal(t, entries[string(item.Key)], item.Value)
			i++
		}
		assert.Equal(t, 4, i)
	})
}

func TestTrieDBKeyIterator(t *testing.T) {
	entries := map[string][]byte{
		"no":           make([]byte, 1),
		"noot":         make([]byte, 2),
		"not":          make([]byte, 3),
		"notable":      make([]byte, 4),
		"notification": make([]byte, 5),
		"test":         make([]byte, 6),
		"dimartiro":    make([]byte, 7),
		"bigvalue":     make([]byte, 33),
		"bigbigvalue":  make([]byte, 66),
	}

	db := NewMemoryDB()
	trieDB := NewEmptyTrieDB[hash.H256, runtime.BlakeTwo256](db)
	trieDB.SetVersion(trie.V1)

	for k, v := range entries {
		err := trieDB.Set([]byte(k), v)
		assert.NoError(t, err)
	}
	assert.NoError(t, trieDB.commit())

	t.Run("iterate_over_all_keys", func(t *testing.T) {
		iter, err := NewTrieDBKeyIterator(trieDB)
		require.NoError(t, err)

		i := 0
		for {
			item, err := iter.Next()
			require.NoError(t, err)
			if item == nil {
				break
			}
			i++
		}
		require.Equal(t, len(entries), i)
	})

	t.Run("seek", func(t *testing.T) {
		iter, err := NewTrieDBKeyIterator(trieDB)
		assert.NoError(t, err)

		err = iter.Seek([]byte("no"))
		assert.NoError(t, err)

		item, err := iter.Next()
		assert.NoError(t, err)
		assert.NotNil(t, item)
		assert.Equal(t, []byte("no"), item)

		item, err = iter.Next()
		assert.NoError(t, err)
		assert.NotNil(t, item)
		assert.Equal(t, []byte("noot"), item)
	})

	t.Run("iterate_over_all_keys_using_Seq", func(t *testing.T) {
		iter, err := NewTrieDBKeyIterator(trieDB)
		require.NoError(t, err)

		i := 0
		for key, err := range iter.Items() {
			require.NoError(t, err)
			if key != nil {
				i++
			}
		}
		require.Equal(t, len(entries), i)
	})

	t.Run("iterate_over_all_prefixed_entries", func(t *testing.T) {
		iter, err := NewPrefixedTrieDBKeyIterator(trieDB, []byte("no"))
		require.NoError(t, err)

		i := 0
		for item, err := range iter.Items() {
			require.NoError(t, err)
			if item == nil {
				continue
			}
			require.Contains(t, entries, string(item))
			i++
		}
		require.Equal(t, 5, i)
	})

	t.Run("iterate_over_all_prefixed_entries_then_seek", func(t *testing.T) {
		iter, err := NewPrefixedTrieDBKeyIteratorThenSeek(trieDB, []byte("no"), []byte("noot"))
		assert.NoError(t, err)

		i := 0
		for item, err := range iter.Items() {
			assert.NoError(t, err)
			if item == nil {
				continue
			}
			require.Contains(t, entries, string(item))
			i++
		}
		assert.Equal(t, 4, i)
	})
}
