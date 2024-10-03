// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package triedb

import (
	"testing"

	"github.com/ChainSafe/gossamer/internal/primitives/core/hash"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime"
	"github.com/ChainSafe/gossamer/pkg/trie"
	"github.com/stretchr/testify/assert"
)

func Test_rawIterator(t *testing.T) {
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

	db := NewMemoryDB[hash.H256, runtime.BlakeTwo256](EmptyNode)
	trieDB := NewEmptyTrieDB[hash.H256, runtime.BlakeTwo256](db)
	trieDB.SetVersion(trie.V1)

	for k, v := range entries {
		err := trieDB.Put([]byte(k), v)
		assert.NoError(t, err)
	}
	assert.NoError(t, trieDB.commit())

	t.Run("iterate_over_all_raw_items", func(t *testing.T) {
		iter, err := newRawIterator(trieDB)
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
		iter, err := newRawIterator(trieDB)
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
		iter, err := newRawIterator(trieDB)
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
		iter, err := newRawIterator(trieDB)
		assert.NoError(t, err)

		found, err := iter.seek([]byte("dimartiro"), true)
		assert.NoError(t, err)
		assert.True(t, found)

		item, err := iter.NextItem()
		assert.NoError(t, err)
		assert.NotNil(t, item)
		assert.Equal(t, "dimartiro", string(item.Key))
	})

	t.Run("iterate_over_all_prefixed_entries", func(t *testing.T) {
		iter, err := newPrefixedRawIterator(trieDB, []byte("no"))
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
		iter, err := newPrefixedRawIterator(trieDB, []byte("noot"))
		assert.NoError(t, err)

		item, err := iter.NextItem()
		assert.NoError(t, err)
		assert.NotNil(t, item)
		assert.Equal(t, "noot", string(item.Key))
	})

	t.Run("iterate_over_all_prefixed_entries_then_seek", func(t *testing.T) {
		iter, err := newPrefixedRawIteratorThenSeek(trieDB, []byte("no"), []byte("noot"))
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
