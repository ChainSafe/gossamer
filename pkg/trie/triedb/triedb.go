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
	val, err := t.lookup(key)
	if err != nil {
		return nil
	}
	return val
}

// Internal methods
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
func (t *TrieDB) lookup(key []byte) ([]byte, error) {
	keyNibbles := nibbles.KeyLEToNibbles(key)
	return t.lookupWithoutCache(keyNibbles)
}

// lookupWithoutCache traverse nodes loading then from DB until reach the one
// we are looking for.
func (t *TrieDB) lookupWithoutCache(nibbleKey []byte) ([]byte, error) {
	// Start from root node and going downwards
	partialKey := nibbleKey
	hash := t.rootHash[:]

	// Iterates through non inlined nodes
	for {
		// Get node from DB
		nodeData, err := t.db.Get(hash)
		if err != nil {
			return nil, ErrIncompleteDB
		}

	InlinedChildrenIterator:
		for {
			// Decode node
			reader := bytes.NewReader(nodeData)
			decodedNode, err := codec.Decode(reader)
			if err != nil {
				return nil, err
			}

			var nextNode codec.MerkleValue

			switch n := decodedNode.(type) {
			case codec.Empty:
				return nil, nil
			case codec.Leaf:
				// We are in the node we were looking for
				if bytes.Equal(partialKey, n.PartialKey) {
					return t.loadValue(partialKey, n.Value)
				}
				return nil, nil
			case codec.Branch:
				nodePartialKey := n.PartialKey

				// This is unusual but could happen if for some reason one
				// branch has a hashed child node that points to a node that
				// doesn't share the prefix we are expecting
				if !bytes.HasPrefix(partialKey, nodePartialKey) {
					return nil, nil
				}

				// We are in the node we were looking for
				if bytes.Equal(partialKey, nodePartialKey) {
					if n.Value != nil {
						return t.loadValue(partialKey, n.Value)
					}
					return nil, nil
				}

				// This is not the node we were looking for but it might be in
				// one of its children
				childIdx := int(partialKey[len(nodePartialKey)])
				nextNode = n.Children[childIdx]
				if nextNode == nil {
					return nil, nil
				}

				// Advance the partial key consuming the part we already checked
				partialKey = partialKey[len(nodePartialKey)+1:]
			}

			// Next node could be inlined or hashed (pointer to a node)
			// https://spec.polkadot.network/chap-state#defn-merkle-value
			switch merkleValue := nextNode.(type) {
			case codec.HashedNode:
				// If it's hashed we set the hash to look for it in next loop
				hash = merkleValue.Data
				break InlinedChildrenIterator
			case codec.InlineNode:
				// If it is inlined we just need to decode it in the next loop
				nodeData = merkleValue.Data
			}
		}
	}
}

// loadValue gets the value from the node, if it is inlined we can return it
// directly. But if it is hashed (V1) we have to look up for its value in the DB
func (t *TrieDB) loadValue(prefix []byte, value codec.NodeValue) ([]byte, error) {
	switch v := value.(type) {
	case codec.InlineValue:
		return v.Data, nil
	case codec.HashedValue:
		prefixedKey := bytes.Join([][]byte{prefix, v.Data}, nil)
		return t.db.Get(prefixedKey)
	default:
		panic("unreachable")
	}
}

var _ trie.TrieRead = (*TrieDB)(nil)
