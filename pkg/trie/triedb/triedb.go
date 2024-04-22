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
	db       db.DBGetter
}

// NewTrieDB creates a new TrieDB using the given root and db
func NewTrieDB(rootHash common.Hash, db db.DBGetter) *TrieDB {
	return &TrieDB{
		rootHash: rootHash,
		db:       db,
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

	lookup := NewTrieLookup(t.db, t.rootHash)
	val, err := lookup.lookupValue(keyNibbles)
	if err != nil {
		return nil
	}
	return val
}

// Internal methods
func (t *TrieDB) loadValue(prefix []byte, value codec.NodeValue) ([]byte, error) {
	lookup := NewTrieLookup(t.db, t.rootHash)
	return lookup.loadValue(prefix, value)
}

func (t *TrieDB) getRootNode() (codec.Node, error) {
	nodeData, err := t.db.Get(t.rootHash[:])
	if err != nil {
		return nil, ErrIncompleteDB
	}

	reader := bytes.NewReader(nodeData)
	decodedNode, err := codec.Decode(reader)
	if err != nil {
		return nil, err
	}

	return decodedNode, nil
}

func (t *TrieDB) getNodeAt(key []byte) (codec.Node, error) {
	lookup := NewTrieLookup(t.db, t.rootHash)
	node, err := lookup.lookupNode(nibbles.KeyLEToNibbles(key))
	if err != nil {
		return nil, err
	}

	return node, nil
}

func (t *TrieDB) getNode(
	merkleValue codec.MerkleValue,
) (node codec.Node, err error) {
	nodeData := []byte{}

	switch n := merkleValue.(type) {
	case codec.InlineNode:
		nodeData = n.Data
	case codec.HashedNode:
		nodeData, err = t.db.Get(n.Data)
		if err != nil {
			return nil, ErrIncompleteDB
		}
	}

	reader := bytes.NewReader(nodeData)
	node, err = codec.Decode(reader)
	if err != nil {
		return nil, err
	}

	return node, err
}

var _ trie.TrieRead = (*TrieDB)(nil)
