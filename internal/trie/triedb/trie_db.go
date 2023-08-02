// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package triedb

import (
	"github.com/ChainSafe/gossamer/internal/trie/hashdb"
	"github.com/ChainSafe/gossamer/internal/trie/triedb/nibble"
)

type TrieDBBuilder struct {
	db   hashdb.HashDB
	root []byte
	//TODO: implement cache and recorder
}

func NewTrieDBBuilder(db hashdb.HashDB, root []byte) *TrieDBBuilder {
	return &TrieDBBuilder{db, root}
}

func (tdbb TrieDBBuilder) Build() *TrieDB {
	return &TrieDB{tdbb.db, tdbb.root}
}

type TrieDB struct {
	db   hashdb.HashDB
	root []byte
	//TODO: implement cache and recorder
}

func (tdb TrieDB) GetValue(key []byte) ([]byte, error) {
	return NewLookup(tdb.db, tdb.root).Lookup(nibble.NewNibbleSlice(key))
}
