package tests

import (
	"testing"

	"github.com/ChainSafe/gossamer/internal/database"
	"github.com/ChainSafe/gossamer/pkg/trie/inmemory"
	"github.com/ChainSafe/gossamer/pkg/trie/triedb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestDB(t *testing.T) database.Table {
	db, err := database.NewPebble("", true)
	require.NoError(t, err)
	return database.NewTable(db, "trie")
}

func TestTrieDB_Get(t *testing.T) {
	db := newTestDB(t)

	entries := map[string]string{
		"no":  "no",
		"not": "not",
		//"nothing": "nothing",
		//"test":    "test",
	}

	inMemoryTrie := inmemory.NewEmptyTrie()

	for k, v := range entries {
		inMemoryTrie.Put([]byte(k), []byte(v))
	}

	err := inMemoryTrie.WriteDirty(db)
	assert.NoError(t, err)

	root, err := inMemoryTrie.Hash()
	assert.NoError(t, err)

	trieDB := triedb.NewTrieDB(root, db)

	//t.Log("trie", inMemoryTrie.String())

	for k, v := range entries {
		value := trieDB.Get([]byte(k))
		assert.Equal(t, []byte(v), value)
	}
}
