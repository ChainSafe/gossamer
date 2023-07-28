// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package triedb

import (
	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/internal/trie/hashdb"
	"github.com/ChainSafe/gossamer/internal/trie/triedb/nibble"
)

var logger = log.NewFromGlobal(log.AddContext("pkg", "trie"))

type TrieDBBuilder struct {
	db     hashdb.HashDB
	root   []byte
	layout TrieLayout
	//TODO: implement cache and recorder
}

func NewTrieDBBuilder(db hashdb.HashDB, root []byte, layout TrieLayout) *TrieDBBuilder {
	return &TrieDBBuilder{db, root, layout}
}

func (tdbb TrieDBBuilder) Build() *TrieDB {
	return &TrieDB{tdbb.db, tdbb.root, tdbb.layout}
}

type TrieDB struct {
	db     hashdb.HashDB
	root   []byte
	Layout TrieLayout
	//TODO: implement cache and recorder
}

func (tdb TrieDB) GetValue(key []byte) ([]byte, error) {
	return NewLookup(tdb.db, tdb.root).Lookup(key, nibble.NewNibbleSlice(key))
}
