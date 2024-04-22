// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package triedb

import (
	"bytes"

	"github.com/ChainSafe/gossamer/pkg/trie/cache"
	"github.com/ChainSafe/gossamer/pkg/trie/db"
	"github.com/ChainSafe/gossamer/pkg/trie/triedb/codec"
)

type TrieBackendDB struct {
	db    db.DBGetter
	cache cache.TrieCache
}

func NewTrieBackendDB(db db.DBGetter, cache cache.TrieCache) TrieBackendDB {
	return TrieBackendDB{
		db:    db,
		cache: cache,
	}
}

func (t *TrieBackendDB) GetNode(key []byte) (codec.Node, error) {
	encodedNode, err := t.GetRawNode(key)
	if err != nil {
		return nil, err
	}

	reader := bytes.NewReader(encodedNode)
	return codec.Decode(reader)
}

func (t *TrieBackendDB) GetRawNode(key []byte) (encodedNode []byte, err error) {
	if t.cache != nil {
		encodedNode = t.cache.GetNode(key)
	}

	if encodedNode == nil {
		encodedNode, err = t.db.Get(key)
		if err != nil {
			return nil, err
		}
		if t.cache != nil {
			t.cache.SetNode(key, encodedNode)
		}
	}

	return encodedNode, nil
}

func (t *TrieBackendDB) GetValue(key []byte) (value []byte, err error) {
	if t.cache != nil {
		value = t.cache.GetValue(key)
	}

	if value == nil {
		value, err = t.db.Get(key)
		if err != nil {
			return nil, err
		}
		if t.cache != nil {
			t.cache.SetValue(key, value)
		}
	}

	return value, nil
}
