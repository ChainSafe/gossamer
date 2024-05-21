package triedb

import (
	"testing"

	"github.com/ChainSafe/gossamer/pkg/trie"
	"github.com/ChainSafe/gossamer/pkg/trie/triedb/db"
	"github.com/stretchr/testify/assert"
)

func TestInsertions(t *testing.T) {
	inmemoryDB := db.NewMemoryDB(make([]byte, 1))
	trieDB := NewTrieDB(trie.EmptyHash, inmemoryDB, nil)

	entries := map[string][]byte{
		"no":        []byte("no"),
		"noot":      []byte("noot"),
		"not":       make([]byte, 30),
		"a":         make([]byte, 40),
		"b":         make([]byte, 50),
		"test":      make([]byte, 60),
		"dimartiro": make([]byte, 70),
	}

	for k, v := range entries {
		err := trieDB.Put([]byte(k), v)
		assert.NoError(t, err)
	}

	for k, v := range entries {
		valueFromTrieDB := trieDB.Get([]byte(k))
		assert.Equal(t, v, valueFromTrieDB)
	}

}
