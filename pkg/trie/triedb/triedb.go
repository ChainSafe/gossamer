// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package triedb

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/trie"
	"github.com/ChainSafe/gossamer/pkg/trie/db"
	"github.com/ChainSafe/gossamer/pkg/trie/node"
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
	val, err := t.lookup(key)
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
	partial := nibbleKey
	hash := l.rootHash[:]
	keyNibbles := 0

	depth := 0

	for {
		// Get node from DB
		nodeData, err := l.db.Get(hash)

		if err != nil {
			if depth == 0 {
				return nil, ErrInvalidStateRoot
			}
			return nil, ErrIncompleteDB
		}

		// Iterates children
		for {
			// Decode node
			reader := bytes.NewReader(nodeData)
			decodedNode, err := node.Decode(reader)
			if err != nil {
				return nil, fmt.Errorf("decoding node error %s", err.Error())
			}

			// Empty Node
			if decodedNode == nil {
				return EmptyValue, nil
			}

			var nextNode *node.Node

			switch decodedNode.Kind() {
			case node.Leaf:
				// If leaf and matches return value
				if bytes.Equal(partial, decodedNode.PartialKey) {
					return l.loadValue(decodedNode.StorageValue)
				}
				return EmptyValue, nil
			// Nibbled branch
			case node.Branch:
				// Get next node
				slice := decodedNode.PartialKey

				if !bytes.HasPrefix(partial, slice) {
					return EmptyValue, nil
				}

				if len(partial) == len(slice) {
					if decodedNode.StorageValue != nil {
						return l.loadValue(decodedNode.StorageValue)
					}
				}

				nextNode = decodedNode.Children[partial[len(slice)]]
				if nextNode == nil {
					return EmptyValue, nil
				}

				partial = partial[len(slice)+1:]
				keyNibbles += len(slice) + 1
			}

			if nextNode.IsHashedValue {
				hash = nextNode.StorageValue
				break
			}

			nodeData = nextNode.StorageValue
		}
		depth++
	}
}

// TODO: change our nodes to use *NodeValue type instead of using []byte and
// stop decoding the value in the Decode method if it is a hashed reference to
// a different node
func (l *TrieDB) loadValue(value []byte) ([]byte, error) {
	// Since we are already decoding the node value when it is a reference this
	// function is trivial
	return value, nil

	// I'll leave the code below for reference regarding the right way to do it
	// But we need to change the node struct to use *NodeValue instead of []byte
	// And the node decode to not decode the value if it is a reference

	/*
		if value == nil {
			return nil, fmt.Errorf("trying to load value from nil node")
		}
		if !value.Hashed {
			return value.Data, nil
		}

		return l.db.Get(value.Data)
	*/
}

var _ trie.ReadOnlyTrie = (*TrieDB)(nil)
