// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package triedb

import (
	"github.com/ChainSafe/gossamer/pkg/trie/hashdb"
	"github.com/ChainSafe/gossamer/pkg/trie/triedb/node"
)

type DBValue = []byte

type TrieDBBuilder[Out node.HashOut] struct {
	db       hashdb.HashDB[Out, DBValue]
	root     Out
	cache    TrieCache[Out]
	recorder TrieRecorder[Out]
}

func NewTrieDBBuilder[Out node.HashOut](
	db hashdb.HashDB[Out, DBValue],
	root Out,
) *TrieDBBuilder[Out] {
	return &TrieDBBuilder[Out]{
		db:       db,
		root:     root,
		cache:    nil,
		recorder: nil,
	}
}

func (tdbb *TrieDBBuilder[Out]) WithCache(cache TrieCache[Out]) *TrieDBBuilder[Out] {
	tdbb.cache = cache
	return tdbb
}

func (tdbb *TrieDBBuilder[Out]) WithRecorder(recorder TrieRecorder[Out]) *TrieDBBuilder[Out] {
	tdbb.recorder = recorder
	return tdbb
}

func (tdbb *TrieDBBuilder[Out]) Build() *TrieDB[Out] {
	return NewTrieDB(tdbb.db, tdbb.root, tdbb.cache, tdbb.recorder)
}
