// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package tests

import (
	"testing"

	"github.com/ChainSafe/gossamer/pkg/trie/memorydb"
	"github.com/ChainSafe/gossamer/pkg/trie/test_support/keccak_hasher"
	"github.com/ChainSafe/gossamer/pkg/trie/test_support/reference_trie"
	"github.com/ChainSafe/gossamer/pkg/trie/triedb"
	"github.com/stretchr/testify/assert"
)

type KeccakHash = keccak_hasher.KeccakHash

var hasher = keccak_hasher.NewKeccakHasher()
var V0Layout = reference_trie.LayoutV0[KeccakHash]{}

func Test_Record(t *testing.T) {
	db := memorydb.NewMemoryDB[KeccakHash](keccak_hasher.NewKeccakHasher(), memorydb.HashKey[KeccakHash])

	rootBytes := make([]byte, 32)
	root := hasher.FromBytes(rootBytes)

	{
		pairs := []struct {
			key   []byte
			value []byte
		}{
			{[]byte("dog"), []byte("cat")},
			{[]byte("lunch"), []byte("time")},
			{[]byte("notdog"), []byte("notcat")},
			{[]byte("hotdog"), []byte("hotcat")},
			{[]byte("letter"), []byte("confusion")},
			{[]byte("insert"), []byte("remove")},
			{[]byte("pirate"), []byte("aargh!")},
			{[]byte("yo ho ho"), []byte("and a bottle of rum")},
		}

		tdb := triedb.NewTrieDBBuilder[KeccakHash](db, root, V0Layout).Build()

		for _, pair := range pairs {
			_, err := tdb.Insert(pair.key, pair.value)
			assert.NoError(t, err)
		}
	}
}
