// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package triedb

import (
	"github.com/ChainSafe/gossamer/pkg/trie/hashdb"
	"github.com/ChainSafe/gossamer/pkg/trie/triedb/nibble"
)

type DBValue = []byte

type TrieDBBuilder[H hashdb.HashOut] struct {
	db       hashdb.HashDB[H, DBValue]
	root     H
	cache    TrieCache[H]
	recorder TrieRecorder[H]
	layout   TrieLayout[H]
}

func NewTrieDBBuilder[H hashdb.HashOut](
	db hashdb.HashDB[H, DBValue],
	root H,
	layout TrieLayout[H],
) *TrieDBBuilder[H] {
	return &TrieDBBuilder[H]{
		db:       db,
		root:     root,
		cache:    nil,
		recorder: nil,
		layout:   layout,
	}
}

func (self *TrieDBBuilder[H]) WithCache(cache TrieCache[H]) *TrieDBBuilder[H] {
	self.cache = cache
	return self
}

func (self *TrieDBBuilder[H]) WithRecorder(recorder TrieRecorder[H]) *TrieDBBuilder[H] {
	self.recorder = recorder
	return self
}

func (self *TrieDBBuilder[H]) Build() *TrieDB[H] {
	rootHandle := Hash[H]{self.root}

	return &TrieDB[H]{
		db:         self.db,
		root:       self.root,
		cache:      self.cache,
		recorder:   self.recorder,
		storage:    NewEmptyNodeStorage[H](),
		deathRow:   make(map[string]nibble.Prefix),
		rootHandle: &rootHandle,
		layout:     self.layout,
	}
}
