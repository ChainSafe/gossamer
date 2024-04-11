package triedb

import (
	"testing"

	"github.com/ChainSafe/gossamer/pkg/trie"
	"github.com/ChainSafe/gossamer/pkg/trie/inmemory"
	"github.com/stretchr/testify/assert"
)

func TestIterator(t *testing.T) {
	db := newTestDB(t)
	inMemoryTrie := inmemory.NewEmptyTrie()
	inMemoryTrie.SetVersion(trie.V1)

	entries := map[string][]byte{
		"no":           make([]byte, 1),
		"noot":         make([]byte, 2),
		"not":          make([]byte, 3),
		"notable":      make([]byte, 4),
		"notification": make([]byte, 5),
		"test":         make([]byte, 6),
		"dimartiro":    make([]byte, 7),
	}

	for k, v := range entries {
		inMemoryTrie.Put([]byte(k), v)
	}

	err := inMemoryTrie.WriteDirty(db)
	assert.NoError(t, err)

	root, err := inMemoryTrie.Hash()
	assert.NoError(t, err)

	trieDB := NewTrieDB(root, db)
	t.Run("iterate_over_all_entries", func(t *testing.T) {
		iter, err := NewTrieDBIterator(trieDB)
		assert.NoError(t, err)

		expected := inMemoryTrie.NextKey([]byte{})
		i := 0
		for key := iter.NextKey(); key != nil; key = iter.NextKey() {
			assert.Equal(t, expected, key)
			expected = inMemoryTrie.NextKey(expected)
			i++
		}

		assert.Equal(t, len(entries), i)
	})

	t.Run("iterate_from_given_key", func(t *testing.T) {
		iter, err := NewTrieDBIterator(trieDB)
		assert.NoError(t, err)

		iter.Seek([]byte("not"))

		expected := inMemoryTrie.NextKey([]byte("not"))
		actual := iter.NextKey()

		assert.Equal(t, expected, actual)
	})
}
