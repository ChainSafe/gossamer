// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package triedb

import (
	"testing"

	"github.com/ChainSafe/gossamer/pkg/trie"
	"github.com/ChainSafe/gossamer/pkg/trie/triedb/db"
	"github.com/stretchr/testify/assert"
)

func TestWrites(t *testing.T) {
	inmemoryDB := db.NewMemoryDB(make([]byte, 1))
	trieDB := NewTrieDB(trie.EmptyHash, inmemoryDB, nil)

	entries := map[string][]byte{
		"no":        []byte("noValue"),
		"noot":      []byte("nootValue"),
		"not":       []byte("notValue"),
		"a":         []byte("aValue"),
		"b":         []byte("bValue"),
		"test":      []byte("testValue"),
		"dimartiro": []byte("dimartiroValue"),
	}

	t.Run("inserts are successful", func(t *testing.T) {
		for k, v := range entries {
			err := trieDB.Put([]byte(k), v)
			assert.NoError(t, err)
		}

		for k, v := range entries {
			valueFromTrieDB := trieDB.Get([]byte(k))
			assert.Equal(t, v, valueFromTrieDB)
		}
	})

	t.Run("delete leaf ok", func(t *testing.T) {

	})

}

func TestDeletes(t *testing.T) {
	inmemoryDB := db.NewMemoryDB(make([]byte, 1))
	trieDB := NewTrieDB(trie.EmptyHash, inmemoryDB, nil)

	entries := map[string][]byte{
		"no":   []byte("noValue"),
		"not":  []byte("notValue"),
		"note": []byte("noteValue"),
		"a":    []byte("aValue"),
	}

	for k, v := range entries {
		err := trieDB.Put([]byte(k), v)
		assert.NoError(t, err)
	}

	err := trieDB.Delete([]byte("not"))
	assert.NoError(t, err)

	valueFromTrieDB := trieDB.Get([]byte("not"))
	assert.Nil(t, valueFromTrieDB)
}
