// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package triedb

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/trie"
	nibbles "github.com/ChainSafe/gossamer/pkg/trie/codec"
	"github.com/ChainSafe/gossamer/pkg/trie/db"

	"github.com/ChainSafe/gossamer/pkg/trie/triedb/codec"
)

var ErrInvalidStateRoot = errors.New("invalid state root")
var ErrIncompleteDB = errors.New("incomplete database")

var EmptyValue = []byte{}

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

// MustHash returns the hashed root of the trie.
func (t *TrieDB) Hash() (common.Hash, error) {
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
	val, err := t.lookup(keyNibbles)
	if err != nil {
		// TODO: do we have to do anything else? maybe change the signature
		// to return an error?
		return nil
	}
	return val
}

// GetKeysWithPrefix returns all keys in little Endian
// format from nodes in the trie that have the given little
// Endian formatted prefix in their key.
func (t *TrieDB) GetKeysWithPrefix(prefix []byte) (keysLE [][]byte) {
	panic("not implemented yet")
}

// Internal methods

func (l *TrieDB) lookup(nibbleKey []byte) ([]byte, error) {
	return l.lookupWithoutCache(nibbleKey)
}

// lookupWithoutCache traverse nodes loading then from DB until reach the one
// we are looking for.
func (l *TrieDB) lookupWithoutCache(nibbleKey []byte) ([]byte, error) {
	partialKey := nibbleKey
	hash := l.rootHash[:]

	depth := 0

	// Iterates through non inlined nodes
	for {
		// Get node from DB
		nodeData, err := l.db.Get(hash)

		if err != nil {
			if depth == 0 {
				return nil, ErrInvalidStateRoot
			}
			return nil, ErrIncompleteDB
		}

	childrenIterator:
		for {
			// Decode node
			reader := bytes.NewReader(nodeData)
			decodedNode, err := codec.Decode(reader)
			if err != nil {
				return nil, fmt.Errorf("decoding node error %s", err.Error())
			}

			// Empty Node
			if decodedNode == nil {
				return EmptyValue, nil
			}

			var nextNode codec.MerkleValue

			switch n := decodedNode.(type) {
			case codec.Leaf:
				// If leaf and matches return value
				if bytes.Equal(partialKey, n.PartialKey) {
					return l.loadValue(n.Value)
				}
				return EmptyValue, nil
			// Nibbled branch
			case codec.Branch:
				// Get next node
				nodePartialKey := n.PartialKey

				if !bytes.HasPrefix(partialKey, nodePartialKey) {
					return EmptyValue, nil
				}

				if bytes.Equal(partialKey, nodePartialKey) {
					if n.Value != nil {
						return l.loadValue(n.Value)
					}
				}

				childIdx := int(partialKey[len(nodePartialKey)])
				nextNode = n.Children[childIdx]
				if nextNode == nil {
					return EmptyValue, nil
				}

				partialKey = partialKey[len(nodePartialKey)+1:]
			}

			switch merkleValue := nextNode.(type) {
			case codec.HashedNode:
				hash = merkleValue.Data
				break childrenIterator
			case codec.InlineNode:
				nodeData = merkleValue.Data
			}
		}
		depth++
	}
}

func (l *TrieDB) loadValue(value codec.NodeValue) ([]byte, error) {
	if value == nil {
		return nil, fmt.Errorf("trying to load value from nil node")
	}

	switch v := value.(type) {
	case codec.InlineValue:
		return v.Data, nil
	case codec.HashedValue:
		return l.db.Get(v.Data)
	default:
		panic("unreachable")
	}
}

var _ trie.ReadOnlyTrie = (*TrieDB)(nil)
