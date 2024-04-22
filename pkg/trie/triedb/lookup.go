// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package triedb

import (
	"bytes"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/trie/db"
	"github.com/ChainSafe/gossamer/pkg/trie/triedb/codec"
)

type TrieLookup struct {
	// Database to query from
	db db.DBGetter
	// Hash to start at
	hash common.Hash
}

func NewTrieLookup(db db.DBGetter, hash common.Hash) TrieLookup {
	return TrieLookup{
		db:   db,
		hash: hash,
	}
}

func (l *TrieLookup) lookupNode(keyNibbles []byte) (codec.Node, error) {
	// Start from root node and going downwards
	partialKey := keyNibbles
	hash := l.hash[:]

	// Iterates through non inlined nodes
	for {
		// Get node from DB
		nodeData, err := l.db.Get(hash)
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
					return n, nil
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
						return n, nil
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

func (l *TrieLookup) lookupValue(keyNibbles []byte) ([]byte, error) {
	node, err := l.lookupNode(keyNibbles)
	if err != nil {
		return nil, err
	}

	if value := node.GetValue(); value != nil {
		return l.loadValue(keyNibbles, value)
	}

	return nil, nil
}

// loadValue gets the value from the node, if it is inlined we can return it
// directly. But if it is hashed (V1) we have to look up for its value in the DB
func (l *TrieLookup) loadValue(prefix []byte, value codec.NodeValue) ([]byte, error) {
	switch v := value.(type) {
	case codec.InlineValue:
		return v.Data, nil
	case codec.HashedValue:
		prefixedKey := bytes.Join([][]byte{prefix, v.Data}, nil)
		return l.db.Get(prefixedKey)
	default:
		panic("unreachable")
	}
}
