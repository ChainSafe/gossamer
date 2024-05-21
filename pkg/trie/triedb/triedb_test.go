// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

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
		"not":       []byte("not"),
		"a":         []byte("a"),
		"b":         []byte("b"),
		"test":      []byte("test"),
		"dimartiro": []byte("dimartiro"),
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
