// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package triedb

import (
	"bytes"
	"errors"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/trie"
	nibbles "github.com/ChainSafe/gossamer/pkg/trie/codec"
	"github.com/ChainSafe/gossamer/pkg/trie/db"

	"github.com/ChainSafe/gossamer/pkg/trie/cache"
	"github.com/ChainSafe/gossamer/pkg/trie/triedb/codec"
)

var ErrIncompleteDB = errors.New("incomplete database")

type entry struct {
	key   []byte
	value []byte
}

// TrieDB is a DB-backed patricia merkle trie implementation
// using lazy loading to fetch nodes
type TrieDB struct {
	rootHash common.Hash
	db       TrieBackendDB
	lookup   TrieLookup
	cache    cache.TrieCache
}

// NewTrieDB creates a new TrieDB using the given root and db
func NewTrieDB(rootHash common.Hash, db db.DBGetter, cache cache.TrieCache) *TrieDB {
	backendDB := NewTrieBackendDB(db, cache)
	return &TrieDB{
		rootHash: rootHash,
		cache:    cache,
		db:       backendDB,
		lookup:   NewTrieLookup(backendDB, rootHash),
	}
}

// Hash returns the hashed root of the trie.
func (t *TrieDB) Hash() (common.Hash, error) {
	// This is trivial since it is a read only trie, but will change when we
	// support writes
	return t.rootHash, nil
}

// MustHash returns the hashed root of the trie.
// It panics if it fails to hash the root node.
func (t *TrieDB) MustHash() common.Hash {
	h, err := t.Hash()
	if err != nil {
		panic(err)
	}

	return h
}

// Get returns the value in the node of the trie
// which matches its key with the key given.
// Note the key argument is given in little Endian format.
func (t *TrieDB) Get(key []byte) []byte {
	keyNibbles := nibbles.KeyLEToNibbles(key)

	val, err := t.lookup.lookupValue(keyNibbles)
	if err != nil {
		return nil
	}

	return val
}

// Internal methods
func (t *TrieDB) loadValue(prefix []byte, value codec.NodeValue) ([]byte, error) {
	return t.lookup.loadValue(prefix, value)
}

func (t *TrieDB) getRootNode() (codec.Node, error) {
	return t.db.GetNode(t.rootHash[:])
}

// Internal methods

func (t *TrieDB) getNodeAt(key []byte) (codec.Node, error) {
	node, err := t.lookup.lookupNode(nibbles.KeyLEToNibbles(key))
	if err != nil {
		return nil, err
	}

	return node, nil
}

func (t *TrieDB) getNode(
	merkleValue codec.MerkleValue,
) (node codec.Node, err error) {
	switch n := merkleValue.(type) {
	case codec.InlineNode:
		reader := bytes.NewReader(n.Data)
		return codec.Decode(reader)
	case codec.HashedNode:
		return t.db.GetNode(n.Data)
	default: // should never happen
		panic("unreachable")
	}
}

var _ trie.TrieRead = (*TrieDB)(nil)
